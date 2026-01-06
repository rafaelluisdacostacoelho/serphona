package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.uber.org/zap"
)

func TestTenantClientSetsTenantHeader(t *testing.T) {
	var capturedHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"agent_id":"agent-1","name":"Test","conversation_flow":{"max_retries":1}}`))
	}))
	defer srv.Close()

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	client := NewClient(srv.URL, mustToken(t, "internal"), "voice-gateway", "vg-1", "internal", logger)

	ctx := authmw.WithTenantID(context.Background(), "tenant-xyz")
	if _, err := client.GetAgentConfig(ctx, uuid.New()); err != nil {
		t.Fatalf("GetAgentConfig falhou: %v", err)
	}

	if got := capturedHeaders.Get(authmw.TenantIDHeader); got != "tenant-xyz" {
		t.Fatalf("esperava X-Tenant-Id tenant-xyz, obtido %q", got)
	}
}

func mustToken(t *testing.T, aud string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Audience: []string{aud}})
	signed, err := tok.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}
