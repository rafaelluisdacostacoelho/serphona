package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockClaims struct {
	UserID string `json:"userId"`
}

func TestNewFromEnv(t *testing.T) {
	t.Setenv(EnvAuthGatewayURL, "")
	if _, err := NewFromEnv(); err == nil {
		t.Fatalf("expected error when %s is missing", EnvAuthGatewayURL)
	}

	t.Setenv(EnvAuthGatewayURL, "http://auth-gateway:8080")
	cl, err := NewFromEnv()
	if err != nil {
		t.Fatalf("expected client to be created, got: %v", err)
	}

	if cl == nil {
		t.Fatalf("expected client instance, got nil")
	}
}

func TestValidateTokenCallsGateway(t *testing.T) {
	var receivedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(mockClaims{UserID: "123"})
	}))
	defer ts.Close()

	cl := New(ts.URL)

	claims, err := cl.ValidateToken("abc")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if claims.UserID != "123" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if receivedAuth != "Bearer abc" {
		t.Fatalf("expected Authorization header, got %s", receivedAuth)
	}
}

func TestGetUserByIDHandles404And401(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch calls {
		case 1:
			w.WriteHeader(http.StatusNotFound)
		case 2:
			w.WriteHeader(http.StatusUnauthorized)
		default:
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "u1"})
		}
	}))
	defer ts.Close()

	cl := New(ts.URL)

	if _, err := cl.GetUserByID("id", "token"); err == nil {
		t.Fatalf("expected not found error")
	}
	if _, err := cl.GetUserByID("id", "token"); err == nil {
		t.Fatalf("expected unauthorized error")
	}
	user, err := cl.GetUserByID("id", "token")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if user.ID != "u1" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestRefreshTokenAndLogout(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{"accessToken": "new", "refreshToken": "r", "expiresIn": 60})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	cl := New(ts.URL)

	if _, err := cl.RefreshToken("refresh"); err != nil {
		t.Fatalf("expected refresh success, got %v", err)
	}
	if err := cl.Logout("token"); err != nil {
		t.Fatalf("expected logout success, got %v", err)
	}
}
