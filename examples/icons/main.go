// Example: icons
//
// Demonstrates the SVG icon pipeline: oksvg parses, rasterx rasterises,
// icons are uploaded as white-on-transparent Texture2D, tinted at draw time.
//
// Section 1 — Shevo icons: the actual SVG icons from joxel/quag/dekk,
//             rendered at 14px and 28px.
//
// Section 2 — Original icons: new icons in the same geometric style,
//             demonstrating a broader vocabulary.
//
// Section 3 — Light / Dark: both sections side by side, showing how
//             a single registered texture produces correct output in
//             any colour scheme by changing only the tint.
package main

import (
	"fmt"
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"
	rc "github.com/ha1tch/raycanvas"
	"github.com/ha1tch/raycanvas/examples/internal/fonts"
)

const (
	W = 1100
	H = 760
)

// ── SVG data ──────────────────────────────────────────────────────────────────
// All icons use currentColor (rendered white, tinted at draw time).
// Shevo icons match shevo-0.2.3 shell.html verbatim.

var svgIcons = map[string][]byte{
	// ── Shevo icons ──────────────────────────────────────────────────────
	"chevron-up": []byte(`<svg width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M2 7.5l4-4 4 4" stroke="white" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`),

	"chevron-down": []byte(`<svg width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M2 4.5l4 4 4-4" stroke="white" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`),

	"align-left": []byte(`<svg width="14" height="14" viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg">
		<line x1="1" y1="3" x2="13" y2="3" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
		<line x1="1" y1="6.5" x2="9" y2="6.5" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
		<line x1="1" y1="10" x2="11" y2="10" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
	</svg>`),

	"align-center": []byte(`<svg width="14" height="14" viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg">
		<line x1="1" y1="3" x2="13" y2="3" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
		<line x1="3" y1="6.5" x2="11" y2="6.5" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
		<line x1="2" y1="10" x2="12" y2="10" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
	</svg>`),

	"align-right": []byte(`<svg width="14" height="14" viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg">
		<line x1="1" y1="3" x2="13" y2="3" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
		<line x1="5" y1="6.5" x2="13" y2="6.5" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
		<line x1="3" y1="10" x2="13" y2="10" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
	</svg>`),

	"upload": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M7.5 10V2M5 4.5L7.5 2 10 4.5" stroke="white" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
		<path d="M2 11v1.5a1 1 0 001 1h9a1 1 0 001-1V11" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
	</svg>`),

	"download": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M7.5 1.5v8M5 7l2.5 3L10 7" stroke="white" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
		<path d="M2 11v1.5a1 1 0 001 1h9a1 1 0 001-1V11" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
	</svg>`),

	"undo": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M2.5 5.5h6a4 4 0 010 8H5" stroke="white" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
		<path d="M5.5 2.5l-3 3 3 3" stroke="white" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`),

	"redo": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M12.5 5.5h-6a4 4 0 000 8H10" stroke="white" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
		<path d="M9.5 2.5l3 3-3 3" stroke="white" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`),

	"anchor": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M2 11.5A1.5 1.5 0 003.5 13h8a1.5 1.5 0 000-3H8l-1-4H6l-1 4H3.5A1.5 1.5 0 002 11.5z" stroke="white" stroke-width="1.3" stroke-linejoin="round"/>
		<path d="M7.5 2v5" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
		<circle cx="7.5" cy="1.5" r="1" fill="white"/>
	</svg>`),

	"layout": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<rect x="1.5" y="2.5" width="5" height="4" rx="0.75" stroke="white" stroke-width="1.3"/>
		<rect x="8.5" y="2.5" width="5" height="4" rx="0.75" stroke="white" stroke-width="1.3"/>
		<rect x="1.5" y="8.5" width="12" height="4" rx="0.75" stroke="white" stroke-width="1.3"/>
		<path d="M5 4.5H10M7.5 3v3" stroke="white" stroke-width="1.2" stroke-linecap="round"/>
	</svg>`),

	"link": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M6.5 8.5a3.5 3.5 0 005 0l2-2a3.536 3.536 0 00-5-5L7.5 2.5" stroke="white" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
		<path d="M8.5 6.5a3.5 3.5 0 00-5 0l-2 2a3.536 3.536 0 005 5L7.5 12.5" stroke="white" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`),

	"globe": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<circle cx="7.5" cy="7.5" r="5.5" stroke="white" stroke-width="1.3"/>
		<path d="M7.5 2C9.5 4 10.5 5.8 10.5 7.5S9.5 11 7.5 13C5.5 11 4.5 9.2 4.5 7.5S5.5 4 7.5 2z" fill="white" opacity="0.35"/>
		<path d="M2 7.5h11" stroke="white" stroke-width="1.1" stroke-linecap="round"/>
	</svg>`),

	"edit": []byte(`<svg width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M8.5 1.5a1.414 1.414 0 012 2L3.5 10.5l-3 .5.5-3 7.5-6.5z" stroke="white" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`),

	// ── Original icons ────────────────────────────────────────────────────
	"pin": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M9.5 1.5l4 4-6 2-3 3-.5-1.5L7.5 7.5l-4-4 2-2zM1.5 13.5l3-3" stroke="white" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
		<path d="M9.5 5.5L5.5 9.5" stroke="white" stroke-width="1.3" stroke-linecap="round"/>
	</svg>`),

	"tag": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M2 2h5.5l5.5 5.5-5 5L2.5 7V2H2z" stroke="white" stroke-width="1.3" stroke-linejoin="round"/>
		<circle cx="5" cy="5" r="1" fill="white"/>
	</svg>`),

	"filter": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M1.5 3.5h12M4 7.5h7M6.5 11.5h2" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
	</svg>`),

	"search": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<circle cx="6.5" cy="6.5" r="4" stroke="white" stroke-width="1.4"/>
		<path d="M10 10l3 3" stroke="white" stroke-width="1.5" stroke-linecap="round"/>
	</svg>`),

	"settings": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<circle cx="7.5" cy="7.5" r="2" stroke="white" stroke-width="1.3"/>
		<path d="M7.5 1.5v1.2M7.5 12.3v1.2M1.5 7.5h1.2M12.3 7.5h1.2M3.4 3.4l.85.85M10.75 10.75l.85.85M10.75 4.25l-.85.85M4.25 10.75l-.85.85" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
	</svg>`),

	"copy": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<rect x="4.5" y="4.5" width="8" height="8" rx="1" stroke="white" stroke-width="1.3"/>
		<path d="M2.5 10.5V3a.5.5 0 01.5-.5h7.5" stroke="white" stroke-width="1.3" stroke-linecap="round"/>
	</svg>`),

	"trash": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M2.5 4.5h10M5.5 4.5V3a.5.5 0 01.5-.5h3a.5.5 0 01.5.5v1.5" stroke="white" stroke-width="1.3" stroke-linecap="round"/>
		<rect x="3.5" y="4.5" width="8" height="8" rx="1" stroke="white" stroke-width="1.3"/>
		<path d="M6 7v3M9 7v3" stroke="white" stroke-width="1.2" stroke-linecap="round"/>
	</svg>`),

	"check": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M2.5 7.5l3.5 3.5 6.5-7" stroke="white" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`),

	"close": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M3 3l9 9M12 3l-9 9" stroke="white" stroke-width="1.4" stroke-linecap="round"/>
	</svg>`),

	"expand": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M9 2h4v4M6 13H2V9M13 6l-4.5 4.5M2 9l4.5-4.5" stroke="white" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`),

	"collapse": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M13 2l-4.5 4.5M2 13l4.5-4.5M8.5 2H13v4.5M6.5 13H2V8.5" stroke="white" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`),

	"star": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M7.5 1.5l1.6 3.9 4.2.4-3.1 2.8.9 4.1-3.6-2.1-3.6 2.1.9-4.1L1.7 5.8l4.2-.4z" stroke="white" stroke-width="1.2" stroke-linejoin="round"/>
	</svg>`),

	"bell": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M7.5 1.5a4 4 0 014 4v3l1 2H2.5l1-2v-3a4 4 0 014-4z" stroke="white" stroke-width="1.3" stroke-linejoin="round"/>
		<path d="M6 11.5a1.5 1.5 0 003 0" stroke="white" stroke-width="1.2"/>
	</svg>`),

	"sort": []byte(`<svg width="15" height="15" viewBox="0 0 15 15" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M4 5l3-3 3 3M10 10l-3 3-3-3" stroke="white" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
		<path d="M7 2v11" stroke="white" stroke-width="1.2" stroke-linecap="round" opacity="0.3"/>
	</svg>`),
}

