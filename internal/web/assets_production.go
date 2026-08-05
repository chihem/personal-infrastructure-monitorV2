//go:build production

package web

import "embed"

//go:embed dist
var embeddedFiles embed.FS

const embeddedRoot = "dist"
