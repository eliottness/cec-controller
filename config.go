package main

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

const (
	configFilePath = "/etc/cec-controller.yaml"
	queueDirEnvVar = "CEC_QUEUE_DIR"
)

// loadConfig loads configuration from file and environment variables
// CLI flags take precedence over config file, which takes precedence over defaults
func loadConfig() (*Config, error) {
	cfg := &Config{}

	// Set up viper to read from config file
	viper.SetConfigFile(configFilePath)
	viper.SetConfigType("yaml")

	// Attempt to read config file (not an error if it doesn't exist)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			slog.Warn("Error reading config file", "path", configFilePath, "error", err)
		}
	}

	// Read all config values with defaults
	cfg.CECAdapter = viper.GetString("cec-adapter")
	cfg.DeviceName = viper.GetString("device-name")
	cfg.Debug = viper.GetBool("debug")
	cfg.NoPowerEvents = viper.GetBool("no-power-events")
	cfg.ConnectionRetries = viper.GetInt("retries")

	// Handle keymap overrides
	if keyMapConfig := viper.Get("keymap"); keyMapConfig != nil {
		switch v := keyMapConfig.(type) {
		case []interface{}:
			var keyMapArgs []string
			for _, item := range v {
				if str, ok := item.(string); ok {
					keyMapArgs = append(keyMapArgs, str)
				}
			}
			cfg.KeyMapOverrides = parseKeyMapFlags(keyMapArgs)
		case []string:
			cfg.KeyMapOverrides = parseKeyMapFlags(v)
		}
	}

	// Handle power devices
	if devicesConfig := viper.Get("devices"); devicesConfig != nil {
		switch v := devicesConfig.(type) {
		case []interface{}:
			var deviceStrs []string
			for _, item := range v {
				switch val := item.(type) {
				case string:
					deviceStrs = append(deviceStrs, val)
				case int:
					deviceStrs = append(deviceStrs, strconv.Itoa(val))
				case int64:
					deviceStrs = append(deviceStrs, strconv.FormatInt(val, 10))
				}
			}
			cfg.PowerDevices = parseDevices(deviceStrs)
		case []string:
			cfg.PowerDevices = parseDevices(v)
		case string:
			cfg.PowerDevices = parseDevices([]string{v})
		}
	}

	// Handle queue directory from environment variable
	if cfg.QueueDir = os.Getenv(queueDirEnvVar); cfg.QueueDir == "" {
		cfg.QueueDir = viper.GetString("queue-dir")
	}

	// Apply defaults for unset values
	if cfg.ConnectionRetries == 0 {
		cfg.ConnectionRetries = 5
	}
	if cfg.DeviceName == "" {
		cfg.DeviceName, _ = os.Hostname()
	}
	if len(cfg.PowerDevices) == 0 && !cfg.NoPowerEvents {
		cfg.PowerDevices = []int{0}
	}
	if cfg.NoPowerEvents || len(cfg.PowerDevices) == 0 {
		cfg.NoPowerEvents = true
	}
	if cfg.QueueDir == "" {
		var err error
		if cfg.QueueDir, err = os.MkdirTemp("", "cec-queue-*"); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

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
