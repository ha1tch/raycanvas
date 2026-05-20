package raycanvas

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Context is a stateful 2D drawing context backed by raylib.
// It mirrors the HTML5 CanvasRenderingContext2D API as closely as practical.
//
// Two kinds of Context exist:
//   - Screen context: draws directly to the framebuffer. Create with NewContext.
//   - Off-screen context: draws into a RenderTexture2D. Create with NewOffscreen.
//
// A single SharedCache should be created per application and passed to all
// Context constructors. It holds all GPU-resident cached resources (fonts,
// icons, shadow textures).
type Context struct {
	rt     *rl.RenderTexture2D // nil = screen context
	width  int32
	height int32

	// Active drawing state.
	state canvasState

	// Save/restore stack.
	stack []canvasState

	// Path accumulator.
	path pathBuffer

	// Shared GPU resource cache.
	cache *SharedCache

	// maskBgColor is the background colour used for roundRect corner overdraw.
	// Set this before a roundRect Clip() call. Defaults to the window background.
	maskBgColor color.RGBA
	maskedRR    roundRectParams
}

// canvasState is the complete drawing state saved and restored by Save/Restore.
type canvasState struct {
	fillStyle    color.RGBA
	strokeStyle  color.RGBA
	lineWidth    float32
	globalAlpha  float32
	font         resolvedFont
	fontString   string // original CSS string, for re-resolution after family fallback
	textAlign    textAlign
	textBaseline textBaseline
	lineDash     []float32
	lineDashOff  float32
	shadowColor  color.RGBA
	shadowBlur   float32
	shadowOffX   float32
	shadowOffY   float32
	lineCap      lineCap
	lineJoin     lineJoin
	direction    textDirection
	smoothing    bool
	transform    matrix3x2
	clip         clipState
}

type textAlign int

const (
	alignLeft textAlign = iota
	alignCenter
	alignRight
)

type textBaseline int

const (
	baselineAlphabetic textBaseline = iota
	baselineTop
	baselineMiddle
	baselineBottom
	baselineHanging
	baselineIdeographic
)

type lineCap int

const (
	lineCapButt lineCap = iota
	lineCapRound
	lineCapSquare
)

type lineJoin int

const (
	lineJoinMiter lineJoin = iota
	lineJoinRound
	lineJoinBevel
)

type textDirection int

const (
	directionLTR textDirection = iota
	directionRTL
)

// defaultState returns the initial canvas state matching browser defaults.
func defaultState(width, height int32) canvasState {
	return canvasState{
		fillStyle:    color.RGBA{R: 0, G: 0, B: 0, A: 255},
		strokeStyle:  color.RGBA{R: 0, G: 0, B: 0, A: 255},
		lineWidth:    1,
		globalAlpha:  1.0,
		textAlign:    alignLeft,
		textBaseline: baselineAlphabetic,
		lineCap:      lineCapButt,
		lineJoin:     lineJoinMiter,
		direction:    directionLTR,
		smoothing:    true,
		transform:    identityMatrix(),
		clip:         noClip(width, height),
	}
}

// NewContext creates a screen-space Context that draws to the raylib framebuffer.
// Call BeginFrame before drawing and EndFrame after.
// width and height should match the window dimensions.
func NewContext(width, height int32, cache *SharedCache) *Context {
	return &Context{
		width:  width,
		height: height,
		state:  defaultState(width, height),
		cache:  cache,
	}
}

// NewOffscreen creates an off-screen Context backed by a RenderTexture2D.
// The texture is allocated immediately; call Unload when done.
//
// Important: RenderTexture2D has Y-axis flipped relative to the screen.
// When blitting via DrawImageCropped, pass srcRect with negative Height to flip.
// This is handled internally by DrawImage* methods.
func NewOffscreen(width, height int32, cache *SharedCache) *Context {
	rt := rl.LoadRenderTexture(width, height)
	return &Context{
		rt:     &rt,
		width:  width,
		height: height,
		state:  defaultState(width, height),
		cache:  cache,
	}
}

// Texture returns the underlying GPU texture for this off-screen context.
// Use this as the source argument to DrawImage* calls.
// Panics if called on a screen context.
func (c *Context) Texture() rl.Texture2D {
	if c.rt == nil {
		panic("raycanvas: Texture() called on a screen context")
	}
	return c.rt.Texture
}

