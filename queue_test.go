package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPowerEventChannel(t *testing.T) {
	// Test power event channel operations
	ch := make(chan PowerEvent, 10)

	testEvent := PowerEvent{Type: PowerSleep, Active: true}
	ch <- testEvent

	receivedEvent := <-ch
	if receivedEvent.Type != testEvent.Type {
		t.Errorf("Expected event type %d, got %d", testEvent.Type, receivedEvent.Type)
	}
	if receivedEvent.Active != testEvent.Active {
		t.Errorf("Expected Active to be %v, got %v", testEvent.Active, receivedEvent.Active)
	}
}

func TestMultiplePowerEvents(t *testing.T) {
	// Test multiple power events through channel
	ch := make(chan PowerEvent, 10)

	events := []PowerEvent{
		{Type: PowerOn, Active: true},
		{Type: PowerSleep, Active: true},
		{Type: PowerResume, Active: false},
		{Type: PowerShutdown, Active: true},
	}

	// Send all events
	for _, event := range events {
		ch <- event
	}

	// Receive and verify all events
	for i, expected := range events {
		received := <-ch
		if received.Type != expected.Type {
			t.Errorf("Event %d: expected type %d, got %d", i, expected.Type, received.Type)
		}
		if received.Active != expected.Active {
			t.Errorf("Event %d: expected Active %v, got %v", i, expected.Active, received.Active)
		}
	}
}

func TestQueueItemSerialization(t *testing.T) {
	// Test queueItem structure
	item := queueItem{
		Type: "power",
		Data: []byte(`{"Type":1,"Active":true}`),
	}

	if item.Type != "power" {
		t.Errorf("Expected type 'power', got '%s'", item.Type)
	}
	if item.Data == nil {
		t.Error("Expected Data to be non-nil")
	}
}

func TestChannelBuffering(t *testing.T) {
	// Test that channels can buffer multiple events
	powerCh := make(chan PowerEvent, 10)

	// Fill buffer partially
	for i := 0; i < 5; i++ {
		powerCh <- PowerEvent{Type: PowerOn, Active: true}
	}

	// Verify we can still send more
	select {
	case powerCh <- PowerEvent{Type: PowerSleep, Active: true}:
		// Success - channel not full
	default:
		t.Error("Channel should not be full after 6 events")
	}

	// Drain channel
	count := 0
	for len(powerCh) > 0 {
		<-powerCh
		count++
	}

	if count != 6 {
		t.Errorf("Expected to drain 6 events, got %d", count)
	}
}

func TestContextCancellationPattern(t *testing.T) {
	// Test context cancellation pattern used in queue
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		<-ctx.Done()
		done <- true
	}()

	cancel()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Goroutine did not respond to context cancellation")
	}
}

func TestTemporaryDirectory(t *testing.T) {
	// Test temporary directory creation and cleanup
	tempDir := filepath.Join(os.TempDir(), "queue-test-temp")

	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Expected directory to exist")
	}

	// Clean up
	err = os.RemoveAll(tempDir)
	if err != nil {
		t.Errorf("Failed to remove temp directory: %v", err)
	}

	// Verify directory is removed
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Error("Expected directory to be removed")
	}
}
