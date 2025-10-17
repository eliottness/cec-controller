package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	DeviceName        string
	CECAdapter        string
	Debug             bool
	KeyMapOverrides   map[string][]int
	NoPowerEvents     bool
	PowerDevices      []int
	ConnectionRetries int
	QueueDir          string
}

func setupLogger(debug bool) {
	var lvl slog.Level
	if debug {
		lvl = slog.LevelDebug
	} else {
		lvl = slog.LevelInfo
	}
	// Remove timestamp from logs, it's not very useful since systemd already adds it
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl, ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey && len(groups) == 0 {
			return slog.Attr{}
		}
		return a
	}})
	slog.SetDefault(slog.New(handler))
}

func runController(cmd *cobra.Command, args []string) error {
	// Load configuration from file first
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
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

	// Create KeyMap object
	keyMapObj, err := NewKeyMap(cfg.KeyMapOverrides)
	if err != nil {
		slog.Error("Failed to initialize virtual keyboard", "error", err)
		return err
	}

	c, err := NewCEC(cfg.CECAdapter, cfg.DeviceName, cfg.ConnectionRetries, queue.InKeyEvents)
	if err != nil {
		slog.Error("Failed to open CEC", "error", err)
		return err
	}
	defer c.Close()

	if !cfg.NoPowerEvents {
		// cec-controller just started alongside the system, so we assume the system has to be powered on
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
				// Ignore key release events
				continue
			}
			keyMapObj.OnKeyPress(kp.KeyCode)
		case ev := <-queue.OutPowerEvents:
			switch ev.Type {
			case PowerOn, PowerResume:
				slog.Info("Powering on devices", "devices", cfg.PowerDevices)
				err = c.PowerOn(cfg.PowerDevices...)
			case PowerSleep, PowerShutdown:
				slog.Info("Putting devices to standby", "devices", cfg.PowerDevices)
				err = c.Standby(cfg.PowerDevices...)
			}
			if err != nil {
				slog.Warn("Failed to send power command after connection reopen, libcec is wierd so we need to restart the current process...")
				cancel()
				queue.RestartProcess()
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

	// Define flags that bind to viper config
	rootCmd.Flags().String("cec-adapter", "", "CEC adapter path (leave empty for auto-detect)")
	rootCmd.Flags().String("device-name", "", "Device name shown on your TV (leave empty for hostname)")
	rootCmd.Flags().Bool("debug", false, "Enable debug output")
	rootCmd.Flags().Bool("no-power-events", false, "Disable power event handling")
	rootCmd.Flags().Int("retries", 5, "Number of times to retry CEC connection on failure")
	rootCmd.Flags().StringSlice("keymap", []string{}, "Custom CEC-to-Linux key mapping (format <cec>:<linux>, e.g. --keymap 1:105)")
	rootCmd.Flags().StringSlice("devices", []string{}, "Power event device addresses (e.g. --devices 0,1). If not specified, power events are disabled")
	rootCmd.Flags().String("queue-dir", "", "Directory for event queue (defaults to temp directory)")

	// Bind flags to viper
	viper.BindPFlag("cec-adapter", rootCmd.Flags().Lookup("cec-adapter"))
	viper.BindPFlag("device-name", rootCmd.Flags().Lookup("device-name"))
	viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug"))
	viper.BindPFlag("no-power-events", rootCmd.Flags().Lookup("no-power-events"))
	viper.BindPFlag("retries", rootCmd.Flags().Lookup("retries"))
	viper.BindPFlag("keymap", rootCmd.Flags().Lookup("keymap"))
	viper.BindPFlag("devices", rootCmd.Flags().Lookup("devices"))
	viper.BindPFlag("queue-dir", rootCmd.Flags().Lookup("queue-dir"))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