// Unload releases the GPU resources held by an off-screen context.
// Must be called before the raylib window is closed.
// No-op on a screen context.
func (c *Context) Unload() {
	if c.rt != nil {
		rl.UnloadRenderTexture(*c.rt)
		c.rt = nil
	}
}

// Width returns the width of the drawing surface in pixels.
func (c *Context) Width() int32 { return c.width }

// Height returns the height of the drawing surface in pixels.
func (c *Context) Height() int32 { return c.height }

// BeginFrame sets up the drawing surface for a new frame.
// For screen contexts: wraps rl.BeginDrawing.
// For off-screen contexts: wraps rl.BeginTextureMode.
// Call exactly once per frame before any draw calls.
//
// EnableSmoothLines is called every frame — it enables GL_LINE_SMOOTH,
// OpenGL's native line antialiasing, which applies to all line primitives
// (DrawLineEx, DrawSplineLinear, DrawCircleLinesV, etc.) at no per-draw cost.
func (c *Context) BeginFrame() {
	if c.rt != nil {
		rl.BeginTextureMode(*c.rt)
	} else {
		rl.BeginDrawing()
	}
	rl.EnableSmoothLines()
}

// EndFrame finalises the current frame.
// For screen contexts: wraps rl.EndDrawing.
// For off-screen contexts: wraps rl.EndTextureMode.
func (c *Context) EndFrame() {
	if c.rt != nil {
		rl.EndTextureMode()
	} else {
		rl.EndDrawing()
	}
}

// --- Save / Restore ----------------------------------------------------------

// Save pushes the current drawing state onto the stack.
// Matches ctx.save() in JavaScript.
func (c *Context) Save() {
	// Deep-copy lineDash slice so restore doesn't alias.
	saved := c.state
	if len(c.state.lineDash) > 0 {
		saved.lineDash = make([]float32, len(c.state.lineDash))
		copy(saved.lineDash, c.state.lineDash)
	}
	c.stack = append(c.stack, saved)
}

// Restore pops the drawing state from the stack.
// Matches ctx.restore() in JavaScript.
// Re-applies the clip from the restored state.
// If a roundRect masked region was open, composites it before restoring.
func (c *Context) Restore() {
	if len(c.stack) == 0 {
		return
	}
	// If a roundRect mask was active at this save level, apply corner overdraw.
	if c.state.clip.hasMask {
		c.applyRoundRectCornerOverdrawBg(c.autoMaskBackground())
	}
	c.state = c.stack[len(c.stack)-1]
	c.stack = c.stack[:len(c.stack)-1]
	// Re-apply scissor for the restored clip state.
	c.state.clip.applyScissor()
}

// --- Style setters -----------------------------------------------------------

// SetFillStyle sets the fill colour from a CSS colour string.
// Cached after first parse. Matches ctx.fillStyle = "..." in JavaScript.
func (c *Context) SetFillStyle(css string) {
	c.state.fillStyle = c.resolveColor(css)
}

// SetStrokeStyle sets the stroke colour from a CSS colour string.
// Cached after first parse. Matches ctx.strokeStyle = "..." in JavaScript.
func (c *Context) SetStrokeStyle(css string) {
	c.state.strokeStyle = c.resolveColor(css)
}

// SetLineWidth sets the stroke line width in pixels.
func (c *Context) SetLineWidth(w float32) {
	c.state.lineWidth = w
}

// SetGlobalAlpha sets the global alpha multiplier (0.0–1.0).
// Applied on top of any alpha embedded in fill/stroke colours.
func (c *Context) SetGlobalAlpha(a float32) {
	c.state.globalAlpha = a
}

// SetFont sets the current font from a CSS font string.
// Example: "italic 700 13px Inter", "500 9px 'Fira Code'".
// Cached after first parse. Matches ctx.font = "..." in JavaScript.
func (c *Context) SetFont(css string) {
	c.state.fontString = css
	key := parseFontString(css)
	c.state.font = c.resolveFont(key)
}

// SetTextAlign sets horizontal text alignment.
// Accepts "left", "center", "right". Matches ctx.textAlign = "..." in JavaScript.
func (c *Context) SetTextAlign(s string) {
	switch s {
	case "center":
		c.state.textAlign = alignCenter
	case "right":
		c.state.textAlign = alignRight
	default:
		c.state.textAlign = alignLeft
	}
}

