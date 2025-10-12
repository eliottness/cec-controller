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
	conn *cec.Connection
}

func (w *CECConnectionWrapper) PowerOn(address int) error {
	return w.conn.PowerOn(address)
}

func (w *CECConnectionWrapper) Standby(address int) error {
	return w.conn.Standby(address)
}

func (w *CECConnectionWrapper) Close() {
	w.conn.Close()
}

// DBusConnection interface abstracts D-Bus connection for testing
type DBusConnection interface {
	AddMatchSignal(options ...interface{}) error
	Signal(ch chan<- interface{})
	Close() error
}
