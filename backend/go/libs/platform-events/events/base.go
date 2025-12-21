package events

import "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/types"

// NewEvent wraps types.NewEvent to keep callers on the events package.
func NewEvent(eventType, source string, payload interface{}) *types.Event {
	return types.NewEvent(eventType, source, payload)
}