// SetTextBaseline sets the vertical text baseline.
// Accepts "alphabetic", "top", "middle", "bottom", "hanging", "ideographic".
func (c *Context) SetTextBaseline(s string) {
	switch s {
	case "top":
		c.state.textBaseline = baselineTop
	case "middle":
		c.state.textBaseline = baselineMiddle
	case "bottom":
		c.state.textBaseline = baselineBottom
	case "hanging":
		c.state.textBaseline = baselineHanging
	case "ideographic":
		c.state.textBaseline = baselineIdeographic
	default:
		c.state.textBaseline = baselineAlphabetic
	}
}

// SetLineDash sets the dash pattern. An empty slice clears the dash.
// Matches ctx.setLineDash([...]) in JavaScript.
func (c *Context) SetLineDash(segments []float32) {
	c.state.lineDash = segments
}

// SetLineDashOffset sets the phase offset for the dash pattern.
func (c *Context) SetLineDashOffset(offset float32) {
	c.state.lineDashOff = offset
}

// SetShadowColor sets the shadow colour from a CSS colour string.
func (c *Context) SetShadowColor(css string) {
	c.state.shadowColor = c.resolveColor(css)
}

// SetShadowBlur sets the shadow blur radius in pixels.
func (c *Context) SetShadowBlur(b float32) {
	c.state.shadowBlur = b
}

// SetShadowOffsetX sets the horizontal shadow offset.
func (c *Context) SetShadowOffsetX(x float32) {
	c.state.shadowOffX = x
}

// SetShadowOffsetY sets the vertical shadow offset.
func (c *Context) SetShadowOffsetY(y float32) {
	c.state.shadowOffY = y
}

// SetLineCap sets the line cap style: "butt", "round", "square".
func (c *Context) SetLineCap(s string) {
	switch s {
	case "round":
		c.state.lineCap = lineCapRound
	case "square":
		c.state.lineCap = lineCapSquare
	default:
		c.state.lineCap = lineCapButt
	}
}

// SetLineJoin sets the line join style: "miter", "round", "bevel".
func (c *Context) SetLineJoin(s string) {
	switch s {
	case "round":
		c.state.lineJoin = lineJoinRound
	case "bevel":
		c.state.lineJoin = lineJoinBevel
	default:
		c.state.lineJoin = lineJoinMiter
	}
}

// SetDirection sets the text direction: "ltr" or "rtl".
// Used for RTL markdown rendering in joxel.
func (c *Context) SetDirection(s string) {
	if s == "rtl" {
		c.state.direction = directionRTL
	} else {
		c.state.direction = directionLTR
	}
}

// SetImageSmoothingEnabled controls bilinear filtering on DrawImage calls.
// When false, nearest-neighbour sampling is used.
func (c *Context) SetImageSmoothingEnabled(b bool) {
	c.state.smoothing = b
}

// --- Effective colour helpers ------------------------------------------------

// autoMaskBackground returns the colour to use for roundRect corner overdraw.
// Prefers an explicitly set maskBgColor; falls back to the current fill style,
// which is typically the card background at the time Clip() is called — making
// SetMaskBackground optional for the common case where the fill style is already
// set to the background colour before building the clip path.
func (c *Context) autoMaskBackground() color.RGBA {
	if c.maskBgColor.A > 0 {
		return c.maskBgColor
	}
	// Fall back to current fill style — usually the card bg colour.
	return c.state.fillStyle
}

// SetMaskBackground sets the background colour used for roundRect clip corner
// overdraw. Call this before Save()/RoundRect()/Clip() when the region behind
// the clipped content is a known solid colour. Defaults to transparent black.
func (c *Context) SetMaskBackground(css string) {
	c.maskBgColor = c.resolveColor(css)
}

// fillColor returns the effective fill colour with globalAlpha applied.
func (c *Context) fillColor() color.RGBA {
	return applyAlpha(c.state.fillStyle, c.state.globalAlpha)
}

// strokeColor returns the effective stroke colour with globalAlpha applied.
func (c *Context) strokeColor() color.RGBA {
	return applyAlpha(c.state.strokeStyle, c.state.globalAlpha)
}
