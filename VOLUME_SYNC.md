# Volume Synchronization Feature

## Overview

This document describes the volume synchronization feature that enables bidirectional audio control between CEC devices and Linux audio systems (PulseAudio/PipeWire).

## How It Works

### CEC → System Audio (Primary Feature)

When you press volume control buttons on your TV remote:
1. The CEC Controller receives the key press event (VolumeUp: 0x41, VolumeDown: 0x42, Mute: 0x43)
2. The `handleVolumeKey()` function intercepts these CEC key codes
3. The `AudioController` detects your audio system (PulseAudio or PipeWire)
4. The appropriate command is executed:
   - **PipeWire**: `wpctl set-volume @DEFAULT_AUDIO_SINK@ X%+` or `X%-`
   - **PulseAudio**: `pactl set-sink-volume @DEFAULT_SINK@ +X%` or `-X%`
5. Your system volume changes accordingly

### System Audio → CEC (Monitoring)

The system includes infrastructure for monitoring system volume changes:
- `AudioController.MonitorVolume()` polls system volume every 500ms
- Changes are sent to a channel for potential future use
- Currently used for logging and future enhancements

Note: CEC devices typically don't support setting absolute volume, only relative changes (up/down).

## Configuration

### Command-Line Flags

- `--no-volume-sync`: Disable volume synchronization entirely
- `--volume-step <1-100>`: Set the volume change increment (default: 5%)
- `--audio-device <address>`: CEC audio device address (default: 5 = Audio System)

### Examples

```bash
# Use default settings (5% volume steps)
./cec-controller

# Use larger volume steps
./cec-controller --volume-step 10

# Disable volume sync
./cec-controller --no-volume-sync

# Custom volume step with debug output
./cec-controller --volume-step 3 --debug
```

## Requirements

One of the following must be installed and running:
- **PipeWire** with `wpctl` command
- **PulseAudio** with `pactl` command

Both are typically pre-installed on modern Linux distributions.

## Testing

### Unit Tests

Run the comprehensive test suite:
```bash
go test -v ./...
```

The tests cover:
- Audio system detection
- Volume parsing for both PipeWire and PulseAudio
- CEC volume method interfaces
- Mock CEC connections with volume functionality

### Manual Testing

To test manually (requires actual CEC hardware):

1. Start the controller:
   ```bash
   ./cec-controller --debug
   ```

2. Press volume buttons on your TV remote and verify:
   - Volume Up increases system volume
   - Volume Down decreases system volume
   - Mute toggles system mute

3. Check debug logs for:
   ```
   CEC Volume Up pressed, increasing system volume step=5
   ```

### Testing Without CEC Hardware

You can test the audio system integration independently:

```go
package main

import (
    "fmt"
    "log"
)

func main() {
    // Create audio controller
    ac, err := NewAudioController()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Detected audio system: %s\n", ac.system)
    
    // Get current volume
    vol, err := ac.GetVolume()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Current volume: %d%%\n", vol)
    
    // Test volume up
    if err := ac.VolumeUp(5); err != nil {
        log.Fatal(err)
    }
    fmt.Println("Volume increased by 5%")
    
    // Test mute
    if err := ac.Mute(); err != nil {
        log.Fatal(err)
    }
    fmt.Println("Mute toggled")
}
```

## Architecture

### Components

1. **AudioController** (`audio.go`)
   - Detects audio system (PipeWire or PulseAudio)
   - Provides unified interface for volume control
   - Monitors volume changes

2. **CEC Integration** (`main.go`)
   - `handleVolumeKey()`: Maps CEC key codes to audio actions
   - Event loop integration for CEC key presses
   - Configuration parsing and validation

3. **CEC Volume Methods** (`cec.go`)
   - `VolumeUp()`: Send volume up command to CEC device
   - `VolumeDown()`: Send volume down command to CEC device
   - `Mute()`: Send mute command to CEC device

### Key Design Decisions

1. **Auto-detection**: Automatically detects PulseAudio or PipeWire without user configuration
2. **Command-line tools**: Uses standard tools (wpctl/pactl) for maximum compatibility
3. **Optional feature**: Can be completely disabled with `--no-volume-sync`
4. **Thread-safe**: Uses channels and proper synchronization
5. **Graceful degradation**: If audio system is not available, the feature is disabled with a warning

## Troubleshooting

### Volume sync not working

1. Check if PipeWire or PulseAudio is running:
   ```bash
   # For PipeWire
   wpctl status
   
   # For PulseAudio
   pactl info
   ```

2. Run with debug logging:
   ```bash
   ./cec-controller --debug
   ```

3. Check for error messages:
   - "Failed to initialize audio controller": No supported audio system found
   - "Failed to increase system volume": Command execution failed

### Volume changes too small/large

Adjust the `--volume-step` parameter:
```bash
./cec-controller --volume-step 10  # Larger steps
./cec-controller --volume-step 2   # Smaller steps
```

## Future Enhancements

Potential improvements for future versions:

1. **Absolute volume sync**: Sync exact volume levels between CEC and system
2. **Multiple sinks**: Support for controlling specific audio outputs
3. **Volume limits**: Configure minimum and maximum volume levels
4. **Smart sync**: Only sync when certain applications are active
5. **Balance control**: Map additional CEC buttons to audio balance
6. **Per-application volume**: Control volume of specific applications
7. **MPRIS integration**: Use MPRIS D-Bus interface for media player control

## References

- [libcec Documentation](https://libcec.pulse-eight.com/)
- [PipeWire Documentation](https://docs.pipewire.org/)
- [PulseAudio Documentation](https://www.freedesktop.org/wiki/Software/PulseAudio/)
- [CEC Key Codes](https://github.com/claes/cec/blob/master/cec.go)
