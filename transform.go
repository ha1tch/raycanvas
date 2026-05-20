package raycanvas

import "math"

// matrix3x2 is a 2D affine transform stored in column-major order compatible
// with the CSS/canvas transform(a,b,c,d,e,f) convention.
//
//	| a c e |
//	| b d f |
//	| 0 0 1 |
type matrix3x2 struct {
	a, b, c, d float32
	e, f       float32
}

func identityMatrix() matrix3x2 {
	return matrix3x2{a: 1, d: 1}
}

// multiply returns m × n.
func (m matrix3x2) multiply(n matrix3x2) matrix3x2 {
	return matrix3x2{
		a: m.a*n.a + m.c*n.b,
		b: m.b*n.a + m.d*n.b,
		c: m.a*n.c + m.c*n.d,
		d: m.b*n.c + m.d*n.d,
		e: m.a*n.e + m.c*n.f + m.e,
		f: m.b*n.e + m.d*n.f + m.f,
	}
}

// transformPoint applies the matrix to a point.
func (m matrix3x2) transformPoint(x, y float32) (float32, float32) {
	return m.a*x + m.c*y + m.e,
		m.b*x + m.d*y + m.f
}

// translate returns a new matrix with a translation applied.
func (m matrix3x2) translate(tx, ty float32) matrix3x2 {
	return m.multiply(matrix3x2{a: 1, d: 1, e: tx, f: ty})
}

// scale returns a new matrix with a scale applied.
func (m matrix3x2) scale(sx, sy float32) matrix3x2 {
	return m.multiply(matrix3x2{a: sx, d: sy})
}

// rotate returns a new matrix with a rotation applied (radians).
func (m matrix3x2) rotate(angle float32) matrix3x2 {
	cos := float32(math.Cos(float64(angle)))
	sin := float32(math.Sin(float64(angle)))
	return m.multiply(matrix3x2{a: cos, b: sin, c: -sin, d: cos})
}

// --- Context transform methods -----------------------------------------------

// Translate applies a translation to the current transform.
func (c *Context) Translate(x, y float32) {
	c.state.transform = c.state.transform.translate(x, y)
}

// Scale applies a scale to the current transform.
func (c *Context) Scale(x, y float32) {
	c.state.transform = c.state.transform.scale(x, y)
}

// ResetTransform resets the current transform to the identity matrix.
func (c *Context) ResetTransform() {
	c.state.transform = identityMatrix()
}

// SetTransform replaces the current transform with the given matrix,
// using the same (a,b,c,d,e,f) parameter order as the HTML5 canvas API.
func (c *Context) SetTransform(a, b, cc, d, e, f float32) {
	c.state.transform = matrix3x2{a: a, b: b, c: cc, d: d, e: e, f: f}
}
