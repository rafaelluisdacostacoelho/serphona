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

	"tenant-manager/internal/domain/tenant"
)

func TestTenantRepository_Update_TenantValidation(t *testing.T) {
	now := time.Now().UTC()
	tenantID := uuid.New()
	tnt := &tenant.Tenant{
		ID:        tenantID,
		Name:      "Acme",
		Slug:      "acme",
		Email:     "acme@example.com",
		Phone:     "+123456789",
		Status:    tenant.StatusActive,
		Plan:      tenant.PlanStarter,
		Settings:  tenant.Settings{},
		Metadata:  tenant.Metadata{},
		CreatedAt: now,
		UpdatedAt: now,
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

			repo := NewTenantRepository(mock)

			if tc.expectExec {
				mock.ExpectExec("UPDATE tenants SET").
					WithArgs(
						tnt.ID,
						tnt.Name,
						tnt.Email,
						tnt.Phone,
						string(tnt.Status),
						string(tnt.Plan),
						pgxmock.AnyArg(), // settings JSON
						pgxmock.AnyArg(), // metadata JSON
						tnt.StripeID,
						tnt.BillingEmail,
						pgxmock.AnyArg(),
					).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			}

			err = repo.Update(tc.ctx, tnt)
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

func TestTenantRepository_GetBySlug_TenantValidation(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()

	settingsJSON := []byte(`{}`)
	metadataJSON := []byte(`{}`)

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
				"id", "name", "slug", "email", "phone", "status", "plan",
				"settings", "metadata", "stripe_id", "billing_email",
				"created_at", "updated_at", "deleted_at",
			}).AddRow(
				tenantID,
				"Acme",
				"acme",
				"acme@example.com",
				"+123456789",
				string(tenant.StatusActive),
				string(tenant.PlanStarter),
				settingsJSON,
				metadataJSON,
				"stripe_123",
				"billing@example.com",
				now,
				now,
				nil,
			)

			mock.ExpectQuery("SELECT ").WithArgs("acme").WillReturnRows(rows)

			repo := NewTenantRepository(mock)
			_, err = repo.GetBySlug(tc.ctx, "acme")

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

func TestTenantRepository_GetQuota_TenantValidation(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()

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

			repo := NewTenantRepository(mock)

			if tc.expectExec {
				rows := pgxmock.NewRows([]string{
					"tenant_id", "max_api_keys", "max_users", "max_calls_per_month",
					"max_minutes_per_month", "max_storage_gb", "used_calls", "used_minutes", "used_storage_gb", "reset_at",
				}).AddRow(
					tenantID,
					10,
					20,
					30,
					40,
					50,
					1,
					2,
					3.5,
					now,
				)

				mock.ExpectQuery("SELECT ").WithArgs(tenantID).WillReturnRows(rows)
			}

			_, err = repo.GetQuota(tc.ctx, tenantID)
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

func TestTenantRepository_UpdateQuota_TenantValidation(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now().UTC()

	q := &tenant.Quota{
		TenantID:           tenantID,
		MaxAPIKeys:         10,
		MaxUsers:           20,
		MaxCallsPerMonth:   30,
		MaxMinutesPerMonth: 40,
		MaxStorageGB:       50,
		UsedCalls:          1,
		UsedMinutes:        2,
		UsedStorageGB:      3.5,
		ResetAt:            now,
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

			repo := NewTenantRepository(mock)

			if tc.expectExec {
				mock.ExpectExec("INSERT INTO tenant_quotas").
					WithArgs(
						q.TenantID,
						q.MaxAPIKeys,
						q.MaxUsers,
						q.MaxCallsPerMonth,
						q.MaxMinutesPerMonth,
						q.MaxStorageGB,
						q.UsedCalls,
						q.UsedMinutes,
						q.UsedStorageGB,
						q.ResetAt,
					).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			}

			err = repo.UpdateQuota(tc.ctx, q)
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

func TestTenantRepository_IncrementUsage_TenantValidation(t *testing.T) {
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

			repo := NewTenantRepository(mock)

			if tc.expectExec {
				mock.ExpectExec("UPDATE tenant_quotas SET").
					WithArgs(tenantID, 5, 7, pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
				mock.ExpectExec("INSERT INTO tenant_usage_history").
					WithArgs(tenantID, pgxmock.AnyArg(), 5, 7).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			}

			err = repo.IncrementUsage(tc.ctx, tenantID, 5, 7)
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
