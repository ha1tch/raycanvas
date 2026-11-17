package raycanvas

import (
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Rounded shapes as triangle meshes.
//
// raylib's own rounded rectangle has two limits that matter to a canvas:
// it cannot carry a texture, so an image cannot be given round corners
// without a known solid colour behind it, and its edge is only as smooth
// as the window's multisampling, which an application may not have asked
// for. The meshes here do both: every vertex carries a texture coordinate,
// and when the context's antialiasing is on (SetAntialias) the edge is a
// one-pixel band whose alpha falls from full to nothing, which is smooth
// with or without multisampling.

// feather is the width in pixels of the soft edge when antialiasing is on.
const feather = 1

// meshPaint says what a mesh is filled with: a flat colour, or a texture
// modulated by that colour.
type meshPaint struct {
	col      color.RGBA
	textured bool
	tex      rl.Texture2D
	src      rl.Rectangle // region of tex, in texels; negative Height flips, as raylib does
	dst      rl.Rectangle // screen rectangle that src is stretched over

	// With gradient set the colour is not flat: it runs from col at the
	// screen height y0 to col2 at y1, alpha included, and stays at the end
	// colours beyond them.
	gradient bool
	col2     color.RGBA
	y0, y1   float32
}

// colourAt is the paint's colour at screen height y.
func (p meshPaint) colourAt(y float32) color.RGBA {
	if !p.gradient || p.y1 == p.y0 {
		return p.col
	}
	t := clamp01((y - p.y0) / (p.y1 - p.y0))
	mix := func(a, b uint8) uint8 { return uint8(float32(a) + (float32(b)-float32(a))*t + 0.5) }
	return color.RGBA{R: mix(p.col.R, p.col2.R), G: mix(p.col.G, p.col2.G), B: mix(p.col.B, p.col2.B), A: mix(p.col.A, p.col2.A)}
}

// uv maps a screen point to a texture coordinate.
func (p meshPaint) uv(x, y float32) (float32, float32) {
	if !p.textured {
		r := rl.GetShapesTextureRectangle()
		t := rl.GetShapesTexture()
		return (r.X + r.Width/2) / float32(t.Width), (r.Y + r.Height/2) / float32(t.Height)
	}
	fx := clamp01((x - p.dst.X) / p.dst.Width)
	fy := clamp01((y - p.dst.Y) / p.dst.Height)
	sy := p.src.Y
	if p.src.Height < 0 {
		sy -= p.src.Height
	}
	return (p.src.X + fx*p.src.Width) / float32(p.tex.Width),
		(sy + fy*p.src.Height) / float32(p.tex.Height)
}

func (p meshPaint) begin() {
	if p.textured {
		rl.SetTexture(p.tex.ID)
	} else {
		rl.SetTexture(rl.GetShapesTexture().ID)
	}
	rl.Begin(rl.Triangles)
}

func (p meshPaint) end() {
	rl.End()
	rl.SetTexture(0)
}

// vertex emits one vertex with the paint's colour at the given share of its alpha.
func (p meshPaint) vertex(v rl.Vector2, alpha float32) {
	u, w := p.uv(v.X, v.Y)
	col := p.colourAt(v.Y)
	rl.Color4ub(col.R, col.G, col.B, uint8(float32(col.A)*alpha+0.5))
	rl.TexCoord2f(u, w)
	rl.Vertex2f(v.X, v.Y)
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// cornerSegments is how many straight pieces approximate a quarter circle
// of the given radius: about one every two pixels of arc.
func cornerSegments(r float32) int {
	n := int(math.Ceil(float64(r) * math.Pi / 4))
	if n < 4 {
		n = 4
	}
	if n > 32 {
		n = 32
	}
	return n
}

// roundRectOutline returns the outline of a rounded rectangle moved outwards
// by d pixels (inwards when d is negative). It always holds 4*(seg+1) points,
// in the same order whatever d is, so two outlines of one rectangle can be
// joined point for point. The order is top-left corner, top-right,
// bottom-right, bottom-left: clockwise as seen on screen.
func roundRectOutline(rect rl.Rectangle, r, d float32, seg int) []rl.Vector2 {
	half := minF32(rect.Width, rect.Height) / 2
	if r > half {
		r = half
	}
	if r < 0 {
		r = 0
	}
	// Corner centres stay where the nominal corners put them unless the
	// outline has moved further inwards than the radius, in which case the
	// corner has become square and its centre follows the outline.
	ci := maxF32(r, -d)
	if ci > half {
		ci = half
	}
	rad := ci + d
	if rad < 0 {
		rad = 0
	}
	centres := [4]rl.Vector2{
		{X: rect.X + ci, Y: rect.Y + ci},
		{X: rect.X + rect.Width - ci, Y: rect.Y + ci},
		{X: rect.X + rect.Width - ci, Y: rect.Y + rect.Height - ci},
		{X: rect.X + ci, Y: rect.Y + rect.Height - ci},
	}
	starts := [4]float64{math.Pi, 1.5 * math.Pi, 0, 0.5 * math.Pi}
	pts := make([]rl.Vector2, 0, 4*(seg+1))
	for k := 0; k < 4; k++ {
		for i := 0; i <= seg; i++ {
			a := starts[k] + float64(i)/float64(seg)*math.Pi/2
			pts = append(pts, rl.Vector2{
				X: centres[k].X + rad*float32(math.Cos(a)),
				Y: centres[k].Y + rad*float32(math.Sin(a)),
			})
		}
	}
	return pts
}

// meshFan fills the inside of a closed outline that is visible from centre.
// The outline runs clockwise on screen; raylib culls anything else.
func meshFan(p meshPaint, centre rl.Vector2, outline []rl.Vector2, alpha float32) {
	n := len(outline)
	for i := 0; i < n; i++ {
		a, b := outline[i], outline[(i+1)%n]
		p.vertex(centre, alpha)
		p.vertex(b, alpha)
		p.vertex(a, alpha)
	}
}

// meshBand fills the strip between two outlines of the same length, inner
// inside outer, with the alpha running from inAlpha to outAlpha across it.
// With closed false the strip is not joined from its last point to its first.
func meshBand(p meshPaint, inner, outer []rl.Vector2, inAlpha, outAlpha float32, closed bool) {
	n := len(inner)
	last := n
	if !closed {
		last = n - 1
	}
	for i := 0; i < last; i++ {
		j := (i + 1) % n
		p.vertex(inner[i], inAlpha)
		p.vertex(inner[j], inAlpha)
		p.vertex(outer[i], outAlpha)

		p.vertex(inner[j], inAlpha)
		p.vertex(outer[j], outAlpha)
		p.vertex(outer[i], outAlpha)
	}
}

// fillRoundMesh fills a rounded rectangle given in screen space.
func (c *Context) fillRoundMesh(rect rl.Rectangle, r float32, p meshPaint) {
	if rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	half := minF32(rect.Width, rect.Height) / 2
	if r > half {
		r = half
	}
	seg := cornerSegments(r)
	centre := rl.Vector2{X: rect.X + rect.Width/2, Y: rect.Y + rect.Height/2}
	p.begin()
	if !c.antialias {
		meshFan(p, centre, roundRectOutline(rect, r, 0, seg), 1)
		p.end()
		return
	}
	inner := roundRectOutline(rect, r, -feather/2.0, seg)
	outer := roundRectOutline(rect, r, feather/2.0, seg)
	meshFan(p, centre, inner, 1)
	meshBand(p, inner, outer, 1, 0, true)
	p.end()
}

// strokeRoundMesh strokes a rounded rectangle given in screen space with a
// line lw pixels wide centred on its outline.
func (c *Context) strokeRoundMesh(rect rl.Rectangle, r, lw float32, col color.RGBA) {
	c.strokeRoundMeshPaint(rect, r, lw, meshPaint{col: col})
}

// strokeRoundMeshPaint is strokeRoundMesh with any paint.
func (c *Context) strokeRoundMeshPaint(rect rl.Rectangle, r, lw float32, p meshPaint) {
	if rect.Width <= 0 || rect.Height <= 0 || lw <= 0 {
		return
	}
	half := minF32(rect.Width, rect.Height) / 2
	if r > half {
		r = half
	}
	seg := cornerSegments(r + lw/2)
	p.begin()
	if !c.antialias {
		meshBand(p, roundRectOutline(rect, r, -lw/2, seg), roundRectOutline(rect, r, lw/2, seg), 1, 1, true)
		p.end()
		return
	}
	f := float32(feather) / 2
	if lw < feather {
		// A line thinner than the soft edge: keep the band a pixel wide and
		// let its alpha carry the thinness.
		a := lw / feather
		mid := roundRectOutline(rect, r, 0, seg)
		meshBand(p, roundRectOutline(rect, r, -feather, seg), mid, 0, a, true)
		meshBand(p, mid, roundRectOutline(rect, r, feather, seg), a, 0, true)
		p.end()
		return
	}
	a0 := roundRectOutline(rect, r, -lw/2-f, seg)
	a1 := roundRectOutline(rect, r, -lw/2+f, seg)
	b1 := roundRectOutline(rect, r, lw/2-f, seg)
	b0 := roundRectOutline(rect, r, lw/2+f, seg)
	meshBand(p, a0, a1, 0, 1, true)
	meshBand(p, a1, b1, 1, 1, true)
	meshBand(p, b1, b0, 1, 0, true)
	p.end()
}

// verticalPaint is a paint that runs from top to bottom down rect, with
// globalAlpha applied to both ends.
func (c *Context) verticalPaint(rect rl.Rectangle, top, bottom color.RGBA) meshPaint {
	return meshPaint{
		col:      applyAlpha(top, c.state.globalAlpha),
		gradient: true,
		col2:     applyAlpha(bottom, c.state.globalAlpha),
		y0:       rect.Y,
		y1:       rect.Y + rect.Height,
	}
}

// screenRect is a rectangle in user space as it lies on the screen.
func (c *Context) screenRect(x, y, w, h float32) rl.Rectangle {
	m := c.state.transform
	tx, ty := m.transformPoint(x, y)
	return rl.Rectangle{X: tx, Y: ty, Width: w * m.a, Height: h * m.d}
}

// FillRoundRectGradient fills a rounded rectangle with a colour that runs
// from top at its upper edge to bottom at its lower edge. The alpha of the
// two colours runs with them, so a panel can be see-through at the top and
// solid at the bottom. The fill style is not used. It is always a mesh,
// with a soft edge when antialiasing is on, and it casts no shadow.
func (c *Context) FillRoundRectGradient(x, y, w, h, r float32, top, bottom color.RGBA) {
	rect := c.screenRect(x, y, w, h)
	c.fillRoundMesh(rect, r*c.state.transform.a, c.verticalPaint(rect, top, bottom))
}

// StrokeRoundRectGradient strokes a rounded rectangle with the current line
// width, centred on the outline, in a colour that runs from top at the
// rectangle's upper edge to bottom at its lower edge, as
// FillRoundRectGradient does. The stroke style is not used.
func (c *Context) StrokeRoundRectGradient(x, y, w, h, r float32, top, bottom color.RGBA) {
	rect := c.screenRect(x, y, w, h)
	c.strokeRoundMeshPaint(rect, r*c.state.transform.a, c.state.lineWidth, c.verticalPaint(rect, top, bottom))
}

// SetAntialias turns the soft edge of rounded rectangles and rounded images
// on or off. It is off by default, which draws exactly what earlier
// versions drew and relies on the window's multisampling (SetupWindow asks
// for 4x). Turn it on in a window opened without multisampling, where the
// corners would otherwise be stepped. It belongs to the context, not to the
// state that Save and Restore keep.
func (c *Context) SetAntialias(on bool) {
	c.antialias = on
}

// imageFilter applies the context's smoothing setting to a texture about to
// be drawn.
func (c *Context) imageFilter(src rl.Texture2D) {
	filter := rl.FilterBilinear
	if !c.state.smoothing {
		filter = rl.FilterPoint
	}
	rl.SetTextureFilter(src, filter)
}

// imageTint is the colour an image is drawn through: white, at globalAlpha.
func (c *Context) imageTint() color.RGBA {
	return applyAlpha(color.RGBA{R: 255, G: 255, B: 255, A: 255}, c.state.globalAlpha)
}

// DrawImageRounded draws an entire texture scaled to fill dst, with corners
// of radius r. The corners are part of the geometry: nothing is drawn
// outside them, so the image may lie over anything, not only a solid
// colour. This is what a rounded clip cannot give (see Clip).
func (c *Context) DrawImageRounded(src rl.Texture2D, dst rl.Rectangle, r float32) {
	c.DrawImageRoundedCropped(src, rl.Rectangle{Width: float32(src.Width), Height: float32(src.Height)}, dst, r)
}

// DrawImageRoundedCropped draws a sub-rectangle of src scaled to fill dst,
// with corners of radius r. As with DrawImageCropped, a negative
// srcRect.Height flips the image, which a render texture needs.
func (c *Context) DrawImageRoundedCropped(src rl.Texture2D, srcRect, dst rl.Rectangle, r float32) {
	m := c.state.transform
	tx, ty := m.transformPoint(dst.X, dst.Y)
	screen := rl.Rectangle{X: tx, Y: ty, Width: dst.Width * m.a, Height: dst.Height * m.d}
	c.imageFilter(src)
	c.fillRoundMesh(screen, r*m.a, meshPaint{
		col:      c.imageTint(),
		textured: true,
		tex:      src,
		src:      srcRect,
		dst:      screen,
	})
}

// DrawImageOffscreenRounded blits an off-screen Context onto this one with
// corners of radius r, the right way up.
func (c *Context) DrawImageOffscreenRounded(src *Context, dst rl.Rectangle, r float32) {
	if src.rt == nil {
		panic("raycanvas: DrawImageOffscreenRounded: source is not an off-screen context")
	}
	tex := src.rt.Texture
	c.DrawImageRoundedCropped(tex, rl.Rectangle{
		Width:  float32(tex.Width),
		Height: -float32(tex.Height),
	}, dst, r)
}
