package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed static/*
var embeddedFiles embed.FS

// Handler returns a same-origin SPA handler backed only by embedded files.
func Handler() http.Handler {
	staticFiles, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		panic("embedded web files are unavailable: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(staticFiles))

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writer.Header().Set("Allow", "GET, HEAD")
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		cleaned := path.Clean("/" + request.URL.Path)
		requested := strings.TrimPrefix(cleaned, "/")
		if requested == "" {
			serveIndex(fileServer, writer, request)
			return
		}
		if info, statErr := fs.Stat(staticFiles, requested); statErr == nil && !info.IsDir() {
			fileServer.ServeHTTP(writer, request)
			return
		}
		if path.Ext(requested) == "" {
			serveIndex(fileServer, writer, request)
			return
		}
		http.NotFound(writer, request)
	})
}

func serveIndex(fileServer http.Handler, writer http.ResponseWriter, request *http.Request) {
	clone := request.Clone(request.Context())
	clone.URL.Path = "/"
	fileServer.ServeHTTP(writer, clone)
}
