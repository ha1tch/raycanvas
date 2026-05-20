// Example: zui
//
// A faithful port of Quag's visual model: pan/zoom infinite canvas, card
// chrome (title bar, drag dots, close/minimise/mode buttons), card drag,
// drop shadow, and the four Quag themes (Kaputccino, Light, Polykai, Dark).
//
// Controls:
//   Pan        — middle-mouse drag, or space+drag
//   Zoom       — scroll wheel (anchored to cursor)
//   Drag card  — drag title bar
//   Focus card — click anywhere on card
//   Cycle theme — T key
//   New card    — N key (places at cursor)
//   Close card  — click × button (appears on hover)
//
// Faithfulness notes:
//   - Theme colour tables match quag/src/constants.js exactly
//   - Grid: fine grid (40wu) fades in above zoom 0.22; major (200wu) always visible
//   - Card: DW=330 DH=230 TH=28 PAD=13 CL_X=13 CL_R=4, r=9
//   - Buttons: close (right slot 0), minimise (right slot 1),
//              mode/wrap (left slots 0,1) — colour animated by btnColorAlpha
//   - Shadow: shadowBlur 12 unfocused, 28 focused
//   - w2s / s2w: wx*zoom+pan.x, (sx-pan.x)/zoom
package main

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
	rc "github.com/ha1tch/raycanvas"
	"github.com/ha1tch/raycanvas/fonts"
)

const (
	W = 1200
	H = 800

	// Card constants — from quag/src/constants.js
	DW      = 330  // default card width (world units)
	DH      = 230  // default card height (world units)
	TH      = 28   // title bar height
	PAD     = 13   // content padding
	SBW     = 5    // scrollbar width
	CL_X    = 13   // button centre x from edge
	CL_R    = 4    // button circle radius
	LH      = 20   // line height
	FSZ     = 13   // font size
	MINI_SZ = 60   // minimised card size

	gridBase  = 40  // fine grid world units
	gridMajor = 200 // major grid world units
)

// ── Theme ─────────────────────────────────────────────────────────────────

type theme struct {
	name string
	// Canvas
	bg, gridFine, gridMajor, cardShadow string
	// Card chrome
	card, border, borderFoc, title, titleSep string
	dragDot, closeCol, minBtn, modeOff, wrapBtn, btnMono string
	// Text
	txt, dim, dimMid string
	// Heading colours (for content preview)
	h1, h2, h3 string
	code, codeBg string
}

