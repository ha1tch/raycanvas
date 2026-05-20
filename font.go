package raycanvas

import (
	"fmt"
	"strconv"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// FontKey uniquely identifies a rasterised font atlas.
// A separate rl.Font (atlas texture) is baked for each unique FontKey.
//
// Critical rule: bake size == draw size. When DrawTextEx is called with
// fontSize == font.BaseSize, raylib renders directly from the atlas with no
// scaling — maximum sharpness. We therefore bake at every pixel size used.
//
// Sizes used in shevo: 8, 9, 10, 11, 12, 13, 14.
type FontKey struct {
	Family string  // "fira", "inter", "georgia", "system"
	Size   float32 // exact pixel size — must match draw call fontSize
	Weight int     // 400=regular, 500=medium, 600=semibold, 700=bold
	Italic bool
}

// fontEntry holds the raw TTF bytes for a font variant so atlases can be
// baked at arbitrary sizes on demand.
type fontEntry struct {
	data []byte
}

// fontRegistry maps (family, weight, italic) → raw TTF bytes.
// Populated at startup via RegisterFont; read-only during the frame loop.
var fontRegistry = map[fontVariantKey]fontEntry{}

type fontVariantKey struct {
	family string
	weight int
	italic bool
}

// DefaultFontSizes lists the pixel sizes baked for every registered font.
// Matches all sizes observed in shevo source. Add to this list if new sizes
// are introduced in ported code.
var DefaultFontSizes = []float32{8, 9, 10, 11, 12, 13, 14}

// DefaultCodepoints is the Unicode range baked into every atlas. Covers
// Latin, Latin-1 Supplement, and Latin Extended-A — sufficient for all
// shevo UI chrome and typical cell content. Extend for non-Latin scripts.
var DefaultCodepoints []rune // nil = raylib default set (ASCII + basic Latin)

// RegisterFont registers a TTF font variant and bakes atlas textures for all
// requested sizes. Must be called after rl.InitWindow and before any draw calls.
//
//	family  — canonical family name: "fira", "inter", "georgia", "system"
//	weight  — numeric weight: 400, 500, 600, 700
//	italic  — true for italic variant
//	data    — raw TTF file bytes (embed with //go:embed or read from disk)
//	sizes   — pixel sizes to bake; pass nil to use DefaultFontSizes
//
// Weights used in shevo:
//   - Fira Code: 400 (regular), 700 (bold). No italic variant exists.
//   - Inter: 400, 500, 600, 700; italic variants for 400 and 700.
//   - Georgia: 400, 700.
func RegisterFont(cache *SharedCache, family string, weight int, italic bool, data []byte, sizes []float32) error {
	if len(data) == 0 {
		return fmt.Errorf("raycanvas: RegisterFont %q: empty data", family)
	}
	if sizes == nil {
		sizes = DefaultFontSizes
	}
	family = strings.ToLower(family)
	fontRegistry[fontVariantKey{family, weight, italic}] = fontEntry{data: data}

	for _, sz := range sizes {
		key := FontKey{Family: family, Size: sz, Weight: weight, Italic: italic}
		if _, ok := cache.lookupFont(key); ok {
			continue // already baked
		}
		font := rl.LoadFontFromMemory(".ttf", data, int32(sz), DefaultCodepoints)
		if !rl.IsFontValid(font) {
			return fmt.Errorf("raycanvas: RegisterFont %q size %.0f: LoadFontFromMemory failed", family, sz)
		}
		// Trilinear filtering for crisp small-size rendering.
		rl.SetTextureFilter(font.Texture, rl.FilterBilinear)
		cache.storeFont(key, font)
	}
	return nil
}

// RegisteredFamilies returns the canonical family names currently registered
// in the font registry. Useful for debugging font name mismatches — the API
// accepts these names (case-insensitive after normalisation) in SetFont calls.
func RegisteredFamilies() []string {
	seen := make(map[string]bool)
	var out []string
	for k := range fontRegistry {
		if !seen[k.family] {
			seen[k.family] = true
			out = append(out, k.family)
		}
	}
	return out
}

// resolvedFont is the fully resolved font for a draw call.
type resolvedFont struct {
	font    rl.Font
	size    float32
	spacing float32 // always 0 — see ARCHITECTURE.md §8
	valid   bool
}

// resolveFont looks up the best matching rl.Font for a FontKey.
// Fallback chain:
//  1. Exact (family, size, weight, italic)
//  2. (family, size, 400, false) — strip weight/italic
//  3. Raylib default font at the requested size
func (c *Context) resolveFont(key FontKey) resolvedFont {
	if f, ok := c.cache.lookupFont(key); ok {
		return resolvedFont{font: f, size: key.Size, valid: true}
	}
	// Try without italic
	if key.Italic {
		k2 := FontKey{Family: key.Family, Size: key.Size, Weight: key.Weight, Italic: false}
		if f, ok := c.cache.lookupFont(k2); ok {
			return resolvedFont{font: f, size: key.Size, valid: true}
		}
	}
	// Try regular weight
	if key.Weight != 400 {
		k2 := FontKey{Family: key.Family, Size: key.Size, Weight: 400, Italic: false}
		if f, ok := c.cache.lookupFont(k2); ok {
			return resolvedFont{font: f, size: key.Size, valid: true}
		}
	}
	// Raylib default font
	return resolvedFont{font: rl.GetFontDefault(), size: key.Size, valid: true}
}

// parseFontString parses a CSS font string into a FontKey.
//
// Supported forms (all appear in shevo):
//
//	"13px 'Fira Code', monospace"
//	"italic 700 13px Inter"
//	"bold 12px system-ui"
//	"500 9px 'Fira Code'"
//	"italic bold 13px Inter"
//
// Parse order: optional style (italic) → optional weight (bold/NNN) →
// size (NNpx) → family name.
func parseFontString(s string) FontKey {
	s = strings.TrimSpace(s)
	key := FontKey{Weight: 400}

	// Pull tokens one at a time from the left until we hit the size.
	tokens := strings.Fields(s)
	i := 0

	for i < len(tokens) {
		t := strings.ToLower(tokens[i])

		if t == "italic" {
			key.Italic = true
			i++
			continue
		}
		if t == "normal" || t == "oblique" {
			i++
			continue
		}
		if t == "bold" {
			key.Weight = 700
			i++
			continue
		}
		// Numeric weight e.g. "500" or "700"
		if w, err := strconv.Atoi(t); err == nil && w >= 100 && w <= 900 {
			key.Weight = w
			i++
			continue
		}
		// Size token: "13px" or "13px," (with trailing comma/quote artifacts)
		if strings.Contains(t, "px") {
			sizeStr := strings.Split(t, "px")[0]
			// Handle composite like "13px" directly in a template string
			sizeStr = strings.Trim(sizeStr, "'\"`)}")
			if sz, err := strconv.ParseFloat(sizeStr, 32); err == nil {
				key.Size = float32(sz)
				i++
				break // family starts here
			}
		}
		// Unrecognised token before size — skip
		i++
	}

	// Remaining tokens form the family name. Strip CSS fallbacks after comma.
	family := strings.Join(tokens[i:], " ")
	if idx := strings.Index(family, ","); idx >= 0 {
		family = family[:idx]
	}
	family = strings.Trim(family, " '\"")
	key.Family = normaliseFamilyName(family)

	if key.Size == 0 {
		key.Size = 12 // safe default
	}
	return key
}

// normaliseFamilyName maps CSS family names to canonical registry keys.
func normaliseFamilyName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case strings.Contains(s, "fira"):
		return "fira"
	case strings.Contains(s, "inter"):
		return "inter"
	case strings.Contains(s, "georgia"):
		return "georgia"
	case strings.Contains(s, "cascadia"):
		return "fira" // treat as monospace fallback
	case strings.Contains(s, "jetbrains"):
		return "fira" // treat as monospace fallback
	case s == "monospace", s == "ui-monospace":
		return "fira"
	case s == "system-ui", s == "-apple-system", s == "sans-serif":
		return "inter" // system-ui maps to Inter in shevo
	case s == "serif":
		return "georgia"
	default:
		return "inter"
	}
}

// measureText returns the rendered width of text in the current font.
// Equivalent to ctx.measureText(text).width in JavaScript.
// spacing is always 0 — see ARCHITECTURE.md §8.
func measureText(f resolvedFont, text string) float32 {
	if !f.valid {
		return 0
	}
	v := rl.MeasureTextEx(f.font, text, f.size, f.spacing)
	return v.X
}
