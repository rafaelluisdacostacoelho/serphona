package tenant

import (
	"context"
	"testing"
	"time"

	"tenant-manager/internal/adapter/redis"
	"tenant-manager/internal/domain/tenant"

	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.uber.org/zap/zaptest"
)

func TestIncrementUsagePublishesUsageReported(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	repo := newUsageRepo(tenantID)
	publisher := &recordingPublisher{}

	svc := NewService(repo, nil, redis.NoopCache{}, publisher, zaptest.NewLogger(t))

	ctx := authmw.WithRequestID(context.Background(), "req-usage-123")

	quota, err := svc.IncrementUsage(ctx, IncrementUsageCommand{TenantID: tenantID, Calls: 2, Minutes: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if quota.UsedCalls != 2 || quota.UsedMinutes != 3 {
		t.Fatalf("quota not updated, got calls=%d minutes=%d", quota.UsedCalls, quota.UsedMinutes)
	}

	if len(publisher.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(publisher.events))
	}

	evt := publisher.events[0]
	if evt.TenantID != tenantID {
		t.Fatalf("tenant mismatch: got %s", evt.TenantID)
	}
	if evt.RequestID != "req-usage-123" {
		t.Fatalf("request_id mismatch: got %s", evt.RequestID)
	}
	if evt.Calls != 2 || evt.Minutes != 3 {
		t.Fatalf("delta mismatch: calls=%d minutes=%d", evt.Calls, evt.Minutes)
	}
	if evt.Period == "" {
		t.Fatalf("period should be set")
	}
	if evt.OccurredAt.IsZero() {
		t.Fatalf("occurred_at should be set")
	}
	if evt.Source != "tenant-manager" {
		t.Fatalf("source mismatch: %s", evt.Source)
	}
}

// recordingPublisher captures events for assertions.
type recordingPublisher struct {
	events []tenant.UsageReportedEvent
}

func (p *recordingPublisher) PublishCreated(context.Context, *tenant.Tenant) error   { return nil }
func (p *recordingPublisher) PublishUpdated(context.Context, *tenant.Tenant) error   { return nil }
func (p *recordingPublisher) PublishDeleted(context.Context, uuid.UUID) error        { return nil }
func (p *recordingPublisher) PublishActivated(context.Context, *tenant.Tenant) error { return nil }
func (p *recordingPublisher) PublishSuspended(context.Context, *tenant.Tenant) error { return nil }
func (p *recordingPublisher) PublishSettingsUpdated(context.Context, uuid.UUID, *tenant.Settings) error {
	return nil
}
func (p *recordingPublisher) PublishUsageReported(_ context.Context, event tenant.UsageReportedEvent) error {
	p.events = append(p.events, event)
	return nil
}

// usageRepoStub implements tenant.Repository minimally for IncrementUsage.
type usageRepoStub struct {
	quota *tenant.Quota
}

func newUsageRepo(tenantID uuid.UUID) *usageRepoStub {
	return &usageRepoStub{
		quota: &tenant.Quota{TenantID: tenantID, ResetAt: time.Now().UTC().Add(30 * 24 * time.Hour)},
	}
}

func (r *usageRepoStub) Create(context.Context, *tenant.Tenant) error { return nil }
func (r *usageRepoStub) GetByID(context.Context, uuid.UUID) (*tenant.Tenant, error) {
	return nil, nil
}
func (r *usageRepoStub) GetBySlug(context.Context, string) (*tenant.Tenant, error) {
	return nil, nil
}
func (r *usageRepoStub) GetByEmail(context.Context, string) (*tenant.Tenant, error) {
	return nil, nil
}
func (r *usageRepoStub) Update(context.Context, *tenant.Tenant) error { return nil }
func (r *usageRepoStub) Delete(context.Context, uuid.UUID) error      { return nil }
func (r *usageRepoStub) List(context.Context, tenant.ListFilter) (*tenant.ListResult, error) {
	return &tenant.ListResult{}, nil
}
func (r *usageRepoStub) UpdateSettings(context.Context, uuid.UUID, tenant.Settings) error {
	return nil
}
func (r *usageRepoStub) GetQuota(context.Context, uuid.UUID) (*tenant.Quota, error) {
	return r.quota, nil
}
func (r *usageRepoStub) UpdateQuota(context.Context, *tenant.Quota) error { return nil }
func (r *usageRepoStub) IncrementUsage(_ context.Context, _ uuid.UUID, calls, minutes int) error {
	r.quota.UsedCalls += calls
	r.quota.UsedMinutes += minutes
	return nil
}
func (r *usageRepoStub) ExistsBySlug(context.Context, string) (bool, error)  { return false, nil }
func (r *usageRepoStub) ExistsByEmail(context.Context, string) (bool, error) { return false, nil }
