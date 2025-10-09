package main

import (
	"log/slog"

	keybd "github.com/micmonay/keybd_event"
)

// KeyMap provides mapping from CEC key codes to Linux key codes and handles virtual key events.
type KeyMap struct {
	cecToLinux map[int]int
	kb         *keybd.KeyBonding
}

// NewKeyMap creates a KeyMap, optionally overriding defaults.
func NewKeyMap(overrides map[int]int) (*KeyMap, error) {
	kb, err := keybd.NewKeyBonding()
	if err != nil {
		return nil, err
	}
	// Base map (can be extended)
	base := map[int]int{
		0x00: 28,  // Select      -> KEY_ENTER
		0x01: 103, // Up          -> KEY_UP
		0x02: 108, // Down        -> KEY_DOWN
		0x03: 105, // Left        -> KEY_LEFT
		0x04: 106, // Right       -> KEY_RIGHT
		0x09: 1,   // Exit        -> KEY_ESC
		0x0D: 57,  // Play        -> KEY_SPACE
		0x20: 63,  // Pause       -> KEY_PAUSE
		0x1B: 102, // Home        -> KEY_HOME
		0x1A: 158, // Back        -> KEY_BACK
		0x21: 119, // Volume Up   -> KEY_VOLUMEUP
		0x22: 114, // Volume Down -> KEY_VOLUMEDOWN
		0x23: 113, // Mute        -> KEY_MUTE
		0x2F: 168, // Red         -> KEY_REDO
		0x30: 169, // Green       -> KEY_GREEN
		0x31: 170, // Yellow      -> KEY_YELLOW
		0x32: 171, // Blue        -> KEY_BLUE
	}
	// Apply overrides
	for k, v := range overrides {
		base[k] = v
	}
	return &KeyMap{
		cecToLinux: base,
		kb:         kb,
	}, nil
}

// OnKeyPress maps a CEC key code to Linux and sends the virtual key event.
func (km *KeyMap) OnKeyPress(cecKeyCode int) {
	linuxKeyCode, ok := km.cecToLinux[cecKeyCode]
	if !ok {
		slog.Warn("Unmapped CEC key code", "cecKeyCode", cecKeyCode)
		return
	}
	slog.Info("Sending virtual key event", "linuxKeyCode", linuxKeyCode)
	km.kb.SetKeys(linuxKeyCode)
	if err := km.kb.Launching(); err != nil {
		slog.Error("Failed to send key event", "error", err)
	}
}
