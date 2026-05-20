package raycanvas

import (
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// roundnessFromRadius converts a pixel corner radius to raylib's 0–1 roundness
// ratio. raylib's roundness = (2 * radius) / min(width, height), clamped to 1.
func roundnessFromRadius(r, w, h float32) float32 {
	minDim := minF32(w, h)
	if minDim <= 0 {
		return 0
	}
	rn := (2 * r) / minDim
	if rn > 1 {
		rn = 1
	}
	return rn
}

const roundRectSegments int32 = 8 // sufficient for small UI radii

// --- Convenience round-rect helpers ------------------------------------------
// These mirror the two-step BeginPath+RoundRect+Fill/Stroke pattern that
// appears constantly in card chrome drawing code (quag uses rr() + fill()).
// Each is a single call matching the intent directly.

// FillRoundRect fills a rounded rectangle with the current fill style.
// Equivalent to: BeginPath(); RoundRect(x,y,w,h,r); Fill()
func (c *Context) FillRoundRect(x, y, w, h, r float32) {
	if c.shadowActive() {
		c.drawShadowForRoundRect(x, y, w, h, r)
	}
	c.BeginPath()
	c.RoundRect(x, y, w, h, r)
	c.Fill()
}

// StrokeRoundRect strokes a rounded rectangle with the current stroke style.
// Equivalent to: BeginPath(); RoundRect(x,y,w,h,r); Stroke()
func (c *Context) StrokeRoundRect(x, y, w, h, r float32) {
	c.BeginPath()
	c.RoundRect(x, y, w, h, r)
	c.Stroke()
}

// FillRoundRectTop fills a rectangle rounded only at the top two corners.
// This is the rrTop primitive from quag — used for title bars drawn over
// a card body. Equivalent to a rounded rect with the bottom squared off.
// Implemented as two stacked raylib calls to avoid path tessellation.
func (c *Context) FillRoundRectTop(x, y, w, h, r float32) {
	c.FillRoundRect(x, y, w, h, r)         // full rounded rect gives top corners
	c.FillRect(x, y+r, w, h-r)             // square off everything below corner radius
}

// StrokeRoundRectTop strokes a rectangle rounded only at the top two corners.
func (c *Context) StrokeRoundRectTop(x, y, w, h, r float32) {
	// Draw as three segments: two corner arcs + straight sides
	// Use path for stroke since we need a single connected outline
	c.BeginPath()
	c.MoveTo(x, y+h)
	c.LineTo(x, y+r)
	c.ArcTo(x, y, x+r, y, r)
	c.LineTo(x+w-r, y)
	c.ArcTo(x+w, y, x+w, y+r, r)
	c.LineTo(x+w, y+h)
	c.Stroke()
}

// FillCircle fills a full circle with the current fill style.
// Equivalent to: BeginPath(); Arc(x,y,r,0,2π,false); Fill()
func (c *Context) FillCircle(x, y, r float32) {
	if c.shadowActive() {
		// Approximate circle shadow as a rounded square of diameter 2r
		c.drawShadowForRoundRect(x-r, y-r, r*2, r*2, r)
	}
	c.BeginPath()
	c.Arc(x, y, r, 0, 2*3.141592653589793, false)
	c.Fill()
}

// StrokeCircle strokes a full circle with the current stroke style.
func (c *Context) StrokeCircle(x, y, r float32) {
	c.BeginPath()
	c.Arc(x, y, r, 0, 2*3.141592653589793, false)
	c.Stroke()
}

// --- Rectangle primitives ----------------------------------------------------

// FillRect fills an axis-aligned rectangle with the current fill style.
// Matches ctx.fillRect(x, y, w, h) in JavaScript.
func (c *Context) FillRect(x, y, w, h float32) {
	if w <= 0 || h <= 0 {
		return
	}
	if c.shadowActive() {
		c.drawShadowForRect(x, y, w, h)
	}
	tx, ty := c.state.transform.transformPoint(x, y)
	sw := w * c.state.transform.a
	sh := h * c.state.transform.d
	rl.DrawRectangleRec(
		rl.Rectangle{X: tx, Y: ty, Width: sw, Height: sh},
		c.fillColor(),
	)
}

// StrokeRect strokes an axis-aligned rectangle with the current stroke style.
// Matches ctx.strokeRect(x, y, w, h) in JavaScript.
func (c *Context) StrokeRect(x, y, w, h float32) {
	if w <= 0 || h <= 0 {
		return
	}
	tx, ty := c.state.transform.transformPoint(x, y)
	sw := w * c.state.transform.a
	sh := h * c.state.transform.d
	rl.DrawRectangleLinesEx(
		rl.Rectangle{X: tx, Y: ty, Width: sw, Height: sh},
		c.state.lineWidth,
		c.strokeColor(),
	)
}

// ClearRect clears a rectangle to transparent black, effectively erasing it.
// Matches ctx.clearRect(x, y, w, h) in JavaScript.
func (c *Context) ClearRect(x, y, w, h float32) {
	if w <= 0 || h <= 0 {
		return
	}
	tx, ty := c.state.transform.transformPoint(x, y)
	sw := w * c.state.transform.a
	sh := h * c.state.transform.d
	rl.DrawRectangleRec(
		rl.Rectangle{X: tx, Y: ty, Width: sw, Height: sh},
		rl.Color{R: 0, G: 0, B: 0, A: 0},
	)
}

// --- Path fill and stroke ----------------------------------------------------

// Fill fills the current path with the current fill style.
// Matches ctx.fill() in JavaScript.
// The path is cleared after this call.
func (c *Context) Fill() {
	if !c.path.hasPath {
		return
	}
	c.fillPath()
	c.path.reset()
}

// Stroke strokes the current path with the current stroke style.
// Matches ctx.stroke() in JavaScript.
// The path is cleared after this call.
func (c *Context) Stroke() {
	if !c.path.hasPath {
		return
	}
	c.strokePath()
	c.path.reset()
}

// fillPath dispatches the current path to the appropriate raylib fill call.
// Single-segment paths (rect, roundRect, full circle) use direct raylib
// primitives. Multi-segment paths tessellate to a triangle fan.
func (c *Context) fillPath() {
	segs := nonMoveSegs(c.path.segs)
	if len(segs) == 1 {
		s := segs[0]
		m := c.state.transform
		col := c.fillColor()
		switch s.kind {
		case segRect:
			tx, ty := m.transformPoint(s.x, s.y)
			rl.DrawRectangleRec(rl.Rectangle{
				X: tx, Y: ty,
				Width:  s.x2 * m.a,
				Height: s.y2 * m.d,
			}, col)
			return

		case segRoundRect:
			tx, ty := m.transformPoint(s.x, s.y)
			sw, sh := s.x2*m.a, s.y2*m.d
			rn := roundnessFromRadius(s.r*m.a, sw, sh)
			rl.DrawRectangleRounded(rl.Rectangle{
				X: tx, Y: ty, Width: sw, Height: sh,
			}, rn, roundRectSegments, col)
			return

		case segArc:
			// Full circle optimisation.
			if isFullCircle(s.x2, s.y2) {
				cx, cy := m.transformPoint(s.x, s.y)
				rl.DrawCircleV(rl.Vector2{X: cx, Y: cy}, s.r*m.a, col)
				return
			}
			// Partial arc: fall through to tessellation.
		}
	}

	pts := c.tessellateFlat()
	if len(pts) < 3 {
		return
	}
	// DrawTriangleFan requires points in CCW order in screen coordinates (Y-down).
	// Our tessellation generates points in increasing-angle order, which is CW
	// in screen coords. Reverse the perimeter points so they are CCW in screen
	// coords, then prepend the centroid as the fan centre.
	for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
		pts[i], pts[j] = pts[j], pts[i]
	}
	var cx, cy float32
	for _, p := range pts {
		cx += p.X
		cy += p.Y
	}
	cx /= float32(len(pts))
	cy /= float32(len(pts))
	fan := make([]rl.Vector2, 0, len(pts)+2)
	fan = append(fan, rl.Vector2{X: cx, Y: cy})
	fan = append(fan, pts...)
	fan = append(fan, pts[0]) // close fan
	rl.DrawTriangleFan(fan, c.fillColor())
}

