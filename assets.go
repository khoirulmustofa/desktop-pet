// Package assets exposes the embedded sprite PNGs.
package assets

import "embed"

//go:embed assets/pet
var Pet embed.FS
