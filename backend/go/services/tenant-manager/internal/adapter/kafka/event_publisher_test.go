package kafka

import (
	"context"
	"errors"
	"testing"

	"tenant-manager/internal/domain/tenant"
)

type fakeProducer struct {
	calls []string
	fail  map[string]error
}

func (p *fakeProducer) SendMessage(_ context.Context, topic string, _ []byte, _ []byte) error {
	p.calls = append(p.calls, topic)
	if err, ok := p.fail[topic]; ok {
		return err
	}
	return nil
}

func TestEventPublisherDLQBehavior(t *testing.T) {
	t.Parallel()

	topicPrefix := "tm"
	dlqTopic := "tm.dlq"

	tests := []struct {
		name        string
		failTopics  map[string]error
		dlqTopic    string
		expectErr   bool
		expectCalls []string
	}{
		{
			name:       "success_no_dlq",
			failTopics: nil,
			dlqTopic:   "",
			expectCalls: []string{
				topicPrefix + ".tenant.updated",
			},
		},
		{
			name: "primary_fail_dlq_success",
			failTopics: map[string]error{
				topicPrefix + ".tenant.updated": errors.New("primary down"),
			},
			dlqTopic: dlqTopic,
			expectCalls: []string{
				topicPrefix + ".tenant.updated",
				dlqTopic,
			},
		},
		{
			name: "primary_and_dlq_fail",
			failTopics: map[string]error{
				topicPrefix + ".tenant.updated": errors.New("primary down"),
				dlqTopic:                        errors.New("dlq down"),
			},
			dlqTopic:  dlqTopic,
			expectErr: true,
			expectCalls: []string{
				topicPrefix + ".tenant.updated",
				dlqTopic,
			},
		},
		{
			name: "primary_fail_no_dlq",
			failTopics: map[string]error{
				topicPrefix + ".tenant.updated": errors.New("primary down"),
			},
			dlqTopic:  "",
			expectErr: true,
			expectCalls: []string{
				topicPrefix + ".tenant.updated",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fp := &fakeProducer{fail: tt.failTopics}
			publisher := NewEventPublisher(fp, topicPrefix, tt.dlqTopic)

			err := publisher.PublishUpdated(context.Background(), mustTenant())

			if tt.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("expected success, got %v", err)
			}

			if len(fp.calls) != len(tt.expectCalls) {
				t.Fatalf("call count mismatch: got %v want %v", fp.calls, tt.expectCalls)
			}
			for i := range tt.expectCalls {
				if fp.calls[i] != tt.expectCalls[i] {
					t.Fatalf("call %d mismatch: got %s want %s", i, fp.calls[i], tt.expectCalls[i])
				}
			}
		})
	}
}

func mustTenant() *tenant.Tenant {
	t := tenant.NewTenant("Acme", "acme@example.com", tenant.PlanStarter)
	return t
}
