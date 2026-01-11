package secret

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// Store encrypts secrets per tenant with a pluggable backend and TTL cache of decrypted values.
type Store struct {
	encKey   []byte
	cacheTTL time.Duration
	backend  Provider

	cache map[string]cacheEntry
}

type encrypted struct {
	Nonce      []byte    `json:"nonce"`
	Ciphertext []byte    `json:"ciphertext"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

// NewStore builds a secret store with AES-GCM encryption and pluggable backend.
func NewStore(encKey string, cacheTTL time.Duration, backend Provider) (*Store, error) {
	if backend == nil {
		backend = NewMemoryProvider()
	}
	if len(encKey) == 0 {
		return nil, errors.New("SECRETS_ENCRYPTION_KEY is required")
	}
	keyBytes := []byte(encKey)
	if l := len(keyBytes); l != 16 && l != 24 && l != 32 {
		return nil, fmt.Errorf("SECRETS_ENCRYPTION_KEY must be 16, 24, or 32 bytes, got %d", l)
	}
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}
	return &Store{
		encKey:   keyBytes,
		cacheTTL: cacheTTL,
		backend:  backend,
		cache:    make(map[string]cacheEntry),
	}, nil
}

// Put stores a secret value for the tenant/id pair.
func (s *Store) Put(tenantID, secretID, value string) error {
	if tenantID == "" || secretID == "" {
		return errors.New("tenant_id and secret_id are required")
	}

	block, err := aes.NewCipher(s.encKey)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(value), nil)

	enc := encrypted{Nonce: nonce, Ciphertext: ciphertext, UpdatedAt: time.Now()}
	payload, err := json.Marshal(enc)
	if err != nil {
		return err
	}

	key := s.key(tenantID, secretID)
	if err := s.backend.Put(context.Background(), key, payload); err != nil {
		return err
	}

	// invalidate cache after rotation
	delete(s.cache, key)

	return nil
}

// Get returns the decrypted secret if present.
func (s *Store) Get(tenantID, secretID string) (string, bool, error) {
	if tenantID == "" || secretID == "" {
		return "", false, errors.New("tenant_id and secret_id are required")
	}
	key := s.key(tenantID, secretID)

	// serve from cache if valid
	if entry, ok := s.cache[key]; ok {
		if time.Now().Before(entry.expiresAt) {
			return entry.value, true, nil
		}
		delete(s.cache, key)
	}

	blob, ok, err := s.backend.Get(context.Background(), key)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}

	var enc encrypted
	if err := json.Unmarshal(blob, &enc); err != nil {
		return "", false, err
	}

	block, err := aes.NewCipher(s.encKey)
	if err != nil {
		return "", false, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", false, err
	}

	plaintext, err := gcm.Open(nil, enc.Nonce, enc.Ciphertext, nil)
	if err != nil {
		return "", false, err
	}

	s.cache[key] = cacheEntry{value: string(plaintext), expiresAt: time.Now().Add(s.cacheTTL)}

	return string(plaintext), true, nil
}

func (s *Store) key(tenantID, secretID string) string {
	return tenantID + "/" + secretID
}
