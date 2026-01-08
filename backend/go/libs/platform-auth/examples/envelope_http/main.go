//go:build examples
// +build examples

package main

import (
	"log"
	"net/http"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
)

// Net/http (chi-compatible) example using envelope helpers.
func main() {
	mux := http.NewServeMux()
	mux.Handle("/reports", middleware.RequireAuthHTTP(http.HandlerFunc(reportsHandler)))

	wrapped := middleware.HTTPRecovery(mux)
	log.Println("Listening on :8082")
	if err := http.ListenAndServe(":8082", wrapped); err != nil {
		log.Fatal(err)
	}
}

func reportsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.WriteError(r.Context(), w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "only GET is supported", nil)
		return
	}

	reports := []map[string]any{{"id": "r1", "status": "ok"}}
	response.WriteSuccess(r.Context(), w, http.StatusOK, reports, response.WithPagination(response.Pagination{Page: 1, PageSize: 50, Total: 1, TotalPages: 1}))
}
