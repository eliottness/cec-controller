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

	"github.com/claes/cec"
)

type Config struct {
	CECAdapter      string
	Debug           bool
	KeyMapOverrides map[int]int
	NoPowerEvents   bool
	PowerDevices    []int
}

type multiFlag []string

func (m *multiFlag) String() string         { return strings.Join(*m, ",") }
func (m *multiFlag) Set(value string) error { *m = append(*m, value); return nil }

func parseKeyMapFlags(keyMapArgs []string) map[int]int {
	m := make(map[int]int)
	for _, entry := range keyMapArgs {
		parts := strings.Split(entry, ":")
		if len(parts) != 2 {
			slog.Warn("Invalid keymap entry", "entry", entry)
			continue
		}
		cecCode, err1 := strconv.Atoi(parts[0])
		linuxCode, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			slog.Warn("Invalid keymap numbers", "entry", entry)
			continue
		}
		m[cecCode] = linuxCode
	}
	return m
}

func parseFlags() *Config {
	var keyMapArgs multiFlag
	var powerDevices multiFlag
	cfg := &Config{}
	flag.StringVar(&cfg.CECAdapter, "cec-adapter", "", "CEC adapter path (leave empty for auto-detect)")
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debug output")
	flag.BoolVar(&cfg.NoPowerEvents, "no-power-events", false, "Disable power event handling")
	flag.Var(&keyMapArgs, "keymap", "Custom CEC-to-Linux key mapping (format <cec>:<linux>, e.g. --keymap 1:105)")
	flag.Var(&powerDevices, "devices", "Power event device addresses (e.g. --devices 0,1). Default to 0")
	flag.Parse()
	cfg.KeyMapOverrides = parseKeyMapFlags(keyMapArgs)
	cfg.PowerDevices = parseDevices(powerDevices)
	cfg.NoPowerEvents = cfg.NoPowerEvents || len(cfg.PowerDevices) == 0
	return cfg
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
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
}

func main() {
	cfg := parseFlags()
	setupLogger(cfg.Debug)

	slog.Info("Starting cec-controller", "config", cfg)

	// Create KeyMap object
	keyMapObj, err := NewKeyMap(cfg.KeyMapOverrides)
	if err != nil {
		slog.Error("Failed to initialize virtual keyboard", "error", err)
		os.Exit(1)
	}

	c, err := cec.Open(cfg.CECAdapter, "cec-controller")
	if err != nil {
		slog.Error("Failed to open CEC", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	c.KeyPresses = make(chan *cec.KeyPress, 10)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Send startup event
	events := make(chan PowerEvent, 8)
	events <- PowerEvent{Type: PowerOn, Active: true}

	if !cfg.NoPowerEvents {
		if err := PowerEventListener(ctx, events); err != nil {
			slog.Error("Failed to start power event listener", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("Listening for CEC key events... (Ctrl+C to exit)")
	for {
		select {
		case kp := <-c.KeyPresses:
			slog.Info("CEC Key pressed", "code", kp.KeyCode, "duration_ms", kp.Duration)
			keyMapObj.OnKeyPress(kp.KeyCode)
		case ev := <-events:
			switch ev.Type {
			case PowerOn, PowerResume:
				for _, dev := range cfg.PowerDevices {
					slog.Info("Sending CEC power on to device", "device", dev)
					if err := c.PowerOn(dev); err != nil {
						slog.Error("Failed to send PowerOn", "device", dev, "error", err)
					}
				}
			case PowerSleep, PowerShutdown:
				for _, dev := range cfg.PowerDevices {
					slog.Info("Sending CEC standby to device", "device", dev)
					if err := c.Standby(dev); err != nil {
						slog.Error("Failed to send Standby", "device", dev, "error", err)
					}
				}
			}
		case <-ctx.Done():
			slog.Info("Shutting down...")
			return
		}
	}
}
