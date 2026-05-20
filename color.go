package raycanvas

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

// ParseColor parses a CSS color string and returns color.RGBA.
// This is the public version of resolveColor for use outside a Context —
// useful when passing colours to DrawIcon or other calls that take color.RGBA directly.
func ParseColor(css string) color.RGBA {
	return parseColor(css)
}

// resolveColor parses a CSS color string and returns color.RGBA.
// Results are cached in the SharedCache — parsing happens only once per
// unique string across the lifetime of the application.
//
// Supported formats (all appear in shevo source):
//   - #rgb          3-digit hex, digits doubled
//   - #rrggbb       6-digit hex
//   - rgba(r,g,b,a) r/g/b 0–255 integers, a 0.0–1.0 float
//   - rgb(r,g,b)    r/g/b 0–255 integers
//
// "currentColor" is NOT handled here — callers substitute the current
// fill or stroke colour before calling resolveColor.
func (c *Context) resolveColor(css string) color.RGBA {
	css = strings.TrimSpace(css)
	if col, ok := c.cache.lookupColor(css); ok {
		return col
	}
	col := parseColor(css)
	c.cache.storeColor(css, col)
	return col
}

// applyAlpha multiplies a resolved color's alpha channel by the context's
// globalAlpha, returning the modulated color for the actual draw call.
// color.RGBA.A is uint8 (0–255); globalAlpha is float32 (0.0–1.0).
func applyAlpha(col color.RGBA, globalAlpha float32) color.RGBA {
	if globalAlpha >= 1.0 {
		return col
	}
	col.A = uint8(float32(col.A) * globalAlpha)
	return col
}

// parseColor is the raw parser, called only on cache miss.
func parseColor(css string) color.RGBA {
	if strings.HasPrefix(css, "#") {
		return parseHex(css)
	}
	lower := strings.ToLower(css)
	if strings.HasPrefix(lower, "rgba(") {
		return parseRGBA(css)
	}
	if strings.HasPrefix(lower, "rgb(") {
		return parseRGB(css)
	}
	if named, ok := namedColors[lower]; ok {
		return named
	}
	// Unrecognised — return transparent magenta as a visible error signal.
	return color.RGBA{R: 255, G: 0, B: 255, A: 0}
}

func parseHex(s string) color.RGBA {
	s = strings.TrimPrefix(s, "#")
	switch len(s) {
	case 3:
		r, _ := strconv.ParseUint(string([]byte{s[0], s[0]}), 16, 8)
		g, _ := strconv.ParseUint(string([]byte{s[1], s[1]}), 16, 8)
		b, _ := strconv.ParseUint(string([]byte{s[2], s[2]}), 16, 8)
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
	case 6:
		r, _ := strconv.ParseUint(s[0:2], 16, 8)
		g, _ := strconv.ParseUint(s[2:4], 16, 8)
		b, _ := strconv.ParseUint(s[4:6], 16, 8)
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
	case 8:
		r, _ := strconv.ParseUint(s[0:2], 16, 8)
		g, _ := strconv.ParseUint(s[2:4], 16, 8)
		b, _ := strconv.ParseUint(s[4:6], 16, 8)
		a, _ := strconv.ParseUint(s[6:8], 16, 8)
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
	}
	return color.RGBA{A: 255}
}

// parseRGBA handles "rgba(r, g, b, a)" where a is 0.0–1.0.
func parseRGBA(s string) color.RGBA {
	inner := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(s), "rgba("), ")")
	parts := splitComma(inner)
	if len(parts) != 4 {
		return color.RGBA{A: 255}
	}
	r := clampUint8(parseIntOrZero(parts[0]))
	g := clampUint8(parseIntOrZero(parts[1]))
	b := clampUint8(parseIntOrZero(parts[2]))
	af, _ := strconv.ParseFloat(strings.TrimSpace(parts[3]), 32)
	if af < 0 {
		af = 0
	} else if af > 1 {
		af = 1
	}
	a := uint8(af * 255)
	return color.RGBA{R: r, G: g, B: b, A: a}
}

// parseRGB handles "rgb(r, g, b)".
func parseRGB(s string) color.RGBA {
	inner := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(s), "rgb("), ")")
	parts := splitComma(inner)
	if len(parts) != 3 {
		return color.RGBA{A: 255}
	}
	r := clampUint8(parseIntOrZero(parts[0]))
	g := clampUint8(parseIntOrZero(parts[1]))
	b := clampUint8(parseIntOrZero(parts[2]))
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parseIntOrZero(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}

func clampUint8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// shadowCacheKey builds a deterministic cache key for the shadow pipeline.
func shadowCacheKey(col color.RGBA, blur, w, h float32, shapeID string) string {
	// blur is formatted as integer (%.0f) because drawShadowTexture snaps to
	// nearest integer before building the texture. This ensures keys are shared
	// across sub-integer blur values and prevents cache churn from animated blurs.
	return fmt.Sprintf("%02x%02x%02x%02x|%.0f|%.0fx%.0f|%s",
		col.R, col.G, col.B, col.A, blur, w, h, shapeID)
}

// namedColors — the subset that actually appears in shevo source.
// Extend as needed.
var namedColors = map[string]color.RGBA{
	"transparent": {0, 0, 0, 0},
	"white":       {255, 255, 255, 255},
	"black":       {0, 0, 0, 255},
	"red":         {255, 0, 0, 255},
	"green":       {0, 128, 0, 255},
	"blue":        {0, 0, 255, 255},
}
