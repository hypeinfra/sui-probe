package templates

import "embed"

//go:embed **/*.gohtml *.gohtml
var Templates embed.FS
