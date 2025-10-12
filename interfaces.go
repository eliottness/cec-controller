package main

import "github.com/claes/cec"

// CECConnection interface abstracts the CEC library connection for testing
type CECConnection interface {
	PowerOn(address int) error
	Standby(address int) error
	Close()
}

// CECConnectionWrapper wraps the real CEC connection
type CECConnectionWrapper struct {
	*cec.Connection
}

func (w *CECConnectionWrapper) SetKeyPressesChan(ch chan *cec.KeyPress) {
	w.Connection.KeyPresses = ch
}

// DBusConnection interface abstracts D-Bus connection for testing
type DBusConnection interface {
	AddMatchSignal(options ...interface{}) error
	Signal(ch chan<- interface{})
	Close() error
}
