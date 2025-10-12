package main

import (
	"context"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

// MockDBusConnection is a mock implementation of D-Bus connection for testing
type MockDBusConnection struct {
	AddMatchSignalFunc func(options ...interface{}) error
	SignalFunc         func(ch chan<- interface{})
	CloseFunc          func() error
	signalChan         chan *dbus.Signal
}

func (m *MockDBusConnection) AddMatchSignal(options ...interface{}) error {
	if m.AddMatchSignalFunc != nil {
		return m.AddMatchSignalFunc(options...)
	}
	return nil
}

func (m *MockDBusConnection) Signal(ch chan<- interface{}) {
	if m.SignalFunc != nil {
		m.SignalFunc(ch)
	}
}

func (m *MockDBusConnection) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func TestPowerEventType_Constants(t *testing.T) {
	// Verify the power event types are defined correctly
	if PowerOn != 0 {
		t.Errorf("Expected PowerOn to be 0, got %d", PowerOn)
	}
	if PowerSleep != 1 {
		t.Errorf("Expected PowerSleep to be 1, got %d", PowerSleep)
	}
	if PowerResume != 2 {
		t.Errorf("Expected PowerResume to be 2, got %d", PowerResume)
	}
	if PowerShutdown != 3 {
		t.Errorf("Expected PowerShutdown to be 3, got %d", PowerShutdown)
	}
}

func TestPowerEvent_Structure(t *testing.T) {
	event := PowerEvent{
		Type:   PowerSleep,
		Active: true,
	}

	if event.Type != PowerSleep {
		t.Errorf("Expected Type to be PowerSleep, got %d", event.Type)
	}
	if !event.Active {
		t.Error("Expected Active to be true")
	}
}

// MockPowerEventListener creates a testable version of PowerEventListener
// that uses a mock D-Bus connection
func MockPowerEventListener(ctx context.Context, events chan<- PowerEvent, signalChan chan *dbus.Signal) error {
	go func() {
		for {
			select {
			case sig := <-signalChan:
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
				case "org.freedesktop.login1.Manager.PrepareForShutdown":
					events <- PowerEvent{Type: PowerShutdown, Active: active}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func TestMockPowerEventListener_PrepareForSleep_Active(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events := make(chan PowerEvent, 10)
	signalChan := make(chan *dbus.Signal, 10)

	err := MockPowerEventListener(ctx, events, signalChan)
	if err != nil {
		t.Fatalf("MockPowerEventListener failed: %v", err)
	}

	// Send a PrepareForSleep signal with active=true (going to sleep)
	signalChan <- &dbus.Signal{
		Name: "org.freedesktop.login1.Manager.PrepareForSleep",
		Body: []interface{}{true},
	}

	select {
	case event := <-events:
		if event.Type != PowerSleep {
			t.Errorf("Expected PowerSleep event, got %d", event.Type)
		}
		if !event.Active {
			t.Error("Expected Active to be true")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for power event")
	}
}

func TestMockPowerEventListener_PrepareForSleep_Inactive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events := make(chan PowerEvent, 10)
	signalChan := make(chan *dbus.Signal, 10)

	err := MockPowerEventListener(ctx, events, signalChan)
	if err != nil {
		t.Fatalf("MockPowerEventListener failed: %v", err)
	}

	// Send a PrepareForSleep signal with active=false (resuming)
	signalChan <- &dbus.Signal{
		Name: "org.freedesktop.login1.Manager.PrepareForSleep",
		Body: []interface{}{false},
	}

	select {
	case event := <-events:
		if event.Type != PowerResume {
			t.Errorf("Expected PowerResume event, got %d", event.Type)
		}
		if event.Active {
			t.Error("Expected Active to be false")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for power event")
	}
}

func TestMockPowerEventListener_PrepareForShutdown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events := make(chan PowerEvent, 10)
	signalChan := make(chan *dbus.Signal, 10)

	err := MockPowerEventListener(ctx, events, signalChan)
	if err != nil {
		t.Fatalf("MockPowerEventListener failed: %v", err)
	}

	// Send a PrepareForShutdown signal with active=true
	signalChan <- &dbus.Signal{
		Name: "org.freedesktop.login1.Manager.PrepareForShutdown",
		Body: []interface{}{true},
	}

	select {
	case event := <-events:
		if event.Type != PowerShutdown {
			t.Errorf("Expected PowerShutdown event, got %d", event.Type)
		}
		if !event.Active {
			t.Error("Expected Active to be true")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for power event")
	}
}

func TestMockPowerEventListener_NilSignal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	events := make(chan PowerEvent, 10)
	signalChan := make(chan *dbus.Signal, 10)

	err := MockPowerEventListener(ctx, events, signalChan)
	if err != nil {
		t.Fatalf("MockPowerEventListener failed: %v", err)
	}

	// Send a nil signal (should be ignored)
	signalChan <- nil

	select {
	case <-events:
		t.Error("Did not expect any event for nil signal")
	case <-time.After(200 * time.Millisecond):
		// Expected - no event should be sent
	}
}

func TestMockPowerEventListener_EmptyBody(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	events := make(chan PowerEvent, 10)
	signalChan := make(chan *dbus.Signal, 10)

	err := MockPowerEventListener(ctx, events, signalChan)
	if err != nil {
		t.Fatalf("MockPowerEventListener failed: %v", err)
	}

	// Send a signal with empty body (should be ignored)
	signalChan <- &dbus.Signal{
		Name: "org.freedesktop.login1.Manager.PrepareForSleep",
		Body: []interface{}{},
	}

	select {
	case <-events:
		t.Error("Did not expect any event for empty body signal")
	case <-time.After(200 * time.Millisecond):
		// Expected - no event should be sent
	}
}

func TestMockPowerEventListener_InvalidBodyType(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	events := make(chan PowerEvent, 10)
	signalChan := make(chan *dbus.Signal, 10)

	err := MockPowerEventListener(ctx, events, signalChan)
	if err != nil {
		t.Fatalf("MockPowerEventListener failed: %v", err)
	}

	// Send a signal with wrong body type (should be ignored)
	signalChan <- &dbus.Signal{
		Name: "org.freedesktop.login1.Manager.PrepareForSleep",
		Body: []interface{}{"not a boolean"},
	}

	select {
	case <-events:
		t.Error("Did not expect any event for invalid body type")
	case <-time.After(200 * time.Millisecond):
		// Expected - no event should be sent
	}
}

func TestMockPowerEventListener_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	events := make(chan PowerEvent, 10)
	signalChan := make(chan *dbus.Signal, 10)

	err := MockPowerEventListener(ctx, events, signalChan)
	if err != nil {
		t.Fatalf("MockPowerEventListener failed: %v", err)
	}

	// Cancel the context
	cancel()

	// Give goroutine time to exit
	time.Sleep(100 * time.Millisecond)

	// Try to send a signal after cancellation
	signalChan <- &dbus.Signal{
		Name: "org.freedesktop.login1.Manager.PrepareForSleep",
		Body: []interface{}{true},
	}

	// Should not receive any event
	select {
	case <-events:
		t.Error("Did not expect any event after context cancellation")
	case <-time.After(200 * time.Millisecond):
		// Expected - goroutine should have stopped
	}
}
