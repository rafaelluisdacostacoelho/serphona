package events

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// ChangeEvent represents a catalog change notification.
type ChangeEvent struct {
	Type      string    `json:"type"`
	TenantID  string    `json:"tenant_id"`
	ToolID    string    `json:"tool_id,omitempty"`
	VersionID string    `json:"version_id,omitempty"`
	DiffHash  string    `json:"diff_hash,omitempty"`
	At        time.Time `json:"at"`
}

// Notifier publishes change events and keeps a recent buffer for polling fallback.
type Notifier interface {
	Notify(event ChangeEvent)
	Recent(since time.Time) []ChangeEvent
}

// NewNotifier builds a notifier with optional webhook sink and in-memory buffer.
func NewNotifier(webhookURL string) Notifier {
	return &multiNotifier{
		sink:  newWebhookSink(webhookURL),
		store: newMemoryStore(256),
	}
}

// multiNotifier fan-outs to sink and stores locally.
type multiNotifier struct {
	sink  sink
	store *memoryStore
}

func (m *multiNotifier) Notify(event ChangeEvent) {
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	m.store.add(event)
	if m.sink != nil {
		m.sink.send(event)
	}
}

func (m *multiNotifier) Recent(since time.Time) []ChangeEvent {
	return m.store.recent(since)
}

// sink abstracts webhook publishing.
type sink interface {
	send(ChangeEvent)
}

type webhookSink struct {
	url    string
	client *http.Client
}

func newWebhookSink(url string) sink {
	if url == "" {
		return nil
	}
	return &webhookSink{url: url, client: &http.Client{Timeout: 5 * time.Second}}
}

func (w *webhookSink) send(evt ChangeEvent) {
	body, _ := json.Marshal(evt)
	req, err := http.NewRequest(http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	_, _ = w.client.Do(req)
}

// memoryStore keeps a ring buffer of events.
type memoryStore struct {
	mu     sync.Mutex
	buf    []ChangeEvent
	size   int
	index  int
	filled bool
}

func newMemoryStore(size int) *memoryStore {
	if size <= 0 {
		size = 256
	}
	return &memoryStore{buf: make([]ChangeEvent, size), size: size}
}

func (m *memoryStore) add(evt ChangeEvent) {
	m.mu.Lock()
	m.buf[m.index] = evt
	m.index = (m.index + 1) % m.size
	if m.index == 0 {
		m.filled = true
	}
	m.mu.Unlock()
}

func (m *memoryStore) recent(since time.Time) []ChangeEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	var out []ChangeEvent
	if !m.filled && m.index == 0 {
		return out
	}

	start := 0
	end := m.size
	if !m.filled {
		end = m.index
	} else {
		start = m.index
	}

	for i := 0; i < m.size; i++ {
		idx := (start + i) % m.size
		if idx == end && !m.filled {
			break
		}
		evt := m.buf[idx]
		if evt.At.IsZero() {
			continue
		}
		if evt.At.After(since) || evt.At.Equal(since) {
			out = append(out, evt)
		}
		if !m.filled && idx+1 == end {
			break
		}
	}
	return out
}
