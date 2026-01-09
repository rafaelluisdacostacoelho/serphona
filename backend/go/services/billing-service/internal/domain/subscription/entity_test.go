package subscription

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSubscriptionTableName(t *testing.T) {
	if got := (Subscription{}).TableName(); got != "subscriptions" {
		t.Fatalf("unexpected table name %s", got)
	}
}

func TestNewSubscriptionDefaults(t *testing.T) {
	customerID := uuid.New()
	sub := NewSubscription(customerID, "plan-basic")

	if sub.CustomerID != customerID || sub.PlanID != "plan-basic" {
		t.Fatalf("unexpected subscription fields %+v", sub)
	}
	if sub.Status != StatusIncomplete {
		t.Fatalf("expected status incomplete, got %s", sub.Status)
	}
	if sub.CancelAtPeriodEnd {
		t.Fatalf("expected cancel at period end to be false")
	}
	if sub.Metadata == nil {
		t.Fatalf("expected metadata map to be initialized")
	}
}

func TestSubscriptionLifecycle(t *testing.T) {
	sub := NewSubscription(uuid.New(), "plan-pro")

	sub.SetStripeSubscriptionID("sub_123")
	if sub.StripeSubscriptionID != "sub_123" {
		t.Fatalf("expected stripe subscription id to be set")
	}

	start := time.Now().Add(-time.Hour)
	end := time.Now().Add(time.Hour)
	sub.Activate(start, end)
	if sub.Status != StatusActive || !sub.CurrentPeriodStart.Equal(start) || !sub.CurrentPeriodEnd.Equal(end) {
		t.Fatalf("activate did not set fields correctly: %+v", sub)
	}

	sub.Cancel(true)
	if !sub.CancelAtPeriodEnd {
		t.Fatalf("expected cancel at period end flag set")
	}

	sub.Cancel(false)
	if sub.Status != StatusCanceled || sub.CanceledAt == nil {
		t.Fatalf("expected immediate cancel to set status and timestamp")
	}
}

func TestSubscriptionStatusAndTrial(t *testing.T) {
	sub := NewSubscription(uuid.New(), "plan-enterprise")

	sub.UpdateStatus(StatusPastDue)
	if sub.Status != StatusPastDue {
		t.Fatalf("expected status to be updated")
	}
	if sub.IsActive() {
		t.Fatalf("expected past due to not be active")
	}

	sub.UpdateStatus(StatusActive)
	if !sub.IsActive() {
		t.Fatalf("expected active status to be considered active")
	}

	trialEnd := time.Now().Add(48 * time.Hour)
	sub.StartTrial(trialEnd)
	if sub.Status != StatusTrialing || sub.TrialStart == nil || sub.TrialEnd == nil {
		t.Fatalf("expected trial fields set")
	}
	if !sub.TrialEnd.Equal(trialEnd) {
		t.Fatalf("unexpected trial end %v", sub.TrialEnd)
	}
}
