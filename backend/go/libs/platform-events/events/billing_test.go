package events

import (
	"testing"
	"time"

	"github.com/serphona/serphona/backend/go/libs/platform-events/topics"
	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

func TestSubscriptionCancelledEventBind(t *testing.T) {
	payload := SubscriptionCancelledEvent{
		SubscriptionID: "sub-123",
		TenantID:       "tenant-456",
		Reason:         "user_request",
		CancelledAt:    time.Now().UTC().Truncate(time.Second),
		CancelledBy:    "user-789",
		Refunded:       true,
	}

	event := NewEvent(topics.SubscriptionCancelled, "billing-service", payload)
	decoded, err := types.Bind[SubscriptionCancelledEvent](event)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.SubscriptionID != payload.SubscriptionID {
		t.Fatalf("SubscriptionID mismatch: got %s want %s", decoded.SubscriptionID, payload.SubscriptionID)
	}
	if decoded.TenantID != payload.TenantID {
		t.Fatalf("TenantID mismatch: got %s want %s", decoded.TenantID, payload.TenantID)
	}
	if decoded.Reason != payload.Reason {
		t.Fatalf("Reason mismatch: got %s want %s", decoded.Reason, payload.Reason)
	}
	if !decoded.CancelledAt.Equal(payload.CancelledAt) {
		t.Fatalf("CancelledAt mismatch: got %s want %s", decoded.CancelledAt, payload.CancelledAt)
	}
	if decoded.CancelledBy != payload.CancelledBy {
		t.Fatalf("CancelledBy mismatch: got %s want %s", decoded.CancelledBy, payload.CancelledBy)
	}
	if decoded.Refunded != payload.Refunded {
		t.Fatalf("Refunded mismatch: got %t want %t", decoded.Refunded, payload.Refunded)
	}
}

func TestPaymentFailedEventBind(t *testing.T) {
	payload := PaymentFailedEvent{
		PaymentID:     "pay-123",
		TenantID:      "tenant-456",
		AmountCents:   1299,
		Currency:      "USD",
		FailureCode:   "card_declined",
		FailureReason: "insufficient_funds",
		FailedAt:      time.Now().UTC().Truncate(time.Second),
		Retryable:     true,
	}

	event := NewEvent(topics.PaymentFailed, "billing-service", payload)
	decoded, err := types.Bind[PaymentFailedEvent](event)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.PaymentID != payload.PaymentID {
		t.Fatalf("PaymentID mismatch: got %s want %s", decoded.PaymentID, payload.PaymentID)
	}
	if decoded.TenantID != payload.TenantID {
		t.Fatalf("TenantID mismatch: got %s want %s", decoded.TenantID, payload.TenantID)
	}
	if decoded.AmountCents != payload.AmountCents {
		t.Fatalf("AmountCents mismatch: got %d want %d", decoded.AmountCents, payload.AmountCents)
	}
	if decoded.Currency != payload.Currency {
		t.Fatalf("Currency mismatch: got %s want %s", decoded.Currency, payload.Currency)
	}
	if decoded.FailureCode != payload.FailureCode {
		t.Fatalf("FailureCode mismatch: got %s want %s", decoded.FailureCode, payload.FailureCode)
	}
	if decoded.FailureReason != payload.FailureReason {
		t.Fatalf("FailureReason mismatch: got %s want %s", decoded.FailureReason, payload.FailureReason)
	}
	if !decoded.FailedAt.Equal(payload.FailedAt) {
		t.Fatalf("FailedAt mismatch: got %s want %s", decoded.FailedAt, payload.FailedAt)
	}
	if decoded.Retryable != payload.Retryable {
		t.Fatalf("Retryable mismatch: got %t want %t", decoded.Retryable, payload.Retryable)
	}
}

func TestInvoiceGeneratedEventBind(t *testing.T) {
	periodStart := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2025, time.January, 31, 23, 59, 59, 0, time.UTC)
	payload := InvoiceGeneratedEvent{
		InvoiceID:   "inv-123",
		TenantID:    "tenant-456",
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		AmountCents: 4599,
		Currency:    "USD",
		DueDate:     periodEnd.Add(7 * 24 * time.Hour),
		Link:        "https://billing.serphona.com/invoices/inv-123",
		GeneratedAt: time.Now().UTC().Truncate(time.Second),
		GeneratedBy: "system",
	}

	event := NewEvent(topics.InvoiceGenerated, "billing-service", payload)
	decoded, err := types.Bind[InvoiceGeneratedEvent](event)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.InvoiceID != payload.InvoiceID {
		t.Fatalf("InvoiceID mismatch: got %s want %s", decoded.InvoiceID, payload.InvoiceID)
	}
	if decoded.TenantID != payload.TenantID {
		t.Fatalf("TenantID mismatch: got %s want %s", decoded.TenantID, payload.TenantID)
	}
	if !decoded.PeriodStart.Equal(payload.PeriodStart) {
		t.Fatalf("PeriodStart mismatch: got %s want %s", decoded.PeriodStart, payload.PeriodStart)
	}
	if !decoded.PeriodEnd.Equal(payload.PeriodEnd) {
		t.Fatalf("PeriodEnd mismatch: got %s want %s", decoded.PeriodEnd, payload.PeriodEnd)
	}
	if decoded.AmountCents != payload.AmountCents {
		t.Fatalf("AmountCents mismatch: got %d want %d", decoded.AmountCents, payload.AmountCents)
	}
	if decoded.Currency != payload.Currency {
		t.Fatalf("Currency mismatch: got %s want %s", decoded.Currency, payload.Currency)
	}
	if !decoded.DueDate.Equal(payload.DueDate) {
		t.Fatalf("DueDate mismatch: got %s want %s", decoded.DueDate, payload.DueDate)
	}
	if decoded.Link != payload.Link {
		t.Fatalf("Link mismatch: got %s want %s", decoded.Link, payload.Link)
	}
	if !decoded.GeneratedAt.Equal(payload.GeneratedAt) {
		t.Fatalf("GeneratedAt mismatch: got %s want %s", decoded.GeneratedAt, payload.GeneratedAt)
	}
	if decoded.GeneratedBy != payload.GeneratedBy {
		t.Fatalf("GeneratedBy mismatch: got %s want %s", decoded.GeneratedBy, payload.GeneratedBy)
	}
}
