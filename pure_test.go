package raycanvas

import (
	"image/color"
	"math"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestParseColor(t *testing.T) {
	cases := []struct {
		in   string
		want color.RGBA
	}{
		{"#fff", color.RGBA{255, 255, 255, 255}},
		{"#204060", color.RGBA{0x20, 0x40, 0x60, 255}},
		{"#20406080", color.RGBA{0x20, 0x40, 0x60, 0x80}},
		{"rgb(1, 2, 3)", color.RGBA{1, 2, 3, 255}},
		{"rgba(300, -5, 3, 0.5)", color.RGBA{255, 0, 3, 127}},
		{"RGBA(1,2,3,2)", color.RGBA{1, 2, 3, 255}},
		{"Black", color.RGBA{0, 0, 0, 255}},
		{"transparent", color.RGBA{}},
		{"no-such-colour", color.RGBA{255, 0, 255, 0}},
		{"#12", color.RGBA{A: 255}},
		{"rgb(1,2)", color.RGBA{A: 255}},
		{"", color.RGBA{255, 0, 255, 0}},
	}
	for _, c := range cases {
		if got := ParseColor(c.in); got != c.want {
			t.Errorf("ParseColor(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseFontString(t *testing.T) {
	cases := []struct {
		in   string
		want FontKey
	}{
		{"13px 'Fira Code', monospace", FontKey{"fira", 13, 400, false}},
		{"italic 700 13px Inter", FontKey{"inter", 13, 700, true}},
		{"bold 12px system-ui", FontKey{"inter", 12, 700, false}},
		{"500 9px 'Fira Code'", FontKey{"fira", 9, 500, false}},
		{"Georgia", FontKey{"inter", 12, 400, false}},
		{"", FontKey{"inter", 12, 400, false}},
		{"10.5px serif", FontKey{"georgia", 10.5, 400, false}},
	}
	for _, c := range cases {
		if got := parseFontString(c.in); got != c.want {
			t.Errorf("parseFontString(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestMatrix(t *testing.T) {
	m := identityMatrix().translate(10, 20).scale(2, 3)
	if x, y := m.transformPoint(1, 1); x != 12 || y != 23 {
		t.Errorf("translate then scale maps (1,1) to (%v,%v), want (12,23)", x, y)
	}
	m = identityMatrix().scale(2, 3).translate(10, 20)
	if x, y := m.transformPoint(1, 1); x != 22 || y != 63 {
		t.Errorf("scale then translate maps (1,1) to (%v,%v), want (22,63)", x, y)
	}
}

func TestScissorIntersection(t *testing.T) {
	cs := noClip(100, 100)
	cs = cs.intersectScissor(rl.Rectangle{X: 10, Y: 10, Width: 50, Height: 50})
	cs = cs.intersectScissor(rl.Rectangle{X: 40, Y: 0, Width: 50, Height: 30})
	if want := (rl.Rectangle{X: 40, Y: 10, Width: 20, Height: 20}); cs.scissor != want {
		t.Errorf("intersection = %v, want %v", cs.scissor, want)
	}
	cs = cs.intersectScissor(rl.Rectangle{X: 80, Y: 80, Width: 10, Height: 10})
	if cs.scissor.Width != 0 || cs.scissor.Height != 0 {
		t.Errorf("disjoint clips leave %v, want an empty rectangle", cs.scissor)
	}
}

func TestSaveRestoreKeepsDashesApart(t *testing.T) {
	// Restore re-applies the scissor, which is an OpenGL call.
	onGL(t, func() {
		c := NewContext(10, 10, NewSharedCache())
		c.SetLineDash([]float32{4, 2})
		c.Save()
		c.state.lineDash[0] = 99
		c.SetLineWidth(7)
		c.Restore()
		if c.state.lineDash[0] != 4 || c.state.lineWidth != 1 {
			t.Errorf("restored dash %v width %v, want [4 2] and 1", c.state.lineDash, c.state.lineWidth)
		}
		c.Restore() // one Restore too many is ignored
	})
}

func area(tris [][3]rl.Vector2) float64 {
	sum := 0.0
	for _, t := range tris {
		sum += math.Abs(float64((t[1].X-t[0].X)*(t[2].Y-t[0].Y)-(t[1].Y-t[0].Y)*(t[2].X-t[0].X))) / 2
	}
	return sum
}

func TestTriangulate(t *testing.T) {
	v := func(x, y float32) rl.Vector2 { return rl.Vector2{X: x, Y: y} }
	u := []rl.Vector2{v(10, 10), v(30, 10), v(30, 70), v(70, 70), v(70, 10), v(90, 10), v(90, 90), v(10, 90)}
	rev := make([]rl.Vector2, len(u))
	for i := range u {
		rev[len(u)-1-i] = u[i]
	}
	closed := append(append([]rl.Vector2{}, u...), u[0])
	for name, poly := range map[string][]rl.Vector2{"clockwise": u, "anticlockwise": rev, "closed": closed} {
		tris := triangulate(poly)
		if len(tris) != 6 {
			t.Errorf("%s U: %d triangles, want 6", name, len(tris))
		}
		if a := area(tris); math.Abs(a-4000) > 0.01 {
			t.Errorf("%s U: area %.2f, want 4000", name, a)
		}
		for _, tr := range tris {
			if c := (tr[1].X-tr[0].X)*(tr[2].Y-tr[0].Y) - (tr[1].Y-tr[0].Y)*(tr[2].X-tr[0].X); c <= 0 {
				t.Errorf("%s U: triangle %v is not clockwise on screen", name, tr)
			}
		}
	}
	for name, poly := range map[string][]rl.Vector2{
		"nothing":    nil,
		"two points": {v(0, 0), v(1, 1)},
		"a line":     {v(0, 0), v(5, 5), v(10, 10)},
		"one point":  {v(3, 3), v(3, 3), v(3, 3), v(3, 3)},
	} {
		if tris := triangulate(poly); len(tris) != 0 {
			t.Errorf("%s: %d triangles, want none", name, len(tris))
		}
	}
	// A figure of eight crosses itself: no promise about the area, but it
	// must come back, and with fewer triangles than points.
	if tris := triangulate([]rl.Vector2{v(0, 0), v(10, 10), v(10, 0), v(0, 10)}); len(tris) > 2 {
		t.Errorf("figure of eight: %d triangles", len(tris))
	}
}

func TestRoundRectOutline(t *testing.T) {
	rect := rl.Rectangle{X: 10, Y: 20, Width: 100, Height: 60}
	for _, d := range []float32{-40, -12, -0.5, 0, 0.5, 3} {
		for _, r := range []float32{0, 8, 30, 500} {
			pts := roundRectOutline(rect, r, d, 6)
			if len(pts) != 28 {
				t.Fatalf("r=%v d=%v: %d points, want 28", r, d, len(pts))
			}
			grow := d
			if grow < -30 {
				grow = -30 // cannot shrink past the middle
			}
			for _, p := range pts {
				if p.X < rect.X-grow-0.01 || p.X > rect.X+rect.Width+grow+0.01 ||
					p.Y < rect.Y-grow-0.01 || p.Y > rect.Y+rect.Height+grow+0.01 {
					t.Errorf("r=%v d=%v: point %v outside the rectangle moved by %v", r, d, p, grow)
					break
				}
			}
			// clockwise on screen: positive shoelace sum with y down
			sum := float32(0)
			for i := range pts {
				a, b := pts[i], pts[(i+1)%len(pts)]
				sum += a.X*b.Y - b.X*a.Y
			}
			if sum < 0 {
				t.Errorf("r=%v d=%v: outline runs anticlockwise", r, d)
			}
		}
	}
}

func TestArcToStartsWhereThePathWas(t *testing.T) {
	c := NewContext(100, 100, NewSharedCache())
	c.Translate(5, 5)
	c.Scale(2, 2)
	c.BeginPath()
	c.MoveTo(0, 20)
	c.BezierCurveTo(0, 10, 0, 10, 0, 10) // ends at (0,10)
	c.ArcTo(0, 0, 10, 0, 10)             // round the corner at the origin
	c.LineTo(20, 0)
	sub := c.tessellate()
	if len(sub) != 1 {
		t.Fatalf("got %d sub-paths, want 1", len(sub))
	}
	for i, p := range sub[0] {
		// everything lies in the quadrant x >= 5, y >= 5 on screen
		if p.X < 4.99 || p.Y < 4.99 || p.X > 45.01 || p.Y > 45.01 {
			t.Errorf("point %d = %v is outside the path's box", i, p)
		}
	}
	last := sub[0][len(sub[0])-1]
	if last.X != 45 || last.Y != 5 {
		t.Errorf("path ends at %v, want (45,5)", last)
	}
}

func TestArcToGeometry(t *testing.T) {
	// A right-angle corner at the origin, both ways round.
	for _, c := range []struct {
		px, py, x2, y2 float32
	}{{0, 10, 10, 0}, {10, 0, 0, 10}, {0, -10, -10, 0}} {
		pts := tessellateArcTo(c.px, c.py, 0, 0, c.x2, c.y2, 4)
		first, last := pts[0], pts[len(pts)-1]
		sx, sy := c.px/10*4, c.py/10*4
		ex, ey := c.x2/10*4, c.y2/10*4
		if math.Hypot(float64(first[0]-sx), float64(first[1]-sy)) > 1e-3 || math.Hypot(float64(last[0]-ex), float64(last[1]-ey)) > 1e-3 {
			t.Errorf("from (%v,%v) to (%v,%v): arc runs %v to %v, want (%v,%v) to (%v,%v)", c.px, c.py, c.x2, c.y2, first, last, sx, sy, ex, ey)
		}
		for _, p := range pts {
			if d := math.Hypot(float64(p[0]-(sx+ex)), float64(p[1]-(sy+ey))); math.Abs(d-4) > 1e-3 {
				t.Errorf("point %v is %.3f from the arc centre, want 4", p, d)
			}
		}
	}
	// A 60 degree turn (the corner of a hexagon): the tangent points lie
	// r*tan(30 degrees) from the corner.
	pts := tessellateArcTo(-10, 0, 0, 0, 5, 8.660254, 3)
	if d := math.Hypot(float64(pts[0][0]), float64(pts[0][1])); math.Abs(d-3*math.Tan(math.Pi/6)) > 1e-3 {
		t.Errorf("hexagon corner: tangent point %.3f from the corner, want %.3f", d, 3*math.Tan(math.Pi/6))
	}
	// Three points on a line: no corner, just the point.
	if pts := tessellateArcTo(0, 0, 5, 0, 10, 0, 3); len(pts) != 1 {
		t.Errorf("straight line gave %d points, want 1", len(pts))
	}
}
