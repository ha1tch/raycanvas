// Example: curves
//
// Exercises: BezierCurveTo, the two-pass glow+stroke pattern used in the
// shevo inter-perspective link layer, animated pulse on the curve,
// control-point visualisation, and multiple simultaneous curves.
//
// This example mirrors the exact _curve() function from shevo-0.2.3's
// integration layer — the primary use case for BezierCurveTo in the codebase.
//
// Visual: draggable-style link curves between "joxel cells" (left panel)
// and "quag cards" (right panel), with animated data-flow pulse dots,
// control-point handles, and a ghost (dashed, lower opacity) variant.
package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
	rc "github.com/ha1tch/raycanvas"
	"github.com/ha1tch/raycanvas/examples/internal/fonts"
)

const (
	W = 900
	H = 600
)

type link struct {
	ax, ay float32 // source anchor (right-centre of a cell)
	bx, by float32 // dest anchor (left-centre of a card)
	color  string
	ghost  bool
}

func main() {
	cache := rc.NewSharedCache()
	ctx := rc.SetupWindow(W, H, "raycanvas — curves (bezierCurveTo)", cache)
	defer rl.CloseWindow()
	defer cache.Unload()
	rl.SetTargetFPS(60)
	if err := fonts.Register(cache); err != nil {
		panic(err)
	}

	// Static links — matching the shevo inter-perspective link geometry.
	links := []link{
		{ax: 310, ay: 140, bx: 560, by: 120, color: "#89b4fa"},
		{ax: 310, ay: 220, bx: 560, by: 200, color: "#a6e3a1"},
		{ax: 310, ay: 300, bx: 560, by: 340, color: "#f38ba8"},
		{ax: 310, ay: 380, bx: 560, by: 280, color: "#fab387"},
		{ax: 310, ay: 460, bx: 560, by: 460, color: "#cba6f7"},
		// Ghost (lower opacity, dashed) — pending / broken link
		{ax: 310, ay: 540, bx: 560, by: 400, color: "#89b4fa", ghost: true},
	}

	// Pulse state per link: 0=idle, 1=peak
	pulses := make([]float32, len(links))

	t := float32(0)
	showHandles := true

	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()
		t += dt

		// Toggle control handles with H key
		if rl.IsKeyPressed(rl.KeyH) {
			showHandles = !showHandles
		}

		// Decay pulses and trigger new ones
		for i := range pulses {
			pulses[i] -= dt * 0.8
			if pulses[i] < 0 {
				pulses[i] = 0
			}
		}
		// Trigger pulses on a staggered cycle
		for i := range links {
			phase := float32(i) * 0.7
			if math.Mod(float64(t+phase), 2.5) < float64(dt) {
				pulses[i] = 1.0
			}
		}

		ctx.BeginFrame()

		// Background
		ctx.SetFillStyle("#1e1e2e")
		ctx.FillRect(0, 0, W, H)

		// ── Left panel: "joxel" cells ─────────────────────────────────────
		ctx.SetFillStyle("rgba(30,30,46,0.9)")
		ctx.BeginPath()
		ctx.RoundRect(20, 20, 300, H-40, 8)
		ctx.Fill()
		ctx.SetStrokeStyle("rgba(255,255,255,0.08)")
		ctx.SetLineWidth(1)
		ctx.BeginPath()
		ctx.RoundRect(20, 20, 300, H-40, 8)
		ctx.Stroke()

		ctx.SetFillStyle("#6c7086")
		ctx.SetFont("10px inter")
		ctx.SetTextAlign("center")
		ctx.SetTextBaseline("middle")
		ctx.FillText("joxel", 170, 42)

		for _, lk := range links {
			// Cell row
			ctx.SetFillStyle("rgba(49,50,68,0.6)")
			ctx.BeginPath()
			ctx.RoundRect(32, lk.ay-18, 266, 36, 4)
			ctx.Fill()
			ctx.SetFillStyle("#cdd6f4")
			ctx.SetFont("11px fira")
			ctx.SetTextAlign("left")
			ctx.SetTextBaseline("middle")
			label := "=A1.value"
			if lk.ghost {
				label = "=B3.total  (unresolved)"
				ctx.SetFillStyle("#585b70")
			}
			ctx.FillText(label, 44, lk.ay)
		}

		// ── Right panel: "quag" cards ─────────────────────────────────────
		ctx.SetFillStyle("rgba(30,30,46,0.9)")
		ctx.BeginPath()
		ctx.RoundRect(580, 20, 300, H-40, 8)
		ctx.Fill()
		ctx.SetStrokeStyle("rgba(255,255,255,0.08)")
		ctx.SetLineWidth(1)
		ctx.BeginPath()
		ctx.RoundRect(580, 20, 300, H-40, 8)
		ctx.Stroke()

		ctx.SetFillStyle("#6c7086")
		ctx.SetFont("10px inter")
		ctx.SetTextAlign("center")
		ctx.SetTextBaseline("middle")
		ctx.FillText("quag", 730, 42)

		for _, lk := range links {
			ctx.SetFillStyle("rgba(49,50,68,0.6)")
			ctx.BeginPath()
			ctx.RoundRect(592, lk.by-24, 276, 48, 6)
			ctx.Fill()
			ctx.SetStrokeStyle(lk.color)
			if lk.ghost {
				ctx.SetGlobalAlpha(0.3)
			} else {
				ctx.SetGlobalAlpha(0.5)
			}
			ctx.SetLineWidth(1)
			ctx.BeginPath()
			ctx.RoundRect(592, lk.by-24, 276, 48, 6)
			ctx.Stroke()
			ctx.SetGlobalAlpha(1.0)

			ctx.SetFillStyle("#a6adc8")
			ctx.SetFont("10px inter")
			ctx.SetTextAlign("left")
			ctx.SetTextBaseline("middle")
			ctx.FillText("Card", 606, lk.by-8)
			ctx.SetFillStyle(lk.color)
			ctx.SetFont("12px fira")
			ctx.FillText("42.7", 606, lk.by+10)

		}

		// ── Draw curves ───────────────────────────────────────────────────
		for i, lk := range links {
			pt := pulses[i]
			drawCurve(ctx, lk, pt, t, showHandles)
		}

		// ── Anchor dots — drawn after curves so they sit on top ───────────
		for _, lk := range links {
			ctx.SetFillStyle(lk.color)
			if lk.ghost {
				ctx.SetGlobalAlpha(0.4)
			}
			// Left anchor (joxel side)
			ctx.BeginPath()
			ctx.Arc(310, lk.ay, 5, 0, 2*math.Pi, false)
			ctx.Fill()
			// Right anchor (quag side)
			ctx.BeginPath()
			ctx.Arc(580, lk.by, 5, 0, 2*math.Pi, false)
			ctx.Fill()
			ctx.SetGlobalAlpha(1.0)
		}

		// ── Legend ────────────────────────────────────────────────────────
		ctx.SetFillStyle("#585b70")
		ctx.SetFont("10px inter")
		ctx.SetTextAlign("left")
		ctx.SetTextBaseline("middle")
		ctx.FillText("H — toggle control handles", 20, H-16)

		ctx.EndFrame()
	}
}

