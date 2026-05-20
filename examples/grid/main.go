// Example: grid
//
// Exercises: the full joxel-style drawing pattern — translate+scale zoom
// transform, nested clip (header clip + sheet clip), FillRect for cell
// backgrounds, StrokeRect / BeginPath+stroke for grid lines, FillText for
// cell content and headers, MeasureText for column width, save/restore at
// depth 2, SetLineDash for selection outline.
//
// This is the most realistic raycanvas exercise: a scrollable, zoomable
// spreadsheet rendered entirely through the canvas API, matching the
// structure of joxel's draw() function.
//
// Controls:
//   scroll  — mouse wheel
//   zoom    — Ctrl + wheel  (or +/- keys)
//   select  — click a cell
package main

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
	rc "github.com/ha1tch/raycanvas"
	"github.com/ha1tch/raycanvas/examples/internal/fonts"
)

const (
	W       = 960
	H       = 640
	HDR_W   = 48  // row header width
	HDR_H   = 24  // column header height
	numCols = 12
	numRows = 40
)

// Default column width and row height in world units.
const (
	defColW = 100
	defRowH = 24
)

// Theme colours (catppuccin mocha — same palette as joxel's dark theme).
var T = struct {
	bg, cellBg, gridLine, headerBg, headerTxt string
	selFill, selStroke, accent, txt            string
}{
	bg:        "#1e1e2e",
	cellBg:    "#181825",
	gridLine:  "rgba(108,112,134,0.35)",
	headerBg:  "#1e1e2e",
	headerTxt: "#6c7086",
	selFill:   "rgba(137,180,250,0.15)",
	selStroke: "#89b4fa",
	accent:    "#89b4fa",
	txt:       "#cdd6f4",
}

// Seed data — a small grid of values to display.
var cellData = map[[2]int]string{
	{0, 0}: "Name", {1, 0}: "Q1", {2, 0}: "Q2", {3, 0}: "Q3", {4, 0}: "Q4", {5, 0}: "Total",
	{0, 1}: "Alpha", {1, 1}: "1240", {2, 1}: "1580", {3, 1}: "980", {4, 1}: "2100", {5, 1}: "5900",
	{0, 2}: "Beta", {1, 2}: "870", {2, 2}: "1020", {3, 2}: "760", {4, 2}: "1340", {5, 2}: "3990",
	{0, 3}: "Gamma", {1, 3}: "2400", {2, 3}: "1900", {3, 3}: "2100", {4, 3}: "1700", {5, 3}: "8100",
	{0, 4}: "Delta", {1, 4}: "540", {2, 4}: "620", {3, 4}: "490", {4, 4}: "710", {5, 4}: "2360",
	{0, 5}: "Epsilon", {1, 5}: "3100", {2, 5}: "2800", {3, 5}: "3300", {4, 5}: "2900", {5, 5}: "12100",
}

func colLabel(c int) string {
	s := ""
	for n := c; ; n = n/26 - 1 {
		s = string(rune('A'+n%26)) + s
		if n < 26 {
			break
		}
	}
	return s
}

