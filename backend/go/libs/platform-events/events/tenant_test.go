package events

import (
	"testing"
	"time"

	"github.com/serphona/serphona/backend/go/libs/platform-events/topics"
	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

func TestTenantMemberAddedEventBind(t *testing.T) {
	payload := TenantMemberAddedEvent{
		TenantID:     "tenant-123",
		MemberID:     "user-456",
		Email:        "member@example.com",
		Role:         "admin",
		AddedAt:      time.Now().UTC().Truncate(time.Second),
		AddedBy:      "user-789",
		InvitationID: "invite-abc",
		Source:       "tenant-manager",
	}

	event := NewEvent(topics.TenantMemberAdded, "tenant-manager", payload)

	decoded, err := types.Bind[TenantMemberAddedEvent](event)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.TenantID != payload.TenantID {
		t.Fatalf("TenantID mismatch: got %s want %s", decoded.TenantID, payload.TenantID)
	}
	if decoded.MemberID != payload.MemberID {
		t.Fatalf("MemberID mismatch: got %s want %s", decoded.MemberID, payload.MemberID)
	}
	if decoded.Email != payload.Email {
		t.Fatalf("Email mismatch: got %s want %s", decoded.Email, payload.Email)
	}
	if decoded.Role != payload.Role {
		t.Fatalf("Role mismatch: got %s want %s", decoded.Role, payload.Role)
	}
	if !decoded.AddedAt.Equal(payload.AddedAt) {
		t.Fatalf("AddedAt mismatch: got %s want %s", decoded.AddedAt, payload.AddedAt)
	}
	if decoded.AddedBy != payload.AddedBy {
		t.Fatalf("AddedBy mismatch: got %s want %s", decoded.AddedBy, payload.AddedBy)
	}
	if decoded.InvitationID != payload.InvitationID {
		t.Fatalf("InvitationID mismatch: got %s want %s", decoded.InvitationID, payload.InvitationID)
	}
	if decoded.Source != payload.Source {
		t.Fatalf("Source mismatch: got %s want %s", decoded.Source, payload.Source)
	}
}

func TestTenantSuspendedEventBind(t *testing.T) {
	expires := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	payload := TenantSuspendedEvent{
		TenantID:    "tenant-123",
		SuspendedAt: time.Now().UTC().Truncate(time.Second),
		SuspendedBy: "user-321",
		Reason:      "payment_failed",
		ExpiresAt:   &expires,
	}

	event := NewEvent(topics.TenantSuspended, "tenant-manager", payload)
	decoded, err := types.Bind[TenantSuspendedEvent](event)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.TenantID != payload.TenantID {
		t.Fatalf("TenantID mismatch: got %s want %s", decoded.TenantID, payload.TenantID)
	}
	if !decoded.SuspendedAt.Equal(payload.SuspendedAt) {
		t.Fatalf("SuspendedAt mismatch: got %s want %s", decoded.SuspendedAt, payload.SuspendedAt)
	}
	if decoded.SuspendedBy != payload.SuspendedBy {
		t.Fatalf("SuspendedBy mismatch: got %s want %s", decoded.SuspendedBy, payload.SuspendedBy)
	}
	if decoded.Reason != payload.Reason {
		t.Fatalf("Reason mismatch: got %s want %s", decoded.Reason, payload.Reason)
	}
	if decoded.ExpiresAt == nil {
		t.Fatalf("ExpiresAt should not be nil")
	}
	if !decoded.ExpiresAt.Equal(expires) {
		t.Fatalf("ExpiresAt mismatch: got %s want %s", decoded.ExpiresAt, expires)
	}
}