// drawCurve renders a single bezierCurveTo link in the shevo style:
// a wide low-opacity glow pass, then a crisp main line, then endpoint dots.
// pt is the pulse intensity (0–1).
func drawCurve(ctx *rc.Context, lk link, pt, t float32, showHandles bool) {
	ax, ay := lk.ax, lk.ay
	bx, by := lk.bx, lk.by

	// Control point offsets — match shevo's _curve() geometry exactly:
	// off = max(70, |dx| * 0.48), sign follows direction of travel.
	dx := bx - ax
	if dx < 0 {
		dx = -dx
	}
	off := float32(70)
	if dx*0.48 > off {
		off = dx * 0.48
	}
	sign := float32(1)
	if lk.bx < lk.ax {
		sign = -1
	}
	cp1x := ax + sign*off
	cp1y := ay
	cp2x := bx - sign*off
	cp2y := by

	alpha := float32(1.0)
	if lk.ghost {
		alpha = 0.48
	}

	// ── Glow pass ─────────────────────────────────────────────────────────
	glowAlpha := float32(0.16)
	if lk.ghost {
		glowAlpha = 0.10
	}
	glowAlpha += pt * 0.28
	glowW := float32(14)
	if lk.ghost {
		glowW = 8
	}

	ctx.Save()
	ctx.SetGlobalAlpha(glowAlpha)
	ctx.SetStrokeStyle(lk.color)
	ctx.SetLineWidth(glowW)
	ctx.SetLineCap("round")
	ctx.BeginPath()
	ctx.MoveTo(ax, ay)
	ctx.BezierCurveTo(cp1x, cp1y, cp2x, cp2y, bx, by)
	ctx.Stroke()
	ctx.Restore()

	// ── Main line ─────────────────────────────────────────────────────────
	ctx.Save()
	ctx.SetGlobalAlpha(alpha + pt*0.12)
	ctx.SetStrokeStyle(lk.color)
	lineW := float32(2.5) + pt*1.5
	ctx.SetLineWidth(lineW)
	ctx.SetLineCap("round")
	if lk.ghost {
		ctx.SetLineDash([]float32{6, 5})
	}
	ctx.BeginPath()
	ctx.MoveTo(ax, ay)
	ctx.BezierCurveTo(cp1x, cp1y, cp2x, cp2y, bx, by)
	ctx.Stroke()
	ctx.SetLineDash(nil)
	ctx.Restore()

	// ── Pulse dot travelling along the curve ─────────────────────────────
	if pt > 0.05 && !lk.ghost {
		// Evaluate cubic Bézier at parameter t_pos
		tPos := float32(math.Mod(float64(t)*0.4, 1.0))
		u := 1 - tPos
		px := u*u*u*ax + 3*u*u*tPos*cp1x + 3*u*tPos*tPos*cp2x + tPos*tPos*tPos*bx
		py := u*u*u*ay + 3*u*u*tPos*cp1y + 3*u*tPos*tPos*cp2y + tPos*tPos*tPos*by

		ctx.Save()
		ctx.SetGlobalAlpha(pt * 0.9)
		ctx.SetFillStyle(lk.color)
		ctx.BeginPath()
		ctx.Arc(px, py, 4+pt*2, 0, 2*math.Pi, false)
		ctx.Fill()
		ctx.Restore()
	}

	// ── Control point handles (toggle with H) ────────────────────────────
	if showHandles {
		ctx.Save()
		ctx.SetGlobalAlpha(0.35)
		ctx.SetStrokeStyle(lk.color)
		ctx.SetLineWidth(1)
		ctx.SetLineDash([]float32{3, 3})

		// Handle lines
		ctx.BeginPath()
		ctx.MoveTo(ax, ay)
		ctx.LineTo(cp1x, cp1y)
		ctx.Stroke()
		ctx.BeginPath()
		ctx.MoveTo(bx, by)
		ctx.LineTo(cp2x, cp2y)
		ctx.Stroke()
		ctx.SetLineDash(nil)

		// Control point dots
		ctx.SetFillStyle(lk.color)
		ctx.SetGlobalAlpha(0.5)
		ctx.BeginPath()
		ctx.Arc(cp1x, cp1y, 3, 0, 2*math.Pi, false)
		ctx.Fill()
		ctx.BeginPath()
		ctx.Arc(cp2x, cp2y, 3, 0, 2*math.Pi, false)
		ctx.Fill()
		ctx.Restore()
	}
}
