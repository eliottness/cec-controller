# Volume Sync Quick Reference

## 🎯 What It Does

Controls your Linux system volume using your TV remote control buttons via HDMI-CEC.

## 🔧 Setup

### Requirements
- PipeWire with `wpctl` **OR** PulseAudio with `pactl`
- CEC-enabled TV and HDMI connection
- cec-controller running

### Zero Configuration
Volume sync works automatically - just start cec-controller:
```bash
./cec-controller
```

## 🎮 Usage

### Remote Control Buttons
| TV Remote Button | System Action |
|-----------------|---------------|
| Volume Up       | Increase system volume by 5% |
| Volume Down     | Decrease system volume by 5% |
| Mute            | Toggle system audio mute |

### Command-Line Options
```bash
# Change volume step
./cec-controller --volume-step 10     # Larger steps (10%)
./cec-controller --volume-step 2      # Smaller steps (2%)

# Disable volume sync
./cec-controller --no-volume-sync

# Enable debug logging
./cec-controller --debug
```

## 🔍 Troubleshooting

### Volume not working?

1. **Check audio system:**
   ```bash
   # For PipeWire
   wpctl status
   
   # For PulseAudio
   pactl info
   ```

2. **Run with debug:**
   ```bash
   ./cec-controller --debug
   ```
   Look for messages like:
   - `Audio system detected system=pipewire`
   - `CEC Volume Up pressed, increasing system volume`

3. **Common issues:**
   - No audio system detected → Install PipeWire or PulseAudio
   - CEC not working → Check HDMI-CEC is enabled on TV
   - Wrong device → Try `--audio-device 0` or `--audio-device 1`

## 📖 More Info

- Full documentation: [VOLUME_SYNC.md](VOLUME_SYNC.md)
- Implementation details: [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)
- General usage: [README.md](README.md)

## 🎉 Quick Test

1. Start cec-controller: `./cec-controller --debug`
2. Press Volume Up on your TV remote
3. Listen for volume increase
4. Check logs for: `CEC Volume Up pressed, increasing system volume`

That's it! 🎊
