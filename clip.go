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

	// rounds lists the rounded clips open at this level, outermost first.
	// Restore closes the ones the restored level does not have.
	rounds []roundClip
}

// roundClip is one rounded clip, fixed when Clip is called: its rectangle
// and radius in screen space, and the colour its corners are painted with.
type roundClip struct {
	rect rl.Rectangle
	r    float32
	bg   color.RGBA
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
//   - If the current path is a roundRect: scissor mode for the bounding box,
//     and when the clip is closed by Restore the four corners are painted
//     over in a background colour. This only looks like a rounded clip over
//     a solid colour: the mask background (SetMaskBackground) or, failing
//     that, the fill style in force when Clip is called. Over anything else
//     (a picture, a gradient, another window's content) draw the content
//     to an off-screen Context and place it with DrawImageOffscreenRounded,
//     or use DrawImageRounded for a texture: those round the geometry itself.
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
		// Fix the geometry and the corner colour now: by the time Restore
		// runs, the transform and the fill style belong to the content.
		m := c.state.transform
		sx, sy := m.transformPoint(rr.x, rr.y)
		rounds := make([]roundClip, len(c.state.clip.rounds), len(c.state.clip.rounds)+1)
		copy(rounds, c.state.clip.rounds)
		c.state.clip.rounds = append(rounds, roundClip{
			rect: rl.Rectangle{X: sx, Y: sy, Width: rr.w * m.a, Height: rr.h * m.d},
			r:    rr.r * m.a,
			bg:   c.autoMaskBackground(),
		})
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

// closeRoundClip paints the four corners of a rounded clip in its background
// colour: for each corner, the part of the corner square that lies outside
// the quarter circle. Restore calls it while the clip's scissor is still in
// force, after the clipped content has been drawn.
//
// With antialiasing on, the painted edge is a soft band, so the clipped
// content meets the background smoothly.
func (c *Context) closeRoundClip(rc roundClip) {
	half := minF32(rc.rect.Width, rc.rect.Height) / 2
	r := rc.r
	if r > half {
		r = half
	}
	if r <= 0 {
		return
	}
	seg := cornerSegments(r)
	// A pixel beyond the rectangle, so the soft edge has no gap at the ends
	// of each arc; the scissor trims it.
	out := rl.Rectangle{X: rc.rect.X - 1, Y: rc.rect.Y - 1, Width: rc.rect.Width + 2, Height: rc.rect.Height + 2}
	corners := [4]rl.Vector2{
		{X: out.X, Y: out.Y},
		{X: out.X + out.Width, Y: out.Y},
		{X: out.X + out.Width, Y: out.Y + out.Height},
		{X: out.X, Y: out.Y + out.Height},
	}
	solid, soft := float32(0), float32(0)
	if c.antialias {
		solid, soft = feather/2.0, -feather/2.0
	}
	edge := roundRectOutline(rc.rect, r, solid, seg)
	var inner []rl.Vector2
	if c.antialias {
		inner = roundRectOutline(rc.rect, r, soft, seg)
	}
	p := meshPaint{col: rc.bg}
	p.begin()
	for k := 0; k < 4; k++ {
		arc := edge[k*(seg+1) : (k+1)*(seg+1)]
		// Square off the arc's ends onto the enlarged rectangle.
		first, last := arc[0], arc[seg]
		switch k {
		case 0:
			first.X, last.Y = out.X, out.Y
		case 1:
			first.Y, last.X = out.Y, out.X+out.Width
		case 2:
			first.X, last.Y = out.X+out.Width, out.Y+out.Height
		case 3:
			first.Y, last.X = out.Y+out.Height, out.X
		}
		p.vertex(corners[k], 1)
		p.vertex(first, 1)
		p.vertex(arc[0], 1)
		for i := 0; i < seg; i++ {
			p.vertex(corners[k], 1)
			p.vertex(arc[i], 1)
			p.vertex(arc[i+1], 1)
		}
		p.vertex(corners[k], 1)
		p.vertex(arc[seg], 1)
		p.vertex(last, 1)
		if c.antialias {
			meshBand(p, inner[k*(seg+1):(k+1)*(seg+1)], arc, 0, 1, false)
		}
	}
	p.end()
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
