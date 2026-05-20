// Example: shadows
//
// Demonstrates the raycanvas shadow pipeline on a light background
// so dark drop shadows are clearly visible:
//
//  1. Drop shadow   — card elevation (blur + offset, gg cached)
//  2. Glow          — coloured shadow without offset
//  3. Text shadow   — offset draw for legibility
//  4. Blur depth    — four panels at increasing blur radii
package main

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
	rc "github.com/ha1tch/raycanvas"
	"github.com/ha1tch/raycanvas/fonts"
)

const (
	W = 960
	H = 700
)

// clearShadow resets all shadow state to prevent bleed between sections.
func clearShadow(ctx *rc.Context) {
	ctx.SetShadowColor("transparent")
	ctx.SetShadowBlur(0)
	ctx.SetShadowOffsetX(0)
	ctx.SetShadowOffsetY(0)
}

func main() {
	cache := rc.NewSharedCache()
	ctx := rc.SetupWindow(W, H, "raycanvas — shadows & blur", cache)
	defer rl.CloseWindow()
	defer cache.Unload()
	rl.SetTargetFPS(60)
	if err := fonts.Register(cache); err != nil {
		panic(err)
	}

	t := float32(0)

	for !rl.WindowShouldClose() {
		t += rl.GetFrameTime()

		ctx.BeginFrame()

		// Light background — dark shadows are visible against light
		ctx.SetFillStyle("#e8e8f0")
		ctx.FillRect(0, 0, W, H)

		// Subtle grid
		ctx.SetStrokeStyle("rgba(0,0,0,0.06)")
		ctx.SetLineWidth(1)
		ctx.BeginPath()
		for x := float32(40); x < W; x += 40 {
			ctx.MoveTo(x, 0); ctx.LineTo(x, H)
		}
		for y := float32(40); y < H; y += 40 {
			ctx.MoveTo(0, y); ctx.LineTo(W, y)
		}
		ctx.Stroke()

		sectionLabel := func(x, y float32, s string) {
			ctx.SetFillStyle("#9090a0")
			ctx.SetFont("10px inter")
			ctx.SetTextAlign("left")
			ctx.SetTextBaseline("top")
			ctx.FillText(s, x, y)
		}

		// ── 1. DROP SHADOW — card elevation ──────────────────────────────
		sectionLabel(20, 14, "drop shadow — card elevation (shadow offset → bottom right)")

		pulse := float32(0.5 + 0.5*math.Sin(float64(t*0.9)))
		animBlur := float32(4) + pulse*16
		animOffY := float32(3) + pulse*10
		animOffX := float32(2) + pulse*6

		cards := []struct {
			x, y, blur, offX, offY float32
			name                   string
		}{
			{50,  40, 4,        2,        4,        "resting\nblur:4 off:2,4"},
			{220, 40, animBlur, animOffX, animOffY, fmt.Sprintf("animated\nblur:%.0f off:%.0f,%.0f", animBlur, animOffX, animOffY)},
			{390, 40, 20,       6,        12,       "lifted\nblur:20 off:6,12"},
		}

		for _, c := range cards {
			cw, ch := float32(120), float32(160)

			// Shadow — drawn first, before the card
			ctx.SetShadowColor("rgba(0,0,0,0.22)")
			ctx.SetShadowBlur(c.blur)
			ctx.SetShadowOffsetX(c.offX)
			ctx.SetShadowOffsetY(c.offY)
			ctx.SetFillStyle("#ffffff")
			ctx.FillRoundRect(c.x, c.y, cw, ch, 10)
			clearShadow(ctx)

			// Card body on top
			ctx.SetFillStyle("#ffffff")
			ctx.FillRoundRect(c.x, c.y, cw, ch, 10)
			ctx.SetStrokeStyle("rgba(0,0,0,0.08)")
			ctx.SetLineWidth(1)
			ctx.StrokeRoundRect(c.x, c.y, cw, ch, 10)

			// Title bar
			ctx.SetFillStyle("#f0f0f8")
			ctx.FillRoundRectTop(c.x, c.y, cw, 28, 10)
			ctx.SetStrokeStyle("rgba(0,0,0,0.06)")
			ctx.SetLineWidth(1)
			ctx.BeginPath()
			ctx.MoveTo(c.x, c.y+28)
			ctx.LineTo(c.x+cw, c.y+28)
			ctx.Stroke()

			// Drag dots
			ctx.SetFillStyle("#c0c0d0")
			for d := 0; d < 3; d++ {
				ctx.FillCircle(c.x+cw/2-8+float32(d)*8, c.y+14, 1.7)
			}

			// Content lines
			ctx.SetFillStyle("rgba(0,0,0,0.10)")
			for li := 0; li < 4; li++ {
				lw := float32(80) - float32(li)*12
				ctx.FillRect(c.x+14, c.y+40+float32(li)*24, lw, 8)
			}

			// Label below
			ctx.SetFillStyle("#808090")
			ctx.SetFont("9px inter")
			ctx.SetTextAlign("center")
			ctx.SetTextBaseline("top")
			ctx.FillText(c.name, c.x+cw/2, c.y+ch+8)
		}

		// ── 2. GLOW — coloured shadow, no offset ─────────────────────────
		sectionLabel(560, 14, "glow — coloured shadow (no offset)")

		glows := []struct{ col, name string }{
			{"#7232c8", "purple"},
			{"#0055d4", "blue"},
			{"#0a8a2e", "green"},
			{"#d42050", "red"},
		}
		glowPulse := float32(0.5 + 0.5*math.Sin(float64(t*1.2)))

		for i, g := range glows {
			cx := float32(590 + i*92)
			cy := float32(130)
			r  := float32(34)

			ctx.SetShadowColor(g.col)
			ctx.SetShadowBlur(12 + glowPulse*10)
			ctx.SetFillStyle(g.col)
			ctx.FillCircle(cx, cy, r)
			clearShadow(ctx)

			ctx.SetFillStyle("#ffffff")
			ctx.FillCircle(cx, cy, r-5)
			ctx.SetStrokeStyle(g.col)
			ctx.SetLineWidth(2)
			ctx.StrokeCircle(cx, cy, r-2)

			ctx.SetFillStyle(g.col)
			ctx.SetFont("10px inter")
			ctx.SetTextAlign("center")
			ctx.SetTextBaseline("middle")
			ctx.FillText(g.name, cx, cy)
		}

		// Glowing card border
		{
			fx, fy := float32(562), float32(192)
			fw, fh := float32(380), float32(56)
			fp := float32(0.5 + 0.5*math.Sin(float64(t*1.0)))

			ctx.SetShadowColor("#7232c8")
			ctx.SetShadowBlur(8 + fp*8)
			ctx.SetFillStyle("#ffffff")
			ctx.FillRoundRect(fx, fy, fw, fh, 9)
			clearShadow(ctx)

			ctx.SetFillStyle("#ffffff")
			ctx.FillRoundRect(fx, fy, fw, fh, 9)
			ctx.SetStrokeStyle("#7232c8")
			ctx.SetLineWidth(1.5)
			ctx.StrokeRoundRect(fx, fy, fw, fh, 9)

			ctx.SetFillStyle("#333")
			ctx.SetFont("12px inter")
			ctx.SetTextAlign("left")
			ctx.SetTextBaseline("middle")
			ctx.FillText("focused card — purple glow border", fx+14, fy+fh/2)
		}

		// ── 3. TEXT SHADOW ────────────────────────────────────────────────
		sectionLabel(20, 278, "text shadow — offset draw (no blur)")

		// Noisy background patch
		for row := 0; row < 4; row++ {
			for col := 0; col < 7; col++ {
				ctx.SetFillStyle([]string{
					"rgba(114,50,200,0.30)", "rgba(0,85,212,0.30)",
					"rgba(10,138,46,0.30)", "rgba(212,32,80,0.30)",
					"rgba(0,112,160,0.30)", "rgba(192,80,0,0.30)",
					"rgba(114,50,200,0.20)",
				}[(row+col)%7])
				ctx.FillCircle(float32(36+col*62), float32(320+row*74), 26)
			}
		}

		ctx.SetFont("700 13px inter")
		ctx.SetTextAlign("left")
		ctx.SetTextBaseline("middle")

		samples := []struct {
			y    float32
			text string
			col  string
			shad string
			ox, oy float32
		}{
			{322, "No shadow — text lost in noise",        "#2d2d2d", "transparent", 0, 0},
			{362, "Dark shadow — legible (1,2)",           "#1a1a1a", "rgba(255,255,255,0.8)", 1, 2},
			{402, "White halo — maximum contrast",         "#1a1a1a", "rgba(255,255,255,0.95)", 0, 0},
			{442, "Colour shadow — vivid",                 "#7232c8", "#7232c8", 2, 2},
			{482, fmt.Sprintf("Animated (%.1f, %.1f)",
				float32(math.Sin(float64(t))*4),
				float32(math.Cos(float64(t*0.7))*3+2)),
				"#d42050", "rgba(0,0,0,0.5)",
				float32(math.Sin(float64(t)) * 4),
				float32(math.Cos(float64(t*0.7))*3 + 2)},
		}

		for _, s := range samples {
			ctx.SetShadowColor(s.shad)
			ctx.SetShadowOffsetX(s.ox)
			ctx.SetShadowOffsetY(s.oy)
			ctx.SetFillStyle(s.col)
			ctx.FillText(s.text, 22, s.y)
			clearShadow(ctx)
		}

		// ── 4. BLUR DEPTH ─────────────────────────────────────────────────
		sectionLabel(560, 278, "blur depth — shadow radius 0 / 4 / 12 / 24")

		blurCfgs := []struct {
			blur float32
			name string
		}{
			{0, "none"},
			{4, "4px"},
			{12, "12px"},
			{24, "24px"},
		}

		for i, bl := range blurCfgs {
			bx := float32(562 + i*96)
			by := float32(300)
			bw := float32(80)
			bh := float32(120)

			// Coloured background to show shadow contrast
			ctx.SetFillStyle([]string{
				"rgba(114,50,200,0.15)",
				"rgba(0,85,212,0.15)",
				"rgba(10,138,46,0.15)",
				"rgba(212,32,80,0.15)",
			}[i])
			ctx.FillRoundRect(bx, by, bw, bh, 8)

			// Shadowed white card on top
			ctx.SetShadowColor("rgba(0,0,0,0.28)")
			ctx.SetShadowBlur(bl.blur)
			ctx.SetShadowOffsetX(3)
			ctx.SetShadowOffsetY(5)
			ctx.SetFillStyle("#ffffff")
			ctx.FillRoundRect(bx+8, by+10, bw-16, bh-20, 6)
			clearShadow(ctx)

			ctx.SetFillStyle("#ffffff")
			ctx.FillRoundRect(bx+8, by+10, bw-16, bh-20, 6)
			ctx.SetStrokeStyle("rgba(0,0,0,0.06)")
			ctx.SetLineWidth(1)
			ctx.StrokeRoundRect(bx+8, by+10, bw-16, bh-20, 6)

			ctx.SetFillStyle("#808090")
			ctx.SetFont("9px inter")
			ctx.SetTextAlign("center")
			ctx.SetTextBaseline("top")
			ctx.FillText(bl.name, bx+bw/2, by+bh+6)
		}

		// Animated glow bar
		aBlur := float32(2 + 14*math.Abs(math.Sin(float64(t*0.5))))
		ctx.SetShadowColor("rgba(114,50,200,0.50)")
		ctx.SetShadowBlur(aBlur)
		ctx.SetShadowOffsetX(0)
		ctx.SetShadowOffsetY(4)
		ctx.SetFillStyle("#ffffff")
		ctx.FillRoundRect(562, 450, 380, 36, 6)
		clearShadow(ctx)

		ctx.SetFillStyle("#ffffff")
		ctx.FillRoundRect(562, 450, 380, 36, 6)
		ctx.SetFillStyle("#7232c8")
		ctx.SetFont("11px inter")
		ctx.SetTextAlign("center")
		ctx.SetTextBaseline("middle")
		ctx.FillText(fmt.Sprintf("animated purple glow — blur: %.1fpx", aBlur), 752, 468)

		// ── HUD ───────────────────────────────────────────────────────────
		ctx.SetFillStyle("rgba(0,0,0,0.65)")
		ctx.FillRoundRect(10, H-30, W-20, 20, 4)
		ctx.SetFillStyle("#c0c0d0")
		ctx.SetFont("9px fira")
		ctx.SetTextAlign("left")
		ctx.SetTextBaseline("middle")
		ctx.FillText(
			"shadow pipeline: shape → gg gaussian blur → Texture2D (cached by color+blur+geometry) | text shadow → offset draw",
			18, H-20,
		)

		ctx.EndFrame()
	}
}