var themes = []theme{
	{
		name:      "kaputccino",
		bg:        "#0f0f17", gridFine: "#1e1e30", gridMajor: "#222236",
		cardShadow: "rgba(0,0,0,0.55)",
		card:      "#1e1e2e", border: "#313244", borderFoc: "#cba6f7",
		title: "#181825", titleSep: "#252538",
		dragDot:  "#45475a", closeCol: "#f38ba8", minBtn: "#f9e2af",
		modeOff:  "#89b4fa", wrapBtn: "#a6e3a1", btnMono: "#3d3d52",
		txt:      "#cdd6f4", dim: "#585b70", dimMid: "#6c7086",
		h1: "#f38ba8", h2: "#fab387", h3: "#f9e2af",
		code: "#f38ba8", codeBg: "rgba(17,17,27,0.85)",
	},
	{
		name:      "light",
		bg:        "#e8e8e2", gridFine: "#d0d0c8", gridMajor: "#b8b8b0",
		cardShadow: "rgba(0,0,0,0.12)",
		card:      "#ffffff", border: "#c8c8c0", borderFoc: "#7232c8",
		title: "#f0f0eb", titleSep: "#d8d8d0",
		dragDot:  "#b0b0a8", closeCol: "#d42050", minBtn: "#c05000",
		modeOff:  "#0055d4", wrapBtn: "#0a8a2e", btnMono: "#d0d0c8",
		txt:      "#2d2d2d", dim: "#7a7a72", dimMid: "#9a9a92",
		h1: "#d42050", h2: "#c05000", h3: "#0055d4",
		code: "#d42050", codeBg: "rgba(0,0,0,0.05)",
	},
	{
		name:      "polykai",
		bg:        "#1c1e1a", gridFine: "#2f3128", gridMajor: "#3e4035",
		cardShadow: "rgba(0,0,0,0.6)",
		card:      "#272822", border: "#3e4035", borderFoc: "#f92672",
		title: "#1c1e1a", titleSep: "#3e4035",
		dragDot:  "#75715e", closeCol: "#f92672", minBtn: "#e6db74",
		modeOff:  "#66d9ef", wrapBtn: "#a6e22e", btnMono: "#3e4035",
		txt:      "#f8f8f2", dim: "#75715e", dimMid: "#90908a",
		h1: "#f92672", h2: "#fd971f", h3: "#e6db74",
		code: "#f92672", codeBg: "rgba(0,0,0,0.3)",
	},
	{
		name:      "dark",
		bg:        "#08080f", gridFine: "#12121c", gridMajor: "#1a1a28",
		cardShadow: "rgba(0,0,0,0.7)",
		card:      "#0e0e18", border: "#1a1a28", borderFoc: "#00d4ff",
		title: "#0a0a14", titleSep: "#1a1a28",
		dragDot:  "#303040", closeCol: "#ff2d8b", minBtn: "#00d4ff",
		modeOff:  "#00d4ff", wrapBtn: "#ff2d8b", btnMono: "#1a1a28",
		txt:      "#b0b0c0", dim: "#404050", dimMid: "#505060",
		h1: "#00d4ff", h2: "#ff2d8b", h3: "#00d4ff",
		code: "#ff2d8b", codeBg: "rgba(0,0,0,0.4)",
	},
}

// ── Card ──────────────────────────────────────────────────────────────────

type cardMode int

const (
	modeCode    cardMode = iota
	modePreview          // rich markdown view
	modeLive             // live shelfdown view (placeholder)
)

type card struct {
	id            int
	x, y          float32 // world position
	w, h          float32 // world size
	title         string
	lines         []string
	mode          cardMode
	scroll        float32
	focused       bool
	minimised     bool
	miniX, miniY  float32
	wrapOn        bool
	btnColorAlpha float32 // 0=mono, 1=colour (animates on hover)
	btnColorTarget float32
	// Drag state (screen space)
	dragging      bool
	dragOffX      float32
	dragOffY      float32
}

var (
	cards   []*card
	nextID  int
	focCard *card
	panX    float32
	panY    float32
	zoomVal float32 = 1.0
	zoomTgt float32 = 1.0

	// Interaction state
	panning      bool
	panStartX    float32
	panStartY    float32
	panStartPanX float32
	panStartPanY float32

	themeIdx int
	T        *theme

	spaceDown bool
)

// World → screen
func w2s(wx, wy float32) (float32, float32) {
	return wx*zoomVal + panX, wy*zoomVal + panY
}

// Screen → world
func s2w(sx, sy float32) (float32, float32) {
	return (sx - panX) / zoomVal, (sy - panY) / zoomVal
}

func newCard(wx, wy float32, title string, lines []string) *card {
	nextID++
	return &card{
		id:    nextID,
		x:     wx,
		y:     wy,
		w:     DW,
		h:     DH,
		title: title,
		lines: lines,
		mode:  modeCode,
	}
}

// ── Font helpers ─────────────────────────────────────────────────────────────

