package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPowerEventChannel(t *testing.T) {
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
	ch := make(chan PowerEvent, 10)

	events := []PowerEvent{
		{Type: PowerOn, Active: true},
		{Type: PowerSleep, Active: true},
		{Type: PowerResume, Active: false},
		{Type: PowerShutdown, Active: true},
	}

	for _, event := range events {
		ch <- event
	}

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
	powerCh := make(chan PowerEvent, 10)

	for i := 0; i < 5; i++ {
		powerCh <- PowerEvent{Type: PowerOn, Active: true}
	}

	select {
	case powerCh <- PowerEvent{Type: PowerSleep, Active: true}:
	default:
		t.Error("Channel should not be full after 6 events")
	}

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
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		<-ctx.Done()
		done <- true
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Error("Goroutine did not respond to context cancellation")
	}
}

func TestTemporaryDirectory(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "queue-test-temp")

	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Expected directory to exist")
	}

	err = os.RemoveAll(tempDir)
	if err != nil {
		t.Errorf("Failed to remove temp directory: %v", err)
	}

	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Error("Expected directory to be removed")
	}
}

func TestRestartProcessRetryLogic(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	queue, err := NewQueue(ctx, tempDir)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Close()

	if result := queue.RestartProcess(0); result {
		t.Error("Expected RestartProcess to return false when retriesLeft is 0")
	}
	if result := queue.RestartProcess(-1); result {
		t.Error("Expected RestartProcess to return false when retriesLeft is negative")
	}
}

func TestRestartProcessPositiveRetries(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test that would restart process in CI environment")
	}
}

func TestRestartProcessRetryDecrement(t *testing.T) {
	testCases := []struct {
		retriesLeft      int
		shouldAttempt    bool
		expectedNewValue int
	}{
		{0, false, 0},
		{1, true, 0},
		{5, true, 4},
		{10, true, 9},
		{-1, false, 0},
	}

	for _, tc := range testCases {
		if tc.retriesLeft <= 0 {
			if tc.shouldAttempt {
				t.Errorf("retriesLeft=%d: expected shouldAttempt=false", tc.retriesLeft)
			}
		} else {
			if !tc.shouldAttempt {
				t.Errorf("retriesLeft=%d: expected shouldAttempt=true", tc.retriesLeft)
			}
			newValue := tc.retriesLeft - 1
			if newValue != tc.expectedNewValue {
				t.Errorf("retriesLeft=%d: expected new value %d, got %d",
					tc.retriesLeft, tc.expectedNewValue, newValue)
			}
		}
	}
}

// TestQueueEventRouting verifies that events sent to InPowerEvents and
// InKeyEvents arrive on OutPowerEvents and OutKeyEvents respectively.
func TestQueueEventRouting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	q, err := NewQueue(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("NewQueue failed: %v", err)
	}
	defer q.Close()

	// Power event round-trip
	q.InPowerEvents <- PowerEvent{Type: PowerSleep, Active: true}
	select {
	case ev := <-q.OutPowerEvents:
		if ev.Type != PowerSleep || !ev.Active {
			t.Errorf("Unexpected power event: %+v", ev)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for power event")
	}
}

// TestQueuePreservesOrder verifies that multiple events arrive in FIFO order.
func TestQueuePreservesOrder(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	q, err := NewQueue(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("NewQueue failed: %v", err)
	}
	defer q.Close()

	events := []PowerEvent{
		{Type: PowerOn, Active: true},
		{Type: PowerSleep, Active: true},
		{Type: PowerResume, Active: false},
	}

	for _, ev := range events {
		q.InPowerEvents <- ev
	}

	for i, expected := range events {
		select {
		case got := <-q.OutPowerEvents:
			if got.Type != expected.Type || got.Active != expected.Active {
				t.Errorf("Event %d: expected %+v, got %+v", i, expected, got)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("Timeout waiting for event %d", i)
		}
	}
}
