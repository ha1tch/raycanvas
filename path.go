package raycanvas

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// TessellationSteps controls the number of line segments used to approximate
// curved path segments (arcs, arcTo). 32 is sufficient for arcs up to ~200px
// radius at typical screen resolutions. Increase for very large arcs.
const TessellationSteps = 32

// segKind identifies the type of a path segment.
type segKind uint8

const (
	segMove segKind = iota
	segLine
	segArc        // ctx.arc(x, y, r, start, end, ccw)
	segArcTo      // ctx.arcTo(x1, y1, x2, y2, r)
	segBezierTo   // ctx.bezierCurveTo(cp1x, cp1y, cp2x, cp2y, x, y)
	segClose
	segRect       // ctx.rect(x, y, w, h)
	segRoundRect  // ctx.roundRect(x, y, w, h, r)
)

type pathSegment struct {
	kind segKind
	// General-purpose coordinate fields.
	// Meaning depends on kind — see comments per case in path methods.
	x, y, x2, y2, x3, y3 float32
	r                      float32
	anticlockwise          bool
}

// pathBuffer accumulates path segments between BeginPath and Fill/Stroke/Clip.
type pathBuffer struct {
	segs    []pathSegment
	curX    float32
	curY    float32
	hasPath bool
}

func (p *pathBuffer) reset() {
	p.segs = p.segs[:0]
	p.curX = 0
	p.curY = 0
	p.hasPath = false
}

// --- Path building methods (called on Context, delegated here) ---------------

func (c *Context) BeginPath() {
	c.path.reset()
}

func (c *Context) ClosePath() {
	c.path.segs = append(c.path.segs, pathSegment{kind: segClose})
}

func (c *Context) MoveTo(x, y float32) {
	c.path.curX, c.path.curY = x, y
	c.path.segs = append(c.path.segs, pathSegment{kind: segMove, x: x, y: y})
	c.path.hasPath = true
}

func (c *Context) LineTo(x, y float32) {
	c.path.segs = append(c.path.segs, pathSegment{kind: segLine, x: x, y: y})
	c.path.curX, c.path.curY = x, y
	c.path.hasPath = true
}

// Arc adds an arc segment.
// x, y: centre; r: radius; startAngle, endAngle: radians; anticlockwise: direction.
// Matches ctx.arc() exactly.
func (c *Context) Arc(x, y, r, startAngle, endAngle float32, anticlockwise bool) {
	c.path.segs = append(c.path.segs, pathSegment{
		kind:          segArc,
		x:             x,
		y:             y,
		r:             r,
		x2:            startAngle,
		y2:            endAngle,
		anticlockwise: anticlockwise,
	})
	// Update current point to end of arc.
	c.path.curX = x + r*float32(math.Cos(float64(endAngle)))
	c.path.curY = y + r*float32(math.Sin(float64(endAngle)))
	c.path.hasPath = true
}

// ArcTo adds an arc-to segment (rounded corner primitive).
// Matches ctx.arcTo(x1, y1, x2, y2, radius).
// x/y = first control point, x2/y2 = second control point, r = radius.
func (c *Context) ArcTo(x1, y1, x2, y2, r float32) {
	c.path.segs = append(c.path.segs, pathSegment{
		kind: segArcTo,
		x:    x1, y: y1,
		x2: x2, y2: y2,
		r: r,
	})
	c.path.curX, c.path.curY = x2, y2
	c.path.hasPath = true
}

// BezierCurveTo adds a cubic Bézier segment.
// Matches ctx.bezierCurveTo(cp1x, cp1y, cp2x, cp2y, x, y).
func (c *Context) BezierCurveTo(cp1x, cp1y, cp2x, cp2y, x, y float32) {
	c.path.segs = append(c.path.segs, pathSegment{
		kind: segBezierTo,
		x: cp1x, y: cp1y,
		x2: cp2x, y2: cp2y,
		x3: x, y3: y,
	})
	c.path.curX, c.path.curY = x, y
	c.path.hasPath = true
}

