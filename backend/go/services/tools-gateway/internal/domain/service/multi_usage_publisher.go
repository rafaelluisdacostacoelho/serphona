package service

import (
	"context"
	"fmt"
	"strings"
)

// MultiUsagePublisher fan-outs events to multiple sinks, returning a combined error if any fail.
type MultiUsagePublisher struct {
	sinks []UsagePublisher
}

func NewMultiUsagePublisher(sinks ...UsagePublisher) UsagePublisher {
	var filtered []UsagePublisher
	for _, s := range sinks {
		if s != nil {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		return NewNoopUsagePublisher()
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return &MultiUsagePublisher{sinks: filtered}
}

func (m *MultiUsagePublisher) PublishUsage(ctx context.Context, evt UsageEvent) error {
	var errs []string
	for _, s := range m.sinks {
		if err := s.PublishUsage(ctx, evt); err != nil {
			errs = append(errs, strings.TrimSpace(err.Error()))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("usage publish errors: %s", strings.Join(errs, "; "))
}
