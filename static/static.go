package static

import "embed"

// content is our static web server content.
//
//go:embed *
var Files embed.FS
