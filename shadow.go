package raycanvas

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/fogleman/gg"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// shadowActive returns true when shadow drawing is enabled.
// A shadow requires non-zero blur or non-zero offset with non-transparent colour.
func (c *Context) shadowActive() bool {
	if c.state.shadowColor.A == 0 {
		return false
	}
	return c.state.shadowBlur > 0 || c.state.shadowOffX != 0 || c.state.shadowOffY != 0
}

// drawShadowForRect draws a shadow for a rectangle using the cached blur pipeline.
// Called from FillRect and StrokeRect when shadow is active.
func (c *Context) drawShadowForRect(x, y, w, h float32) {
	if !c.shadowActive() {
		return
	}
	m := c.state.transform
	tx, ty := m.transformPoint(x, y)
	sw, sh := w*m.a, h*m.d
	key := shadowCacheKey(c.state.shadowColor, c.state.shadowBlur, sw, sh, "rect")
	pad := c.state.shadowBlur * 2
	c.drawShadowTexture(key, tx, ty, sw, sh, func(gc *gg.Context) {
		gc.SetColor(colorToGG(c.state.shadowColor))
		gc.DrawRectangle(float64(pad), float64(pad), float64(sw), float64(sh))
		gc.Fill()
	})
}

// drawShadowForRoundRect draws a shadow for a rounded rectangle.
func (c *Context) drawShadowForRoundRect(x, y, w, h, r float32) {
	if !c.shadowActive() {
		return
	}
	m := c.state.transform
	tx, ty := m.transformPoint(x, y)
	sw, sh := w*m.a, h*m.d
	sr := r * m.a
	key := shadowCacheKey(c.state.shadowColor, c.state.shadowBlur, sw, sh,
		shadowCacheKey(c.state.shadowColor, sr, 0, 0, "rrect"))
	pad2 := c.state.shadowBlur * 2
	c.drawShadowTexture(key, tx, ty, sw, sh, func(gc *gg.Context) {
		gc.SetColor(colorToGG(c.state.shadowColor))
		gc.DrawRoundedRectangle(
			float64(pad2), float64(pad2),
			float64(sw), float64(sh), float64(sr))
		gc.Fill()
	})
}

// drawShadowTexture is the core shadow pipeline.
// It retrieves or builds a blurred shadow texture and draws it offset from
// the shape position. The draw function renders the shadow shape into a gg
// context; the result is blurred and uploaded to GPU once, then cached.
func (c *Context) drawShadowTexture(key string, x, y, w, h float32, drawFn func(*gg.Context)) {
	// Snap blur to the nearest integer to ensure:
	//   1. The ImageBlurGaussian kernel (blurSize = int(blur/2)) steps smoothly
	//      without visual jumps between consecutive blur values.
	//   2. Cache entries are shared across sub-integer blur values, preventing
	//      the animated card from filling the FIFO cache and evicting other entries.
	blur := float32(math.Round(float64(c.state.shadowBlur)))
	if blur < 1 {
		blur = 1
	}
	pad := blur * 2 // padding so blur doesn't clip at the edges

	tex, ok := c.cache.lookupShadow(key)
	if !ok {
		tex = buildShadowTexture(w, h, blur, pad, drawFn)
		c.cache.storeShadow(key, tex)
	}

	// Draw shadow offset from shape position.
	rl.DrawTexturePro(
		tex,
		rl.Rectangle{X: 0, Y: 0, Width: float32(tex.Width), Height: float32(tex.Height)},
		rl.Rectangle{
			X:      x + c.state.shadowOffX - pad,
			Y:      y + c.state.shadowOffY - pad,
			Width:  float32(tex.Width),
			Height: float32(tex.Height),
		},
		rl.Vector2{},
		0,
		rl.White,
	)
}

// buildShadowTexture renders a shadow shape into a gg context, applies
// gaussian blur, uploads to GPU, and returns the Texture2D.
//
// Pipeline (CPU, runs once per unique shadow configuration):
//  1. Allocate gg context large enough to hold shape + blur padding
//  2. Call drawFn to render the shadow shape (positioned with padding offset)
//  3. Convert gg image → raylib Image
//  4. Apply ImageBlurGaussian (blurSize = blur/2, minimum 1)
//  5. LoadTextureFromImage → Texture2D
//  6. Unload intermediate Image
func buildShadowTexture(w, h, blur, pad float32, drawFn func(*gg.Context)) rl.Texture2D {
	imgW := int(w + pad*2 + 0.5)
	imgH := int(h + pad*2 + 0.5)
	if imgW < 1 {
		imgW = 1
	}
	if imgH < 1 {
		imgH = 1
	}

	gc := gg.NewContext(imgW, imgH)
	drawFn(gc)

	rgba := gc.Image().(*image.RGBA)
	rlImg := rl.NewImageFromImage(rgba)

	blurSize := int32(blur / 2)
	if blurSize < 1 {
		blurSize = 1
	}
	rl.ImageBlurGaussian(rlImg, blurSize)

	tex := rl.LoadTextureFromImage(rlImg)
	rl.UnloadImage(rlImg)
	return tex
}

