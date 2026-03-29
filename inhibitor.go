package main

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
)

// inhibitorLock holds a systemd-logind delay inhibitor file descriptor.
// Closing the fd releases the lock and allows the system to proceed.
type inhibitorLock struct {
	fd *os.File
}

// openSystemBus opens a connection to the D-Bus system bus for inhibitor use.
func openSystemBus() (*dbus.Conn, error) {
	return dbus.SystemBus()
}

// acquireInhibitor acquires a systemd-logind delay inhibitor lock via D-Bus.
// what is a colon-separated list of inhibition targets (e.g. "sleep:shutdown").
// The returned lock must be released by calling Release() once the protected
// operation completes, allowing the system to proceed.
func acquireInhibitor(conn *dbus.Conn, what, why string) (*inhibitorLock, error) {
	if conn == nil {
		return nil, nil
	}
	obj := conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")
	var fd dbus.UnixFD
	if err := obj.Call("org.freedesktop.login1.Manager.Inhibit", 0,
		what, "cec-controller", why, "delay",
	).Store(&fd); err != nil {
		return nil, fmt.Errorf("failed to acquire inhibitor lock: %w", err)
	}
	return &inhibitorLock{fd: os.NewFile(uintptr(fd), "inhibitor-lock")}, nil
}

// Release releases the inhibitor lock. Safe to call on a nil receiver.
func (l *inhibitorLock) Release() {
	if l != nil && l.fd != nil {
		l.fd.Close()
		l.fd = nil
	}
}