// strokePath dispatches the current path to the appropriate raylib stroke call.
func (c *Context) strokePath() {
	segs := nonMoveSegs(c.path.segs)
	if len(segs) == 1 {
		s := segs[0]
		m := c.state.transform
		col := c.strokeColor()
		lw := c.state.lineWidth
		switch s.kind {
		case segRect:
			tx, ty := m.transformPoint(s.x, s.y)
			rl.DrawRectangleLinesEx(rl.Rectangle{
				X: tx, Y: ty,
				Width:  s.x2 * m.a,
				Height: s.y2 * m.d,
			}, lw, col)
			return

		case segRoundRect:
			tx, ty := m.transformPoint(s.x, s.y)
			sw, sh := s.x2*m.a, s.y2*m.d
			rn := roundnessFromRadius(s.r*m.a, sw, sh)
			rl.DrawRectangleRoundedLinesEx(rl.Rectangle{
				X: tx, Y: ty, Width: sw, Height: sh,
			}, rn, roundRectSegments, lw, col)
			return

		case segArc:
			cx, cy := m.transformPoint(s.x, s.y)
			sr := s.r * m.a
			if isFullCircle(s.x2, s.y2) {
				// Full circle: DrawRing with inner=outer-lw gives a clean ring stroke.
				inner := sr - lw/2
				if inner < 0 {
					inner = 0
				}
				rl.DrawRing(rl.Vector2{X: cx, Y: cy}, inner, sr+lw/2, 0, 360, 64, col)
				return
			}
			// Partial arc: DrawRing over the swept angle range.
			// DrawRing angles are in degrees; canvas uses radians.
			start := s.x2 * (180 / math.Pi)
			end   := s.y2 * (180 / math.Pi)
			if s.anticlockwise {
				start, end = end, start
			}
			inner := sr - lw/2
			if inner < 0 {
				inner = 0
			}
			segs := int32(sr * math.Pi / 4) // ~1 segment per 4px of arc length
			if segs < 16 {
				segs = 16
			}
			if segs > 128 {
				segs = 128
			}
			rl.DrawRing(rl.Vector2{X: cx, Y: cy}, inner, sr+lw/2, start, end, segs, col)
			return

		case segBezierTo:
			// Cubic Bézier stroke — rendered anti-aliased via gg, cached as
			// Texture2D keyed on geometry+lineWidth. Color/alpha applied as
			// tint at blit time so animated alpha changes don't bust the cache.
			var p0x, p0y float32
			for i := len(c.path.segs) - 2; i >= 0; i-- {
				prev := c.path.segs[i]
				if prev.kind == segMove || prev.kind == segLine {
					p0x, p0y = prev.x, prev.y
					break
				}
			}
			p0  := rl.Vector2{X: m.a*p0x + m.e, Y: m.d*p0y + m.f}
			cp1 := rl.Vector2{X: m.a*s.x + m.e,  Y: m.d*s.y + m.f}
			cp2 := rl.Vector2{X: m.a*s.x2 + m.e, Y: m.d*s.y2 + m.f}
			p1  := rl.Vector2{X: m.a*s.x3 + m.e, Y: m.d*s.y3 + m.f}
			cacheKey := bezierCacheKey(p0, cp1, cp2, p1, lw)
			tex, ok := c.cache.lookupShadow(cacheKey)
			var origin rl.Vector2
			if !ok {
				tex, origin = buildBezierTexture(p0, cp1, cp2, p1, lw)
				c.cache.storeShadow(cacheKey, tex)
				// Store origin alongside — encode in a second key.
				// We re-derive origin from the bounding box at draw time below.
			} else {
				// Re-derive origin (same formula as buildBezierTexture).
				pad := lw*2 + 2
				origin = rl.Vector2{
					X: min4(p0.X, cp1.X, cp2.X, p1.X) - pad,
					Y: min4(p0.Y, cp1.Y, cp2.Y, p1.Y) - pad,
				}
			}
			rl.DrawTexturePro(
				tex,
				rl.Rectangle{X: 0, Y: 0, Width: float32(tex.Width), Height: float32(tex.Height)},
				rl.Rectangle{X: origin.X, Y: origin.Y, Width: float32(tex.Width), Height: float32(tex.Height)},
				rl.Vector2{},
				0,
				col,
			)
			return
		}
	}

	subPaths := c.tessellate()
	if len(subPaths) == 0 {
		return
	}
	for _, pts := range subPaths {
		if len(pts) < 2 {
			continue
		}
		if len(c.state.lineDash) > 0 {
			c.strokeDashed(pts)
			continue
		}
		strokeSubPath(pts, c.state.lineWidth, c.strokeColor())
	}
}

