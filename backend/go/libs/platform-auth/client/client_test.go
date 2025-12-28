package client

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
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

func TestValidateTokenStructuredError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad"}`))
	}))
	defer ts.Close()

	c := New(ts.URL)
	_, err := c.ValidateToken("token")
	if err == nil {
		t.Fatalf("expected error")
	}
	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected status %d", httpErr.StatusCode)
	}
	if httpErr.Body == "" {
		t.Fatalf("expected body in error")
	}
}

func TestTLSConfigFromFilesEmpty(t *testing.T) {
	cfg, err := TLSConfigFromFiles("", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("expected min version TLS1.2")
	}
}

func TestClientTLSConfigAllowsCustomCA(t *testing.T) {
	materials := generateTLSMaterials(t)

	serverCert, err := tls.X509KeyPair(materials.serverCertPEM, materials.serverKeyPEM)
	if err != nil {
		t.Fatalf("failed to load server key pair: %v", err)
	}

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(mockClaims{UserID: "tls"})
	}))
	ts.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}}
	ts.StartTLS()
	defer ts.Close()

	caFile := writePEMTemp(t, "ca-*.pem", materials.caCertPEM)

	cfg, err := TLSConfigFromFiles(caFile, "", "")
	if err != nil {
		t.Fatalf("unexpected error loading CA: %v", err)
	}

	cl := NewWithOptions(ts.URL, WithTLSConfig(cfg))
	claims, err := cl.ValidateToken("token")
	if err != nil {
		t.Fatalf("expected TLS client success, got %v", err)
	}
	if claims.UserID != "tls" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestClientMTLS(t *testing.T) {
	materials := generateTLSMaterials(t)

	serverCert, err := tls.X509KeyPair(materials.serverCertPEM, materials.serverKeyPEM)
	if err != nil {
		t.Fatalf("failed to load server key pair: %v", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(materials.caCertPEM) {
		t.Fatalf("failed to append CA cert")
	}

	sawClientCert := false
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			sawClientCert = true
		}
		_ = json.NewEncoder(w).Encode(mockClaims{UserID: "mtls"})
	}))
	ts.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
	}
	ts.StartTLS()
	defer ts.Close()

	caFile := writePEMTemp(t, "ca-*.pem", materials.caCertPEM)
	clientCertFile := writePEMTemp(t, "client-*.pem", materials.clientCertPEM)
	clientKeyFile := writePEMTemp(t, "client-key-*.pem", materials.clientKeyPEM)

	cfg, err := TLSConfigFromFiles(caFile, clientCertFile, clientKeyFile)
	if err != nil {
		t.Fatalf("unexpected error loading mTLS config: %v", err)
	}

	cl := NewWithOptions(ts.URL, WithTLSConfig(cfg))
	claims, err := cl.ValidateToken("token")
	if err != nil {
		t.Fatalf("expected mTLS success, got %v", err)
	}
	if claims.UserID != "mtls" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if !sawClientCert {
		t.Fatalf("expected client certificate to be presented")
	}
}

type tlsMaterials struct {
	caCertPEM     []byte
	serverCertPEM []byte
	serverKeyPEM  []byte
	clientCertPEM []byte
	clientKeyPEM  []byte
}

func generateTLSMaterials(t *testing.T) tlsMaterials {
	t.Helper()

	now := time.Now()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		Subject:               pkix.Name{CommonName: "test-ca"},
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA cert: %v", err)
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate server key: %v", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(24 * time.Hour),
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		Subject:      pkix.Name{CommonName: "test-server"},
	}

	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create server cert: %v", err)
	}

	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate client key: %v", err)
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Subject:      pkix.Name{CommonName: "test-client"},
	}

	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create client cert: %v", err)
	}

	return tlsMaterials{
		caCertPEM:     pemEncodeCert(caDER),
		serverCertPEM: pemEncodeCert(serverDER),
		serverKeyPEM:  pemEncodeKey(serverKey),
		clientCertPEM: pemEncodeCert(clientDER),
		clientKeyPEM:  pemEncodeKey(clientKey),
	}
}

func pemEncodeCert(der []byte) []byte {
	var b bytes.Buffer
	_ = pem.Encode(&b, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	return b.Bytes()
}

func pemEncodeKey(key *rsa.PrivateKey) []byte {
	var b bytes.Buffer
	_ = pem.Encode(&b, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return b.Bytes()
}

func writePEMTemp(t *testing.T, pattern string, data []byte) string {
	t.Helper()
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		t.Fatalf("failed to write temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(f.Name()) })
	return f.Name()
}
