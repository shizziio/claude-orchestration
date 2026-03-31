// Package logbroker provides an in-memory log buffer that fans out
// live run output to polling HTTP clients and persists it for completed runs.
package logbroker

import (
	"io"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// Broker stores per-run log buffers and delivers chunks to subscribers.
// It is safe for concurrent use.
type Broker struct {
	mu      sync.Mutex
	entries map[uuid.UUID]*entry
}

type entry struct {
	buf    strings.Builder
	subs   []chan struct{} // signal channels: closed when new data or run ends
	closed bool           // true once the run is complete
}

// New returns a ready Broker.
func New() *Broker {
	return &Broker{entries: make(map[uuid.UUID]*entry)}
}

// Writer returns an io.WriteCloser that writes log data for the given run.
// Calling Close() signals that the run has finished.
func (b *Broker) Writer(runID uuid.UUID) io.WriteCloser {
	b.mu.Lock()
	if b.entries[runID] == nil {
		b.entries[runID] = &entry{}
	}
	b.mu.Unlock()
	return &runWriter{b: b, id: runID}
}

type runWriter struct {
	b  *Broker
	id uuid.UUID
}

func (w *runWriter) Write(p []byte) (int, error) {
	w.b.mu.Lock()
	defer w.b.mu.Unlock()
	e := w.b.entries[w.id]
	if e == nil || e.closed {
		return 0, io.ErrClosedPipe
	}
	e.buf.Write(p)
	w.b.notifyLocked(e)
	return len(p), nil
}

func (w *runWriter) Close() error {
	w.b.mu.Lock()
	defer w.b.mu.Unlock()
	e := w.b.entries[w.id]
	if e == nil {
		return nil
	}
	e.closed = true
	w.b.notifyLocked(e)
	return nil
}

// notifyLocked wakes all subscribers and prunes closed channels.
// Must be called with b.mu held.
func (b *Broker) notifyLocked(e *entry) {
	live := e.subs[:0]
	for _, ch := range e.subs {
		select {
		case ch <- struct{}{}:
			live = append(live, ch)
		default:
			// subscriber is already pending a notification — keep it
			live = append(live, ch)
		}
	}
	e.subs = live
}

// Subscribe returns a signal channel and the current buffered log.
// The channel receives a value whenever new data is written or the run ends.
// For completed runs (closed == true) the channel is nil.
// Callers must call Unsubscribe when done.
func (b *Broker) Subscribe(runID uuid.UUID) (signal <-chan struct{}, current string, live bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.entries[runID]
	if e == nil {
		return nil, "", false
	}
	cur := e.buf.String()
	if e.closed {
		return nil, cur, true
	}
	ch := make(chan struct{}, 1)
	e.subs = append(e.subs, ch)
	return ch, cur, true
}

// Unsubscribe removes a subscriber channel from the entry.
func (b *Broker) Unsubscribe(runID uuid.UUID, ch <-chan struct{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.entries[runID]
	if e == nil {
		return
	}
	filtered := e.subs[:0]
	for _, s := range e.subs {
		if s != ch {
			filtered = append(filtered, s)
		}
	}
	e.subs = filtered
}

// GetLog returns the current log for a run (buffered so far, even if still running).
// Returns ("", false) if the run is not in the broker.
func (b *Broker) GetLog(runID uuid.UUID) (string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.entries[runID]
	if e == nil {
		return "", false
	}
	return e.buf.String(), true
}

// IsClosed reports whether a run has been closed (i.e. the runner finished).
func (b *Broker) IsClosed(runID uuid.UUID) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.entries[runID]
	return e != nil && e.closed
}

// Evict removes the entry from memory after the log has been persisted to DB.
func (b *Broker) Evict(runID uuid.UUID) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.entries, runID)
}
