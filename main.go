package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
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
	NoVolumeSync      bool
	VolumeStep        int
	AudioDevice       int
}

type multiFlag []string

func (m *multiFlag) String() string         { return strings.Join(*m, ",") }
func (m *multiFlag) Set(value string) error { *m = append(*m, value); return nil }

func parseKeyMapFlags(keyMapArgs []string) map[string][]int {
	m := make(map[string][]int)
	for _, entry := range keyMapArgs {
		parts := strings.Split(entry, ":")
		if len(parts) != 2 {
			slog.Warn("Invalid keymap entry", "entry", entry)
			continue
		}

		codes := strings.Split(parts[1], "+")
		var linuxCodes []int
		for _, codeStr := range codes {
			code, err := strconv.Atoi(codeStr)
			if err != nil {
				slog.Warn("Invalid linux key code", "code", codeStr, "error", err)
				continue
			}
			linuxCodes = append(linuxCodes, code)
		}

		m[parts[0]] = linuxCodes
	}
	return m
}

const queueDirEnvVar = "CEC_QUEUE_DIR"

func parseFlags() *Config {
	var (
		keyMapArgs   multiFlag
		powerDevices multiFlag
		cfg          Config
	)

	flag.StringVar(&cfg.CECAdapter, "cec-adapter", "", "CEC adapter path (leave empty for auto-detect)")
	flag.StringVar(&cfg.DeviceName, "device-name", "", "Device name shown on your TV (leave empty for hostname)")
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debug output")
	flag.BoolVar(&cfg.NoPowerEvents, "no-power-events", false, "Disable power event handling")
	flag.IntVar(&cfg.ConnectionRetries, "retries", 5, "Number of times to retry CEC connection on failure")
	flag.Var(&keyMapArgs, "keymap", "Custom CEC-to-Linux key mapping (format <cec>:<linux>, e.g. --keymap 1:105)")
	flag.Var(&powerDevices, "devices", "Power event device addresses (e.g. --devices 0,1). Default to 0")
	flag.BoolVar(&cfg.NoVolumeSync, "no-volume-sync", false, "Disable volume synchronization")
	flag.IntVar(&cfg.VolumeStep, "volume-step", 5, "Volume change step percentage (1-100)")
	flag.IntVar(&cfg.AudioDevice, "audio-device", 5, "CEC audio device address for volume sync (default: 5 = Audio System)")
	flag.Parse()
	cfg.KeyMapOverrides = parseKeyMapFlags(keyMapArgs)
	cfg.PowerDevices = parseDevices(powerDevices)
	cfg.NoPowerEvents = cfg.NoPowerEvents || len(cfg.PowerDevices) == 0
	if cfg.DeviceName == "" {
		cfg.DeviceName, _ = os.Hostname()
	}
	if cfg.VolumeStep < 1 || cfg.VolumeStep > 100 {
		slog.Warn("Volume step must be between 1 and 100, using default 5")
		cfg.VolumeStep = 5
	}
	if cfg.QueueDir = os.Getenv(queueDirEnvVar); cfg.QueueDir == "" {
		var err error
		if cfg.QueueDir, err = os.MkdirTemp("", "cec-queue-*"); err != nil {
			slog.Error("Failed to create temporary queue dir", "error", err)
			os.Exit(1)
		}
	}
	return &cfg
}

func parseDevices(devices []string) []int {
	if len(devices) == 0 {
		return []int{0} // Default to device 0
	}
	var result []int
	for _, devStr := range devices {
		parts := strings.Split(devStr, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			dev, err := strconv.Atoi(part)
			if err != nil {
				slog.Warn("Invalid device address", "device", part, "error", err)
				continue
			}
			result = append(result, dev)
		}
	}
	return result
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

func handleVolumeKey(cecKeyCode int, audioCtrl *AudioController, volumeStep int) bool {
	volumeUpKey := 0x41   // CEC key code for VolumeUp
	volumeDownKey := 0x42 // CEC key code for VolumeDown
	muteKey := 0x43       // CEC key code for Mute

	switch cecKeyCode {
	case volumeUpKey:
		slog.Info("CEC Volume Up pressed, increasing system volume", "step", volumeStep)
		if err := audioCtrl.VolumeUp(volumeStep); err != nil {
			slog.Error("Failed to increase system volume", "error", err)
		}
		return true
	case volumeDownKey:
		slog.Info("CEC Volume Down pressed, decreasing system volume", "step", volumeStep)
		if err := audioCtrl.VolumeDown(volumeStep); err != nil {
			slog.Error("Failed to decrease system volume", "error", err)
		}
		return true
	case muteKey:
		slog.Info("CEC Mute pressed, toggling system mute")
		if err := audioCtrl.Mute(); err != nil {
			slog.Error("Failed to toggle system mute", "error", err)
		}
		return true
	}
	return false
}

func main() {
	cfg := parseFlags()
	setupLogger(cfg.Debug)

	slog.Info("Starting cec-controller", "config", cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	queue, err := NewQueue(ctx, cfg.QueueDir)
	if err != nil {
		slog.Error("Failed to initialize event queue", "dir", cfg.QueueDir, "error", err)
		os.Exit(1)
	}
	defer queue.Close()

	// Create KeyMap object
	keyMapObj, err := NewKeyMap(cfg.KeyMapOverrides)
	if err != nil {
		slog.Error("Failed to initialize virtual keyboard", "error", err)
		os.Exit(1)
	}

	c, err := NewCEC(cfg.CECAdapter, cfg.DeviceName, cfg.ConnectionRetries, queue.InKeyEvents)
	if err != nil {
		slog.Error("Failed to open CEC", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	if !cfg.NoPowerEvents {
		// cec-controller just started alongside the system, so we assume the system has to be powered on
		queue.InPowerEvents <- PowerEvent{Type: PowerOn, Active: true}
		if err := PowerEventListener(ctx, queue.InPowerEvents); err != nil {
			slog.Error("Failed to start power event listener", "error", err)
			os.Exit(1)
		}
	}

	var audioCtrl *AudioController
	var volumeChanges chan int
	if !cfg.NoVolumeSync {
		audioCtrl, err = NewAudioController()
		if err != nil {
			slog.Warn("Failed to initialize audio controller, volume sync disabled", "error", err)
			cfg.NoVolumeSync = true
		} else {
			slog.Info("Volume synchronization enabled", "volume-step", cfg.VolumeStep, "audio-device", cfg.AudioDevice)
			volumeChanges = make(chan int, 10)
			go func() {
				if err := audioCtrl.MonitorVolume(ctx, volumeChanges); err != nil && err != context.Canceled {
					slog.Error("Volume monitoring stopped", "error", err)
				}
			}()
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

			// Handle volume keys if volume sync is enabled
			if !cfg.NoVolumeSync && audioCtrl != nil {
				handled := handleVolumeKey(kp.KeyCode, audioCtrl, cfg.VolumeStep)
				if handled {
					continue
				}
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
		case vol := <-volumeChanges:
			if !cfg.NoVolumeSync {
				slog.Debug("System volume changed, syncing to CEC device", "volume", vol)
				// Note: CEC devices don't support setting absolute volume, only relative changes
				// This channel is here for future enhancements or logging
			}
		case <-ctx.Done():
			slog.Info("Shutting down...")
			return
		}
	}
}