func main() {
	cache := rc.NewSharedCache()
	ctx := rc.SetupWindow(W, H, "raycanvas — icons", cache)
	defer rl.CloseWindow()
	defer cache.Unload()
	rl.SetTargetFPS(60)
	if err := fonts.Register(cache); err != nil {
		panic(err)
	}

	// Register all icons at two sizes
	for name, data := range svgIcons {
		if err := rc.RegisterIcon(cache, name, data, 14); err != nil {
			panic("register " + name + ": " + err.Error())
		}
		if err := rc.RegisterIcon(cache, name, data, 24); err != nil {
			panic("register " + name + "@24: " + err.Error())
		}
		if err := rc.RegisterIcon(cache, name, data, 40); err != nil {
			panic("register " + name + "@40: " + err.Error())
		}
	}

	// Icon group definitions
	shevoIcons := []string{
		"chevron-up", "chevron-down",
		"align-left", "align-center", "align-right",
		"upload", "download",
		"undo", "redo",
		"anchor", "layout", "link", "globe", "edit",
	}

	originalIcons := []string{
		"search", "filter", "settings",
		"copy", "trash",
		"check", "close",
		"pin", "tag",
		"expand", "collapse",
		"star", "bell", "sort",
	}

	// Dark theme colours (kaputccino)
	darkBg      := "#0f0f17"
	darkPanel   := "#1e1e2e"
	darkBorder  := "#313244"
	darkLabel   := "#6c7086"
	darkAccent  := "#cba6f7"
	darkTxt     := "#cdd6f4"
	darkHover   := "#89b4fa"

	// Light theme colours
	lightBg     := "#f0f0f4"
	lightPanel  := "#ffffff"
	lightBorder := "rgba(0,0,0,0.10)"
	lightLabel  := "#9090a0"
	lightTxt    := "#2d2d2d"


	// Hover tracking
	var hoverIcon string

	for !rl.WindowShouldClose() {
		mp := rl.GetMousePosition()
		mx, my := mp.X, mp.Y
		_ = mx
		_ = my

		ctx.BeginFrame()

		// ── Background ────────────────────────────────────────────────────
		ctx.SetFillStyle("#12121e")
		ctx.FillRect(0, 0, W, H)

		// Section header helper
		sectionHeader := func(x, y float32, title string) {
			ctx.SetFillStyle("#1e1e2e")
			ctx.FillRect(x, y, W-x*2, 22)
			ctx.SetFillStyle("#585b70")
			ctx.SetFont("10px inter")
			ctx.SetTextAlign("left")
			ctx.SetTextBaseline("middle")
			ctx.FillText(title, x+10, y+11)
		}

		// Icon grid helper — renders a row of icons with labels below
		iconRow := func(
			icons []string, x, y, size, spacing float32,
			tint color.RGBA,
			bgCol, borderCol, labelCol, hoverCol string,
		) float32 {
			ix := x
			for _, name := range icons {
				// Hit test for hover
				isHover := mx >= ix && mx <= ix+spacing && my >= y-4 && my <= y+size+20
				if isHover {
					hoverIcon = name
				}

				// Button background
				pad := float32(6)
				bx := ix - pad
				bw := size + pad*2
				bh := size + pad*2 + 14

				if isHover {
					ctx.SetFillStyle(hoverCol)
					ctx.SetGlobalAlpha(0.15)
					ctx.FillRoundRect(bx, y-pad, bw, bh, 5)
					ctx.SetGlobalAlpha(1)
				} else {
					ctx.SetFillStyle(bgCol)
					ctx.SetGlobalAlpha(0.5)
					ctx.FillRoundRect(bx, y-pad, bw, bh, 5)
					ctx.SetGlobalAlpha(1)
				}

				ctx.SetStrokeStyle(borderCol)
				ctx.SetLineWidth(0.5)
				ctx.StrokeRoundRect(bx, y-pad, bw, bh, 5)

				// Icon — tinted with provided colour
				iconTint := tint
				if isHover {
					iconTint = rc.ParseColor(hoverCol)
				}
				ctx.DrawIcon(name, ix, y, size, iconTint)

				// Label
				ctx.SetFont("8px inter")
				ctx.SetTextAlign("center")
				ctx.SetTextBaseline("top")
				if isHover {
					ctx.SetFillStyle(hoverCol)
				} else {
					ctx.SetFillStyle(labelCol)
				}
				ctx.FillText(name, ix+size/2, y+size+4)

				ix += spacing
			}
			return ix
		}

		// ── Section 1: Shevo icons ────────────────────────────────────────
		sectionHeader(10, 10, "shevo icons — from joxel / quag / dekk (14px, 24px)")

		// 14px row
		ctx.SetFillStyle(darkLabel)
		ctx.SetFont("9px inter")
		ctx.SetTextAlign("left")
		ctx.SetTextBaseline("middle")
		ctx.FillText("14px", 14, 52)
		iconRow(shevoIcons, 50, 42, 14, 48,
			rc.ParseColor(darkTxt),
			darkPanel, darkBorder, darkLabel, darkHover)

		// 24px row
		ctx.SetFillStyle(darkLabel)
		ctx.FillText("24px", 14, 108)
		iconRow(shevoIcons, 50, 94, 24, 52,
			rc.ParseColor(darkTxt),
			darkPanel, darkBorder, darkLabel, darkHover)

		// ── Section 2: Original icons ─────────────────────────────────────
		sectionHeader(10, 148, "original icons — same geometric style (14px, 24px)")

		ctx.SetFillStyle(darkLabel)
		ctx.SetFont("9px inter")
		ctx.SetTextAlign("left")
		ctx.SetTextBaseline("middle")
		ctx.FillText("14px", 14, 188)
		iconRow(originalIcons, 50, 178, 14, 52,
			rc.ParseColor(darkAccent),
			darkPanel, darkBorder, darkLabel, darkHover)

		ctx.SetFillStyle(darkLabel)
		ctx.FillText("24px", 14, 244)
		iconRow(originalIcons, 50, 230, 24, 56,
			rc.ParseColor(darkAccent),
			darkPanel, darkBorder, darkLabel, darkHover)

		// ── Section 3: Large icon showcase (40px, both groups) ───────────
		sectionHeader(10, 284, "40px — large icon showcase")

		allIcons := append(shevoIcons, originalIcons...)
		ix := float32(20)
		for _, name := range allIcons {
			size := float32(40)
			ctx.SetFillStyle(darkPanel)
			ctx.FillRoundRect(ix, 300, 52, 68, 6)
			ctx.SetStrokeStyle(darkBorder)
			ctx.SetLineWidth(0.5)
			ctx.StrokeRoundRect(ix, 300, 52, 68, 6)

			// Alternate accent colours
			cols := []string{darkTxt, darkHover, darkAccent, "#a6e3a1", "#f38ba8", "#fab387", "#f9e2af"}
			idx := 0
			for i, n := range allIcons {
				if n == name {
					idx = i
					break
				}
			}
			ctx.DrawIcon(name, ix+6, 308, size, rc.ParseColor(cols[idx%len(cols)]))

			ctx.SetFont("7px inter")
			ctx.SetTextAlign("center")
			ctx.SetTextBaseline("top")
			ctx.SetFillStyle(darkLabel)
			ctx.FillText(name, ix+26, 354)

			ix += 56
			if ix > float32(W-56) {
				break
			}
		}

		// ── Section 4: Light / Dark theme comparison ──────────────────────
		sectionHeader(10, 392, "light ↔ dark — same texture, different tint")

		half := float32(W-20) / 2
		panelY := float32(420)
		panelH := float32(290)

		// Dark panel
		ctx.SetFillStyle(darkBg)
		ctx.FillRoundRect(10, panelY, half-5, panelH, 10)
		ctx.SetStrokeStyle(darkBorder)
		ctx.SetLineWidth(1)
		ctx.StrokeRoundRect(10, panelY, half-5, panelH, 10)

		// Dark panel header bar
		ctx.SetFillStyle(darkPanel)
		ctx.FillRoundRectTop(10, panelY, half-5, 32, 10)
		ctx.SetStrokeStyle(darkBorder)
		ctx.SetLineWidth(0.5)
		ctx.BeginPath()
		ctx.MoveTo(10, panelY+32)
		ctx.LineTo(10+half-5, panelY+32)
		ctx.Stroke()
		ctx.SetFillStyle(darkLabel)
		ctx.SetFont("10px inter")
		ctx.SetTextAlign("left")
		ctx.SetTextBaseline("middle")
		ctx.FillText("Dark — kaputccino", 24, panelY+16)

		// Dark panel: toolbar simulation
		toolbarY := panelY + 48
		toolbarIcons := []string{"undo", "redo", "copy", "trash", "upload", "download", "link", "search", "settings"}
		for i, name := range toolbarIcons {
			ix2 := float32(24 + i*36)
			ctx.DrawIcon(name, ix2, toolbarY, 14, rc.ParseColor(darkTxt))
		}
		// Divider
		ctx.SetStrokeStyle(darkBorder)
		ctx.SetLineWidth(0.5)
		ctx.BeginPath()
		ctx.MoveTo(20, toolbarY+24)
		ctx.LineTo(10+half-15, toolbarY+24)
		ctx.Stroke()

		// Dark panel: button row with icons
		btnY := toolbarY + 36
		for i, cfg := range []struct {
			name, col, label string
		}{
			{"check", "#a6e3a1", "Confirm"},
			{"close", "#f38ba8", "Dismiss"},
			{"link", "#89b4fa", "Copy link"},
			{"star", "#f9e2af", "Favourite"},
			{"trash", "#f38ba8", "Delete"},
		} {
			bx := float32(20 + i*80)

			ctx.SetFillStyle(darkPanel)
			ctx.FillRoundRect(bx, btnY, 72, 28, 5)
			ctx.SetStrokeStyle(cfg.col)
			ctx.SetGlobalAlpha(0.5)
			ctx.SetLineWidth(1)
			ctx.StrokeRoundRect(bx, btnY, 72, 28, 5)
			ctx.SetGlobalAlpha(1)

			ctx.DrawIcon(cfg.name, bx+8, btnY+7, 14, rc.ParseColor(cfg.col))
			ctx.SetFont("10px inter")
			ctx.SetTextAlign("left")
			ctx.SetTextBaseline("middle")
			ctx.SetFillStyle(cfg.col)
			ctx.FillText(cfg.label, bx+28, btnY+14)
		}

		// Dark panel: icon size comparison
		sizeY := btnY + 48
		ctx.SetFillStyle(darkLabel)
		ctx.SetFont("9px inter")
		ctx.SetTextAlign("left")
		ctx.SetTextBaseline("middle")
		ctx.FillText("size comparison:", 20, sizeY-4)
		for i, sz := range []float32{10, 14, 18, 24, 32, 40} {
			ctx.DrawIcon("globe", float32(120+i*50), sizeY-sz/2, sz, rc.ParseColor(darkTxt))
			ctx.SetFillStyle(darkLabel)
			ctx.SetFont("8px inter")
			ctx.SetTextAlign("center")
			ctx.SetTextBaseline("top")
			ctx.FillText(fmt.Sprintf("%.0f", sz), float32(120+i*50)+sz/2, sizeY+sz/2+3)
		}

		// Light panel
		lx := float32(W/2) + 5
		ctx.SetFillStyle(lightBg)
		ctx.FillRoundRect(lx, panelY, half-5, panelH, 10)
		ctx.SetStrokeStyle(lightBorder)
		ctx.SetLineWidth(1)
		ctx.StrokeRoundRect(lx, panelY, half-5, panelH, 10)

		// Light panel header
		ctx.SetFillStyle(lightPanel)
		ctx.FillRoundRectTop(lx, panelY, half-5, 32, 10)
		ctx.SetStrokeStyle(lightBorder)
		ctx.SetLineWidth(0.5)
		ctx.BeginPath()
		ctx.MoveTo(lx, panelY+32)
		ctx.LineTo(lx+half-5, panelY+32)
		ctx.Stroke()
		ctx.SetFillStyle(lightLabel)
		ctx.SetFont("10px inter")
		ctx.SetTextAlign("left")
		ctx.SetTextBaseline("middle")
		ctx.FillText("Light — same icons, different tint", lx+14, panelY+16)

		// Light panel: toolbar
		for i, name := range toolbarIcons {
			ix3 := lx + float32(14+i*36)
			ctx.DrawIcon(name, ix3, toolbarY, 14, rc.ParseColor(lightTxt))
		}
		ctx.SetStrokeStyle(lightBorder)
		ctx.SetLineWidth(0.5)
		ctx.BeginPath()
		ctx.MoveTo(lx+10, toolbarY+24)
		ctx.LineTo(lx+half-15, toolbarY+24)
		ctx.Stroke()

		// Light panel: button row
		for i, cfg := range []struct {
			name, col, label string
		}{
			{"check", "#0a8a2e", "Confirm"},
			{"close", "#d42050", "Dismiss"},
			{"link", "#0055d4", "Copy link"},
			{"star", "#c05000", "Favourite"},
			{"trash", "#d42050", "Delete"},
		} {
			bx := lx + float32(10+i*80)

			ctx.SetFillStyle(lightPanel)
			ctx.FillRoundRect(bx, btnY, 72, 28, 5)
			ctx.SetStrokeStyle(cfg.col)
			ctx.SetGlobalAlpha(0.4)
			ctx.SetLineWidth(1)
			ctx.StrokeRoundRect(bx, btnY, 72, 28, 5)
			ctx.SetGlobalAlpha(1)

			ctx.DrawIcon(cfg.name, bx+8, btnY+7, 14, rc.ParseColor(cfg.col))
			ctx.SetFont("10px inter")
			ctx.SetTextAlign("left")
			ctx.SetTextBaseline("middle")
			ctx.SetFillStyle(cfg.col)
			ctx.FillText(cfg.label, bx+28, btnY+14)
		}

		// Light panel: size comparison
		ctx.SetFillStyle(lightLabel)
		ctx.SetFont("9px inter")
		ctx.SetTextAlign("left")
		ctx.SetTextBaseline("middle")
		ctx.FillText("size comparison:", lx+10, sizeY-4)
		for i, sz := range []float32{10, 14, 18, 24, 32, 40} {
			ctx.DrawIcon("globe", lx+float32(110+i*50), sizeY-sz/2, sz, rc.ParseColor(lightTxt))
			ctx.SetFillStyle(lightLabel)
			ctx.SetFont("8px inter")
			ctx.SetTextAlign("center")
			ctx.SetTextBaseline("top")
			ctx.FillText(fmt.Sprintf("%.0f", sz), lx+float32(110+i*50)+sz/2, sizeY+sz/2+3)
		}

		// ── HUD ───────────────────────────────────────────────────────────
		ctx.SetFillStyle("rgba(15,15,23,0.92)")
		ctx.FillRoundRect(10, H-30, W-20, 20, 4)
		ctx.SetFillStyle("#45475a")
		ctx.SetFont("9px fira")
		ctx.SetTextAlign("left")
		ctx.SetTextBaseline("middle")
		hudText := "icons: oksvg parse → rasterx render → Texture2D (white-on-transparent) → tinted at draw time"
		if hoverIcon != "" {
			hudText += "   |   hover: " + hoverIcon
		}
		ctx.FillText(hudText, 18, H-20)

		ctx.EndFrame()
	}
}
