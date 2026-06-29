package job

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"sync"
	"time"

	"convertthis/internal/convert"
	"convertthis/internal/storage"
)

type Status string

const (
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
	StatusError      Status = "error"
)

type Job struct {
	ID         string         `json:"id"`
	Status     Status         `json:"status"`
	From       convert.Format `json:"from"`
	To         convert.Format `json:"to"`
	OutputKey  string         `json:"-"`
	OutputName string         `json:"filename"`
	Err        string         `json:"error,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
}

type Request struct {
	Input    io.Reader
	From     convert.Format
	To       convert.Format
	BaseName string
}

type Queue interface {
	Enqueue(ctx context.Context, req Request) (Job, error)
	Get(id string) (Job, bool)
}

type SyncQueue struct {
	conv  convert.Converter
	store storage.Storage

	mu   sync.RWMutex
	jobs map[string]Job
}

func NewSyncQueue(c convert.Converter, s storage.Storage) *SyncQueue {
	return &SyncQueue{conv: c, store: s, jobs: make(map[string]Job)}
}

func (q *SyncQueue) set(j Job) {
	q.mu.Lock()
	q.jobs[j.ID] = j
	q.mu.Unlock()
}

func (q *SyncQueue) Get(id string) (Job, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	j, ok := q.jobs[id]
	return j, ok
}

func (q *SyncQueue) Enqueue(ctx context.Context, req Request) (Job, error) {
	j := Job{
		ID:        newID(),
		Status:    StatusProcessing,
		From:      req.From,
		To:        req.To,
		CreatedAt: time.Now(),
	}
	q.set(j)

	out, err := q.conv.Convert(ctx, req.Input, req.From, req.To)
	if err != nil {
		j.Status, j.Err = StatusError, err.Error()
		q.set(j)
		return j, err
	}

	key := j.ID + "." + req.To.Ext()
	if err := q.store.Save(ctx, key, out); err != nil {
		j.Status, j.Err = StatusError, err.Error()
		q.set(j)
		return j, err
	}

	j.Status = StatusDone
	j.OutputKey = key
	j.OutputName = req.BaseName + "." + req.To.Ext()
	q.set(j)
	return j, nil
}

func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
