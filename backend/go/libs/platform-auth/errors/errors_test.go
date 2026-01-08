package errors_test

import (
	"errors"
	"testing"

	platformErrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
)

func TestAuthError_Error(t *testing.T) {
	tests := []struct {
		name     string
		input    *platformErrors.AuthError
		expected string
	}{
		{
			name: "Error with wrapped error",
			input: &platformErrors.AuthError{
				Code:    platformErrors.CodeInvalidToken,
				Message: "Invalid token provided",
				Err:     errors.New("token signature mismatch"),
			},
			expected: "Invalid token provided: token signature mismatch",
		},
		{
			name: "Error without wrapped error",
			input: &platformErrors.AuthError{
				Code:    platformErrors.CodeUnauthorized,
				Message: "Unauthorized access",
			},
			expected: "Unauthorized access",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.Error(); got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAuthError_Unwrap(t *testing.T) {
	wrappedErr := errors.New("wrapped error")
	authErr := &platformErrors.AuthError{
		Code:    platformErrors.CodeInvalidToken,
		Message: "Invalid token",
		Err:     wrappedErr,
	}

	if unwrapped := errors.Unwrap(authErr); unwrapped != wrappedErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, wrappedErr)
	}
}

func TestNewAuthError(t *testing.T) {
	code := platformErrors.CodeInvalidToken
	message := "Invalid token"
	wrappedErr := errors.New("wrapped error")
	authErr := platformErrors.NewAuthError(code, message, wrappedErr)

	if authErr.Code != code {
		t.Errorf("NewAuthError() Code = %v, want %v", authErr.Code, code)
	}
	if authErr.Message != message {
		t.Errorf("NewAuthError() Message = %v, want %v", authErr.Message, message)
	}
	if authErr.Err != wrappedErr {
		t.Errorf("NewAuthError() Err = %v, want %v", authErr.Err, wrappedErr)
	}
}
