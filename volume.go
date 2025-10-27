package main

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
)

// VolumeController interface abstracts volume control for testing
type VolumeController interface {
	VolumeUp() error
	VolumeDown() error
	Mute() error
	SetVolume(percent int) error
	GetVolume() (int, error)
	IsMuted() (bool, error)
}

// PulseAudioVolumeController controls system volume using pactl (PulseAudio/PipeWire)
type PulseAudioVolumeController struct {
	step int // Volume adjustment step in percent
}

// NewVolumeController creates a new volume controller
func NewVolumeController(step int) VolumeController {
	if step <= 0 || step > 100 {
		slog.Warn("Invalid volume step, defaulting to 5%", "step", step)
		step = 5
	}
	return &PulseAudioVolumeController{step: step}
}

// VolumeUp increases volume by the configured step
func (vc *PulseAudioVolumeController) VolumeUp() error {
	cmd := exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@", fmt.Sprintf("+%d%%", vc.step))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to increase volume: %w (output: %s)", err, string(output))
	}
	slog.Debug("Volume increased", "step", vc.step)
	return nil
}

// VolumeDown decreases volume by the configured step
func (vc *PulseAudioVolumeController) VolumeDown() error {
	cmd := exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@", fmt.Sprintf("-%d%%", vc.step))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to decrease volume: %w (output: %s)", err, string(output))
	}
	slog.Debug("Volume decreased", "step", vc.step)
	return nil
}

// Mute toggles mute state
func (vc *PulseAudioVolumeController) Mute() error {
	cmd := exec.Command("pactl", "set-sink-mute", "@DEFAULT_SINK@", "toggle")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to toggle mute: %w (output: %s)", err, string(output))
	}
	slog.Debug("Mute toggled")
	return nil
}

// SetVolume sets volume to a specific percentage
func (vc *PulseAudioVolumeController) SetVolume(percent int) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("invalid volume percentage: %d", percent)
	}
	cmd := exec.Command("pactl", "set-sink-volume", "@DEFAULT_SINK@", fmt.Sprintf("%d%%", percent))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set volume: %w (output: %s)", err, string(output))
	}
	slog.Debug("Volume set", "percent", percent)
	return nil
}

// GetVolume returns the current volume percentage
func (vc *PulseAudioVolumeController) GetVolume() (int, error) {
	cmd := exec.Command("pactl", "get-sink-volume", "@DEFAULT_SINK@")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to get volume: %w (output: %s)", err, string(output))
	}

	// Parse output like: "Volume: front-left: 65536 / 100% / 0.00 dB,   front-right: 65536 / 100% / 0.00 dB"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "Volume:") {
			continue
		}
		// Extract the first percentage value
		parts := strings.Split(line, "/")
		if len(parts) >= 2 {
			percentStr := strings.TrimSpace(parts[1])
			percentStr = strings.TrimSuffix(percentStr, "%")
			percent, err := strconv.Atoi(percentStr)
			if err != nil {
				return 0, fmt.Errorf("failed to parse volume percentage from '%s': %w", percentStr, err)
			}
			if percent < 0 || percent > 150 { // Allow some headroom but validate
				return 0, fmt.Errorf("invalid volume percentage parsed: %d", percent)
			}
			return percent, nil
		}
	}
	return 0, fmt.Errorf("could not parse volume from output: %s", string(output))
}

// IsMuted returns whether the audio is muted
func (vc *PulseAudioVolumeController) IsMuted() (bool, error) {
	cmd := exec.Command("pactl", "get-sink-mute", "@DEFAULT_SINK@")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to get mute state: %w (output: %s)", err, string(output))
	}

	// Parse output like: "Mute: yes" or "Mute: no"
	outputStr := strings.TrimSpace(string(output))
	if strings.HasPrefix(outputStr, "Mute: yes") {
		return true, nil
	} else if strings.HasPrefix(outputStr, "Mute: no") {
		return false, nil
	}
	return false, fmt.Errorf("unexpected mute state format: %s", outputStr)
}
