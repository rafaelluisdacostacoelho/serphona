package events

import (
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/topics"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/types"
)

func TestUserCreatedEventBind(t *testing.T) {
	payload := UserCreatedEvent{
		UserID:    "user-123",
		TenantID:  "tenant-456",
		Email:     "user@example.com",
		Name:      "John Doe",
		Role:      "member",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}

	evt := NewEvent(topics.UserCreated, "auth-gateway", payload)
	decoded, err := types.Bind[UserCreatedEvent](evt)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.UserID != payload.UserID || decoded.TenantID != payload.TenantID || decoded.Email != payload.Email || decoded.Name != payload.Name || decoded.Role != payload.Role {
		t.Fatalf("Decoded payload mismatch: %+v", decoded)
	}
	if !decoded.CreatedAt.Equal(payload.CreatedAt) {
		t.Fatalf("CreatedAt mismatch: got %s want %s", decoded.CreatedAt, payload.CreatedAt)
	}
}

func TestPasswordResetEventBind(t *testing.T) {
	payload := PasswordResetEvent{
		UserID:   "user-123",
		TenantID: "tenant-456",
		ResetAt:  time.Now().UTC().Truncate(time.Second),
		ResetBy:  "user-ops",
		Method:   "admin_reset",
		TokenID:  "token-1",
	}

	evt := NewEvent(topics.PasswordReset, "auth-gateway", payload)
	decoded, err := types.Bind[PasswordResetEvent](evt)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.UserID != payload.UserID || decoded.TenantID != payload.TenantID || decoded.ResetBy != payload.ResetBy || decoded.Method != payload.Method || decoded.TokenID != payload.TokenID {
		t.Fatalf("Decoded payload mismatch: %+v", decoded)
	}
	if !decoded.ResetAt.Equal(payload.ResetAt) {
		t.Fatalf("ResetAt mismatch: got %s want %s", decoded.ResetAt, payload.ResetAt)
	}
}
