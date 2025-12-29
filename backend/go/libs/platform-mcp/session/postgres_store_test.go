package session

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestPostgresStoreCreateGetEnd(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	s := Session{ID: "s1", TenantID: "t1", ExpiresAt: time.Now().Add(time.Minute), Attrs: map[string]string{"k": "v"}}
	attrs, _ := json.Marshal(s.Attrs)

	mock.ExpectExec("INSERT INTO mcp_sessions").WithArgs(s.ID, s.TenantID, pgxmock.AnyArg(), s.ExpiresAt, attrs).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectQuery("SELECT id, tenant_id").WithArgs(s.TenantID, s.ID).
		WillReturnRows(mock.NewRows([]string{"id", "tenant_id", "created_at", "expires_at", "attrs"}).
			AddRow(s.ID, s.TenantID, time.Now(), s.ExpiresAt, attrs))
	mock.ExpectExec("DELETE FROM mcp_sessions").WithArgs(s.TenantID, s.ID).WillReturnResult(pgxmock.NewResult("DELETE", 1))

	store := NewPostgresStore(mock)
	if _, err := store.Create(context.Background(), s); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := store.Get(context.Background(), s.TenantID, s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != s.ID || got.TenantID != s.TenantID {
		t.Fatalf("unexpected session: %+v", got)
	}
	if err := store.End(context.Background(), s.TenantID, s.ID); err != nil {
		t.Fatalf("end: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestPostgresStoreTouch(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	s := Session{ID: "s1", TenantID: "t1", ExpiresAt: time.Now().Add(time.Minute)}
	attrs, _ := json.Marshal(s.Attrs)

	newExp := time.Now().Add(2 * time.Minute)
	mock.ExpectQuery("UPDATE mcp_sessions SET expires_at").WithArgs(s.TenantID, s.ID, pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"expires_at"}).AddRow(newExp))
	mock.ExpectQuery("SELECT id, tenant_id").WithArgs(s.TenantID, s.ID).
		WillReturnRows(mock.NewRows([]string{"id", "tenant_id", "created_at", "expires_at", "attrs"}).
			AddRow(s.ID, s.TenantID, time.Now(), newExp, attrs))

	store := NewPostgresStore(mock)
	got, err := store.Touch(context.Background(), s.TenantID, s.ID, time.Minute)
	if err != nil {
		t.Fatalf("touch: %v", err)
	}
	if got.ExpiresAt.Before(newExp.Add(-time.Second)) {
		t.Fatalf("expected expiry extended")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestPostgresStoreCreateValidation(t *testing.T) {
	store := NewPostgresStore(nil)
	if _, err := store.Create(context.Background(), Session{}); err == nil {
		t.Fatalf("expected validation error for missing ids")
	}
}

func TestPostgresStoreTouchInvalidExtend(t *testing.T) {
	store := NewPostgresStore(nil)
	if _, err := store.Touch(context.Background(), "t1", "s1", 0); err == nil {
		t.Fatalf("expected extend error")
	}
}

func TestPostgresStoreGetExpired(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	attrs := []byte(`{"k":"v"}`)
	expired := time.Now().Add(-time.Minute)
	mock.ExpectQuery("SELECT id, tenant_id").WithArgs("t1", "s1").
		WillReturnRows(mock.NewRows([]string{"id", "tenant_id", "created_at", "expires_at", "attrs"}).
			AddRow("s1", "t1", time.Now().Add(-time.Hour), expired, attrs))
	mock.ExpectExec("DELETE FROM mcp_sessions").WithArgs("t1", "s1").WillReturnResult(pgxmock.NewResult("DELETE", 1))

	store := NewPostgresStore(mock)
	if _, err := store.Get(context.Background(), "t1", "s1"); err == nil || !errors.Is(err, errExpired) {
		t.Fatalf("expected expired error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestPostgresStoreCreateExecError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	s := Session{ID: "s1", TenantID: "t1"}
	mock.ExpectExec("INSERT INTO mcp_sessions").WithArgs(s.ID, s.TenantID, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("boom"))

	store := NewPostgresStore(mock)
	if _, err := store.Create(context.Background(), s); err == nil {
		t.Fatalf("expected exec error")
	}
}

func TestPostgresStoreGetQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT id, tenant_id").WithArgs("t1", "s1").WillReturnError(errors.New("boom"))
	store := NewPostgresStore(mock)
	if _, err := store.Get(context.Background(), "t1", "s1"); err == nil {
		t.Fatalf("expected query error")
	}
}

func TestPostgresStoreTouchQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery("UPDATE mcp_sessions SET expires_at").WithArgs("t1", "s1", pgxmock.AnyArg()).WillReturnError(errors.New("boom"))
	store := NewPostgresStore(mock)
	if _, err := store.Touch(context.Background(), "t1", "s1", time.Second); err == nil {
		t.Fatalf("expected touch query error")
	}
}