// buildRoundRectMaskImage builds a CPU-side RGBA mask image for a roundRect clip.
// White pixels inside the rounded rect, transparent outside. Cached as *image.RGBA
// in the SharedCache so it is built at most once per unique geometry.
// Used by closeMaskedRegion: applied via rl.ImageAlphaMask to the offscreen content.
func buildRoundRectMaskImage(rr roundRectParams, canvasW, canvasH int32) *image.RGBA {
	gc := gg.NewContext(int(canvasW), int(canvasH))
	gc.SetColor(color.White)
	gc.DrawRoundedRectangle(
		float64(rr.x), float64(rr.y),
		float64(rr.w), float64(rr.h),
		float64(rr.r),
	)
	gc.Fill()
	return gc.Image().(*image.RGBA)
}

// buildRoundRectMask builds a GPU texture version of the mask — kept for
// any future shader-based compositing path.
func buildRoundRectMask(rr roundRectParams, canvasW, canvasH int32) rl.Texture2D {
	rgba := buildRoundRectMaskImage(rr, canvasW, canvasH)
	rlImg := rl.NewImageFromImage(rgba)
	rl.ImageFormat(rlImg, rl.UncompressedGrayscale)
	tex := rl.LoadTextureFromImage(rlImg)
	rl.UnloadImage(rlImg)
	return tex
}

// colorToGG converts a color.RGBA to gg's color interface.
func colorToGG(col color.RGBA) color.Color {
	return col
}

// --- Anti-aliased Bézier stroke cache ----------------------------------------

// bezierCacheKey builds a cache key for an anti-aliased cubic Bézier stroke.
// Keyed on geometry and lineWidth only — color/alpha applied as tint at blit time.
func bezierCacheKey(p0, cp1, cp2, p1 rl.Vector2, lineWidth float32) string {
	return fmt.Sprintf("bezier|%.2f,%.2f|%.2f,%.2f|%.2f,%.2f|%.2f,%.2f|%.2f",
		p0.X, p0.Y, cp1.X, cp1.Y, cp2.X, cp2.Y, p1.X, p1.Y, lineWidth)
}

// buildBezierTexture renders a cubic Bézier stroke into a gg context with
// anti-aliasing, uploads to GPU, and returns the Texture2D.
//
// The texture covers the bounding box of the curve expanded by pad pixels on
// each side to accommodate the stroke width and glow falloff. The curve is
// rendered white-on-transparent so tinting works at draw time.
//
// origin is the top-left of the bounding box in screen space — needed by the
// caller to position the blit correctly.
func buildBezierTexture(p0, cp1, cp2, p1 rl.Vector2, lineWidth float32) (rl.Texture2D, rl.Vector2) {
	pad := lineWidth*2 + 2

	// Bounding box of the four control points.
	minX := min4(p0.X, cp1.X, cp2.X, p1.X) - pad
	minY := min4(p0.Y, cp1.Y, cp2.Y, p1.Y) - pad
	maxX := max4(p0.X, cp1.X, cp2.X, p1.X) + pad
	maxY := max4(p0.Y, cp1.Y, cp2.Y, p1.Y) + pad

	imgW := int(maxX-minX+1)
	imgH := int(maxY-minY+1)
	if imgW < 1 {
		imgW = 1
	}
	if imgH < 1 {
		imgH = 1
	}

	gc := gg.NewContext(imgW, imgH)
	// Translate so that curve coords are relative to the texture origin.
	gc.Translate(float64(-minX), float64(-minY))
	gc.SetRGBA(1, 1, 1, 1)
	gc.SetLineWidth(float64(lineWidth))
	gc.SetLineCapRound()
	gc.MoveTo(float64(p0.X), float64(p0.Y))
	gc.CubicTo(
		float64(cp1.X), float64(cp1.Y),
		float64(cp2.X), float64(cp2.Y),
		float64(p1.X), float64(p1.Y),
	)
	gc.Stroke()

	rgba := gc.Image().(*image.RGBA)
	rlImg := rl.NewImageFromImage(rgba)
	tex := rl.LoadTextureFromImage(rlImg)
	rl.UnloadImage(rlImg)
	rl.SetTextureFilter(tex, rl.FilterBilinear)

	return tex, rl.Vector2{X: minX, Y: minY}
}

func min4(a, b, c, d float32) float32 {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	if d < m {
		m = d
	}
	return m
}

func max4(a, b, c, d float32) float32 {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	if d > m {
		m = d
	}
	return m
}