// Rect adds a closed rectangle sub-path.
// Matches ctx.rect(x, y, w, h).
func (c *Context) Rect(x, y, w, h float32) {
	c.path.segs = append(c.path.segs, pathSegment{
		kind: segRect,
		x: x, y: y, x2: w, y2: h,
	})
	c.path.hasPath = true
}

// RoundRect adds a closed rounded-rectangle sub-path.
// r is a pixel radius (NOT a 0–1 ratio; that conversion happens inside
// draw.go when dispatching to rl.DrawRectangleRounded).
// Matches ctx.roundRect(x, y, w, h, r).
func (c *Context) RoundRect(x, y, w, h, r float32) {
	c.path.segs = append(c.path.segs, pathSegment{
		kind: segRoundRect,
		x: x, y: y, x2: w, y2: h,
		r: r,
	})
	c.path.hasPath = true
}

// --- Clip shape detection ----------------------------------------------------

type clipShape int

const (
	clipShapeOther     clipShape = iota
	clipShapeRect
	clipShapeRoundRect
)

// clipShape inspects the path to determine whether it is a single rect,
// a single roundRect, or something more complex.
func (p *pathBuffer) clipShape() clipShape {
	// Filter out leading Move segments.
	segs := p.segs
	for len(segs) > 0 && segs[0].kind == segMove {
		segs = segs[1:]
	}
	if len(segs) == 1 {
		switch segs[0].kind {
		case segRect:
			return clipShapeRect
		case segRoundRect:
			return clipShapeRoundRect
		}
	}
	return clipShapeOther
}

// roundRectParams holds the geometry for a roundRect clip.
type roundRectParams struct {
	x, y, w, h, r float32
}

// asRoundRect extracts rounded rect parameters from the path.
// Returns false if the path is not a single roundRect.
func (p *pathBuffer) asRoundRect() (roundRectParams, bool) {
	for _, s := range p.segs {
		if s.kind == segRoundRect {
			return roundRectParams{x: s.x, y: s.y, w: s.x2, h: s.y2, r: s.r}, true
		}
	}
	return roundRectParams{}, false
}

