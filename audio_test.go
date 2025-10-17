package main

import (
	"testing"
)

func TestAudioSystem_Constants(t *testing.T) {
	tests := []struct {
		name     string
		system   AudioSystem
		expected string
	}{
		{"PulseAudio", AudioSystemPulseAudio, "pulseaudio"},
		{"PipeWire", AudioSystemPipeWire, "pipewire"},
		{"Unknown", AudioSystemUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.system) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.system))
			}
		})
	}
}

func TestParseVolume_PipeWire(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected int
		wantErr  bool
	}{
		{
			name:     "Valid volume 50%",
			output:   "Volume: 0.50",
			expected: 50,
			wantErr:  false,
		},
		{
			name:     "Valid volume 75%",
			output:   "Volume: 0.75",
			expected: 75,
			wantErr:  false,
		},
		{
			name:     "Valid volume 100%",
			output:   "Volume: 1.00",
			expected: 100,
			wantErr:  false,
		},
		{
			name:     "Valid volume 0%",
			output:   "Volume: 0.00",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "Invalid output",
			output:   "Invalid",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVolume(tt.output, AudioSystemPipeWire)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseVolume() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseVolume_PulseAudio(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected int
		wantErr  bool
	}{
		{
			name:     "Valid volume 50%",
			output:   "Volume: front-left: 32768 /  50% / -18.06 dB,   front-right: 32768 /  50% / -18.06 dB",
			expected: 50,
			wantErr:  false,
		},
		{
			name:     "Valid volume 75%",
			output:   "Volume: front-left: 49152 /  75% / -7.50 dB,   front-right: 49152 /  75% / -7.50 dB",
			expected: 75,
			wantErr:  false,
		},
		{
			name:     "Valid volume 100%",
			output:   "Volume: front-left: 65536 / 100% / 0.00 dB,   front-right: 65536 / 100% / 0.00 dB",
			expected: 100,
			wantErr:  false,
		},
		{
			name:     "Invalid output",
			output:   "Invalid",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVolume(tt.output, AudioSystemPulseAudio)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseVolume() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDetectAudioSystem(t *testing.T) {
	// This is a basic test that just checks the function runs
	// In a real environment, it will detect the actual audio system
	system := detectAudioSystem()
	
	// We can't make assumptions about the test environment, so just verify
	// that it returns one of the valid types
	validSystems := []AudioSystem{AudioSystemPulseAudio, AudioSystemPipeWire, AudioSystemUnknown}
	found := false
	for _, valid := range validSystems {
		if system == valid {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("detectAudioSystem() returned invalid system: %s", system)
	}
}

func TestAudioController_Structure(t *testing.T) {
	// Test that AudioController can be created with a specific system
	ac := &AudioController{system: AudioSystemPulseAudio}
	if ac.system != AudioSystemPulseAudio {
		t.Errorf("Expected system to be PulseAudio, got %s", ac.system)
	}

	ac = &AudioController{system: AudioSystemPipeWire}
	if ac.system != AudioSystemPipeWire {
		t.Errorf("Expected system to be PipeWire, got %s", ac.system)
	}
}
