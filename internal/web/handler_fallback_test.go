//go:build !production

package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFallbackStylesheetIsEmbedded(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/app.css", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "background: #0d1117") {
		t.Fatalf("fallback stylesheet body = %s", response.Body.String())
	}
}
