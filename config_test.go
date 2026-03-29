package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestConfigLoading(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "cec-controller.yaml")

	configContent := `
cec-adapter: "/dev/ttyACM0"
device-name: "TestDevice"
debug: true
no-power-events: false
retries: 10
keymap:
  "1": "105"
  "2": "106"
devices:
  - "0"
  - "1"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	viper.Reset()
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if adapter := viper.GetString("cec-adapter"); adapter != "/dev/ttyACM0" {
		t.Errorf("Expected cec-adapter to be '/dev/ttyACM0', got '%s'", adapter)
	}
	if deviceName := viper.GetString("device-name"); deviceName != "TestDevice" {
		t.Errorf("Expected device-name to be 'TestDevice', got '%s'", deviceName)
	}
	if debug := viper.GetBool("debug"); !debug {
		t.Error("Expected debug to be true")
	}
	if retries := viper.GetInt("retries"); retries != 10 {
		t.Errorf("Expected retries to be 10, got %d", retries)
	}
	keymap := viper.GetStringMapString("keymap")
	if len(keymap) != 2 {
		t.Errorf("Expected 2 keymap entries, got %d", len(keymap))
	}
	if keymap["1"] != "105" {
		t.Errorf("Expected keymap['1'] to be '105', got '%s'", keymap["1"])
	}
}

func TestParseKeyMapFromMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string][]int
	}{
		{
			name:     "Single key mapping",
			input:    map[string]interface{}{"1": "105"},
			expected: map[string][]int{"1": {105}},
		},
		{
			name:     "Multiple key codes",
			input:    map[string]interface{}{"1": "29+105"},
			expected: map[string][]int{"1": {29, 105}},
		},
		{
			name:     "Multiple mappings",
			input:    map[string]interface{}{"1": "105", "2": "106"},
			expected: map[string][]int{"1": {105}, "2": {106}},
		},
		{
			name:     "Invalid value type",
			input:    map[string]interface{}{"1": 105},
			expected: map[string][]int{},
		},
		{
			name:     "Partially invalid codes skips entire entry",
			input:    map[string]interface{}{"1": "29+abc+105"},
			expected: map[string][]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseKeyMapFromMap(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d mappings, got %d", len(tt.expected), len(result))
			}
			for key, expectedCodes := range tt.expected {
				resultCodes, ok := result[key]
				if !ok {
					t.Errorf("Expected key '%s' not found in result", key)
					continue
				}
				if len(resultCodes) != len(expectedCodes) {
					t.Errorf("For key '%s', expected %d codes, got %d", key, len(expectedCodes), len(resultCodes))
					continue
				}
				for i, code := range expectedCodes {
					if resultCodes[i] != code {
						t.Errorf("For key '%s' at index %d, expected code %d, got %d", key, i, code, resultCodes[i])
					}
				}
			}
		})
	}
}

func TestParseKeyMapFlags(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string][]int
	}{
		{
			name:     "Single key mapping",
			input:    []string{"1:105"},
			expected: map[string][]int{"1": {105}},
		},
		{
			name:     "Multiple key codes",
			input:    []string{"1:29+105"},
			expected: map[string][]int{"1": {29, 105}},
		},
		{
			name:     "Multiple mappings",
			input:    []string{"1:105", "2:106"},
			expected: map[string][]int{"1": {105}, "2": {106}},
		},
		{
			name:     "Invalid format",
			input:    []string{"invalid"},
			expected: map[string][]int{},
		},
		{
			name:     "Partially invalid codes skips entire entry",
			input:    []string{"1:29+abc+105"},
			expected: map[string][]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseKeyMapFlags(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d mappings, got %d", len(tt.expected), len(result))
			}
			for key, expectedCodes := range tt.expected {
				resultCodes, ok := result[key]
				if !ok {
					t.Errorf("Expected key '%s' not found in result", key)
					continue
				}
				if len(resultCodes) != len(expectedCodes) {
					t.Errorf("For key '%s', expected %d codes, got %d", key, len(expectedCodes), len(resultCodes))
					continue
				}
				for i, code := range expectedCodes {
					if resultCodes[i] != code {
						t.Errorf("For key '%s' at index %d, expected code %d, got %d", key, i, code, resultCodes[i])
					}
				}
			}
		})
	}
}

func TestParseDevices(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []int
	}{
		{"Empty input defaults to device 0", []string{}, []int{0}},
		{"Single device", []string{"1"}, []int{1}},
		{"Multiple devices in one string", []string{"0,1,2"}, []int{0, 1, 2}},
		{"Multiple devices in separate strings", []string{"0", "1", "2"}, []int{0, 1, 2}},
		{"Devices with spaces", []string{"0, 1, 2"}, []int{0, 1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDevices(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d devices, got %d", len(tt.expected), len(result))
			}
			for i, expected := range tt.expected {
				if i >= len(result) {
					t.Errorf("Missing device at index %d", i)
					continue
				}
				if result[i] != expected {
					t.Errorf("At index %d, expected device %d, got %d", i, expected, result[i])
				}
			}
		})
	}
}

func TestDefaultValues(t *testing.T) {
	viper.Reset()

	tempDir := t.TempDir()
	os.Setenv(queueDirEnvVar, tempDir)
	defer os.Unsetenv(queueDirEnvVar)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.ConnectionRetries != 5 {
		t.Errorf("Expected default retries to be 5, got %d", cfg.ConnectionRetries)
	}
	if cfg.DeviceName == "" {
		t.Error("Expected device name to be set to hostname")
	}
	if cfg.NoPowerEvents || len(cfg.PowerDevices) != 1 || cfg.PowerDevices[0] != 0 {
		t.Errorf("Expected NoPowerEvents=false and PowerDevices=[0], got NoPowerEvents=%v, PowerDevices=%v", cfg.NoPowerEvents, cfg.PowerDevices)
	}
	if cfg.QueueDir != tempDir {
		t.Errorf("Expected queue dir to be '%s', got '%s'", tempDir, cfg.QueueDir)
	}
	if cfg.RestartRetries != 3 {
		t.Errorf("Expected default restart retries to be 3, got %d", cfg.RestartRetries)
	}
	if cfg.ActiveSourceDeviceType != CECDeviceTypePlayback {
		t.Errorf("Expected default active source device type to be %d (Playback), got %d", CECDeviceTypePlayback, cfg.ActiveSourceDeviceType)
	}
	if cfg.SetActiveSource {
		t.Error("Expected set-active-source to be false by default")
	}
}

func TestRestartRetriesFromEnvVar(t *testing.T) {
	viper.Reset()
	tempDir := t.TempDir()
	os.Setenv(queueDirEnvVar, tempDir)
	defer os.Unsetenv(queueDirEnvVar)
	os.Setenv(restartRetriesEnvVar, "7")
	defer os.Unsetenv(restartRetriesEnvVar)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if cfg.RestartRetries != 7 {
		t.Errorf("Expected RestartRetries to be 7 from env var, got %d", cfg.RestartRetries)
	}
}

// TestExampleConfigFile verifies that the shipped example config file parses
// cleanly and contains all known configuration keys, preventing silent drift.
func TestExampleConfigFile(t *testing.T) {
	viper.Reset()
	viper.SetConfigFile("cec-controller.yaml.example")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read example config file: %v", err)
	}

	tempDir := t.TempDir()
	os.Setenv(queueDirEnvVar, tempDir)
	defer os.Unsetenv(queueDirEnvVar)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed on example file: %v", err)
	}
	if err := validateConfig(cfg); err != nil {
		t.Fatalf("validateConfig failed on example file: %v", err)
	}

	// Verify all known keys are present in the example file so drift is caught.
	knownKeys := []string{
		"cec-adapter", "device-name", "debug", "no-power-events",
		"retries", "restart-retries", "set-active-source", "active-source-type",
		"keymap", "devices", "queue-dir",
	}
	for _, key := range knownKeys {
		if !viper.IsSet(key) {
			t.Errorf("Example config file is missing key %q — add it to cec-controller.yaml.example", key)
		}
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid defaults",
			cfg:     Config{ConnectionRetries: 5, RestartRetries: 3, ActiveSourceDeviceType: CECDeviceTypePlayback},
			wantErr: false,
		},
		{
			name:    "zero retries",
			cfg:     Config{ConnectionRetries: 0, RestartRetries: 3, ActiveSourceDeviceType: CECDeviceTypePlayback},
			wantErr: true,
		},
		{
			name:    "negative restart retries",
			cfg:     Config{ConnectionRetries: 5, RestartRetries: -1, ActiveSourceDeviceType: CECDeviceTypePlayback},
			wantErr: true,
		},
		{
			name:    "invalid device type",
			cfg:     Config{ConnectionRetries: 5, RestartRetries: 3, ActiveSourceDeviceType: 9},
			wantErr: true,
		},
		{
			name:    "valid TV device type",
			cfg:     Config{ConnectionRetries: 5, RestartRetries: 0, ActiveSourceDeviceType: CECDeviceTypeTV},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
