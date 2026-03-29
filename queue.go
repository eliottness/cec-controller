package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"syscall"

	"github.com/beeker1121/goque"
	"github.com/claes/cec"
)

type Queue struct {
	InPowerEvents chan PowerEvent
	InKeyEvents   chan *cec.KeyPress

	OutPowerEvents chan PowerEvent
	OutKeyEvents   chan *cec.KeyPress

	fsQueue     *goque.Queue
	dir         string
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	cleanupOnce sync.Once
	notify      chan struct{} // closed/signalled by writer when an item is enqueued
}

type queueItem struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func NewQueue(ctx context.Context, dir string) (*Queue, error) {
	queue, err := goque.OpenQueue(dir)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	inPowerEvents := make(chan PowerEvent, 10)
	inKeyEvents := make(chan *cec.KeyPress, 100)
	outPowerEvents := make(chan PowerEvent, 10)
	outKeyEvents := make(chan *cec.KeyPress, 100)

	q := &Queue{
		InPowerEvents:  inPowerEvents,
		InKeyEvents:    inKeyEvents,
		OutPowerEvents: outPowerEvents,
		OutKeyEvents:   outKeyEvents,
		fsQueue:        queue,
		dir:            dir,
		cancel:         cancel,
		notify:         make(chan struct{}, 1),
	}

	// signal wakes the reader goroutine after an item is written to disk.
	// The channel is buffered(1): if a signal is already pending the send is
	// dropped, which is fine — the reader will drain all available items.
	signal := func() {
		select {
		case q.notify <- struct{}{}:
		default:
		}
	}

	// Writer goroutine: drains InPowerEvents and InKeyEvents to disk.
	// Blocking select ensures no busy-wait when idle.
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case pe := <-inPowerEvents:
				data, err := json.Marshal(pe)
				if err != nil {
					slog.Error("Error marshaling power event", "error", err)
					continue
				}
				if _, err := queue.EnqueueObjectAsJSON(queueItem{Type: "power", Data: data}); err != nil {
					slog.Error("Error enqueuing power event", "error", err)
				} else {
					signal()
				}
			case ke := <-inKeyEvents:
				data, err := json.Marshal(ke)
				if err != nil {
					slog.Error("Error marshaling key event", "error", err)
					continue
				}
				if _, err := queue.EnqueueObjectAsJSON(queueItem{Type: "key", Data: data}); err != nil {
					slog.Error("Error enqueuing key event", "error", err)
				} else {
					signal()
				}
			}
		}
	}()

	// Reader goroutine: dequeues items from disk and sends them to out channels.
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			item, err := queue.Dequeue()
			if errors.Is(err, goque.ErrEmpty) {
				select {
				case <-ctx.Done():
					return
				case <-q.notify:
				}
				continue
			}
			if err != nil {
				slog.Error("Error dequeuing item", "error", err)
				continue
			}

			var qItem queueItem
			if err := json.Unmarshal(item.Value, &qItem); err != nil {
				slog.Error("Error parsing dequeued item", "error", err)
				continue
			}

			switch qItem.Type {
			case "power":
				var powerEvent PowerEvent
				if err := json.Unmarshal(qItem.Data, &powerEvent); err != nil {
					slog.Error("Error parsing power event", "error", err)
					continue
				}
				select {
				case outPowerEvents <- powerEvent:
				case <-ctx.Done():
					return
				}
			case "key":
				var keyEvent cec.KeyPress
				if err := json.Unmarshal(qItem.Data, &keyEvent); err != nil {
					slog.Error("Error parsing key event", "error", err)
					continue
				}
				select {
				case outKeyEvents <- &keyEvent:
				case <-ctx.Done():
					return
				}
			default:
				slog.Warn("Unknown queue item type", "type", qItem.Type)
			}
		}
	}()

	return q, nil
}

// RestartProcess sometimes the cec library gets stuck and stops receiving events.
// This function restarts the entire process making sure the queue is preserved between processes.
// Returns true if restart was attempted, false if no retries left.
func (q *Queue) RestartProcess(retriesLeft int) bool {
	if retriesLeft <= 0 {
		slog.Error("No process restarts remaining, cannot restart")
		return false
	}

	execPath, err := os.Executable()
	if err != nil {
		slog.Error("Failed to get executable path, cannot restart", "error", err)
		return false
	}

	slog.Warn("Restarting process", "retriesLeft", retriesLeft-1)
	q.cleanup()

	// Pass the decremented retry count via environment variable
	env := os.Environ()
	env = append(env, queueDirEnvVar+"="+q.dir)
	env = append(env, restartRetriesEnvVar+"="+fmt.Sprintf("%d", retriesLeft-1))

	if err := syscall.Exec(execPath, os.Args, env); err != nil {
		slog.Error("Failed to restart", "error", err)
		return false
	}
	// syscall.Exec only returns on failure - success replaces the current process
	return true
}

func (q *Queue) Close() {
	q.cleanup()
	if err := os.RemoveAll(q.dir); err != nil {
		slog.Error("Failed to remove queue directory", "dir", q.dir, "error", err)
	}
}

// cleanup cancels the internal context, waits for goroutines to exit, and
// closes the underlying store. Safe to call multiple times.
func (q *Queue) cleanup() {
	q.cleanupOnce.Do(func() {
		q.cancel()
		q.wg.Wait()
		q.fsQueue.Close()
	})
}
