package raycanvas

import "image/color"

// SetFillColor sets the fill colour from a colour value rather than a CSS
// string, for callers that already hold one (a theme, say). The alpha in
// the value is kept and globalAlpha still applies on top of it.
func (c *Context) SetFillColor(col color.RGBA) {
	c.state.fillStyle = col
}

// SetStrokeColor sets the stroke colour from a colour value.
func (c *Context) SetStrokeColor(col color.RGBA) {
	c.state.strokeStyle = col
}

// SetMaskBackgroundColor sets the colour that the corners of a rounded clip
// are painted with, from a colour value. See SetMaskBackground.
func (c *Context) SetMaskBackgroundColor(col color.RGBA) {
	c.maskBgColor = col
}
