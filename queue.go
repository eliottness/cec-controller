package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"syscall"
	"time"

	"github.com/beeker1121/goque"
	"github.com/claes/cec"
)

type Queue struct {
	InPowerEvents chan PowerEvent
	InKeyEvents   chan *cec.KeyPress

	OutPowerEvents chan PowerEvent
	OutKeyEvents   chan *cec.KeyPress

	fsQueue *goque.Queue
	dir     string
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

	inPowerEvents := make(chan PowerEvent, 10)
	inKeyEvents := make(chan *cec.KeyPress, 10)
	outPowerEvents := make(chan PowerEvent, 10)
	outKeyEvents := make(chan *cec.KeyPress, 10)

	q := &Queue{
		InPowerEvents:  inPowerEvents,
		InKeyEvents:    inKeyEvents,
		OutPowerEvents: outPowerEvents,
		OutKeyEvents:   outKeyEvents,
		fsQueue:        queue,
		dir:            dir,
	}

	go func() {
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
				}
			case ke := <-inKeyEvents:
				data, err := json.Marshal(ke)
				if err != nil {
					slog.Error("Error marshaling key event", "error", err)
					continue
				}

				if _, err := queue.EnqueueObjectAsJSON(queueItem{Type: "key", Data: data}); err != nil {
					slog.Error("Error enqueuing key event", "error", err)
				}
			default:
				item, err := queue.Dequeue()
				if errors.Is(err, goque.ErrEmpty) {
					time.Sleep(1 * time.Millisecond)
					continue
				}
				if err != nil {
					slog.Error("Error dequeuing item", "error", err)
				}

				var qItem queueItem
				if err := json.Unmarshal(item.Value, &qItem); err != nil {
					slog.Error("Error parsing dequeued item", "error", err)
					continue
				}

				switch qItem.Type {
				case "power":
					var powerEvent PowerEvent
					err = json.Unmarshal(qItem.Data, &powerEvent)
					if err == nil {
						q.OutPowerEvents <- powerEvent
					}
				case "key":
					var keyEvent cec.KeyPress
					err = json.Unmarshal(qItem.Data, &keyEvent)
					if err == nil {
						q.OutKeyEvents <- &keyEvent
					}
				default:
					slog.Warn("Unknown queue item type", "type", qItem.Type)
				}

				if err != nil {
					slog.Error("Error parsing dequeued item", "error", err)
				}
			}
		}
	}()

	return q, nil
}

// RestartProcess sometimes the cec library gets stuck and stops receiving events.
// This function restarts the entire process making sure the queue is preserved between processes
func (q *Queue) RestartProcess() {
	execPath, err := os.Executable()
	if err != nil {
		slog.Error("Failed to get executable path, cannot restart", "error", err)
		return
	}

	q.close(false)
	if err := syscall.Exec(execPath, os.Args, append([]string{queueDirEnvVar + "=" + q.dir}, os.Environ()...)); err != nil {
		slog.Error("Failed to restart", "error", err)
	}
}

func (q *Queue) Close() {
	q.close(true)
}

func (q *Queue) close(delete bool) {
	close(q.InPowerEvents)
	close(q.InKeyEvents)
	close(q.OutPowerEvents)
	close(q.OutKeyEvents)
	q.fsQueue.Close()

	if delete {
		if err := os.RemoveAll(q.dir); err != nil {
			slog.Error("Failed to remove queue directory", "dir", q.dir, "error", err)
		}
	}
}
