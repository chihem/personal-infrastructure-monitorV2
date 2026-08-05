package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmbeddedWebHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "index", method: http.MethodGet, path: "/", wantStatus: http.StatusOK, wantBody: "Infrastructure Monitor"},
		{name: "SPA fallback", method: http.MethodGet, path: "/cpu", wantStatus: http.StatusOK, wantBody: "Infrastructure Monitor"},
		{name: "missing asset", method: http.MethodGet, path: "/missing.js", wantStatus: http.StatusNotFound},
		{name: "write method rejected", method: http.MethodPost, path: "/", wantStatus: http.StatusMethodNotAllowed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			response := httptest.NewRecorder()
			Handler().ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if test.wantBody != "" && !strings.Contains(response.Body.String(), test.wantBody) {
				t.Fatalf("body does not contain %q: %s", test.wantBody, response.Body.String())
			}
		})
	}
}
