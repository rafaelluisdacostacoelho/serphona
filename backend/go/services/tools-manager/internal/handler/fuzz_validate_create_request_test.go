//go:build go1.18

package handler

import (
	"encoding/json"
	"testing"
)

// FuzzValidateCreateRequest exercises schema depth/size and allowlist validation.
func FuzzValidateCreateRequest(f *testing.F) {
	seed := baseCreateReq()
	seedBytes, _ := json.Marshal(seed)
	f.Add(string(seedBytes))

	f.Fuzz(func(t *testing.T, raw string) {
		var req createToolRequest
		if err := json.Unmarshal([]byte(raw), &req); err != nil {
			t.Skip() // ignore invalid JSON; focus on validator behavior
		}

		_ = validateCreateRequest(req)
	})
}