// strokeDashed draws a dashed polyline by splitting the tessellated points
// into on/off segments according to the lineDash pattern.
func (c *Context) strokeDashed(pts []rl.Vector2) {
	if len(pts) < 2 {
		return
	}
	dash := c.state.lineDash
	dashLen := float32(0)
	for _, d := range dash {
		dashLen += d
	}
	if dashLen <= 0 {
		rl.DrawSplineLinear(pts, c.state.lineWidth, c.strokeColor())
		return
	}

	col := c.strokeColor()
	lw := c.state.lineWidth
	di := 0
	remaining := dash[0] - c.state.lineDashOff
	drawing := true
	var segStart rl.Vector2 = pts[0]

	for i := 1; i < len(pts); i++ {
		dx := pts[i].X - pts[i-1].X
		dy := pts[i].Y - pts[i-1].Y
		segLen := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		ux, uy := dx/segLen, dy/segLen
		pos := float32(0)

		for pos < segLen {
			step := minF32(remaining, segLen-pos)
			endX := pts[i-1].X + ux*(pos+step)
			endY := pts[i-1].Y + uy*(pos+step)
			end := rl.Vector2{X: endX, Y: endY}
			if drawing {
				rl.DrawLineEx(segStart, end, lw, col)
			}
			segStart = end
			pos += step
			remaining -= step
			if remaining <= 0 {
				di = (di + 1) % len(dash)
				remaining = dash[di]
				drawing = !drawing
			}
		}
	}
}

