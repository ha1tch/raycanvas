package raycanvas

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// clipState holds the clip region active at one save/restore level.
// Rectangular clips use the raylib scissor mode; roundRect clips additionally
// carry a gg-backed alpha mask texture.
type clipState struct {
	// scissor is the axis-aligned bounding rect of all active clips at this
	// level. Stored in screen (pixel) space. Updated by intersectScissor.
	scissor rl.Rectangle
	// hasScissor is true when any clip has been applied at this level.
	hasScissor bool

	// maskTex is non-zero when a non-rectangular (e.g. roundRect) clip is
	// active. The texture is UNCOMPRESSED_GRAYSCALE; white = visible.
	// Zero value (ID==0) means no mask is active.
	maskTex rl.Texture2D
	hasMask bool
}

// noClip returns a clipState that allows the full canvas.
func noClip(width, height int32) clipState {
	return clipState{
		scissor: rl.Rectangle{X: 0, Y: 0, Width: float32(width), Height: float32(height)},
	}
}

// intersectScissor returns the intersection of the current scissor rect and r.
// If there was no previous scissor, r is taken as-is.
func (cs clipState) intersectScissor(r rl.Rectangle) clipState {
	if !cs.hasScissor {
		cs.scissor = r
		cs.hasScissor = true
		return cs
	}
	x1 := maxF32(cs.scissor.X, r.X)
	y1 := maxF32(cs.scissor.Y, r.Y)
	x2 := minF32(cs.scissor.X+cs.scissor.Width, r.X+r.Width)
	y2 := minF32(cs.scissor.Y+cs.scissor.Height, r.Y+r.Height)
	if x2 < x1 {
		x2 = x1
	}
	if y2 < y1 {
		y2 = y1
	}
	cs.scissor = rl.Rectangle{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}
	cs.hasScissor = true
	return cs
}

// applyScissor issues BeginScissorMode for the current clip state.
// If no clip is active, EndScissorMode is called.
// BeginScissorMode takes int32; we cast here — the only place.
func (cs clipState) applyScissor() {
	if !cs.hasScissor {
		rl.EndScissorMode()
		return
	}
	rl.BeginScissorMode(
		int32(cs.scissor.X),
		int32(cs.scissor.Y),
		int32(cs.scissor.Width),
		int32(cs.scissor.Height),
	)
}

// --- Clip method on Context --------------------------------------------------

// Clip uses the current path as a clipping region, intersecting with any
// existing clip. The current path is consumed (cleared after this call),
// matching HTML5 canvas behaviour.
//
// Implementation dispatch:
//   - If the current path is a single axis-aligned rect: scissor mode.
//   - If the current path is a roundRect: scissor mode for the bounding box
//     + gg alpha-mask texture for the rounded shape.
//   - All other paths: bounding-box scissor only (sufficient for shevo).
func (c *Context) Clip() {
	switch c.path.clipShape() {
	case clipShapeRect:
		r := c.path.boundingRect()
		r = c.transformRect(r)
		c.state.clip = c.state.clip.intersectScissor(r)
		c.state.clip.applyScissor()

	case clipShapeRoundRect:
		rr, _ := c.path.asRoundRect()
		// Scissor to the bounding box for coarse culling.
		r := c.path.boundingRect()
		r = c.transformRect(r)
		c.state.clip = c.state.clip.intersectScissor(r)
		c.state.clip.applyScissor()
		// Store the roundRect geometry for corner overdraw on Restore().
		c.maskedRR = rr
		c.state.clip.hasMask = true

	default:
		// Fallback: bounding box scissor.
		r := c.path.boundingRect()
		r = c.transformRect(r)
		c.state.clip = c.state.clip.intersectScissor(r)
		c.state.clip.applyScissor()
	}
	c.path.reset()
}

// applyRoundRectCornerOverdraw draws four small filled rounded-corner "caps"
// using the background colour over the corners of the clipped region, creating
// the visual effect of a rounded clip without GPU readback or nested RTs.
//
// This is called from Restore() when hasMask was true. It requires knowing the
// background colour — callers set c.maskBgColor before the clip.
//
// The technique: after all clipped content is drawn inside the scissor rect,
// we draw four quarter-circle "masks" in the background colour at each corner.
// Each cap is drawn as a filled circle sector positioned at the corner centre,
// covering only the pixels outside the rounded rect border.
func (c *Context) applyRoundRectCornerOverdrawBg(sc color.RGBA) {
	rr := c.maskedRR
	m := c.state.transform

	x, y, w, h, r := rr.x, rr.y, rr.w, rr.h, rr.r
	sr := r * m.a

	// Transform corners to screen space.
	sx, sy := m.transformPoint(x, y)
	// sw, sh := w*m.a, h*m.d  // not needed directly

	_ = w
	_ = h

	// Each corner: draw a solid square the size of the radius, then
	// punch the rounded corner back in using a filled circle in bg colour.
	// Top-left
	c.drawCornerCap(sx+sr, sy+sr, sr, 180, 270, sc)
	// Top-right
	c.drawCornerCap(sx+w*m.a-sr, sy+sr, sr, 270, 360, sc)
	// Bottom-right
	c.drawCornerCap(sx+w*m.a-sr, sy+h*m.d-sr, sr, 0, 90, sc)
	// Bottom-left
	c.drawCornerCap(sx+sr, sy+h*m.d-sr, sr, 90, 180, sc)
}

// drawCornerCap paints a corner overdraw cap:
// fills the corner square with bgCol, then draws a filled arc in bgCol
// to restore the rounded cutout.
func (c *Context) drawCornerCap(cx, cy, r, startDeg, endDeg float32, bgCol color.RGBA) {
	// The square corner fill
	var rx, ry float32
	switch {
	case startDeg == 180: // top-left: square is to the left and above
		rx, ry = cx-r, cy-r
	case startDeg == 270: // top-right: square is to the right and above
		rx, ry = cx, cy-r
	case startDeg == 0: // bottom-right: square is to the right and below
		rx, ry = cx, cy
	default: // bottom-left: square is to the left and below
		rx, ry = cx-r, cy
	}
	rl.DrawRectangleRec(rl.Rectangle{X: rx, Y: ry, Width: r, Height: r}, bgCol)
	// Re-carve the rounded corner: draw the filled arc (the part that should
	// remain transparent/background) — this IS the background colour circle sector.
	rl.DrawCircleSector(
		rl.Vector2{X: cx, Y: cy},
		r,
		startDeg, endDeg,
		16,
		bgCol,
	)
}

// transformRect applies the current transform to an axis-aligned rectangle,
// returning the screen-space bounding box. For non-rotated transforms (which
// is the only case in shevo — only translate+scale are used), this is exact.
func (c *Context) transformRect(r rl.Rectangle) rl.Rectangle {
	m := c.state.transform
	x1, y1 := m.transformPoint(r.X, r.Y)
	x2, y2 := m.transformPoint(r.X+r.Width, r.Y+r.Height)
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	return rl.Rectangle{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}
}

// --- helpers -----------------------------------------------------------------

func minF32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxF32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

