package raycanvas

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// DrawImage draws an entire texture at the given position.
// Equivalent to ctx.drawImage(src, dx, dy) in JavaScript.
func (c *Context) DrawImage(src rl.Texture2D, dx, dy float32) {
	tx, ty := c.state.transform.transformPoint(dx, dy)
	filter := rl.FilterBilinear
	if !c.state.smoothing {
		filter = rl.FilterPoint
	}
	rl.SetTextureFilter(src, filter)
	rl.DrawTextureV(src, rl.Vector2{X: tx, Y: ty}, rl.White)
}

// DrawImageScaled draws an entire texture scaled to fill dst.
// Equivalent to ctx.drawImage(src, dx, dy, dw, dh) in JavaScript.
func (c *Context) DrawImageScaled(src rl.Texture2D, dst rl.Rectangle) {
	m := c.state.transform
	tx, ty := m.transformPoint(dst.X, dst.Y)
	sw := dst.Width * m.a
	sh := dst.Height * m.d
	filter := rl.FilterBilinear
	if !c.state.smoothing {
		filter = rl.FilterPoint
	}
	rl.SetTextureFilter(src, filter)
	srcRect := rl.Rectangle{
		X:      0,
		Y:      0,
		Width:  float32(src.Width),
		Height: float32(src.Height),
	}
	rl.DrawTexturePro(
		src,
		srcRect,
		rl.Rectangle{X: tx, Y: ty, Width: sw, Height: sh},
		rl.Vector2{},
		0,
		rl.White,
	)
}

// DrawImageCropped draws a sub-rectangle of src scaled to fill dst.
// Equivalent to ctx.drawImage(src, sx, sy, sw, sh, dx, dy, dw, dh) in JavaScript.
//
// RenderTexture2D Y-flip: when src comes from an off-screen Context (.Texture()),
// pass srcRect with negative Height to flip the image right-way-up:
//
//	srcRect := rl.Rectangle{X: 0, Y: 0, Width: float32(w), Height: -float32(h)}
//
// This is handled automatically when the source is an off-screen Context —
// use DrawImageOffscreen for that case.
func (c *Context) DrawImageCropped(src rl.Texture2D, srcRect, dst rl.Rectangle) {
	m := c.state.transform
	tx, ty := m.transformPoint(dst.X, dst.Y)
	sw := dst.Width * m.a
	sh := dst.Height * m.d
	filter := rl.FilterBilinear
	if !c.state.smoothing {
		filter = rl.FilterPoint
	}
	rl.SetTextureFilter(src, filter)
	rl.DrawTexturePro(
		src,
		srcRect,
		rl.Rectangle{X: tx, Y: ty, Width: sw, Height: sh},
		rl.Vector2{},
		0,
		rl.White,
	)
}

// DrawImageOffscreen blits an off-screen Context onto this Context.
// Handles the RenderTexture2D Y-flip automatically.
// dst specifies the destination rectangle in this Context's local space.
func (c *Context) DrawImageOffscreen(src *Context, dst rl.Rectangle) {
	if src.rt == nil {
		panic("raycanvas: DrawImageOffscreen: source is not an off-screen context")
	}
	tex := src.rt.Texture
	// Negative Height flips the Y axis to correct the render texture inversion.
	srcRect := rl.Rectangle{
		X:      0,
		Y:      0,
		Width:  float32(tex.Width),
		Height: -float32(tex.Height),
	}
	c.DrawImageCropped(tex, srcRect, dst)
}

// DrawImageOffscreenCropped blits a sub-rectangle of an off-screen Context.
// srcRect is in the source Context's local space (before Y-flip correction).
func (c *Context) DrawImageOffscreenCropped(src *Context, srcRect, dst rl.Rectangle) {
	if src.rt == nil {
		panic("raycanvas: DrawImageOffscreenCropped: source is not an off-screen context")
	}
	// Flip Y: in a RenderTexture2D, row 0 is at the bottom.
	// Negate height and adjust Y to sample the correct region.
	tex := src.rt.Texture
	flipped := rl.Rectangle{
		X:      srcRect.X,
		Y:      float32(tex.Height) - srcRect.Y - srcRect.Height,
		Width:  srcRect.Width,
		Height: -srcRect.Height,
	}
	c.DrawImageCropped(tex, flipped, dst)
}
