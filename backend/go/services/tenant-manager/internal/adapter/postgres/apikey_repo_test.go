package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	pgxmock "github.com/pashagolub/pgxmock/v4"
	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"

	"tenant-manager/internal/domain/apikey"
)

func TestAPIKeyRepository_Save_TenantValidation(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()
	key := &apikey.APIKey{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        "test",
		KeyPrefix:   "prefix",
		KeyHash:     "hash",
		Permissions: []string{"tenants:write"},
		Metadata:    apikey.Metadata{RateLimit: 100},
		ExpiresAt:   ptrTime(now.Add(time.Hour)),
		CreatedAt:   now,
	}

	cases := []struct {
		name       string
		ctx        context.Context
		expectErr  error
		expectExec bool
	}{
		{name: "no tenant context", ctx: context.Background(), expectErr: autherrors.ErrUnauthorized},
		{name: "foreign tenant", ctx: authmw.WithTenantID(context.Background(), uuid.NewString()), expectErr: autherrors.ErrInsufficientPermissions},
		{name: "matching tenant", ctx: authmw.WithTenantID(context.Background(), tenantID.String()), expectExec: true},
		{name: "platform tenant", ctx: authmw.WithTenantID(context.Background(), "platform"), expectExec: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock: %v", err)
			}
			defer mock.Close()

			repo := NewAPIKeyRepository(mock)

			if tc.expectExec {
				mock.ExpectExec("INSERT INTO api_keys").
					WithArgs(
						key.ID,
						key.TenantID,
						key.Name,
						key.KeyHash,
						key.KeyPrefix,
						key.Permissions,
						key.Metadata.RateLimit,
						key.ExpiresAt,
						key.CreatedAt,
						key.LastUsedAt,
						key.RevokedAt,
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			}

			err = repo.Save(tc.ctx, key)
			if tc.expectErr != nil {
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("expected error %v, got %v", tc.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations were not met: %v", err)
			}
		})
	}
}

func TestAPIKeyRepository_FindByID_TenantValidation(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()
	expiresAt := now.Add(time.Hour)

	key := &apikey.APIKey{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        "test",
		KeyPrefix:   "prefix",
		KeyHash:     "hash",
		Permissions: []string{"tenants:write"},
		Metadata:    apikey.Metadata{RateLimit: 50},
		ExpiresAt:   &expiresAt,
		CreatedAt:   now,
	}

	cases := []struct {
		name      string
		ctx       context.Context
		expectErr error
	}{
		{name: "no tenant context", ctx: context.Background(), expectErr: autherrors.ErrUnauthorized},
		{name: "foreign tenant", ctx: authmw.WithTenantID(context.Background(), uuid.NewString()), expectErr: autherrors.ErrInsufficientPermissions},
		{name: "matching tenant", ctx: authmw.WithTenantID(context.Background(), tenantID.String())},
		{name: "platform tenant", ctx: authmw.WithTenantID(context.Background(), "platform")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock: %v", err)
			}
			defer mock.Close()

			rows := pgxmock.NewRows([]string{
				"id", "tenant_id", "name", "key_hash", "key_prefix", "scopes", "rate_limit", "expires_at", "last_used_at", "created_at", "revoked_at",
			}).AddRow(
				key.ID,
				key.TenantID,
				key.Name,
				key.KeyHash,
				key.KeyPrefix,
				key.Permissions,
				key.Metadata.RateLimit,
				key.ExpiresAt,
				key.LastUsedAt,
				key.CreatedAt,
				key.RevokedAt,
			)

			mock.ExpectQuery("SELECT id, tenant_id, name").WithArgs(key.ID).WillReturnRows(rows)

			repo := NewAPIKeyRepository(mock)
			_, err = repo.FindByID(tc.ctx, key.ID)

			if tc.expectErr != nil {
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("expected error %v, got %v", tc.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations were not met: %v", err)
			}
		})
	}
}

