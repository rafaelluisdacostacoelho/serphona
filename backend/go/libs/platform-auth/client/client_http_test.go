package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

func newTestServer(t *testing.T) (*httptest.Server, *http.ServeMux) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, mux
}

func TestValidateTokenUnauthorized(t *testing.T) {
	srv, mux := newTestServer(t)
	mux.HandleFunc("/api/v1/auth/validate", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	c := New(srv.URL)
	_, err := c.ValidateToken("tok")
	if !errors.Is(err, autherrors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestValidateTokenHTTPError(t *testing.T) {
	srv, mux := newTestServer(t)
	mux.HandleFunc("/api/v1/auth/validate", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	})

	c := New(srv.URL)
	_, err := c.ValidateToken("tok")
	if err == nil {
		t.Fatalf("expected error")
	}
	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != http.StatusInternalServerError || httpErr.Body != "boom" {
		t.Fatalf("unexpected http error: %+v", httpErr)
	}
}

func TestGetMeSuccess(t *testing.T) {
	srv, mux := newTestServer(t)
	mux.HandleFunc("/api/v1/auth/me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(types.User{ID: "u1", Email: "e", Name: "n", Role: "user", TenantID: "t"})
	})

	c := New(srv.URL)
	user, err := c.GetMe("tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "u1" || user.TenantID != "t" {
		t.Fatalf("unexpected user %+v", user)
	}
}

func TestGetUserByIDNotFound(t *testing.T) {
	srv, mux := newTestServer(t)
	mux.HandleFunc("/api/v1/auth/users/abc", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	c := New(srv.URL)
	_, err := c.GetUserByID("abc", "tok")
	if !errors.Is(err, autherrors.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestRefreshTokenUnauthorized(t *testing.T) {
	srv, mux := newTestServer(t)
	mux.HandleFunc("/api/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	c := New(srv.URL)
	_, err := c.RefreshToken("rt")
	if !errors.Is(err, autherrors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestRefreshTokenSuccess(t *testing.T) {
	srv, mux := newTestServer(t)
	mux.HandleFunc("/api/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(types.TokenResponse{AccessToken: "a", RefreshToken: "r", ExpiresIn: 60})
	})

	c := New(srv.URL)
	tok, err := c.RefreshToken("rt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "a" || tok.RefreshToken != "r" {
		t.Fatalf("unexpected token %+v", tok)
	}
}

func TestLogoutUnauthorized(t *testing.T) {
	srv, mux := newTestServer(t)
	mux.HandleFunc("/api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	c := New(srv.URL)
	err := c.Logout("tok")
	if !errors.Is(err, autherrors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLogoutSuccess(t *testing.T) {
	srv, mux := newTestServer(t)
	mux.HandleFunc("/api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	c := New(srv.URL)
	if err := c.Logout("tok"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
