// Example: paths
//
// Exercises: BeginPath, MoveTo, LineTo, Arc (full circles and sectors),
// ArcTo (rounded corners via tangent arcs), RoundRect, ClosePath,
// Fill, Stroke, Clip (rect and roundRect), SetLineDash, nested save/clip.
//
// Visual: a panel of path-drawing demos — circles, pie slices, rounded
// polygons via arcTo, clipped content, and animated dash offset.
package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
	rc "github.com/ha1tch/raycanvas"
	"github.com/ha1tch/raycanvas/fonts"
)

const (
	W = 900
	H = 650
)

func main() {
	cache := rc.NewSharedCache()
	ctx := rc.SetupWindow(W, H, "raycanvas — paths", cache)
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

		// Background
		ctx.SetFillStyle("#1e1e2e")
		ctx.FillRect(0, 0, W, H)

		// ── Panel helper ──────────────────────────────────────────────────
		panel := func(x, y, w, h float32, title string) {
			ctx.SetFillStyle("rgba(49,50,68,0.7)")
			ctx.BeginPath()
			ctx.RoundRect(x, y, w, h, 6)
			ctx.Fill()
			ctx.SetStrokeStyle("rgba(255,255,255,0.08)")
			ctx.SetLineWidth(1)
			ctx.BeginPath()
			ctx.RoundRect(x, y, w, h, 6)
			ctx.Stroke()
			if title != "" {
				ctx.SetFillStyle("#6c7086")
				ctx.SetFont("10px inter")
				ctx.SetTextAlign("left")
				ctx.SetTextBaseline("top")
				ctx.FillText(title, x+10, y+8)
			}
		}

		// ── 1. Full circles ───────────────────────────────────────────────
		panel(16, 16, 200, 160, "arc — full circles")
		for i := 0; i < 5; i++ {
			r := float32(10 + i*8)
			cx := float32(56 + i*36)
			cy := float32(88)
			ctx.SetFillStyle([]string{"#f38ba8", "#fab387", "#f9e2af", "#a6e3a1", "#89b4fa"}[i])
			ctx.SetGlobalAlpha(0.7)
			ctx.BeginPath()
			ctx.Arc(cx, cy, r, 0, 2*math.Pi, false)
			ctx.Fill()
			ctx.SetGlobalAlpha(1.0)
			ctx.SetStrokeStyle([]string{"#f38ba8", "#fab387", "#f9e2af", "#a6e3a1", "#89b4fa"}[i])
			ctx.SetLineWidth(1.5)
			ctx.BeginPath()
			ctx.Arc(cx, cy, r+4, 0, 2*math.Pi, false)
			ctx.Stroke()
		}

		// ── 2. Pie sectors (partial arcs) ─────────────────────────────────
		panel(232, 16, 200, 160, "arc — pie chart")
		{
			cx, cy, r := float32(332), float32(96), float32(60)
			slices := []struct {
				start, end float32
				color      string
			}{
				{0, 1.2, "#f38ba8"},
				{1.2, 2.8, "#fab387"},
				{2.8, 4.5, "#a6e3a1"},
				{4.5, 2 * math.Pi, "#89b4fa"},
			}
			for _, s := range slices {
				ctx.SetFillStyle(s.color)
				ctx.BeginPath()
				ctx.MoveTo(cx, cy)
				ctx.Arc(cx, cy, r, s.start, s.end, false)
				ctx.ClosePath()
				ctx.Fill()
			}
			// Centre hole → donut
			ctx.SetFillStyle("#1e1e2e")
			ctx.BeginPath()
			ctx.Arc(cx, cy, r*0.45, 0, 2*math.Pi, false)
			ctx.Fill()
		}

		// ── 3. ArcTo — rounded polygon ────────────────────────────────────
		panel(448, 16, 200, 160, "arcTo — rounded corners")
		{
			// Draw a rounded triangle using arcTo
			ctx.SetFillStyle("rgba(203,166,247,0.3)")
			ctx.SetStrokeStyle("#cba6f7")
			ctx.SetLineWidth(2)
			points := [][2]float32{
				{548, 36}, {628, 156}, {468, 156},
			}
			r := float32(16)
			ctx.BeginPath()
			// Start at midpoint of last→first edge
			mx := (points[2][0] + points[0][0]) / 2
			my := (points[2][1] + points[0][1]) / 2
			ctx.MoveTo(mx, my)
			for i := range points {
				p1 := points[i]
				p2 := points[(i+1)%len(points)]
				ctx.ArcTo(p1[0], p1[1], p2[0], p2[1], r)
			}
			ctx.ClosePath()
			ctx.Fill()
			ctx.Stroke()

			// Rounded rectangle via arcTo (manual, to verify vs RoundRect)
			x, y, w, h, rad := float32(468), float32(36), float32(80), float32(40), float32(10)
			ctx.SetFillStyle("rgba(137,180,250,0.25)")
			ctx.SetStrokeStyle("#89b4fa")
			ctx.SetLineWidth(1.5)
			ctx.BeginPath()
			ctx.MoveTo(x+rad, y)
			ctx.ArcTo(x+w, y, x+w, y+h, rad)
			ctx.ArcTo(x+w, y+h, x, y+h, rad)
			ctx.ArcTo(x, y+h, x, y, rad)
			ctx.ArcTo(x, y, x+w, y, rad)
			ctx.ClosePath()
			ctx.Fill()
			ctx.Stroke()
		}

		// ── 4. Clip to rect ───────────────────────────────────────────────
		panel(664, 16, 220, 160, "clip — rect")
		{
			// Draw content, clip to inner rect; content outside is hidden
			ctx.Save()
			ctx.BeginPath()
			ctx.Rect(690, 44, 168, 112)
			ctx.Clip()

			// Circles that extend beyond the clip rect
			for i := 0; i < 6; i++ {
				angle := float32(i) * math.Pi / 3
				cx2 := float32(774) + float32(60)*float32(math.Cos(float64(angle)))
				cy2 := float32(100) + float32(60)*float32(math.Sin(float64(angle)))
				ctx.SetFillStyle([]string{
					"rgba(243,139,168,0.6)", "rgba(250,179,135,0.6)",
					"rgba(249,226,175,0.6)", "rgba(166,227,161,0.6)",
					"rgba(137,220,235,0.6)", "rgba(137,180,250,0.6)",
				}[i])
				ctx.BeginPath()
				ctx.Arc(cx2, cy2, 45, 0, 2*math.Pi, false)
				ctx.Fill()
			}
			ctx.Restore()

			// Clip boundary marker
			ctx.SetStrokeStyle("rgba(243,139,168,0.6)")
			ctx.SetLineWidth(1)
			ctx.SetLineDash([]float32{4, 3})
			ctx.StrokeRect(690, 44, 168, 112)
			ctx.SetLineDash(nil)
		}

		// ── 5. Clip to roundRect (the hard case) ──────────────────────────
		panel(16, 192, 200, 200, "clip — roundRect")
		{
			ctx.Save()
			ctx.BeginPath()
			ctx.RoundRect(28, 212, 176, 164, 20)
			ctx.Clip()

			// Diagonal stripes clipped to the rounded shape
			ctx.SetStrokeStyle("rgba(166,227,161,0.5)")
			ctx.SetLineWidth(8)
			for i := -20; i < 30; i += 3 {
				sx := float32(28 + i*12)
				ctx.BeginPath()
				ctx.MoveTo(sx, 212)
				ctx.LineTo(sx+164, 376)
				ctx.Stroke()
			}

			// Filled circle in centre, also clipped
			ctx.SetFillStyle("rgba(166,227,161,0.4)")
			ctx.BeginPath()
			ctx.Arc(116, 294, 55, 0, 2*math.Pi, false)
			ctx.Fill()
			ctx.Restore()

			// RoundRect outline
			ctx.SetStrokeStyle("rgba(166,227,161,0.6)")
			ctx.SetLineWidth(1.5)
			ctx.BeginPath()
			ctx.RoundRect(28, 212, 176, 164, 20)
			ctx.Stroke()
		}

		// ── 6. LineDash with animated offset ─────────────────────────────
		panel(232, 192, 420, 200, "setLineDash — animated dash offset")
		{
			patterns := []struct {
				dash  []float32
				color string
				label string
			}{
				{[]float32{8, 4}, "#f38ba8", "[8, 4]"},
				{[]float32{4, 4}, "#fab387", "[4, 4]"},
				{[]float32{12, 3, 3, 3}, "#f9e2af", "[12, 3, 3, 3]"},
				{[]float32{2, 6}, "#89b4fa", "[2, 6]"},
				{[]float32{16, 4, 4, 4, 4, 4}, "#cba6f7", "[16,4,4,4,4,4]"},
			}
			offset := t * 30 // animate the offset

			for i, p := range patterns {
				y2 := float32(220 + i*36)
				ctx.SetStrokeStyle(p.color)
				ctx.SetLineWidth(2)
				ctx.SetLineDash(p.dash)
				ctx.SetLineDashOffset(offset)
				ctx.BeginPath()
				ctx.MoveTo(310, y2)
				ctx.LineTo(630, y2)
				ctx.Stroke()
				ctx.SetLineDash(nil)
				ctx.SetLineDashOffset(0)

				ctx.SetFillStyle(p.color)
				ctx.SetFont("9px fira")
				ctx.SetTextAlign("left")
				ctx.SetTextBaseline("middle")
				ctx.FillText(p.label, 244, y2)
			}
		}

		// ── 7. Animated arc sweep ─────────────────────────────────────────
		panel(664, 192, 220, 200, "arc — animated sweep")
		{
			cx2, cy2, r2 := float32(774), float32(292), float32(70)
			sweep := float32(math.Pi * 2 * (0.5 + 0.5*math.Sin(float64(t*0.7))))

			// Background ring
			ctx.SetStrokeStyle("rgba(255,255,255,0.08)")
			ctx.SetLineWidth(12)
			ctx.BeginPath()
			ctx.Arc(cx2, cy2, r2, 0, 2*math.Pi, false)
			ctx.Stroke()

			// Animated arc
			ctx.SetStrokeStyle("#89b4fa")
			ctx.SetLineWidth(12)
			ctx.BeginPath()
			ctx.Arc(cx2, cy2, r2, -math.Pi/2, -math.Pi/2+sweep, false)
			ctx.Stroke()

			// Inner fill
			ctx.SetFillStyle("rgba(137,180,250,0.15)")
			ctx.BeginPath()
			ctx.Arc(cx2, cy2, r2-6, 0, 2*math.Pi, false)
			ctx.Fill()
		}

		// ── 8. Nested clip (depth 2) ──────────────────────────────────────
		panel(16, 408, 868, 220, "nested clip — depth 2 (rect inside rect)")
		{
			// Outer clip
			ctx.Save()
			ctx.BeginPath()
			ctx.Rect(30, 430, 840, 186)
			ctx.Clip()

			// Content: a gradient of overlapping circles
			for i := 0; i < 20; i++ {
				cx3 := float32(30 + i*45)
				ctx.SetFillStyle([]string{
					"rgba(243,139,168,0.4)", "rgba(250,179,135,0.4)",
					"rgba(249,226,175,0.4)", "rgba(166,227,161,0.4)",
					"rgba(137,220,235,0.4)", "rgba(137,180,250,0.4)",
					"rgba(203,166,247,0.4)",
				}[i%7])
				ctx.BeginPath()
				ctx.Arc(cx3, 523, 60, 0, 2*math.Pi, false)
				ctx.Fill()
			}

			// Inner clip — further restricts to left half
			ctx.Save()
			ctx.BeginPath()
			ctx.Rect(30, 430, 420, 186)
			ctx.Clip()

			// Tint the clipped region to show the intersection
			ctx.SetFillStyle("rgba(255,255,255,0.07)")
			ctx.FillRect(30, 430, 420, 186)
			ctx.SetStrokeStyle("rgba(255,255,255,0.3)")
			ctx.SetLineWidth(1)
			ctx.SetLineDash([]float32{4, 3})
			ctx.StrokeRect(30, 430, 420, 186)
			ctx.SetLineDash(nil)
			ctx.Restore()

			ctx.Restore()

			// Outer clip boundary
			ctx.SetStrokeStyle("rgba(255,255,255,0.15)")
			ctx.SetLineWidth(1)
			ctx.SetLineDash([]float32{6, 4})
			ctx.StrokeRect(30, 430, 840, 186)
			ctx.SetLineDash(nil)
		}

		ctx.EndFrame()
	}
}