func TestAPIKeyRepository_CountByTenantID_TenantValidation(t *testing.T) {
	tenantID := uuid.New()

	cases := []struct {
		name       string
		ctx        context.Context
		expectErr  error
		expectExec bool
	}{
		{name: "no tenant context", ctx: context.Background(), expectErr: autherrors.ErrUnauthorized},
		{name: "foreign tenant", ctx: authmw.WithTenantID(context.Background(), uuid.NewString()), expectErr: autherrors.ErrInsufficientPermissions},
		{name: "matching tenant", ctx: authmw.WithTenantID(context.Background(), tenantID.String()), expectExec: true},
		{name: "platform tenant", ctx: authmw.WithTenantID(context.Background(), "platform"), expectExec: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock: %v", err)
			}
			defer mock.Close()

			repo := NewAPIKeyRepository(mock)

			if tc.expectExec {
				rows := pgxmock.NewRows([]string{"count"}).AddRow(int64(2))
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM api_keys").WithArgs(tenantID).WillReturnRows(rows)
			}

			_, err = repo.CountByTenantID(tc.ctx, tenantID)
			if tc.expectErr != nil {
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("expected error %v, got %v", tc.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations were not met: %v", err)
			}
		})
	}
}

func TestAPIKeyRepository_Delete_TenantValidation(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()
	expiresAt := now.Add(time.Hour)

	key := &apikey.APIKey{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        "test",
		KeyPrefix:   "prefix",
		KeyHash:     "hash",
		Permissions: []string{"tenants:write"},
		Metadata:    apikey.Metadata{RateLimit: 50},
		ExpiresAt:   &expiresAt,
		CreatedAt:   now,
	}

	cases := []struct {
		name       string
		ctx        context.Context
		expectErr  error
		expectExec bool
		expectRows bool
	}{
		{name: "no tenant context", ctx: context.Background(), expectErr: autherrors.ErrUnauthorized, expectRows: true},
		{name: "foreign tenant", ctx: authmw.WithTenantID(context.Background(), uuid.NewString()), expectErr: autherrors.ErrInsufficientPermissions, expectRows: true},
		{name: "matching tenant", ctx: authmw.WithTenantID(context.Background(), tenantID.String()), expectExec: true, expectRows: true},
		{name: "platform tenant", ctx: authmw.WithTenantID(context.Background(), "platform"), expectExec: true, expectRows: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock: %v", err)
			}
			defer mock.Close()

			repo := NewAPIKeyRepository(mock)

			if tc.expectRows {
				rows := pgxmock.NewRows([]string{
					"id", "tenant_id", "name", "key_hash", "key_prefix", "scopes", "rate_limit", "expires_at", "last_used_at", "created_at", "revoked_at",
				}).AddRow(
					key.ID,
					key.TenantID,
					key.Name,
					key.KeyHash,
					key.KeyPrefix,
					key.Permissions,
					key.Metadata.RateLimit,
					key.ExpiresAt,
					key.LastUsedAt,
					key.CreatedAt,
					key.RevokedAt,
				)

				mock.ExpectQuery("SELECT id, tenant_id, name").WithArgs(key.ID).WillReturnRows(rows)
			}

			if tc.expectExec {
				mock.ExpectExec("DELETE FROM api_keys").WithArgs(key.ID).WillReturnResult(pgxmock.NewResult("DELETE", 1))
			}

			err = repo.Delete(tc.ctx, key.ID)
			if tc.expectErr != nil {
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("expected error %v, got %v", tc.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations were not met: %v", err)
			}
		})
	}
}

func TestAPIKeyRepository_ExistsByName_TenantValidation(t *testing.T) {
	tenantID := uuid.New()

	cases := []struct {
		name       string
		ctx        context.Context
		expectErr  error
		expectExec bool
	}{
		{name: "no tenant context", ctx: context.Background(), expectErr: autherrors.ErrUnauthorized},
		{name: "foreign tenant", ctx: authmw.WithTenantID(context.Background(), uuid.NewString()), expectErr: autherrors.ErrInsufficientPermissions},
		{name: "matching tenant", ctx: authmw.WithTenantID(context.Background(), tenantID.String()), expectExec: true},
		{name: "platform tenant", ctx: authmw.WithTenantID(context.Background(), "platform"), expectExec: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock: %v", err)
			}
			defer mock.Close()

			repo := NewAPIKeyRepository(mock)

			if tc.expectExec {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery("SELECT EXISTS\\(").WithArgs(tenantID, "name").WillReturnRows(rows)
			}

			_, err = repo.ExistsByName(tc.ctx, tenantID, "name")
			if tc.expectErr != nil {
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("expected error %v, got %v", tc.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations were not met: %v", err)
			}
		})
	}
}

