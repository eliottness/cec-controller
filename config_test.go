package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestConfigLoading(t *testing.T) {
	// Create a temporary config file
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

	// Override the config file path for testing
	viper.Reset()
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Test basic string values
	if adapter := viper.GetString("cec-adapter"); adapter != "/dev/ttyACM0" {
		t.Errorf("Expected cec-adapter to be '/dev/ttyACM0', got '%s'", adapter)
	}

	if deviceName := viper.GetString("device-name"); deviceName != "TestDevice" {
		t.Errorf("Expected device-name to be 'TestDevice', got '%s'", deviceName)
	}

	// Test boolean value
	if debug := viper.GetBool("debug"); !debug {
		t.Error("Expected debug to be true")
	}

	// Test integer value
	if retries := viper.GetInt("retries"); retries != 10 {
		t.Errorf("Expected retries to be 10, got %d", retries)
	}

	// Test map values
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
			name: "Single key mapping",
			input: map[string]interface{}{
				"1": "105",
			},
			expected: map[string][]int{
				"1": {105},
			},
		},
		{
			name: "Multiple key codes",
			input: map[string]interface{}{
				"1": "29+105",
			},
			expected: map[string][]int{
				"1": {29, 105},
			},
		},
		{
			name: "Multiple mappings",
			input: map[string]interface{}{
				"1": "105",
				"2": "106",
			},
			expected: map[string][]int{
				"1": {105},
				"2": {106},
			},
		},
		{
			name: "Invalid value type",
			input: map[string]interface{}{
				"1": 105, // Should be string
			},
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
			name:  "Single key mapping",
			input: []string{"1:105"},
			expected: map[string][]int{
				"1": {105},
			},
		},
		{
			name:  "Multiple key codes",
			input: []string{"1:29+105"},
			expected: map[string][]int{
				"1": {29, 105},
			},
		},
		{
			name:  "Multiple mappings",
			input: []string{"1:105", "2:106"},
			expected: map[string][]int{
				"1": {105},
				"2": {106},
			},
		},
		{
			name:     "Invalid format",
			input:    []string{"invalid"},
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
		{
			name:     "Empty input defaults to device 0",
			input:    []string{},
			expected: []int{0},
		},
		{
			name:     "Single device",
			input:    []string{"1"},
			expected: []int{1},
		},
		{
			name:     "Multiple devices in one string",
			input:    []string{"0,1,2"},
			expected: []int{0, 1, 2},
		},
		{
			name:     "Multiple devices in separate strings",
			input:    []string{"0", "1", "2"},
			expected: []int{0, 1, 2},
		},
		{
			name:     "Devices with spaces",
			input:    []string{"0, 1, 2"},
			expected: []int{0, 1, 2},
		},
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
	// Test with empty viper config
	viper.Reset()

	// Create a temporary directory for queue
	tempDir := t.TempDir()
	os.Setenv(queueDirEnvVar, tempDir)
	defer os.Unsetenv(queueDirEnvVar)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check default values
	if cfg.ConnectionRetries != 5 {
		t.Errorf("Expected default retries to be 5, got %d", cfg.ConnectionRetries)
	}

	if cfg.DeviceName == "" {
		t.Error("Expected device name to be set to hostname")
	}

	if cfg.NoPowerEvents || len(cfg.PowerDevices) != 1 || cfg.PowerDevices[0] != 0 {
		t.Errorf("Expected NoPowerEvents to be false and PowerDevices to be [0], got NoPowerEvents=%v, PowerDevices=%v", cfg.NoPowerEvents, cfg.PowerDevices)
	}

	if cfg.QueueDir != tempDir {
		t.Errorf("Expected queue dir to be '%s', got '%s'", tempDir, cfg.QueueDir)
	}
}
