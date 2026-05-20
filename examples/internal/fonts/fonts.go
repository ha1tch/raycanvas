// Package fonts embeds the TTF font files used by raycanvas examples and
// provides a single Register call that loads them all into a SharedCache.
//
// Fonts included:
//   - Fira Code Regular (400) and Bold (700)
//   - Inter Regular (400), Medium (500), SemiBold (600), Bold (700)
//
// These are the exact weights used by the Shevo perspectives (joxel, quag, dekk).
// All fonts are loaded at the sizes defined in rc.DefaultFontSizes (8–14px),
// matching the actual sizes used in Shevo's UI chrome and cell rendering.
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
// Must be called after rl.InitWindow and before any draw calls.
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
