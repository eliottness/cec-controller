package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

const (
	configFilePath       = "/etc/cec-controller.yaml"
	queueDirEnvVar       = "CEC_QUEUE_DIR"
	restartRetriesEnvVar = "CEC_RESTART_RETRIES"
)

// loadConfig loads configuration from file and environment variables.
// CLI flags take precedence over config file, which takes precedence over defaults.
func loadConfig() (*Config, error) {
	cfg := &Config{}

	viper.SetConfigFile(configFilePath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("Error reading config file", "path", configFilePath, "error", err)
		}
	}

	cfg.CECAdapter = viper.GetString("cec-adapter")
	cfg.DeviceName = viper.GetString("device-name")
	cfg.Debug = viper.GetBool("debug")
	cfg.NoPowerEvents = viper.GetBool("no-power-events")
	cfg.ConnectionRetries = viper.GetInt("retries")
	cfg.SetActiveSource = viper.GetBool("set-active-source")
	cfg.ActiveSourceDeviceType = viper.GetInt("active-source-type")

	// Handle keymap overrides
	if keyMapConfig := viper.Get("keymap"); keyMapConfig != nil {
		switch v := keyMapConfig.(type) {
		case map[string]interface{}:
			cfg.KeyMapOverrides = parseKeyMapFromMap(v)
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

	// Queue directory: env var takes precedence (set by RestartProcess)
	if cfg.QueueDir = os.Getenv(queueDirEnvVar); cfg.QueueDir == "" {
		cfg.QueueDir = viper.GetString("queue-dir")
	}

	// Restart retries: env var takes precedence (decremented by previous process on restart)
	if retriesStr := os.Getenv(restartRetriesEnvVar); retriesStr != "" {
		if retries, err := strconv.Atoi(retriesStr); err == nil {
			cfg.RestartRetries = retries
		} else {
			slog.Warn("Invalid CEC_RESTART_RETRIES value", "value", retriesStr, "error", err)
			cfg.RestartRetries = viper.GetInt("restart-retries")
		}
	} else {
		cfg.RestartRetries = viper.GetInt("restart-retries")
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
	if cfg.RestartRetries == 0 {
		cfg.RestartRetries = 3
	}
	if cfg.ActiveSourceDeviceType == 0 {
		cfg.ActiveSourceDeviceType = CECDeviceTypePlayback
	}

	return cfg, nil
}

// validateConfig checks that all config values are within acceptable ranges.
func validateConfig(cfg *Config) error {
	if cfg.ConnectionRetries < 1 {
		return fmt.Errorf("--retries must be at least 1 (got %d)", cfg.ConnectionRetries)
	}
	if cfg.RestartRetries < 0 {
		return fmt.Errorf("--restart-retries must be non-negative (got %d)", cfg.RestartRetries)
	}
	validDeviceTypes := map[int]bool{
		CECDeviceTypeTV: true, CECDeviceTypeRecording: true,
		CECDeviceTypeTuner: true, CECDeviceTypePlayback: true,
		CECDeviceTypeAudioSystem: true,
	}
	if !validDeviceTypes[cfg.ActiveSourceDeviceType] {
		return fmt.Errorf("--active-source-type must be one of 0,1,3,4,5 (got %d)", cfg.ActiveSourceDeviceType)
	}
	return nil
}

func parseKeyMapFromMap(keyMapConfig map[string]interface{}) map[string][]int {
	m := make(map[string][]int)
	for cecKey, value := range keyMapConfig {
		var linuxCodesStr string
		switch v := value.(type) {
		case string:
			linuxCodesStr = v
		default:
			slog.Warn("Invalid keymap value type", "key", cecKey, "value", value)
			continue
		}

		codes := strings.Split(linuxCodesStr, "+")
		var linuxCodes []int
		valid := true
		for _, codeStr := range codes {
			code, err := strconv.Atoi(codeStr)
			if err != nil {
				slog.Warn("Invalid keymap entry, skipping", "key", cecKey, "value", linuxCodesStr, "badCode", codeStr)
				valid = false
				break
			}
			linuxCodes = append(linuxCodes, code)
		}
		if valid {
			m[cecKey] = linuxCodes
		}
	}
	return m
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
		valid := true
		for _, codeStr := range codes {
			code, err := strconv.Atoi(codeStr)
			if err != nil {
				slog.Warn("Invalid keymap entry, skipping", "entry", entry, "badCode", codeStr)
				valid = false
				break
			}
			linuxCodes = append(linuxCodes, code)
		}
		if valid {
			m[parts[0]] = linuxCodes
		}
	}
	return m
}

func parseDevices(devices []string) []int {
	if len(devices) == 0 {
		return []int{0}
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
