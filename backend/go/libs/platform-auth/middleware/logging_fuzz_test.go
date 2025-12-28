package middleware

import (
	"net/http"
	"testing"
)

func FuzzRedactHeaders(f *testing.F) {
	f.Add("Authorization", "secret")
	f.Add("X-Trace", "abc")
	f.Add("Cookie", "sess=1")

	f.Fuzz(func(t *testing.T, key, val string) {
		h := http.Header{}
		h.Set(key, val)
		redacted := RedactHeaders(h)
		if key != "" && redacted == nil {
			t.Fatalf("expected header map")
		}
	})
}