// boundingRect returns the axis-aligned bounding box of the path in local space.
// Only handles rect and roundRect segments; other segments use their endpoints.
func (p *pathBuffer) boundingRect() rl.Rectangle {
	var minX, minY, maxX, maxY float32
	first := true
	expand := func(x, y float32) {
		if first {
			minX, minY, maxX, maxY = x, y, x, y
			first = false
			return
		}
		if x < minX {
			minX = x
		}
		if y < minY {
			minY = y
		}
		if x > maxX {
			maxX = x
		}
		if y > maxY {
			maxY = y
		}
	}
	for _, s := range p.segs {
		switch s.kind {
		case segRect, segRoundRect:
			expand(s.x, s.y)
			expand(s.x+s.x2, s.y+s.y2)
		case segMove, segLine:
			expand(s.x, s.y)
		case segArc:
			expand(s.x-s.r, s.y-s.r)
			expand(s.x+s.r, s.y+s.r)
		default:
			expand(s.x, s.y)
			expand(s.x2, s.y2)
		}
	}
	if first {
		return rl.Rectangle{}
	}
	return rl.Rectangle{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// roundRectMaskKey builds a cache key for a roundRect mask texture.
func roundRectMaskKey(rr roundRectParams, w, h int32) string {
	return fmt.Sprintf("rrmask|%.2f|%.2f|%.2f|%.2f|%.2f|%d|%d",
		rr.x, rr.y, rr.w, rr.h, rr.r, w, h)
}

// --- Tessellation ------------------------------------------------------------

// tessellate converts the path buffer into a slice of sub-paths, each being
// a []rl.Vector2. A new sub-path begins at every segMove. This is the correct
// representation for stroke dispatch: each sub-path is drawn independently,
// preventing spurious connecting lines between MoveTo discontinuities.
//
// For fill, use tessellateFlat which returns a single merged point slice.
func (c *Context) tessellate() [][]rl.Vector2 {
	var subPaths [][]rl.Vector2
	var cur []rl.Vector2
	m := c.state.transform

	add := func(x, y float32) {
		tx, ty := m.transformPoint(x, y)
		cur = append(cur, rl.Vector2{X: tx, Y: ty})
	}
	flush := func() {
		if len(cur) > 0 {
			subPaths = append(subPaths, cur)
			cur = nil
		}
	}

	for _, s := range c.path.segs {
		switch s.kind {
		case segMove:
			flush() // end previous sub-path at every MoveTo
			tx, ty := m.transformPoint(s.x, s.y)
			cur = append(cur, rl.Vector2{X: tx, Y: ty})

		case segLine:
			add(s.x, s.y)

		case segClose:
			if len(cur) > 0 {
				cur = append(cur, cur[0]) // close by repeating first point
			}
			flush()

		case segRect:
			flush()
			add(s.x, s.y)
			add(s.x+s.x2, s.y)
			add(s.x+s.x2, s.y+s.y2)
			add(s.x, s.y+s.y2)
			add(s.x, s.y)
			flush()

		case segRoundRect:
			flush()
			cur = append(cur, tessellateRoundRect(s.x, s.y, s.x2, s.y2, s.r, m)...)
			flush()

		case segArc:
			// Arc without a preceding MoveTo starts a new sub-path.
			if len(cur) == 0 {
				flush()
			}
			cur = append(cur, tessellateArc(s.x, s.y, s.r, s.x2, s.y2, s.anticlockwise, m)...)

		case segArcTo:
			var px, py float32
			if len(cur) > 0 {
				last := cur[len(cur)-1]
				px = (last.X - m.e) / m.a
				py = (last.Y - m.f) / m.d
			}
			arcPts := tessellateArcTo(px, py, s.x, s.y, s.x2, s.y2, s.r)
			for _, p := range arcPts {
				tx, ty := m.transformPoint(p[0], p[1])
				cur = append(cur, rl.Vector2{X: tx, Y: ty})
			}

		case segBezierTo:
			cur = append(cur, tessellateCubicBezier(
				c.path.curX, c.path.curY,
				s.x, s.y, s.x2, s.y2, s.x3, s.y3, m,
			)...)
		}
	}
	flush()
	return subPaths
}

// tessellateFlat returns all sub-paths merged into a single []rl.Vector2,
// used for fill operations (DrawTriangleFan) where connectivity is desired.
func (c *Context) tessellateFlat() []rl.Vector2 {
	subPaths := c.tessellate()
	var pts []rl.Vector2
	for _, sp := range subPaths {
		pts = append(pts, sp...)
	}
	return pts
}

// tessellateArc returns tessellated points for an arc segment.
func tessellateArc(cx, cy, r, start, end float32, ccw bool, m matrix3x2) []rl.Vector2 {
	if ccw {
		for end > start {
			end -= 2 * math.Pi
		}
	} else {
		for end < start {
			end += 2 * math.Pi
		}
	}
	sweep := end - start
	steps := TessellationSteps
	pts := make([]rl.Vector2, 0, steps+1)
	for i := 0; i <= steps; i++ {
		t := start + sweep*float32(i)/float32(steps)
		lx := cx + r*float32(math.Cos(float64(t)))
		ly := cy + r*float32(math.Sin(float64(t)))
		tx, ty := m.transformPoint(lx, ly)
		pts = append(pts, rl.Vector2{X: tx, Y: ty})
	}
	return pts
}

// tessellateRoundRect returns tessellated points for a rounded rect.
func tessellateRoundRect(x, y, w, h, r float32, m matrix3x2) []rl.Vector2 {
	r = minF32(r, minF32(w/2, h/2))
	var pts []rl.Vector2
	add := func(lx, ly float32) {
		tx, ty := m.transformPoint(lx, ly)
		pts = append(pts, rl.Vector2{X: tx, Y: ty})
	}
	addArc := func(cx, cy, start, end float32) {
		for i := 0; i <= TessellationSteps/4; i++ {
			t := start + (end-start)*float32(i)/float32(TessellationSteps/4)
			add(cx+r*float32(math.Cos(float64(t))), cy+r*float32(math.Sin(float64(t))))
		}
	}
	pi := float32(math.Pi)
	// Top edge → top-right corner → right edge → bottom-right → bottom → bottom-left → left → top-left
	add(x+r, y)
	addArc(x+w-r, y+r, -pi/2, 0)
	addArc(x+w-r, y+h-r, 0, pi/2)
	addArc(x+r, y+h-r, pi/2, pi)
	addArc(x+r, y+r, pi, 3*pi/2)
	add(x+r, y) // close
	return pts
}

// tessellateArcTo computes the arc-to geometry.
// Returns local-space [x,y] pairs (not yet transformed).
func tessellateArcTo(px, py, x1, y1, x2, y2, r float32) [][2]float32 {
	// Standard arcTo algorithm: compute tangent points, then arc.
	// Vector from current point to first control.
	d1x, d1y := x1-px, y1-py
	d2x, d2y := x2-x1, y2-y1

	len1 := float32(math.Sqrt(float64(d1x*d1x + d1y*d1y)))
	len2 := float32(math.Sqrt(float64(d2x*d2x + d2y*d2y)))
	if len1 < 1e-6 || len2 < 1e-6 || r < 1e-6 {
		return [][2]float32{{x1, y1}}
	}

	// Normalise
	d1x, d1y = d1x/len1, d1y/len1
	d2x, d2y = d2x/len2, d2y/len2

	// Angle between the two directions
	cosAngle := d1x*d2x + d1y*d2y
	if cosAngle >= 1.0 {
		return [][2]float32{{x1, y1}}
	}
	sinAngle := float32(math.Sqrt(float64(1 - cosAngle*cosAngle)))
	if sinAngle < 1e-6 {
		return [][2]float32{{x1, y1}}
	}

	// Distance from control point to tangent points
	tanLen := r * (1 + cosAngle) / sinAngle
	if tanLen < 0 {
		tanLen = -tanLen
	}

	// Tangent points
	t1x := x1 - d1x*tanLen
	t1y := y1 - d1y*tanLen
	t2x := x1 + d2x*tanLen
	t2y := y1 + d2y*tanLen

	// Arc centre: perpendicular to d1 at t1
	perpX := -d1y
	perpY := d1x
	// Determine which side the centre is on
	cross := d1x*d2y - d1y*d2x
	if cross > 0 {
		perpX, perpY = -perpX, -perpY
	}
	cx := t1x + perpX*r
	cy := t1y + perpY*r

	// Start and end angles
	startAngle := float32(math.Atan2(float64(t1y-cy), float64(t1x-cx)))
	endAngle := float32(math.Atan2(float64(t2y-cy), float64(t2x-cx)))
	ccw := cross > 0

	steps := TessellationSteps / 4
	if steps < 4 {
		steps = 4
	}
	pts := make([][2]float32, 0, steps+2)
	pts = append(pts, [2]float32{t1x, t1y})

	sweep := endAngle - startAngle
	if !ccw && sweep > 0 {
		sweep -= 2 * math.Pi
	}
	if ccw && sweep < 0 {
		sweep += 2 * math.Pi
	}
	for i := 1; i <= steps; i++ {
		t := startAngle + sweep*float32(i)/float32(steps)
		pts = append(pts, [2]float32{
			cx + r*float32(math.Cos(float64(t))),
			cy + r*float32(math.Sin(float64(t))),
		})
	}
	return pts
}

// tessellateCubicBezier returns tessellated screen-space points for a cubic Bézier.
// Used for fill/stroke paths; for direct stroke use DrawSplineSegmentBezierCubic instead.
func tessellateCubicBezier(p0x, p0y, cp1x, cp1y, cp2x, cp2y, p1x, p1y float32, m matrix3x2) []rl.Vector2 {
	pts := make([]rl.Vector2, 0, TessellationSteps+1)
	for i := 0; i <= TessellationSteps; i++ {
		t := float32(i) / float32(TessellationSteps)
		u := 1 - t
		x := u*u*u*p0x + 3*u*u*t*cp1x + 3*u*t*t*cp2x + t*t*t*p1x
		y := u*u*u*p0y + 3*u*u*t*cp1y + 3*u*t*t*cp2y + t*t*t*p1y
		tx, ty := m.transformPoint(x, y)
		pts = append(pts, rl.Vector2{X: tx, Y: ty})
	}
	return pts
}