// zoomedFont returns a CSS font string with size scaled by the current zoom,
// clamped to the baked atlas sizes (8–14px). This gives sharp text at each
// zoom level by selecting the nearest pre-baked atlas rather than scaling
// from a single atlas (which would blur at non-native sizes).
func zoomedFont(weight, family string, worldSz float32) string {
	sz := worldSz * zoomVal
	// Snap to nearest baked size
	baked := []float32{8, 9, 10, 11, 12, 13, 14}
	best := baked[0]
	for _, b := range baked {
		if abs32(b-sz) < abs32(best-sz) {
			best = b
		}
	}
	if weight != "" {
		return fmt.Sprintf("%s %.0fpx %s", weight, best, family)
	}
	return fmt.Sprintf("%.0fpx %s", best, family)
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// ── Rendering ─────────────────────────────────────────────────────────────────

func drawGrid(ctx *rc.Context) {
	ctx.SetFillStyle(T.bg)
	ctx.FillRect(0, 0, W, H)

	// Fine grid — fades in above zoom 0.22
	fineAlpha := float32(math.Max(0, math.Min(1, float64((zoomVal-0.22)/0.28))))
	if fineAlpha > 0.005 {
		gs := float32(gridBase) * zoomVal
		fox := float32(math.Mod(float64(panX), float64(gs)))
		if fox < 0 {
			fox += gs
		}
		foy := float32(math.Mod(float64(panY), float64(gs)))
		if foy < 0 {
			foy += gs
		}
		ctx.Save()
		ctx.SetGlobalAlpha(fineAlpha * 0.55)
		ctx.SetStrokeStyle(T.gridFine)
		ctx.SetLineWidth(0.5)
		ctx.BeginPath()
		for x := fox; x < W+1; x += gs {
			ctx.MoveTo(x, 0)
			ctx.LineTo(x, H)
		}
		for y := foy; y < H+1; y += gs {
			ctx.MoveTo(0, y)
			ctx.LineTo(W, y)
		}
		ctx.Stroke()
		ctx.Restore()
	}

	// Major grid — always visible
	{
		mgs := float32(gridMajor) * zoomVal
		mox := float32(math.Mod(float64(panX), float64(mgs)))
		if mox < 0 {
			mox += mgs
		}
		moy := float32(math.Mod(float64(panY), float64(mgs)))
		if moy < 0 {
			moy += mgs
		}
		ctx.SetStrokeStyle(T.gridMajor)
		ctx.SetLineWidth(1)
		ctx.BeginPath()
		for x := mox; x < W+1; x += mgs {
			ctx.MoveTo(x, 0)
			ctx.LineTo(x, H)
		}
		for y := moy; y < H+1; y += mgs {
			ctx.MoveTo(0, y)
			ctx.LineTo(W, y)
		}
		ctx.Stroke()
	}
}

func drawCard(ctx *rc.Context, c *card) {
	var sx, sy float32
	if c.minimised {
		sx, sy = w2s(c.miniX, c.miniY)
	} else {
		sx, sy = w2s(c.x, c.y)
	}

	cw, ch := c.w, c.h
	if c.minimised {
		cw = MINI_SZ
		ch = MINI_SZ
	}
	sw := cw * zoomVal
	sh := ch * zoomVal

	// Viewport cull
	if sx+sw < 0 || sx > W || sy+sh < 0 || sy > H {
		return
	}

	ctx.Save()
	ctx.Translate(sx, sy)
	ctx.Scale(zoomVal, zoomVal)

	// ── Shadow ───────────────────────────────────────────────────────────
	ctx.SetShadowColor(T.cardShadow)
	if c.focused {
		ctx.SetShadowBlur(28)
		ctx.SetShadowOffsetY(6)
	} else {
		ctx.SetShadowBlur(12)
		ctx.SetShadowOffsetY(2)
	}
	ctx.SetFillStyle(T.card)
	ctx.FillRoundRect(0, 0, cw, ch, 9)
	ctx.SetShadowColor("transparent")
	ctx.SetShadowBlur(0)
	ctx.SetShadowOffsetY(0)

	// ── Border / focus glow ──────────────────────────────────────────────
	ctx.SetLineWidth(1 / zoomVal)
	ctx.BeginPath()
	ctx.RoundRect(0, 0, cw, ch, 9)
	if c.focused {
		ctx.SetStrokeStyle(T.borderFoc)
		ctx.SetLineWidth(1.5 / zoomVal)
	} else {
		ctx.SetStrokeStyle(T.border)
	}
	ctx.Stroke()

	if c.minimised {
		// Mini label
		ctx.SetFont(zoomedFont("700", "inter", 9))
		ctx.SetTextAlign("center")
		ctx.SetTextBaseline("middle")
		ctx.SetFillStyle(T.txt)
		if c.title != "" {
			ctx.FillText(c.title, cw/2, ch/2)
		} else {
			// Three dots
			ctx.SetFillStyle(T.dim)
			for d := 0; d < 3; d++ {
				ctx.FillCircle(cw/2-6+float32(d)*6, ch/2, 2)
			}
		}
		ctx.Restore()
		return
	}

	// ── Title bar — drawn as rounded-top rect using two stacked primitives ──
	// This avoids the custom ArcTo path which goes through DrawTriangleFan
	// and produces distortion at high zoom. Two raylib calls instead:
	//   1. FillRoundRect for the full-height shape (gives rounded top corners)
	//   2. FillRect to square off everything below the title bar height
	ctx.SetFillStyle(T.title)
	ctx.FillRoundRectTop(0, 0, cw, TH, 9)

	// Title sep line
	ctx.SetStrokeStyle(T.titleSep)
	ctx.SetLineWidth(1 / zoomVal)
	ctx.BeginPath()
	ctx.MoveTo(0, TH)
	ctx.LineTo(cw, TH)
	ctx.Stroke()

	// Drag dots — three circles at centre of title bar
	ctx.SetFillStyle(T.dragDot)
	for d := 0; d < 3; d++ {
		ctx.FillCircle(cw/2-8+float32(d)*8, TH/2, 1.7)
	}

	// ── Title bar buttons ─────────────────────────────────────────────────
	bca := c.btnColorAlpha
	drawCardButtons(ctx, c, cw, bca)

	// ── Content area ──────────────────────────────────────────────────────
	ctx.Save()
	ctx.SetMaskBackground(T.card)
	ctx.BeginPath()
	ctx.Rect(0, TH+0.5, cw, ch-TH-0.5)
	ctx.Clip()

	switch c.mode {
	case modeCode:
		drawCodeContent(ctx, c, cw, ch)
	case modePreview:
		drawPreviewContent(ctx, c, cw, ch)
	case modeLive:
		drawLivePlaceholder(ctx, c, cw, ch)
	}

	ctx.Restore()

	ctx.Restore()
}

func drawCardButtons(ctx *rc.Context, c *card, cw, bca float32) {
	lerpColor := func(mono, col string, t float32) string {
		// Simple: just switch between mono and colour based on threshold
		if t < 0.5 {
			return mono
		}
		return col
	}

	// Left buttons
	leftSlots := []struct {
		col    string
		active bool
	}{
		{lerpColor(T.btnMono, T.modeOff, bca), false},  // mode
		{lerpColor(T.btnMono, T.wrapBtn, bca), c.wrapOn}, // wrap
	}
	for i, btn := range leftSlots {
		x := float32(CL_X) + float32(i)*(CL_R*2+5)
		ctx.SetFillStyle(btn.col)
		ctx.FillCircle(x, TH/2, CL_R)
		if btn.active {
			// Inner dot for active state
			ctx.SetFillStyle(T.card)
			ctx.FillCircle(x, TH/2, 1.5)
		}
	}

	// Right buttons: minimise (slot 1), close (slot 0)
	rightSlots := []struct {
		col string
		vis float32
	}{
		{lerpColor(T.btnMono, T.closeCol, bca), bca},         // close
		{lerpColor(T.btnMono, T.minBtn, bca), 1},              // minimise
	}
	for i, btn := range rightSlots {
		if btn.vis < 0.005 {
			continue
		}
		x := cw - float32(CL_X) - float32(i)*(CL_R*2+5)
		r := CL_R * btn.vis
		ctx.SetFillStyle(btn.col)
		ctx.FillCircle(x, TH/2, r)
	}
}

func drawCodeContent(ctx *rc.Context, c *card, cw, ch float32) {
	// Monospaced font, plain text lines — matches quag code mode
	ctx.SetFont(zoomedFont("400", "fira", FSZ))
	ctx.SetTextAlign("left")
	ctx.SetTextBaseline("middle")

	lineCount := len(c.lines)
	totH := float32(lineCount*LH) + PAD*2
	maxScr := float32(math.Max(0, float64(totH-(ch-TH))))
	if c.scroll > maxScr {
		c.scroll = maxScr
	}

	for li, raw := range c.lines {
		ly := float32(TH+PAD) + float32(li)*LH - c.scroll
		my := ly + LH/2
		if ly+LH < TH || ly > ch {
			continue
		}
		// Line colour heuristic — matches quag's parseLine colour logic
		ctx.SetFillStyle(lineColor(raw))
		ctx.FillText(raw, PAD, my)
	}

	// Scrollbar
	cH := ch - TH
	if maxScr > 0 {
		tH := cH * cH / totH
		tY := TH + (c.scroll/maxScr)*(cH-tH)
		ctx.SetFillStyle("rgba(49,50,68,0.35)")
		ctx.FillRect(cw-SBW-3, TH, SBW, cH)
		ctx.SetFillStyle("#45475a")
		ctx.FillRect(cw-SBW-3, tY, SBW, tH)
	}
}

func drawPreviewContent(ctx *rc.Context, c *card, cw, ch float32) {
	// Rich preview — headings coloured per theme, body text in T.txt
	ctx.SetTextAlign("left")
	ctx.SetTextBaseline("middle")

	lineCount := len(c.lines)
	totH := float32(lineCount*LH) + PAD*2
	maxScr := float32(math.Max(0, float64(totH-(ch-TH))))
	if c.scroll > maxScr {
		c.scroll = maxScr
	}

	for li, raw := range c.lines {
		ly := float32(TH+PAD) + float32(li)*LH - c.scroll
		my := ly + LH/2
		if ly+LH < TH || ly > ch {
			continue
		}

		switch {
		case len(raw) > 2 && raw[:2] == "# ":
			ctx.SetFont(zoomedFont("700", "inter", FSZ+1))
			ctx.SetFillStyle(T.h1)
			ctx.FillText(raw[2:], PAD, my)
		case len(raw) > 3 && raw[:3] == "## ":
			ctx.SetFont(zoomedFont("700", "inter", FSZ))
			ctx.SetFillStyle(T.h2)
			ctx.FillText(raw[3:], PAD, my)
		case len(raw) > 4 && raw[:4] == "### ":
			ctx.SetFont(zoomedFont("600", "inter", FSZ))
			ctx.SetFillStyle(T.h3)
			ctx.FillText(raw[4:], PAD, my)
		case len(raw) > 0 && raw[0] == '`':
			ctx.SetFont(zoomedFont("", "fira", FSZ-1))
			ctx.SetFillStyle(T.code)
			ctx.FillText(raw, PAD, my)
		case raw == "---" || raw == "":
			if raw == "---" {
				ctx.SetStrokeStyle(T.dim)
				ctx.SetLineWidth(0.5)
				ctx.BeginPath()
				ctx.MoveTo(PAD, my)
				ctx.LineTo(cw-PAD-SBW-2, my)
				ctx.Stroke()
			}
		default:
			ctx.SetFont(zoomedFont("", "inter", FSZ))
			ctx.SetFillStyle(T.txt)
			ctx.FillText(raw, PAD, my)
		}
	}
}

func drawLivePlaceholder(ctx *rc.Context, c *card, cw, ch float32) {
	ctx.SetFont(zoomedFont("400", "inter", 11))
	ctx.SetTextAlign("center")
	ctx.SetTextBaseline("middle")
	ctx.SetFillStyle(T.dimMid)
	ctx.FillText("live view", cw/2, (TH+ch)/2)
}

// lineColor returns a colour for a code line — rough heuristic matching quag.
func lineColor(raw string) string {
	if len(raw) == 0 {
		return T.txt
	}
	switch raw[0] {
	case '#':
		return T.dim
	case '/', '-', '*':
		return T.dim
	}
	// Keywords
	for _, kw := range []string{"func ", "type ", "var ", "const ", "import ", "package ", "return ", "if ", "for ", "range "} {
		if len(raw) >= len(kw) && raw[:len(kw)] == kw {
			return T.h1
		}
	}
	return T.txt
}

// ── Input handling ────────────────────────────────────────────────────────

// hitCard returns the topmost card under screen position (sx, sy), or nil.
func hitCard(sx, sy float32) *card {
	// Iterate in reverse (topmost drawn last = highest z)
	for i := len(cards) - 1; i >= 0; i-- {
		c := cards[i]
		var cx, cy, cw, ch float32
		if c.minimised {
			cx, cy = w2s(c.miniX, c.miniY)
			cw = MINI_SZ * zoomVal
			ch = MINI_SZ * zoomVal
		} else {
			cx, cy = w2s(c.x, c.y)
			cw = c.w * zoomVal
			ch = c.h * zoomVal
		}
		if sx >= cx && sx <= cx+cw && sy >= cy && sy <= cy+ch {
			return c
		}
	}
	return nil
}

// hitTitleBar returns true if (sx, sy) is in the title bar of card c.
func hitTitleBar(c *card, sx, sy float32) bool {
	if c.minimised {
		return false
	}
	cx, cy := w2s(c.x, c.y)
	return sx >= cx && sx <= cx+c.w*zoomVal && sy >= cy && sy <= cy+float32(TH)*zoomVal
}

// hitCloseButton returns true if close button was hit.
func hitCloseBtn(c *card, sx, sy float32) bool {
	if c.minimised {
		return false
	}
	cx, cy := w2s(c.x, c.y)
	// Close button: right slot 0 = cw - CL_X in card space
	bx := cx + (c.w-CL_X)*zoomVal
	by := cy + (TH/2)*zoomVal
	r := float32(CL_R+4) * zoomVal
	dx := sx - bx
	dy := sy - by
	return dx*dx+dy*dy <= r*r
}

func hitMinBtn(c *card, sx, sy float32) bool {
	if c.minimised {
		return false
	}
	cx, cy := w2s(c.x, c.y)
	bx := cx + (c.w-CL_X-(CL_R*2+5))*zoomVal
	by := cy + (TH/2)*zoomVal
	r := float32(CL_R+4) * zoomVal
	dx := sx - bx
	dy := sy - by
	return dx*dx+dy*dy <= r*r
}

func hitModeBtn(c *card, sx, sy float32) bool {
	if c.minimised {
		return false
	}
	cx, cy := w2s(c.x, c.y)
	bx := cx + CL_X*zoomVal
	by := cy + (TH/2)*zoomVal
	r := float32(CL_R+4) * zoomVal
	dx := sx - bx
	dy := sy - by
	return dx*dx+dy*dy <= r*r
}

// bringToFront moves c to the end of the cards slice (drawn last = on top).
func bringToFront(c *card) {
	for i, cc := range cards {
		if cc == c {
			cards = append(cards[:i], cards[i+1:]...)
			cards = append(cards, c)
			return
		}
	}
}

func main() {
	T = &themes[0]

	cache := rc.NewSharedCache()
	ctx := rc.SetupWindow(W, H, "raycanvas — zui (quag)", cache)
	defer rl.CloseWindow()
	defer cache.Unload()
	rl.SetTargetFPS(60)
	if err := fonts.Register(cache); err != nil {
		panic(err)
	}

	// Centre the canvas
	panX = W / 2
	panY = H / 2

	// Seed cards
	cards = []*card{
		newCard(-500, -300, "Overview", []string{
			"# Project Overview",
			"",
			"## Goals",
			"Build a ZUI for Shelf AMS",
			"Port Shevo to Go via raycanvas",
			"",
			"## Status",
			"raycanvas 0.1.x — in progress",
			"joxel port — pending",
			"quag port — pending",
		}),
		newCard(-100, -280, "Notes", []string{
			"// Canvas API ported to Go",
			"// Backed by raylib-go",
			"",
			"func BeginPath() {",
			"  c.path.reset()",
			"}",
			"",
			"func BezierCurveTo(...) {",
			"  // cached gg texture",
			"}",
		}),
		newCard(-500, 80, "Tasks", []string{
			"## Active",
			"- [x] raycanvas scaffold",
			"- [x] font system",
			"- [x] path tessellation",
			"- [ ] joxel port",
			"- [ ] quag port",
			"",
			"## Backlog",
			"- [ ] shelfdown renderer",
			"- [ ] IoT ingest API",
		}),
		newCard(-100, 60, "API Surface", []string{
			"# Canvas API",
			"",
			"FillRect  StrokeRect",
			"BeginPath  MoveTo  LineTo",
			"Arc  ArcTo  RoundRect",
			"BezierCurveTo",
			"Fill  Stroke  Clip",
			"",
			"# Text",
			"FillText  MeasureText",
			"SetFont  SetTextAlign",
		}),
		newCard(330, -280, "Metrics", []string{
			"## Performance",
			"",
			"grid: 60fps ✓",
			"bezier: cached texture ✓",
			"fonts: atlas per size ✓",
			"clip: scissor + overdraw ✓",
			"",
			"## Next",
			"mipmap tiers",
			"cluster cache",
		}),
	}

	// Set mode on some cards
	cards[0].mode = modePreview
	cards[2].mode = modePreview
	cards[3].mode = modePreview
	cards[4].mode = modePreview

	for !rl.WindowShouldClose() {
		mp := rl.GetMousePosition()
		sx, sy := mp.X, mp.Y
		dt := rl.GetFrameTime()
		wheel := rl.GetMouseWheelMove()

		// Theme cycle
		if rl.IsKeyPressed(rl.KeyT) {
			themeIdx = (themeIdx + 1) % len(themes)
			T = &themes[themeIdx]
		}

		// Space key for pan mode
		spaceDown = rl.IsKeyDown(rl.KeySpace)

		// Zoom (wheel, anchored to cursor)
		if wheel != 0 {
			oldZoom := zoomVal
			factor := float32(math.Pow(1.12, float64(wheel)))
			zoomVal *= factor
			if zoomVal < 0.08 {
				zoomVal = 0.08
			}
			if zoomVal > 6 {
				zoomVal = 6
			}
			// Anchor zoom to cursor
			panX = sx - (sx-panX)*zoomVal/oldZoom
			panY = sy - (sy-panY)*zoomVal/oldZoom
		}

		// Pan — middle mouse or space+left mouse
		isPanGesture := rl.IsMouseButtonDown(rl.MouseButtonMiddle) ||
			(spaceDown && rl.IsMouseButtonDown(rl.MouseButtonLeft))
		if isPanGesture && !panning {
			panning = true
			panStartX, panStartY = sx, sy
			panStartPanX, panStartPanY = panX, panY
		}
		if !isPanGesture {
			panning = false
		}
		if panning {
			panX = panStartPanX + (sx - panStartX)
			panY = panStartPanY + (sy - panStartY)
		}

		// Button colour alpha — animate toward target
		for _, c := range cards {
			if c.minimised {
				continue
			}
			// Check proximity for button colour reveal
			cx, cy := w2s(c.x, c.y)
			cw := c.w * zoomVal
			inCard := sx >= cx && sx <= cx+cw && sy >= cy && sy <= cy+float32(TH)*zoomVal*2
			if inCard {
				c.btnColorTarget = 1
			} else {
				c.btnColorTarget = 0
			}
			c.btnColorAlpha += (c.btnColorTarget - c.btnColorAlpha) * 0.22
			if c.btnColorAlpha < 0.001 {
				c.btnColorAlpha = 0
			}
			if c.btnColorAlpha > 0.999 {
				c.btnColorAlpha = 1
			}
		}

		// Left mouse button
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && !panning {
			hit := hitCard(sx, sy)
			if hit != nil {
				// Check buttons first
				if hit.btnColorAlpha > 0.3 && hitCloseBtn(hit, sx, sy) {
					// Close: remove card
					for i, c := range cards {
						if c == hit {
							cards = append(cards[:i], cards[i+1:]...)
							break
						}
					}
					if focCard == hit {
						focCard = nil
					}
					hit = nil
				} else if hit != nil && hit.btnColorAlpha > 0.3 && hitMinBtn(hit, sx, sy) {
					if hit.minimised {
						hit.minimised = false
					} else {
						hit.minimised = true
						hit.miniX = hit.x + hit.w/2 - MINI_SZ/2
						hit.miniY = hit.y + hit.h/2 - MINI_SZ/2
					}
					hit = nil
				} else if hit != nil && hit.btnColorAlpha > 0.3 && hitModeBtn(hit, sx, sy) {
					switch hit.mode {
					case modeCode:
						hit.mode = modePreview
					case modePreview:
						hit.mode = modeLive
					case modeLive:
						hit.mode = modeCode
					}
					hit = nil
				}

				if hit != nil {
					// Focus and possibly start drag
					if focCard != nil {
						focCard.focused = false
					}
					focCard = hit
					hit.focused = true
					bringToFront(hit)

					if hitTitleBar(hit, sx, sy) || hit.minimised {
						hit.dragging = true
						cx, cy := w2s(hit.x, hit.y)
						if hit.minimised {
							cx, cy = w2s(hit.miniX, hit.miniY)
						}
						hit.dragOffX = sx - cx
						hit.dragOffY = sy - cy
					}
				}
			} else {
				// Click on empty canvas — deselect
				if focCard != nil {
					focCard.focused = false
					focCard = nil
				}
			}
		}

		// Drag
		for _, c := range cards {
			if c.dragging {
				wx, wy := s2w(sx-c.dragOffX, sy-c.dragOffY)
				if c.minimised {
					c.miniX = wx
					c.miniY = wy
				} else {
					c.x = wx
					c.y = wy
				}
			}
		}

		if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			for _, c := range cards {
				c.dragging = false
			}
		}

		// Card scroll (when hovering and not dragging)
		if !panning {
			hit := hitCard(sx, sy)
			if hit != nil && !hit.minimised {
				_, cy := w2s(hit.x, hit.y)
				inContent := sy > cy+float32(TH)*zoomVal
				_ = inContent
				// Scroll with shift+wheel
				if rl.IsKeyDown(rl.KeyLeftShift) && wheel != 0 {
					hit.scroll -= wheel * LH * 3
					if hit.scroll < 0 {
						hit.scroll = 0
					}
				}
			}
		}

		// New card at cursor
		if rl.IsKeyPressed(rl.KeyN) {
			wx, wy := s2w(sx, sy)
			nc := newCard(wx-DW/2, wy-DH/2, "New Card", []string{
				"// new card",
				"// press T to cycle themes",
				"// drag title bar to move",
				"// scroll on content area",
			})
			cards = append(cards, nc)
			if focCard != nil {
				focCard.focused = false
			}
			focCard = nc
			nc.focused = true
		}

		_ = dt

		// ── Render ────────────────────────────────────────────────────────
		ctx.BeginFrame()

		drawGrid(ctx)

		// Draw unfocused cards first, focused last
		for _, c := range cards {
			if !c.focused {
				drawCard(ctx, c)
			}
		}
		for _, c := range cards {
			if c.focused {
				drawCard(ctx, c)
			}
		}

		// ── HUD ────────────────────────────────────────────────────────────
		ctx.SetFillStyle("rgba(17,17,27,0.88)")
		ctx.BeginPath()
		ctx.RoundRect(12, H-40, 340, 28, 4)
		ctx.Fill()

		ctx.SetFont("10px fira")
		ctx.SetTextAlign("left")
		ctx.SetTextBaseline("middle")
		ctx.SetFillStyle("#6c7086")
		hud := fmt.Sprintf("theme: %s  zoom: %.0f%%  pan: %.0f,%.0f  cards: %d  T=theme  N=new  space+drag=pan",
			T.name, zoomVal*100, panX, panY, len(cards))
		ctx.FillText(hud, 20, H-26)

		ctx.EndFrame()
	}
}
