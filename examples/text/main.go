// Example: text
//
// Exercises: SetFont, FillText, MeasureText, SetTextAlign, SetTextBaseline,
// CSS font string parsing (size, weight, family), font fallback chain,
// measured-width-based layout, and word-wrap using MeasureText.
//
// Fonts are embedded via the internal/fonts package (Inter + Fira Code),
// so this example renders correctly on any platform without system fonts.
package main

import (
	"fmt"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/ha1tch/raycanvas/fonts"
	rc "github.com/ha1tch/raycanvas"
)

const (
	W = 900
	H = 700
)

func main() {
	cache := rc.NewSharedCache()
	ctx := rc.SetupWindow(W, H, "raycanvas — text", cache)
	defer rl.CloseWindow()
	defer cache.Unload()
	rl.SetTargetFPS(60)
	if err := fonts.Register(cache); err != nil {
		panic(err)
	}

	for !rl.WindowShouldClose() {
		ctx.BeginFrame()

		// Background
		ctx.SetFillStyle("#1e1e2e")
		ctx.FillRect(0, 0, W, H)

		// ── Section header ────────────────────────────────────────────────
		sectionY := float32(12)
		section := func(title string) float32 {
			ctx.SetFillStyle("rgba(255,255,255,0.06)")
			ctx.FillRect(16, sectionY, W-32, 20)
			ctx.SetFillStyle("#6c7086")
			ctx.SetFont("400 10px inter")
			ctx.SetTextAlign("left")
			ctx.SetTextBaseline("middle")
			ctx.FillText(title, 24, sectionY+10)
			y := sectionY + 28
			sectionY += 28
			return y
		}

		// ── Section 1: Fira Code at all shevo sizes ───────────────────────
		y := section("Fira Code — all sizes used in shevo (8–14px)")

		sizes := []float32{8, 9, 10, 11, 12, 13, 14}
		for _, sz := range sizes {
			ctx.SetFont(fmt.Sprintf("400 %.0fpx fira", sz))
			ctx.SetFillStyle("#cdd6f4")
			ctx.SetTextBaseline("alphabetic")
			label := fmt.Sprintf("%.0fpx — The quick brown fox jumps over the lazy dog  0123456789", sz)
			ctx.FillText(label, 24, y)
			// Underline at measured width verifies MeasureText matches DrawTextEx
			w := ctx.MeasureText(label)
			ctx.SetStrokeStyle("rgba(137,180,250,0.25)")
			ctx.SetLineWidth(0.5)
			ctx.BeginPath()
			ctx.MoveTo(24, y+2)
			ctx.LineTo(24+w, y+2)
			ctx.Stroke()
			y += sz*1.9 + 2
		}
		sectionY = y + 6

		// ── Section 2: Inter weight variants ─────────────────────────────
		y = section("Inter — weight variants at 13px")

		for _, w := range []struct{ css, label string }{
			{"400 13px inter", "Regular 400 — Inter: AaBbCcDdEeFf 0123456789"},
			{"500 13px inter", "Medium 500 — Inter: AaBbCcDdEeFf 0123456789"},
			{"600 13px inter", "SemiBold 600 — Inter: AaBbCcDdEeFf 0123456789"},
			{"700 13px inter", "Bold 700 — Inter: AaBbCcDdEeFf 0123456789"},
		} {
			ctx.SetFont(w.css)
			ctx.SetFillStyle("#cdd6f4")
			ctx.SetTextBaseline("alphabetic")
			ctx.FillText(w.label, 24, y)
			y += 22
		}
		sectionY = y + 6

		// ── Section 3: textAlign ──────────────────────────────────────────
		y = section("textAlign: left / center / right")

		midX := float32(W / 2)
		ctx.SetStrokeStyle("rgba(243,139,168,0.35)")
		ctx.SetLineWidth(1)
		ctx.SetLineDash([]float32{4, 4})
		ctx.BeginPath()
		ctx.MoveTo(midX, y-4)
		ctx.LineTo(midX, y+56)
		ctx.Stroke()
		ctx.SetLineDash(nil)

		ctx.SetFont("13px inter")
		ctx.SetFillStyle("#cdd6f4")
		ctx.SetTextBaseline("alphabetic")

		ctx.SetTextAlign("left")
		ctx.FillText("left-aligned — starts at the guide →", midX, y+13)
		ctx.SetTextAlign("center")
		ctx.FillText("← centered on the guide →", midX, y+31)
		ctx.SetTextAlign("right")
		ctx.FillText("← right-aligned — ends at the guide", midX, y+49)
		ctx.SetTextAlign("left")
		sectionY = y + 66

		// ── Section 4: textBaseline ───────────────────────────────────────
		y = section("textBaseline: top / middle / alphabetic / bottom")

		baseY := y + 28
		ctx.SetStrokeStyle("rgba(166,227,161,0.4)")
		ctx.SetLineWidth(1)
		ctx.BeginPath()
		ctx.MoveTo(24, baseY)
		ctx.LineTo(float32(W-24), baseY)
		ctx.Stroke()

		ctx.SetFont("14px inter")
		x := float32(24)
		for _, bl := range []struct{ name, color string }{
			{"top", "#f38ba8"},
			{"middle", "#fab387"},
			{"alphabetic", "#a6e3a1"},
			{"bottom", "#89b4fa"},
		} {
			ctx.SetTextBaseline(bl.name)
			ctx.SetFillStyle(bl.color)
			ctx.FillText(bl.name, x, baseY)
			x += ctx.MeasureText(bl.name) + 28
		}
		sectionY = baseY + 28

		// ── Section 5: MeasureText word-wrap ─────────────────────────────
		y = section("Word-wrap via MeasureText")

		ctx.SetFont("13px inter")
		ctx.SetFillStyle("#cdd6f4")
		ctx.SetTextBaseline("alphabetic")
		maxW := float32(W - 48)
		paragraph := "Joxel is a canvas-based spreadsheet where cells contain JSON documents rather than simple scalar values. Cell references use dot-path syntax to traverse nested structures, and a formula engine evaluates expressions lazily across the dependency graph."
		line := ""
		lineY := y
		for _, word := range strings.Fields(paragraph) {
			test := line
			if test != "" {
				test += " "
			}
			test += word
			if ctx.MeasureText(test) > maxW && line != "" {
				ctx.FillText(line, 24, lineY)
				line = word
				lineY += 20
			} else {
				line = test
			}
		}
		if line != "" {
			ctx.FillText(line, 24, lineY)
		}

		ctx.EndFrame()
	}
}
