package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	keybd "github.com/micmonay/keybd_event"
	"github.com/chbmuc/cec"
)

type Config struct {
	CECAdapter      string
	Debug           bool
	KeyMapOverrides map[int]int
}

type multiFlag []string

func (m *multiFlag) String() string        { return strings.Join(*m, ",") }
func (m *multiFlag) Set(value string) error { *m = append(*m, value); return nil }

func ParseKeyMapFlags(keyMapArgs []string) map[int]int {
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

func ParseFlags() *Config {
	var keyMapArgs multiFlag
	cfg := &Config{}
	flag.StringVar(&cfg.CECAdapter, "cec-adapter", "", "CEC adapter path (leave empty for auto-detect)")
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debug output")
	flag.Var(&keyMapArgs, "keymap", "Custom CEC-to-Linux key mapping (format <cec>:<linux>, e.g. --keymap 1:105)")
	flag.Parse()
	cfg.KeyMapOverrides = ParseKeyMapFlags(keyMapArgs)
	return cfg
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
	cfg := ParseFlags()
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

	// --- Power Events Integration ---
	events := make(chan PowerEvent, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Send startup event
	events <- PowerEvent{Type: PowerStartup, Active: true}

	if err := PowerEventListener(ctx, events); err != nil {
		slog.Error("Failed to start power event listener", "error", err)
		os.Exit(1)
	}

	go func() {
		for ev := range events {
			switch ev.Type {
			case PowerStartup:
				slog.Info("System startup event")
				// Add your startup hook logic here
			case PowerSleep:
				slog.Info("System is going to sleep")
				// Add your sleep hook logic here
			case PowerResume:
				slog.Info("System has resumed")
				// Add your resume hook logic here
			case PowerShutdown:
				slog.Info("System is shutting down")
				// Add your shutdown hook logic here
			}
		}
	}()

	slog.Info("Listening for CEC key events... (Ctrl+C to exit)")
	for {
		select {
		case kp := <-c.KeyPresses:
			slog.Info("CEC Key pressed", "code", kp.KeyCode, "duration_ms", kp.Duration)
			keyMapObj.OnKeyPress(kp.KeyCode)
		case <-time.After(5 * time.Second):
			slog.Debug("No key event...")
		}
	}
}
