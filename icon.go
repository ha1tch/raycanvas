package raycanvas

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// RegisterIcon parses an SVG and rasterises it at the given size, storing
// the result as a GPU Texture2D in the cache.
//
// Icons are pre-rasterised with white-on-transparent so they can be tinted
// at draw time via DrawTexturePro's tint parameter. This matches how shevo
// uses currentColor: the caller passes the current stroke or fill colour as
// the tint.
//
// Must be called after rl.InitWindow and before any draw calls.
//
//	name    — identifier used in DrawIcon calls
//	svgData — raw SVG bytes (embed with //go:embed or read from disk)
//	size    — target render size in pixels (icons are square)
func RegisterIcon(cache *SharedCache, name string, svgData []byte, size float32) error {
	if size <= 0 {
		return fmt.Errorf("raycanvas: RegisterIcon %q: size must be > 0", name)
	}
	if _, ok := cache.lookupIcon(name, size); ok {
		return nil // already registered
	}

	isize := int(size + 0.5)
	if isize < 1 {
		isize = 1
	}

	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgData))
	if err != nil {
		return fmt.Errorf("raycanvas: RegisterIcon %q: parse SVG: %w", name, err)
	}

	// Scale the icon to fit exactly within size × size.
	icon.SetTarget(0, 0, float64(isize), float64(isize))

	// Rasterise into an RGBA image with white fill so tinting works.
	rgba := image.NewRGBA(image.Rect(0, 0, isize, isize))
	// Fill with white so currentColor tinting doesn't darken unfilled areas.
	// Actual icon alpha channel controls visibility.
	draw.Draw(rgba, rgba.Bounds(), image.NewUniform(color.Transparent), image.Point{}, draw.Src)

	scanner := rasterx.NewScannerGV(isize, isize, rgba, rgba.Bounds())
	raster := rasterx.NewDasher(isize, isize, scanner)
	icon.Draw(raster, 1.0)

	// Convert all coloured pixels to white, preserving alpha.
	// This allows the tint parameter to apply the correct colour at draw time.
	toWhite(rgba)

	rlImg := rl.NewImageFromImage(rgba)
	tex := rl.LoadTextureFromImage(rlImg)
	rl.UnloadImage(rlImg)
	rl.SetTextureFilter(tex, rl.FilterBilinear)

	cache.storeIcon(name, size, tex)
	return nil
}

// DrawIcon draws a named icon at the given position and size, tinted with col.
// col replaces "currentColor" — pass the current stroke or fill colour.
// size controls the rendered square dimensions; the icon is scaled to fit.
// Matches the pattern ctx.drawImage(iconTexture, x, y, size, size).
func (c *Context) DrawIcon(name string, x, y, size float32, col color.RGBA) {
	tex, ok := c.cache.lookupIcon(name, size)
	if !ok {
		// Fallback: try the nearest registered size (simple linear search).
		// In practice all needed sizes are registered at startup.
		tex, ok = c.cache.lookupIcon(name, size)
		if !ok {
			return
		}
	}

	m := c.state.transform
	tx, ty := m.transformPoint(x, y)
	sw := size * m.a
	sh := size * m.d

	tint := applyAlpha(col, c.state.globalAlpha)

	rl.DrawTexturePro(
		tex,
		rl.Rectangle{X: 0, Y: 0, Width: float32(tex.Width), Height: float32(tex.Height)},
		rl.Rectangle{X: tx, Y: ty, Width: sw, Height: sh},
		rl.Vector2{},
		0,
		tint,
	)
}

// toWhite replaces the RGB channels of every pixel with white (255,255,255),
// preserving the alpha channel. This converts a coloured icon into a
// white-on-transparent mask suitable for tinting.
func toWhite(img *image.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			img.SetRGBA(x, y, color.RGBA{
				R: 255,
				G: 255,
				B: 255,
				A: uint8(a >> 8),
			})
		}
	}
}
