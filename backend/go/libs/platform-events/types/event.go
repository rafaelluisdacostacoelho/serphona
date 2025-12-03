package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event representa um evento no sistema
type Event struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Source    string            `json:"source"`
	Timestamp time.Time         `json:"timestamp"`
	TenantID  string            `json:"tenant_id,omitempty"`
	UserID    string            `json:"user_id,omitempty"`
	Data      interface{}       `json:"data"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	TraceID   string            `json:"trace_id,omitempty"`
	SpanID    string            `json:"span_id,omitempty"`
	Version   string            `json:"version"`
}

// NewEvent cria um novo evento com valores padrão
func NewEvent(eventType, source string, data interface{}) *Event {
	return &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().UTC(),
		Data:      data,
		Metadata:  make(map[string]string),
		Version:   "1.0",
	}
}

// WithTenantID adiciona tenant ID ao evento
func (e *Event) WithTenantID(tenantID string) *Event {
	e.TenantID = tenantID
	return e
}

// WithUserID adiciona user ID ao evento
func (e *Event) WithUserID(userID string) *Event {
	e.UserID = userID
	return e
}

// WithMetadata adiciona metadata ao evento
func (e *Event) WithMetadata(key, value string) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// WithTrace adiciona trace context ao evento
func (e *Event) WithTrace(traceID, spanID string) *Event {
	e.TraceID = traceID
	e.SpanID = spanID
	return e
}

// ToJSON serializa o evento para JSON
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON desserializa o evento de JSON
func FromJSON(data []byte) (*Event, error) {
	var event Event
	err := json.Unmarshal(data, &event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// EventHandler é a função que processa eventos
type EventHandler func(*Event) error

// EventFilter permite filtrar eventos antes de processar
type EventFilter func(*Event) bool
