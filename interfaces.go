package main

import (
	"fmt"

	"github.com/claes/cec"
	keybd "github.com/micmonay/keybd_event"
)

// CECConnection interface abstracts the CEC library connection for testing.
// Semantics: nil = success, non-nil = failure (standard Go).
type CECConnection interface {
	PowerOn(address int) error
	Standby(address int) error
	SetActiveSource(deviceType int) bool
	SetKeyPressesChan(ch chan *cec.KeyPress)
	Close()
}

// CECConnectionWrapper wraps the real CEC connection and normalises its error
// semantics: libcec returns non-nil on success and nil on failure; this wrapper
// inverts that so callers see standard Go conventions (nil = success).
type CECConnectionWrapper struct {
	*cec.Connection
}

func (w *CECConnectionWrapper) PowerOn(address int) error {
	if w.Connection.PowerOn(address) == nil {
		return fmt.Errorf("libcec PowerOn failed for address %d", address)
	}
	return nil
}

func (w *CECConnectionWrapper) Standby(address int) error {
	if w.Connection.Standby(address) == nil {
		return fmt.Errorf("libcec Standby failed for address %d", address)
	}
	return nil
}

func (w *CECConnectionWrapper) SetActiveSource(deviceType int) bool {
	return w.Connection.SetActiveSource(deviceType)
}

func (w *CECConnectionWrapper) SetKeyPressesChan(ch chan *cec.KeyPress) {
	w.Connection.KeyPresses = ch
}

// KeyboardEmitter abstracts virtual key event emission for testing.
type KeyboardEmitter interface {
	Emit(keyCodes []int) error
}

// keybdEmitter is the real KeyboardEmitter using keybd_event.
type keybdEmitter struct{}

func (k *keybdEmitter) Emit(keyCodes []int) error {
	kb, err := keybd.NewKeyBonding()
	if err != nil {
		return fmt.Errorf("failed to create KeyBonding: %w", err)
	}
	kb.SetKeys(keyCodes...)
	return kb.Launching()
}
