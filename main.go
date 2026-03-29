package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
)

type Config struct {
	DeviceName             string
	CECAdapter             string
	Debug                  bool
	KeyMapOverrides        map[string][]int
	NoPowerEvents          bool
	PowerDevices           []int
	ConnectionRetries      int
	QueueDir               string
	RestartRetries         int
	SetActiveSource        bool
	ActiveSourceDeviceType int
}

func setupLogger(debug bool) {
	var lvl slog.Level
	if debug {
		lvl = slog.LevelDebug
	} else {
		lvl = slog.LevelInfo
	}
	// Remove timestamp from logs, it's not very useful since systemd already adds it
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		}})
	slog.SetDefault(slog.New(handler))
}

func runController(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		return err
	}

	if err := validateConfig(cfg); err != nil {
		slog.Error("Invalid configuration", "error", err)
		return err
	}

	setupLogger(cfg.Debug)

	slog.Info("Starting cec-controller", "config", cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	queue, err := NewQueue(ctx, cfg.QueueDir)
	if err != nil {
		slog.Error("Failed to initialize event queue", "dir", cfg.QueueDir, "error", err)
		return err
	}
	defer queue.Close()

	c, err := NewCEC(cfg.CECAdapter, cfg.DeviceName, cfg.ConnectionRetries, queue.InKeyEvents)
	if err != nil {
		slog.Error("Failed to open CEC, you can specify a cec-adapter since auto-detect does not work", "cec-adapter", cfg.CECAdapter, "error", err)
		return err
	}
	defer c.Close()

	keyMapObj, err := NewKeyMap(cfg.KeyMapOverrides)
	if err != nil {
		slog.Error("Failed to initialize virtual keyboard", "error", err)
		return err
	}

	// Claim active source on startup so the TV switches input to this device.
	if cfg.SetActiveSource {
		if !c.SetActiveSource(cfg.ActiveSourceDeviceType) {
			slog.Warn("Failed to set active source on startup")
		} else {
			slog.Info("Active source set", "deviceType", cfg.ActiveSourceDeviceType)
		}
	}

	// Open a D-Bus connection for logind inhibitor locks (sleep/shutdown protection).
	// Non-fatal: if unavailable, CEC commands run without holding a delay lock.
	var dbusConn, dbusErr = openSystemBus()
	if dbusErr != nil {
		slog.Warn("Failed to connect to D-Bus, inhibitor locks will be skipped", "error", dbusErr)
		dbusConn = nil
	}

	if !cfg.NoPowerEvents {
		// Send an initial PowerOn so devices wake up when this service starts.
		queue.InPowerEvents <- PowerEvent{Type: PowerOn, Active: true}
		if err := PowerEventListener(ctx, queue.InPowerEvents); err != nil {
			slog.Error("Failed to start power event listener", "error", err)
			return err
		}
	}

	slog.Info("Listening for CEC key and power events... (Ctrl+C to exit)")
	for {
		select {
		case kp := <-queue.OutKeyEvents:
			if kp == nil || kp.Duration != 0 {
				continue
			}
			keyMapObj.OnKeyPress(kp.KeyCode)
		case ev := <-queue.OutPowerEvents:
			var err error
			switch ev.Type {
			case PowerOn, PowerResume:
				slog.Info("Powering on devices", "devices", cfg.PowerDevices)
				err = c.PowerOn(cfg.PowerDevices...)
			case PowerSleep, PowerShutdown:
				slog.Info("Putting devices to standby", "devices", cfg.PowerDevices)
				// Hold a logind delay inhibitor so the system waits for CEC
				// standby to complete before proceeding with sleep/shutdown.
				lock, lockErr := acquireInhibitor(dbusConn, "sleep:shutdown", "Sending CEC standby command")
				if lockErr != nil {
					slog.Warn("Failed to acquire inhibitor lock", "error", lockErr)
				}
				err = c.Standby(cfg.PowerDevices...)
				lock.Release()
			}
			if err != nil {
				slog.Warn("Failed to send power command after connection reopen, libcec is weird so we need to restart the current process...")
				cancel()
				if !queue.RestartProcess(cfg.RestartRetries) {
					slog.Error("Process restart failed or no retries left, exiting")
					return fmt.Errorf("too many restarts")
				}
			}
		case <-ctx.Done():
			slog.Info("Shutting down...")
			return nil
		}
	}
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "cec-controller",
		Short: "HDMI-CEC controller for Linux",
		Long: `CEC Controller is a Linux CLI application that listens for HDMI-CEC key events
and translates them to Linux virtual keyboard actions. It also reacts to system
power events (startup, shutdown, sleep, resume).`,
		RunE: runController,
	}

	rootCmd.Flags().String("cec-adapter", "", "CEC adapter path (leave empty for auto-detect)")
	rootCmd.Flags().String("device-name", "", "Device name shown on your TV (leave empty for hostname)")
	rootCmd.Flags().Bool("debug", false, "Enable debug output")
	rootCmd.Flags().Bool("no-power-events", false, "Disable power event handling")
	rootCmd.Flags().Int("retries", 5, "Number of times to retry opening the CEC adapter on failure (each attempt may take up to 10s)")
	rootCmd.Flags().StringSlice("keymap", []string{}, "Custom CEC-to-Linux key mapping (format <cec>:<linux>, e.g. --keymap 1:105)")
	rootCmd.Flags().StringSlice("devices", []string{}, "Power event device addresses (e.g. --devices 0,1). Defaults to 0.")
	rootCmd.Flags().String("queue-dir", "", "Directory for event queue (defaults to temp directory)")
	rootCmd.Flags().Int("restart-retries", 3, "Maximum number of process restarts when the CEC library gets stuck (0 disables restart)")
	rootCmd.Flags().Bool("set-active-source", false, "Claim active source on startup so the TV switches input to this device")
	rootCmd.Flags().Int("active-source-type", CECDeviceTypePlayback, "CEC device type for active source claim (0=TV 1=Recording 3=Tuner 4=Playback 5=AudioSystem)")

	mustBind := func(key, flag string) {
		if err := viper.BindPFlag(key, rootCmd.Flags().Lookup(flag)); err != nil {
			slog.Warn("Failed to bind flag", "key", key, "flag", flag, "error", err)
		}
	}
	mustBind("cec-adapter", "cec-adapter")
	mustBind("device-name", "device-name")
	mustBind("debug", "debug")
	mustBind("no-power-events", "no-power-events")
	mustBind("retries", "retries")
	mustBind("keymap", "keymap")
	mustBind("devices", "devices")
	mustBind("queue-dir", "queue-dir")
	mustBind("restart-retries", "restart-retries")
	mustBind("set-active-source", "set-active-source")
	mustBind("active-source-type", "active-source-type")

	// Hidden subcommand to generate man pages into a target directory.
	// Usage: cec-controller generate-docs --output-dir /usr/share/man/man1
	var outputDir string
	generateDocsCmd := &cobra.Command{
		Use:    "generate-docs",
		Short:  "Generate man pages for cec-controller",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
			header := &doc.GenManHeader{
				Title:   "CEC-CONTROLLER",
				Section: "1",
				Source:  "cec-controller",
				Manual:  "General Commands Manual",
			}
			return doc.GenManTree(rootCmd, header, outputDir)
		},
	}
	generateDocsCmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory to write man pages into")
	rootCmd.AddCommand(generateDocsCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
