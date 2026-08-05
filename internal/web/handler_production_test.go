//go:build production

package web

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

var scriptSource = regexp.MustCompile(`<script[^>]+src="([^"]+)"`)

func TestProductionFrontendAssetsAreEmbedded(t *testing.T) {
	t.Parallel()

	handler := Handler()
	indexResponse := httptest.NewRecorder()
	handler.ServeHTTP(indexResponse, httptest.NewRequest(http.MethodGet, "/", nil))
	if indexResponse.Code != http.StatusOK {
		t.Fatalf("index status = %d", indexResponse.Code)
	}

	match := scriptSource.FindStringSubmatch(indexResponse.Body.String())
	if len(match) != 2 {
		t.Fatalf("production index has no script asset: %s", indexResponse.Body.String())
	}
	assetResponse := httptest.NewRecorder()
	handler.ServeHTTP(assetResponse, httptest.NewRequest(http.MethodGet, match[1], nil))
	if assetResponse.Code != http.StatusOK {
		t.Fatalf("script asset %s status = %d", match[1], assetResponse.Code)
	}
	if assetResponse.Body.Len() == 0 {
		t.Fatal("production script asset is empty")
	}
}
