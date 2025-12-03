package call

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewCall(t *testing.T) {
	tenantID := uuid.New()
	callerNumber := "+5511999887766"
	calleeNumber := "+5511988776655"

	call := NewCall(tenantID, DirectionInbound, callerNumber, calleeNumber)

	if call.ID == uuid.Nil {
		t.Error("Call ID should not be nil")
	}

	if call.TenantID != tenantID {
		t.Errorf("Expected TenantID %s, got %s", tenantID, call.TenantID)
	}

	if call.CallerNumber != callerNumber {
		t.Errorf("Expected CallerNumber %s, got %s", callerNumber, call.CallerNumber)
	}

	if call.CalleeNumber != calleeNumber {
		t.Errorf("Expected CalleeNumber %s, got %s", calleeNumber, call.CalleeNumber)
	}

	if call.Direction != DirectionInbound {
		t.Errorf("Expected Direction %s, got %s", DirectionInbound, call.Direction)
	}

	if call.State != StateRinging {
		t.Errorf("Expected State %s, got %s", StateRinging, call.State)
	}

	if call.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	if call.Metadata == nil {
		t.Error("Metadata map should be initialized")
	}
}

func TestCall_Answer(t *testing.T) {
	call := NewCall(uuid.New(), DirectionInbound, "+5511999887766", "+5511988776655")

	if call.State != StateRinging {
		t.Errorf("Initial state should be %s, got %s", StateRinging, call.State)
	}

	call.Answer()

	if call.State != StateAnswered {
		t.Errorf("After Answer, state should be %s, got %s", StateAnswered, call.State)
	}

	if call.AnsweredAt == nil {
		t.Error("AnsweredAt should be set after Answer")
	}
}

func TestCall_Activate(t *testing.T) {
	call := NewCall(uuid.New(), DirectionInbound, "+5511999887766", "+5511988776655")
	call.Answer()

	call.Activate()

	if call.State != StateActive {
		t.Errorf("After Activate, state should be %s, got %s", StateActive, call.State)
	}
}

func TestCall_End(t *testing.T) {
	call := NewCall(uuid.New(), DirectionInbound, "+5511999887766", "+5511988776655")
	call.Answer()

	time.Sleep(10 * time.Millisecond) // Simulate call duration

	call.End()

	if call.State != StateEnded {
		t.Errorf("After End, state should be %s, got %s", StateEnded, call.State)
	}

	if call.EndedAt == nil {
		t.Error("EndedAt should be set after End")
	}

	if call.Duration <= 0 {
		t.Error("Duration should be greater than 0 after End")
	}
}

func TestCall_HoldAndResume(t *testing.T) {
	call := NewCall(uuid.New(), DirectionInbound, "+5511999887766", "+5511988776655")
	call.Answer()
	call.Activate()

	call.Hold()
	if call.State != StateHold {
		t.Errorf("After Hold, state should be %s, got %s", StateHold, call.State)
	}

	call.Resume()
	if call.State != StateActive {
		t.Errorf("After Resume, state should be %s, got %s", StateActive, call.State)
	}
}

func TestCall_Transfer(t *testing.T) {
	call := NewCall(uuid.New(), DirectionInbound, "+5511999887766", "+5511988776655")
	call.Answer()

	call.Transfer()

	if call.State != StateTransferred {
		t.Errorf("After Transfer, state should be %s, got %s", StateTransferred, call.State)
	}
}

func TestCall_SetError(t *testing.T) {
	call := NewCall(uuid.New(), DirectionInbound, "+5511999887766", "+5511988776655")

	call.SetError()

	if call.State != StateError {
		t.Errorf("After SetError, state should be %s, got %s", StateError, call.State)
	}

	if call.EndedAt == nil {
		t.Error("EndedAt should be set after SetError")
	}
}

func TestCall_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected bool
	}{
		{"Active", StateActive, true},
		{"Answered", StateAnswered, true},
		{"Hold", StateHold, true},
		{"Ringing", StateRinging, false},
		{"Ended", StateEnded, false},
		{"Error", StateError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := NewCall(uuid.New(), DirectionInbound, "+5511999887766", "+5511988776655")
			call.State = tt.state

			if call.IsActive() != tt.expected {
				t.Errorf("IsActive() for state %s: expected %v, got %v", tt.state, tt.expected, call.IsActive())
			}
		})
	}
}

func TestCall_IsEnded(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected bool
	}{
		{"Ended", StateEnded, true},
		{"Error", StateError, true},
		{"Active", StateActive, false},
		{"Ringing", StateRinging, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call := NewCall(uuid.New(), DirectionInbound, "+5511999887766", "+5511988776655")
			call.State = tt.state

			if call.IsEnded() != tt.expected {
				t.Errorf("IsEnded() for state %s: expected %v, got %v", tt.state, tt.expected, call.IsEnded())
			}
		})
	}
}

func TestCall_Metadata(t *testing.T) {
	call := NewCall(uuid.New(), DirectionInbound, "+5511999887766", "+5511988776655")

	key := "customer_name"
	value := "John Doe"

	call.Metadata[key] = value

	if call.Metadata[key] != value {
		t.Errorf("Expected Metadata[%s] = %v, got %v", key, value, call.Metadata[key])
	}
}
