package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/godbus/dbus/v5"
)

type PowerEventType int

const (
	PowerOn PowerEventType = iota
	PowerSleep
	PowerResume
	PowerShutdown
)

type PowerEvent struct {
	Type   PowerEventType
	Active bool // true if the event is starting (e.g., going to sleep), false if ending (e.g., resuming)
}

// PowerEventListener subscribes to systemd-logind D-Bus signals and sends events on the channel.
func PowerEventListener(ctx context.Context, events chan<- PowerEvent) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}

	// Subscribe to PrepareForSleep and PrepareForShutdown signals from logind
	if err := conn.AddMatchSignal(dbus.WithMatchSender("org.freedesktop.login1"),
		dbus.WithMatchInterface("org.freedesktop.login1.Manager"),
		dbus.WithMatchMember("PrepareForSleep"),
	); err != nil {
		return fmt.Errorf("failed to add match for sleep signals: %w", err)
	}
	if err := conn.AddMatchSignal(dbus.WithMatchSender("org.freedesktop.login1"),
		dbus.WithMatchInterface("org.freedesktop.login1.Manager"),
		dbus.WithMatchMember("PrepareForShutdown"),
	); err != nil {
		return fmt.Errorf("failed to add match for shutdown signals: %w", err)
	}

	signalCh := make(chan *dbus.Signal, 10)
	conn.Signal(signalCh)

	go func() {
		for {
			select {
			case sig := <-signalCh:
				if sig == nil || len(sig.Body) == 0 {
					continue
				}
				active, ok := sig.Body[0].(bool)
				if !ok {
					continue
				}
				switch sig.Name {
				case "org.freedesktop.login1.Manager.PrepareForSleep":
					evType := PowerResume
					if active {
						evType = PowerSleep
					}
					events <- PowerEvent{Type: evType, Active: active}
					slog.Debug("Power event", "type", evType, "active", active)
				case "org.freedesktop.login1.Manager.PrepareForShutdown":
					events <- PowerEvent{Type: PowerShutdown, Active: active}
					slog.Debug("Power event", "type", PowerShutdown, "active", active)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}
