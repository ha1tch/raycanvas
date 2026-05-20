// Example: basic
//
// Exercises: FillRect, StrokeRect, ClearRect, SetGlobalAlpha, Save/Restore,
// SetFillStyle, SetStrokeStyle, SetLineWidth, RoundRect fill and stroke.
//
// Visual: a grid of coloured rectangles with varying alpha, stroked frames,
// rounded cards at increasing radii, nested save/restore state demo, a
// ClearRect "hole", and an animated oscillating box.
package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
	rc "github.com/ha1tch/raycanvas"
	"github.com/ha1tch/raycanvas/fonts"
)

const (
	W = 800
	H = 600
)

func main() {
	cache := rc.NewSharedCache()
	ctx := rc.SetupWindow(W, H, "raycanvas — basic", cache)
	defer rl.CloseWindow()
	defer cache.Unload()
	rl.SetTargetFPS(60)
	if err := fonts.Register(cache); err != nil {
		panic(err)
	}

	t := float32(0)

	colors := []string{
		"#f38ba8", "#fab387", "#f9e2af",
		"#a6e3a1", "#89dceb", "#89b4fa", "#cba6f7",
	}

	for !rl.WindowShouldClose() {
		t += rl.GetFrameTime()

		ctx.BeginFrame()

		// Background
		ctx.SetFillStyle("#1e1e2e")
		ctx.FillRect(0, 0, W, H)

		// ── Row 1: solid colour blocks ────────────────────────────────────
		for i, col := range colors {
			ctx.SetFillStyle(col)
			ctx.FillRect(float32(20+i*110), 20, 90, 60)
		}

		// ── Row 2: same blocks with pulsing global alpha ──────────────────
		pulse := float32(0.3 + 0.7*math.Abs(math.Sin(float64(t*0.8))))
		ctx.Save()
		ctx.SetGlobalAlpha(pulse)
		for i, col := range colors {
			ctx.SetFillStyle(col)
			ctx.FillRect(float32(20+i*110), 100, 90, 60)
		}
		ctx.Restore() // globalAlpha restored to 1.0

		// ── Row 3: stroked rectangles, line width 1–7 ────────────────────
		for i, col := range colors {
			ctx.SetStrokeStyle(col)
			ctx.SetLineWidth(float32(i+1) * 0.9)
			ctx.StrokeRect(float32(20+i*110), 180, 90, 60)
		}

		// ── Row 4: rounded rects — fill + stroke at increasing radii ─────
		radii := []float32{2, 6, 10, 14, 18, 22, 28}
		for i, r := range radii {
			x := float32(20 + i*110)
			// Semi-transparent fill
			ctx.SetFillStyle(colors[i])
			ctx.SetGlobalAlpha(0.22)
			ctx.BeginPath()
			ctx.RoundRect(x, 260, 90, 60, r)
			ctx.Fill()
			// Opaque stroke
			ctx.SetGlobalAlpha(1.0)
			ctx.SetStrokeStyle(colors[i])
			ctx.SetLineWidth(1.5)
			ctx.BeginPath()
			ctx.RoundRect(x, 260, 90, 60, r)
			ctx.Stroke()
		}

		// ── Save/restore nesting: three levels, each with different state ─
		// Outer: red, alpha 0.9
		ctx.Save()
		ctx.SetFillStyle("#f38ba8")
		ctx.SetGlobalAlpha(0.9)
		ctx.FillRect(20, 350, 200, 110)

		// Middle: blue, alpha 0.75
		ctx.Save()
		ctx.SetFillStyle("#89b4fa")
		ctx.SetGlobalAlpha(0.75)
		ctx.FillRect(45, 368, 150, 75)

		// Inner: green, alpha 0.95
		ctx.Save()
		ctx.SetFillStyle("#a6e3a1")
		ctx.SetGlobalAlpha(0.95)
		ctx.FillRect(70, 386, 100, 40)
		ctx.Restore() // back to blue, 0.75

		// Confirm restore: draw a thin stripe in the restored blue
		ctx.FillRect(45, 434, 150, 8)
		ctx.Restore() // back to red, 0.9

		// Confirm: another stripe in restored red
		ctx.FillRect(20, 448, 200, 8)
		ctx.Restore() // back to defaults

		// ── ClearRect punches a transparent hole ──────────────────────────
		ctx.SetFillStyle("#cba6f7")
		ctx.SetGlobalAlpha(0.85)
		ctx.FillRect(260, 350, 200, 110)
		ctx.SetGlobalAlpha(1.0)
		ctx.ClearRect(305, 378, 110, 54)

		// ── Dashed stroke rect ────────────────────────────────────────────
		ctx.SetStrokeStyle("#f9e2af")
		ctx.SetLineWidth(1.5)
		ctx.SetLineDash([]float32{6, 4})
		ctx.StrokeRect(500, 350, 270, 110)
		ctx.SetLineDash(nil)

		// Inside the dashed box: a stroked rounded rect
		ctx.SetStrokeStyle("rgba(249,226,175,0.5)")
		ctx.SetLineWidth(1)
		ctx.BeginPath()
		ctx.RoundRect(512, 362, 246, 86, 8)
		ctx.Stroke()

		// ── Animated oscillating filled+stroked box ───────────────────────
		ox := float32(550) + float32(90)*float32(math.Sin(float64(t*1.4)))
		oy := float32(500) + float32(32)*float32(math.Cos(float64(t*2.1)))
		ctx.SetFillStyle("rgba(137,180,250,0.55)")
		ctx.FillRect(ox-35, oy-22, 70, 44)
		ctx.SetStrokeStyle("#89b4fa")
		ctx.SetLineWidth(2)
		ctx.StrokeRect(ox-35, oy-22, 70, 44)

		// ── Hairline border around the whole canvas ───────────────────────
		ctx.SetStrokeStyle("rgba(255,255,255,0.10)")
		ctx.SetLineWidth(1)
		ctx.StrokeRect(0.5, 0.5, W-1, H-1)

		ctx.EndFrame()
	}
}
