package raycanvas

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// triangulate cuts a simple polygon (one outline that does not cross
// itself, convex or not) into triangles by clipping ears. The triangles
// come back clockwise as seen on screen, which is the order the mesh
// routines draw in. It returns nil when the outline has fewer than three
// distinct points or no area.
//
// It does not handle holes or outlines that cross themselves; for those
// the result covers the outline's points but not necessarily the right
// area. A canvas fills such paths by a winding rule, which this is not.
func triangulate(pts []rl.Vector2) [][3]rl.Vector2 {
	// Drop repeated points, including a closing point equal to the first.
	poly := make([]rl.Vector2, 0, len(pts))
	for _, p := range pts {
		if n := len(poly); n > 0 && poly[n-1] == p {
			continue
		}
		poly = append(poly, p)
	}
	for len(poly) > 1 && poly[0] == poly[len(poly)-1] {
		poly = poly[:len(poly)-1]
	}
	if len(poly) < 3 {
		return nil
	}
	area := float32(0)
	for i := range poly {
		a, b := poly[i], poly[(i+1)%len(poly)]
		area += a.X*b.Y - b.X*a.Y
	}
	if area == 0 {
		return nil
	}
	// With y pointing down a positive area is clockwise on screen. Work on
	// a clockwise outline throughout.
	if area < 0 {
		for i, j := 0, len(poly)-1; i < j; i, j = i+1, j-1 {
			poly[i], poly[j] = poly[j], poly[i]
		}
	}
	cross := func(a, b, c rl.Vector2) float32 {
		return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
	}
	inside := func(p, a, b, c rl.Vector2) bool {
		return cross(a, b, p) >= 0 && cross(b, c, p) >= 0 && cross(c, a, p) >= 0
	}
	out := make([][3]rl.Vector2, 0, len(poly)-2)
	for len(poly) > 3 {
		n := len(poly)
		clipped := false
		for i := 0; i < n; i++ {
			a, b, c := poly[(i+n-1)%n], poly[i], poly[(i+1)%n]
			if cross(a, b, c) <= 0 {
				continue // a reflex or flat corner is not an ear
			}
			ear := true
			for j := 0; j < n; j++ {
				p := poly[j]
				if p == a || p == b || p == c {
					continue
				}
				if inside(p, a, b, c) {
					ear = false
					break
				}
			}
			if !ear {
				continue
			}
			out = append(out, [3]rl.Vector2{a, b, c})
			poly = append(poly[:i], poly[i+1:]...)
			clipped = true
			break
		}
		if !clipped {
			// Only flat corners are left, or the outline crosses itself:
			// drop a point so the loop always ends.
			poly = poly[:n-1]
		}
	}
	if cross(poly[0], poly[1], poly[2]) > 0 {
		out = append(out, [3]rl.Vector2{poly[0], poly[1], poly[2]})
	}
	return out
}

// fillTriangles draws clockwise triangles in one colour.
func fillTriangles(tris [][3]rl.Vector2, p meshPaint) {
	p.begin()
	for _, t := range tris {
		p.vertex(t[0], 1)
		p.vertex(t[2], 1)
		p.vertex(t[1], 1)
	}
	p.end()
}
