package ui

import "embed"

// FS exposes the embedded UI assets.
//
//go:embed template.html styles.css logo.png
var FS embed.FS
