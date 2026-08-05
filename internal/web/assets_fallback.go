//go:build !production

package web

import "embed"

// The fallback keeps ordinary Go tests and development builds independent of
// Node. Release builds use assets_production.go after Vite has produced dist.
//
//go:embed static/*
var embeddedFiles embed.FS

const embeddedRoot = "static"
