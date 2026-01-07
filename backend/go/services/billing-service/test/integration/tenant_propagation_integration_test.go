package integration

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/adapter/httpclient"
	pgrepo "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/adapter/postgres"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/config"
	domain "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/domain/wallet"
)

func TestTenantGuardsEnforceIsolation(t *testing.T) {
	db := newSQLite(t)
	repo := pgrepo.NewWalletRepository(db)

	tenantA := uuid.New()
	tenantB := uuid.New()

	ctxTenantA := authmw.WithTenantID(context.Background(), tenantA.String())
	wallet := &domain.Wallet{ID: uuid.New(), TenantID: tenantA, Balance: 100, Currency: "USD"}

	if err := repo.Create(ctxTenantA, wallet); err != nil {
		t.Fatalf("create wallet: %v", err)
	}

	ctxTenantB := authmw.WithTenantID(context.Background(), tenantB.String())
	if _, err := repo.FindByTenantID(ctxTenantB, tenantA); !errors.Is(err, autherrors.ErrInsufficientPermissions) {
		t.Fatalf("expected cross-tenant access to be denied, got %v", err)
	}

	platformCtx := authmw.WithTenantID(context.Background(), "platform")
	if _, err := repo.FindByTenantID(platformCtx, tenantA); err != nil {
		t.Fatalf("platform tenant should bypass guard, got %v", err)
	}
}

func TestServiceClientInjectsTenantHeader(t *testing.T) {
	tenantID := "tenant-integration"
	token := signServiceToken(t, "svc-audience")

	captured := make(chan http.Header, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured <- r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.ServiceConfig{Name: "billing-service", Instance: "it-1", Audience: "svc-audience"}
	client := httpclient.NewServiceClient(cfg, 5*time.Second, token)

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req = req.WithContext(authmw.WithTenantID(req.Context(), tenantID))

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client do: %v", err)
	}
	_ = resp.Body.Close()

	select {
	case hdr := <-captured:
		if hdr.Get(authmw.TenantIDHeader) != tenantID {
			t.Fatalf("expected tenant header %s, got %s", tenantID, hdr.Get(authmw.TenantIDHeader))
		}
		if hdr.Get("Authorization") == "" {
			t.Fatalf("expected Authorization to be set from service token")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("did not receive request headers")
	}
}

func signServiceToken(t *testing.T, audience string) string {
	t.Helper()

	claims := jwt.RegisteredClaims{Audience: jwt.ClaimStrings{audience}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}
