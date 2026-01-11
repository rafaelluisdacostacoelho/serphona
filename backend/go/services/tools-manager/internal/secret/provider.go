package secret

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Provider persists encrypted secret payloads.
type Provider interface {
	Put(ctx context.Context, path string, data []byte) error
	Get(ctx context.Context, path string) ([]byte, bool, error)
}

// memoryProvider keeps data in-memory (tests/default).
type memoryProvider struct {
	mu   sync.Mutex
	data map[string][]byte
}

func NewMemoryProvider() Provider {
	return &memoryProvider{data: make(map[string][]byte)}
}

func (m *memoryProvider) Put(ctx context.Context, path string, data []byte) error {
	m.mu.Lock()
	m.data[path] = append([]byte(nil), data...)
	m.mu.Unlock()
	return nil
}

func (m *memoryProvider) Get(ctx context.Context, path string) ([]byte, bool, error) {
	m.mu.Lock()
	b, ok := m.data[path]
	if ok {
		b = append([]byte(nil), b...)
	}
	m.mu.Unlock()
	return b, ok, nil
}

// vaultProvider writes to HashiCorp Vault KV v2 using HTTP.
type vaultProvider struct {
	addr   string
	token  string
	mount  string
	prefix string
	client *http.Client
}

type vaultWriteRequest struct {
	Data map[string]string `json:"data"`
}

type vaultReadResponse struct {
	Data struct {
		Data map[string]string `json:"data"`
	} `json:"data"`
}

func NewVaultProvider(addr, token, mount, prefix string) (Provider, error) {
	if addr == "" || token == "" {
		return nil, errors.New("vault addr and token are required")
	}
	if mount == "" {
		mount = "secret"
	}
	if prefix == "" {
		prefix = "tools-manager"
	}
	return &vaultProvider{
		addr:   strings.TrimRight(addr, "/"),
		token:  token,
		mount:  mount,
		prefix: strings.Trim(prefix, "/"),
		client: &http.Client{Timeout: 5 * time.Second},
	}, nil
}

func (v *vaultProvider) Put(ctx context.Context, path string, data []byte) error {
	url := fmt.Sprintf("%s/v1/%s/data/%s", v.addr, v.mount, v.fullPath(path))
	payload := vaultWriteRequest{Data: map[string]string{"value": base64.StdEncoding.EncodeToString(data)}}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("X-Vault-Token", v.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("vault put failed: %s", resp.Status)
	}
	return nil
}

func (v *vaultProvider) Get(ctx context.Context, path string) ([]byte, bool, error) {
	url := fmt.Sprintf("%s/v1/%s/data/%s", v.addr, v.mount, v.fullPath(path))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("X-Vault-Token", v.token)

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode >= 300 {
		return nil, false, fmt.Errorf("vault get failed: %s", resp.Status)
	}

	var out vaultReadResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, false, err
	}
	encoded := out.Data.Data["value"]
	if encoded == "" {
		return nil, false, errors.New("vault response missing value")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, false, err
	}
	return decoded, true, nil
}

func (v *vaultProvider) fullPath(path string) string {
	if v.prefix == "" {
		return strings.TrimLeft(path, "/")
	}
	return fmt.Sprintf("%s/%s", v.prefix, strings.TrimLeft(path, "/"))
}