// --- Text --------------------------------------------------------------------

// FillText draws text at the given position using the current font and fill style.
// Matches ctx.fillText(text, x, y) in JavaScript.
//
// textAlign and textBaseline are applied to compute the final draw position.
func (c *Context) FillText(text string, x, y float32) {
	if text == "" || !c.state.font.valid {
		return
	}
	tx, ty := c.state.transform.transformPoint(x, y)
	tx, ty = c.applyTextAlign(text, tx, ty)
	// Text shadow: render the text at the shadow offset in the shadow colour.
	// This matches browser text-shadow behaviour exactly — a full copy of the
	// text rendered at (tx+offX, ty+offY) in the shadow colour, drawn first
	// so the main text sits on top. No blur (text blur would require gg rasterisation).
	if c.shadowActive() && (c.state.shadowOffX != 0 || c.state.shadowOffY != 0) {
		shadowCol := applyAlpha(c.state.shadowColor, c.state.globalAlpha)
		rl.DrawTextEx(
			c.state.font.font,
			text,
			rl.Vector2{X: tx + c.state.shadowOffX, Y: ty + c.state.shadowOffY},
			c.state.font.size,
			c.state.font.spacing,
			shadowCol,
		)
	}
	rl.DrawTextEx(
		c.state.font.font,
		text,
		rl.Vector2{X: tx, Y: ty},
		c.state.font.size,
		c.state.font.spacing,
		c.fillColor(),
	)
}

// MeasureText returns the rendered width of text in the current font.
// Matches ctx.measureText(text).width in JavaScript.
// Returns 0 if no font is set.
func (c *Context) MeasureText(text string) float32 {
	if !c.state.font.valid {
		return 0
	}
	return measureText(c.state.font, text)
}

// applyTextAlign adjusts the draw origin for textAlign and textBaseline.
func (c *Context) applyTextAlign(text string, x, y float32) (float32, float32) {
	f := c.state.font
	if !f.valid {
		return x, y
	}

	// Horizontal alignment
	switch c.state.textAlign {
	case alignCenter:
		w := measureText(f, text)
		x -= w / 2
	case alignRight:
		w := measureText(f, text)
		x -= w
	}

	// Vertical alignment — approximate using fontSize.
	// raylib DrawTextEx draws from the top-left of the glyph bounding box.
	// The canvas "alphabetic" baseline is approximately 0.8 * fontSize from top.
	sz := f.size
	// Baseline offsets for raylib fonts (DrawTextEx draws from glyph box top).
	// Raylib bakes fonts so the glyph box top is approximately at the ascender.
	// For Inter and Fira Code at 8–14px, empirical measurements give:
	//   alphabetic baseline ≈ 0.75 * fontSize from the top of the bounding box
	//   middle              ≈ 0.5  * fontSize
	//   bottom              ≈ 1.0  * fontSize (full descent)
	switch c.state.textBaseline {
	case baselineAlphabetic:
		y -= sz * 0.75
	case baselineMiddle:
		y -= sz * 0.5
	case baselineBottom:
		y -= sz
	case baselineTop, baselineHanging:
		// y is already the top of the glyph box — no adjustment
	}

	return x, y
}

// --- Helpers -----------------------------------------------------------------

// nonMoveSegs returns the path segments excluding leading Move segments.
func nonMoveSegs(segs []pathSegment) []pathSegment {
	for len(segs) > 0 && segs[0].kind == segMove {
		segs = segs[1:]
	}
	return segs
}

// strokeSubPath draws a single tessellated sub-path.
// Two-point paths (a single line segment) use DrawLineEx — crisp, 1px-capable,
// no polygon overhead. Longer paths use DrawSplineLinear.
// This is the key dispatch that fixes both the aliasing and the diagonal artifacts.
func strokeSubPath(pts []rl.Vector2, lineWidth float32, col color.RGBA) {
	if len(pts) < 2 {
		return
	}
	if len(pts) == 2 {
		rl.DrawLineEx(pts[0], pts[1], lineWidth, col)
		return
	}
	rl.DrawSplineLinear(pts, lineWidth, col)
}

// isFullCircle returns true when start..end spans a full revolution (2π),
// which is the only arc form used in shevo outside the chevron animation.
func isFullCircle(start, end float32) bool {
	const twoPi = 2 * math.Pi
	diff := float32(math.Abs(float64(end - start)))
	return diff >= float32(twoPi)-1e-4
}