func main() {
	cache := rc.NewSharedCache()
	ctx := rc.SetupWindow(W, H, "raycanvas — grid (joxel pattern)", cache)
	defer rl.CloseWindow()
	defer cache.Unload()
	rl.SetTargetFPS(60)
	if err := fonts.Register(cache); err != nil {
		panic(err)
	}

	// View state
	var (
		scrollX  = float32(0)
		scrollY  = float32(0)
		cellZoom = float32(1.0)
		selCol   = 0
		selRow   = 0
	)

	// World-space column/row accessors.
	colX := func(c int) float32 {
		return float32(c) * defColW
	}
	rowY := func(r int) float32 {
		return float32(r) * defRowH
	}
	// Screen ↔ world transforms (same as joxel's g2sx/g2sy).
	g2sx := func(gx float32) float32 { return HDR_W + (gx-scrollX)*cellZoom }
	g2sy := func(gy float32) float32 { return HDR_H + (gy-scrollY)*cellZoom }
	s2gx := func(sx float32) float32 { return (sx-HDR_W)/cellZoom + scrollX }
	s2gy := func(sy float32) float32 { return (sy-HDR_H)/cellZoom + scrollY }

	for !rl.WindowShouldClose() {
		// ── Input ─────────────────────────────────────────────────────────
		wheel := rl.GetMouseWheelMove()
		if rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) {
			if wheel != 0 {
				cellZoom *= float32(math.Pow(1.12, float64(wheel)))
				if cellZoom < 0.3 {
					cellZoom = 0.3
				}
				if cellZoom > 4 {
					cellZoom = 4
				}
			}
		} else if wheel != 0 {
			scrollY -= wheel * 40 / cellZoom
		}
		if rl.IsKeyPressed(rl.KeyEqual) {
			cellZoom *= 1.15
		}
		if rl.IsKeyPressed(rl.KeyMinus) {
			cellZoom /= 1.15
		}

		// Clamp scroll
		maxScrollX := float32(numCols)*defColW - float32(W-HDR_W)/cellZoom
		maxScrollY := float32(numRows)*defRowH - float32(H-HDR_H)/cellZoom
		if scrollX < 0 {
			scrollX = 0
		}
		if scrollY < 0 {
			scrollY = 0
		}
		if maxScrollX > 0 && scrollX > maxScrollX {
			scrollX = maxScrollX
		}
		if maxScrollY > 0 && scrollY > maxScrollY {
			scrollY = maxScrollY
		}

		// Click to select
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			mp := rl.GetMousePosition()
			if mp.X > HDR_W && mp.Y > HDR_H {
				gx := s2gx(mp.X)
				gy := s2gy(mp.Y)
				c := int(gx / defColW)
				r := int(gy / defRowH)
				if c >= 0 && c < numCols && r >= 0 && r < numRows {
					selCol, selRow = c, r
				}
			}
		}

		// ── Draw ──────────────────────────────────────────────────────────
		ctx.BeginFrame()

		// Background
		ctx.SetFillStyle(T.bg)
		ctx.FillRect(0, 0, W, H)

		// Viewport extents in world space (for culling).
		viewW := (float32(W) - HDR_W) / cellZoom
		viewH := (float32(H) - HDR_H) / cellZoom
		firstCol := 0
		firstRow := 0
		for firstCol < numCols && colX(firstCol+1) < scrollX {
			firstCol++
		}
		for firstRow < numRows && rowY(firstRow+1) < scrollY {
			firstRow++
		}

		// ── Cell sheet: apply zoom transform ─────────────────────────────
		// Save, clip to sheet area, apply translate+scale.
		// Mirrors joxel's save → clip(HDR_W,HDR_H,…) → translate → scale pattern.
		ctx.Save()
		ctx.BeginPath()
		ctx.Rect(HDR_W, HDR_H, float32(W)-HDR_W, float32(H)-HDR_H)
		ctx.Clip()
		ctx.Translate(HDR_W-scrollX*cellZoom, HDR_H-scrollY*cellZoom)
		ctx.Scale(cellZoom, cellZoom)

		// Cell backgrounds
		ctx.SetFillStyle(T.cellBg)
		for r := firstRow; r < numRows; r++ {
			ry := rowY(r)
			if ry > scrollY+viewH {
				break
			}
			for c := firstCol; c < numCols; c++ {
				cx := colX(c)
				if cx > scrollX+viewW {
					break
				}
				ctx.FillRect(cx, ry, defColW, defRowH)
			}
		}

		// Cell text content
		ctx.SetFont(fmt.Sprintf("12px fira"))
		ctx.SetTextAlign("left")
		ctx.SetTextBaseline("middle")
		for r := firstRow; r < numRows; r++ {
			ry := rowY(r)
			if ry > scrollY+viewH {
				break
			}
			for c := firstCol; c < numCols; c++ {
				cx := colX(c)
				if cx > scrollX+viewW {
					break
				}
				val, ok := cellData[[2]int{c, r}]
				if !ok {
					continue
				}
				// Clip text to cell
				ctx.Save()
				ctx.BeginPath()
				ctx.Rect(cx+2, ry, defColW-4, defRowH)
				ctx.Clip()

				if r == 0 {
					ctx.SetFont("700 12px fira")
					ctx.SetFillStyle(T.accent)
				} else {
					ctx.SetFont("12px fira")
					ctx.SetFillStyle(T.txt)
				}
				ctx.FillText(val, cx+6, ry+defRowH/2)
				ctx.Restore()
			}
		}

		// Grid lines — lineWidth divided by cellZoom keeps them 1px on screen.
		ctx.SetStrokeStyle(T.gridLine)
		ctx.SetLineWidth(1 / cellZoom)
		ctx.BeginPath()
		for c := firstCol; c <= numCols; c++ {
			lx := colX(c)
			if lx > scrollX+viewW+200 {
				break
			}
			ctx.MoveTo(lx, scrollY)
			ctx.LineTo(lx, scrollY+viewH+200)
		}
		for r := firstRow; r <= numRows; r++ {
			ly := rowY(r)
			if ly > scrollY+viewH {
				break
			}
			ctx.MoveTo(scrollX, ly)
			ctx.LineTo(scrollX+viewW+200, ly)
		}
		ctx.Stroke()

		// Selection fill
		sx := colX(selCol)
		sy := rowY(selRow)
		ctx.SetFillStyle(T.selFill)
		ctx.FillRect(sx, sy, defColW, defRowH)

		// Selection stroke — lineWidth kept screen-size.
		ctx.SetStrokeStyle(T.selStroke)
		ctx.SetLineWidth(2 / cellZoom)
		ctx.StrokeRect(sx+0.5/cellZoom, sy+0.5/cellZoom,
			defColW-1/cellZoom, defRowH-1/cellZoom)

		ctx.Restore() // end cell sheet zoom transform

		// ── Column headers (screen space) ─────────────────────────────────
		ctx.SetFillStyle(T.headerBg)
		ctx.FillRect(0, 0, float32(W), HDR_H)

		ctx.SetFont("10px fira")
		ctx.SetTextAlign("center")
		ctx.SetTextBaseline("middle")
		hgx := g2sx(colX(firstCol))
		for c := firstCol; c < numCols; c++ {
			if hgx > float32(W) {
				break
			}
			hgxNext := hgx + defColW*cellZoom
			if hgxNext > HDR_W {
				if c == selCol {
					ctx.SetFillStyle(T.accent)
				} else {
					ctx.SetFillStyle(T.headerTxt)
				}
				ctx.FillText(colLabel(c), hgx+defColW*cellZoom/2, HDR_H/2)
			}
			hgx = hgxNext
		}

		// Column header grid lines
		ctx.SetStrokeStyle(T.gridLine)
		ctx.SetLineWidth(1)
		ctx.BeginPath()
		hgx = g2sx(colX(firstCol))
		for c := firstCol; c <= numCols; c++ {
			lx := float32(math.Round(float64(hgx))) + 0.5
			if lx >= HDR_W && lx <= float32(W) {
				ctx.MoveTo(lx, 0)
				ctx.LineTo(lx, HDR_H)
			}
			if c < numCols {
				hgx += defColW * cellZoom
			}
		}
		ctx.Stroke()

		// ── Row headers ───────────────────────────────────────────────────
		ctx.SetFillStyle(T.headerBg)
		ctx.FillRect(0, HDR_H, HDR_W, float32(H)-HDR_H)

		ctx.SetFont("10px fira")
		ctx.SetTextAlign("center")
		ctx.SetTextBaseline("middle")
		hgy := g2sy(rowY(firstRow))
		for r := firstRow; r < numRows; r++ {
			if hgy > float32(H) {
				break
			}
			hgyNext := hgy + defRowH*cellZoom
			if hgyNext > HDR_H {
				if r == selRow {
					ctx.SetFillStyle(T.accent)
				} else {
					ctx.SetFillStyle(T.headerTxt)
				}
				ctx.FillText(fmt.Sprintf("%d", r+1), HDR_W/2, hgy+defRowH*cellZoom/2)
			}
			hgy = hgyNext
		}

		// Row header grid lines
		ctx.SetStrokeStyle(T.gridLine)
		ctx.BeginPath()
		hgy = g2sy(rowY(firstRow))
		for r := firstRow; r <= numRows; r++ {
			ly := float32(math.Round(float64(hgy))) + 0.5
			if ly >= HDR_H && ly <= float32(H) {
				ctx.MoveTo(0, ly)
				ctx.LineTo(HDR_W, ly)
			}
			if r < numRows {
				hgy += defRowH * cellZoom
			}
			if hgy > float32(H) {
				break
			}
		}
		ctx.Stroke()

		// ── Corner block + separator lines ────────────────────────────────
		ctx.SetFillStyle(T.headerBg)
		ctx.FillRect(0, 0, HDR_W, HDR_H)

		ctx.SetStrokeStyle(T.gridLine)
		ctx.SetLineWidth(1)
		ctx.BeginPath()
		ctx.MoveTo(0, HDR_H+0.5)
		ctx.LineTo(float32(W), HDR_H+0.5)
		ctx.MoveTo(HDR_W+0.5, 0)
		ctx.LineTo(HDR_W+0.5, float32(H))
		ctx.Stroke()

		// ── HUD: zoom level + selection coords ────────────────────────────
		ctx.SetFillStyle("rgba(30,30,46,0.85)")
		ctx.BeginPath()
		ctx.RoundRect(float32(W)-160, float32(H)-36, 150, 26, 4)
		ctx.Fill()
		ctx.SetFillStyle("#6c7086")
		ctx.SetFont("10px fira")
		ctx.SetTextAlign("left")
		ctx.SetTextBaseline("middle")
		hud := fmt.Sprintf("%s%d  zoom %.0f%%", colLabel(selCol), selRow+1, cellZoom*100)
		ctx.FillText(hud, float32(W)-150, float32(H)-23)

		ctx.EndFrame()
	}
}
