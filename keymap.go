package main

import (
	"log/slog"

	"github.com/claes/cec"
	keybd "github.com/micmonay/keybd_event"
)

// KeyMap provides mapping from CEC key codes to Linux key codes and handles virtual key events.
type KeyMap struct {
	cecToLinux map[int][]int
}

var base = map[int]int{
	// Navigation
	cec.GetKeyCodeByName("Select"): keybd.VK_ENTER,
	cec.GetKeyCodeByName("Enter"):  keybd.VK_ENTER,
	cec.GetKeyCodeByName("Up"):     keybd.VK_UP,
	cec.GetKeyCodeByName("Down"):   keybd.VK_DOWN,
	cec.GetKeyCodeByName("Left"):   keybd.VK_LEFT,
	cec.GetKeyCodeByName("Right"):  keybd.VK_RIGHT,
	cec.GetKeyCodeByName("Exit"):   keybd.VK_ESC,
	cec.GetKeyCodeByName("Play"):   keybd.VK_PLAY,
	cec.GetKeyCodeByName("Pause"):  keybd.VK_PAUSE,
	cec.GetKeyCodeByName("Stop"):   keybd.VK_STOP,
	cec.GetKeyCodeByName("Home"):   keybd.VK_HOME,

	// Numbers
	cec.GetKeyCodeByName("0"): keybd.VK_0,
	cec.GetKeyCodeByName("1"): keybd.VK_1,
	cec.GetKeyCodeByName("2"): keybd.VK_2,
	cec.GetKeyCodeByName("3"): keybd.VK_3,
	cec.GetKeyCodeByName("4"): keybd.VK_4,
	cec.GetKeyCodeByName("5"): keybd.VK_5,
	cec.GetKeyCodeByName("6"): keybd.VK_6,
	cec.GetKeyCodeByName("7"): keybd.VK_7,
	cec.GetKeyCodeByName("8"): keybd.VK_8,
	cec.GetKeyCodeByName("9"): keybd.VK_9,

	// TODO: send MPRIS messages
	//cec.GetKeyCodeByName("Volume Up"): keybd.VK_VOLUMEUP,
	//cec.GetKeyCodeByName("Volume Down"): keybd.VK_VOLUMEDOWN,
	//cec.GetKeyCodeByName("Mute"): keybd.VK_MUTE,
}

// NewKeyMap creates a KeyMap, optionally overriding defaults.
func NewKeyMap(overrides map[string][]int) (*KeyMap, error) {
	// Base map (can be extended)

	var keyMap = make(map[int][]int, len(base)+len(overrides))

	for k, v := range base {
		keyMap[k] = []int{v}
	}

	// Apply overrides
	for k, v := range overrides {
		cecCode := cec.GetKeyCodeByName(k)
		if cecCode == -1 {
			slog.Warn("Invalid CEC key name in overrides", "key", k)
			continue
		}
		keyMap[cecCode] = v
	}

	slog.Debug("Key map initialized", "mapping", base)

	return &KeyMap{
		cecToLinux: keyMap,
	}, nil
}

// OnKeyPress maps a CEC key code to Linux and sends the virtual key event.
func (km *KeyMap) OnKeyPress(cecKeyCode int) {
	linuxKeyCode, ok := km.cecToLinux[cecKeyCode]
	if !ok {
		slog.Warn("Unmapped CEC key code", "cec-key-code", cecKeyCode)
		return
	}

	kb, err := keybd.NewKeyBonding()
	if err != nil {
		slog.Error("Failed to create KeyBonding", "error", err)
		return
	}

	slog.Debug("Sending virtual key event", "cec-key-code", cecKeyCode, "linux-key-code", linuxKeyCode)
	kb.SetKeys(linuxKeyCode...)
	if err := kb.Launching(); err != nil {
		slog.Error("Failed to send key event", "error", err)
	}
}