func TestAPIKeyRepository_FindActiveByTenantID_TenantValidation(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()
	expiresAt := now.Add(time.Hour)

	cases := []struct {
		name       string
		ctx        context.Context
		expectErr  error
		expectExec bool
	}{
		{name: "no tenant context", ctx: context.Background(), expectErr: autherrors.ErrUnauthorized},
		{name: "foreign tenant", ctx: authmw.WithTenantID(context.Background(), uuid.NewString()), expectErr: autherrors.ErrInsufficientPermissions},
		{name: "matching tenant", ctx: authmw.WithTenantID(context.Background(), tenantID.String()), expectExec: true},
		{name: "platform tenant", ctx: authmw.WithTenantID(context.Background(), "platform"), expectExec: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock: %v", err)
			}
			defer mock.Close()

			repo := NewAPIKeyRepository(mock)

			if tc.expectExec {
				rows := pgxmock.NewRows([]string{
					"id", "tenant_id", "name", "key_hash", "key_prefix", "scopes", "rate_limit", "expires_at", "last_used_at", "created_at", "revoked_at",
				}).AddRow(
					uuid.New(),
					tenantID,
					"k1",
					"hash",
					"prefix",
					[]string{"tenants:read"},
					100,
					&expiresAt,
					nil,
					now,
					nil,
				)

				mock.ExpectQuery("SELECT id, tenant_id, name").WithArgs(tenantID, pgxmock.AnyArg()).WillReturnRows(rows)
			}

			_, err = repo.FindActiveByTenantID(tc.ctx, tenantID)
			if tc.expectErr != nil {
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("expected error %v, got %v", tc.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations were not met: %v", err)
			}
		})
	}
}

func TestAPIKeyRepository_CountActiveByTenantID_TenantValidation(t *testing.T) {
	tenantID := uuid.New()

	cases := []struct {
		name       string
		ctx        context.Context
		expectErr  error
		expectExec bool
	}{
		{name: "no tenant context", ctx: context.Background(), expectErr: autherrors.ErrUnauthorized},
		{name: "foreign tenant", ctx: authmw.WithTenantID(context.Background(), uuid.NewString()), expectErr: autherrors.ErrInsufficientPermissions},
		{name: "matching tenant", ctx: authmw.WithTenantID(context.Background(), tenantID.String()), expectExec: true},
		{name: "platform tenant", ctx: authmw.WithTenantID(context.Background(), "platform"), expectExec: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock: %v", err)
			}
			defer mock.Close()

			repo := NewAPIKeyRepository(mock)

			if tc.expectExec {
				rows := pgxmock.NewRows([]string{"count"}).AddRow(int64(1))
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM api_keys[[:space:]]+WHERE tenant_id = \\$1 AND revoked_at IS NULL").WithArgs(tenantID, pgxmock.AnyArg()).WillReturnRows(rows)
			}

			_, err = repo.CountActiveByTenantID(tc.ctx, tenantID)
			if tc.expectErr != nil {
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("expected error %v, got %v", tc.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations were not met: %v", err)
			}
		})
	}
}

func TestAPIKeyRepository_RevokeAll_TenantValidation(t *testing.T) {
	tenantID := uuid.New()
	revokedBy := uuid.New()

	cases := []struct {
		name       string
		ctx        context.Context
		expectErr  error
		expectExec bool
	}{
		{name: "no tenant context", ctx: context.Background(), expectErr: autherrors.ErrUnauthorized},
		{name: "foreign tenant", ctx: authmw.WithTenantID(context.Background(), uuid.NewString()), expectErr: autherrors.ErrInsufficientPermissions},
		{name: "matching tenant", ctx: authmw.WithTenantID(context.Background(), tenantID.String()), expectExec: true},
		{name: "platform tenant", ctx: authmw.WithTenantID(context.Background(), "platform"), expectExec: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock: %v", err)
			}
			defer mock.Close()

			repo := NewAPIKeyRepository(mock)

			if tc.expectExec {
				mock.ExpectExec("UPDATE api_keys[[:space:]]+SET revoked_at").WithArgs(tenantID, pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 2))
			}

			err = repo.RevokeAll(tc.ctx, tenantID, revokedBy)
			if tc.expectErr != nil {
				if !errors.Is(err, tc.expectErr) {
					t.Fatalf("expected error %v, got %v", tc.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("expectations were not met: %v", err)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
