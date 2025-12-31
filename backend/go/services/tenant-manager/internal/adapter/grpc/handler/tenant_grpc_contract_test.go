package handler

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
	"github.com/serphona/backend/go/libs/platform-observability/tracing"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"tenant-manager/internal/adapter/kafka"
	"tenant-manager/internal/adapter/redis"
	"tenant-manager/internal/application/tenant"
	domain "tenant-manager/internal/domain/tenant"
	tenantpb "tenant-manager/proto"
)

const bufSize = 1024 * 1024

// minimalTenantRepo implements tenant.Repository for contract tests.
type minimalTenantRepo struct {
	items map[uuid.UUID]*domain.Tenant
}

func newMinimalTenantRepo() *minimalTenantRepo {
	return &minimalTenantRepo{items: make(map[uuid.UUID]*domain.Tenant)}
}

func (r *minimalTenantRepo) Create(_ context.Context, t *domain.Tenant) error {
	r.items[t.ID] = t
	return nil
}
func (r *minimalTenantRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	if t, ok := r.items[id]; ok {
		return t, nil
	}
	return nil, domain.ErrTenantNotFound
}
func (r *minimalTenantRepo) GetBySlug(_ context.Context, slug string) (*domain.Tenant, error) {
	return nil, domain.ErrTenantNotFound
}
func (r *minimalTenantRepo) GetByEmail(_ context.Context, email string) (*domain.Tenant, error) {
	return nil, domain.ErrTenantNotFound
}
func (r *minimalTenantRepo) Update(_ context.Context, t *domain.Tenant) error {
	r.items[t.ID] = t
	return nil
}
func (r *minimalTenantRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}
func (r *minimalTenantRepo) List(_ context.Context, filter domain.ListFilter) (*domain.ListResult, error) {
	return &domain.ListResult{Tenants: []*domain.Tenant{}, Total: 0, PageNumber: filter.PageNumber, PageSize: filter.PageSize, TotalPages: 0}, nil
}
func (r *minimalTenantRepo) UpdateSettings(_ context.Context, _ uuid.UUID, _ domain.Settings) error {
	return nil
}
func (r *minimalTenantRepo) GetSettings(_ context.Context, _ uuid.UUID) (*domain.Settings, error) {
	return nil, nil
}
func (r *minimalTenantRepo) GetQuota(_ context.Context, _ uuid.UUID) (*domain.Quota, error) {
	return nil, domain.ErrTenantNotFound
}
func (r *minimalTenantRepo) UpdateQuota(_ context.Context, _ *domain.Quota) error { return nil }
func (r *minimalTenantRepo) IncrementUsage(_ context.Context, _ uuid.UUID, _, _ int) error {
	return nil
}
func (r *minimalTenantRepo) ExistsBySlug(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (r *minimalTenantRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func TestTenantGRPCEnvelopeMetadata(t *testing.T) {
	// Tracing provider to ensure traceparent emission
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	tracing.Tracer() // touch to ensure tracer is initialized

	authjwt.SetSecret("test-secret")
	authjwt.ResetSecretOnceForTests()
	authjwt.ResetValidationConfig()

	repo := newMinimalTenantRepo()
	seedID := uuid.New()
	repo.items[seedID] = &domain.Tenant{ID: seedID, Name: "Acme", Email: "acme@example.com"}

	svc := tenant.NewService(repo, nil, redis.NoopCache{}, kafka.NewNoopPublisher(), zaptest.NewLogger(t))
	h := NewTenantHandler(svc)

	lis := bufconn.Listen(bufSize)
	server := grpc.NewServer(grpc.UnaryInterceptor(authmw.UnaryAuthInterceptor()))
	tenantpb.RegisterTenantServiceServer(server, h)

	go func() {
		_ = server.Serve(lis)
	}()
	t.Cleanup(server.Stop)

	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to dial bufnet: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := tenantpb.NewTenantServiceClient(conn)

	token := signedTestToken(t)
	md := metadata.Pairs(
		"authorization", "Bearer "+token,
		"x-request-id", "req-grpc-contract",
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	var header metadata.MD
	resp, err := client.GetTenant(ctx, &tenantpb.GetTenantRequest{Id: seedID.String()}, grpc.Header(&header))
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if resp == nil || resp.Tenant == nil || resp.Tenant.Id != seedID.String() {
		t.Fatalf("expected tenant payload with correct ID")
	}

	if got := header.Get("x-request-id"); len(got) == 0 || got[0] != "req-grpc-contract" {
		t.Fatalf("expected x-request-id in header, got %v", got)
	}
	if got := header.Get("traceparent"); len(got) == 0 || got[0] == "" {
		t.Fatalf("expected traceparent header to be set")
	}
}

func signedTestToken(t *testing.T) string {
	t.Helper()

	claims := types.Claims{
		UserID:   uuid.NewString(),
		Email:    "user@example.com",
		Name:     "User",
		Role:     "user",
		TenantID: uuid.NewString(),
		Scopes:   []string{"read:tenants"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(authjwt.GetSecret()))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}
