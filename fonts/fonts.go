// Package fonts embeds Inter and Fira Code TTF files and registers them
// with a raycanvas SharedCache in one call.
//
// This package is publicly importable from any module:
//
//	import "github.com/ha1tch/raycanvas/fonts"
//	fonts.Register(cache)
//
// Fonts included:
//   - Fira Code: Regular (400), Bold (700)
//   - Inter: Regular (400), Medium (500), SemiBold (600), Bold (700)
//
// All variants are baked at the default atlas sizes (8–14px).
// Must be called after rc.SetupWindow and before any draw calls.
package fonts

import (
	_ "embed"
	"fmt"

	rc "github.com/ha1tch/raycanvas"
)

//go:embed FiraCode-Regular.ttf
var firaRegular []byte

//go:embed FiraCode-Bold.ttf
var firaBold []byte

//go:embed Inter-Regular.ttf
var interRegular []byte

//go:embed Inter-Medium.ttf
var interMedium []byte

//go:embed Inter-SemiBold.ttf
var interSemiBold []byte

//go:embed Inter-Bold.ttf
var interBold []byte

// Register loads all embedded fonts into cache at all default sizes.
// Returns the first error encountered, or nil on success.
func Register(cache *rc.SharedCache) error {
	type entry struct {
		family string
		weight int
		italic bool
		data   []byte
	}
	entries := []entry{
		{"fira", 400, false, firaRegular},
		{"fira", 700, false, firaBold},
		{"inter", 400, false, interRegular},
		{"inter", 500, false, interMedium},
		{"inter", 600, false, interSemiBold},
		{"inter", 700, false, interBold},
	}
	for _, e := range entries {
		if err := rc.RegisterFont(cache, e.family, e.weight, e.italic, e.data, nil); err != nil {
			return fmt.Errorf("fonts.Register: %w", err)
		}
	}
	return nil
}
