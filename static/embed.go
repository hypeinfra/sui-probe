package static

import "embed"

//go:embed *.css logo/*.svg
var FS embed.FS
