package main

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// AudioSystem represents the type of audio system in use
type AudioSystem string

const (
	AudioSystemPulseAudio AudioSystem = "pulseaudio"
	AudioSystemPipeWire   AudioSystem = "pipewire"
	AudioSystemUnknown    AudioSystem = "unknown"
)

// AudioController manages system audio volume
type AudioController struct {
	system AudioSystem
}

// NewAudioController creates a new audio controller and detects the audio system
func NewAudioController() (*AudioController, error) {
	system := detectAudioSystem()
	if system == AudioSystemUnknown {
		return nil, fmt.Errorf("no supported audio system detected (PulseAudio or PipeWire)")
	}

	slog.Info("Audio system detected", "system", system)
	return &AudioController{system: system}, nil
}

// detectAudioSystem determines which audio system is running
func detectAudioSystem() AudioSystem {
	// Check for wpctl (PipeWire)
	if _, err := exec.LookPath("wpctl"); err == nil {
		if err := exec.Command("wpctl", "status").Run(); err == nil {
			return AudioSystemPipeWire
		}
	}

	// Check for pactl (PulseAudio)
	if _, err := exec.LookPath("pactl"); err == nil {
		if err := exec.Command("pactl", "info").Run(); err == nil {
			return AudioSystemPulseAudio
		}
	}

	return AudioSystemUnknown
}

// VolumeUp increases the system volume by the specified percentage
func (a *AudioController) VolumeUp(percentage int) error {
	slog.Debug("Increasing system volume", "percentage", percentage)
	
	switch a.system {
	case AudioSystemPipeWire:
		return a.executeCommand("wpctl", "set-volume", "@DEFAULT_AUDIO_SINK@", fmt.Sprintf("%d%%+", percentage))
	case AudioSystemPulseAudio:
		return a.executeCommand("pactl", "set-sink-volume", "@DEFAULT_SINK@", fmt.Sprintf("+%d%%", percentage))
	default:
		return fmt.Errorf("unsupported audio system: %s", a.system)
	}
}

// VolumeDown decreases the system volume by the specified percentage
func (a *AudioController) VolumeDown(percentage int) error {
	slog.Debug("Decreasing system volume", "percentage", percentage)
	
	switch a.system {
	case AudioSystemPipeWire:
		return a.executeCommand("wpctl", "set-volume", "@DEFAULT_AUDIO_SINK@", fmt.Sprintf("%d%%-", percentage))
	case AudioSystemPulseAudio:
		return a.executeCommand("pactl", "set-sink-volume", "@DEFAULT_SINK@", fmt.Sprintf("-%d%%", percentage))
	default:
		return fmt.Errorf("unsupported audio system: %s", a.system)
	}
}

// Mute toggles the mute state of the system audio
func (a *AudioController) Mute() error {
	slog.Debug("Toggling system mute")
	
	switch a.system {
	case AudioSystemPipeWire:
		return a.executeCommand("wpctl", "set-mute", "@DEFAULT_AUDIO_SINK@", "toggle")
	case AudioSystemPulseAudio:
		return a.executeCommand("pactl", "set-sink-mute", "@DEFAULT_SINK@", "toggle")
	default:
		return fmt.Errorf("unsupported audio system: %s", a.system)
	}
}

// GetVolume retrieves the current system volume as a percentage (0-100)
func (a *AudioController) GetVolume() (int, error) {
	var cmd *exec.Cmd
	
	switch a.system {
	case AudioSystemPipeWire:
		cmd = exec.Command("wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@")
	case AudioSystemPulseAudio:
		cmd = exec.Command("pactl", "get-sink-volume", "@DEFAULT_SINK@")
	default:
		return 0, fmt.Errorf("unsupported audio system: %s", a.system)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to get volume: %w", err)
	}

	return parseVolume(string(output), a.system)
}

// parseVolume extracts the volume percentage from command output
func parseVolume(output string, system AudioSystem) (int, error) {
	output = strings.TrimSpace(output)
	
	switch system {
	case AudioSystemPipeWire:
		// wpctl returns "Volume: 0.50" format
		parts := strings.Fields(output)
		if len(parts) >= 2 {
			volStr := strings.TrimSpace(parts[1])
			vol, err := strconv.ParseFloat(volStr, 64)
			if err == nil {
				return int(vol * 100), nil
			}
		}
	case AudioSystemPulseAudio:
		// pactl returns lines like "Volume: front-left: 32768 /  50% / -18.06 dB"
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Volume:") {
				parts := strings.Split(line, "/")
				if len(parts) >= 2 {
					volStr := strings.TrimSpace(parts[1])
					volStr = strings.TrimSuffix(volStr, "%")
					vol, err := strconv.Atoi(volStr)
					if err == nil {
						return vol, nil
					}
				}
			}
		}
	}
	
	return 0, fmt.Errorf("failed to parse volume from output: %s", output)
}

// MonitorVolume monitors system volume changes and sends them to the channel
func (a *AudioController) MonitorVolume(ctx context.Context, changes chan<- int) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	lastVolume := -1
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			currentVolume, err := a.GetVolume()
			if err != nil {
				slog.Debug("Failed to get current volume", "error", err)
				continue
			}
			
			if currentVolume != lastVolume && lastVolume != -1 {
				slog.Debug("Volume changed", "from", lastVolume, "to", currentVolume)
				select {
				case changes <- currentVolume:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			lastVolume = currentVolume
		}
	}
}

// executeCommand runs a command and returns any error
func (a *AudioController) executeCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command %s failed: %w, output: %s", name, err, string(output))
	}
	return nil
}
