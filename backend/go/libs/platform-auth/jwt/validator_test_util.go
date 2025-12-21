package jwt

import "sync"

// ResetSecretOnceForTests resets the once guard so EnsureSecretLoaded can run in tests.
func ResetSecretOnceForTests() {
	ensureSecretOnce = sync.Once{}
	ensureSecretErr = nil
}
