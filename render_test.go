package raycanvas

import (
	"image/color"
	"math"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestBezierStartsWhereThePathWas(t *testing.T) {
	c := NewContext(100, 100, NewSharedCache())
	c.BeginPath()
	c.MoveTo(10, 50)
	c.BezierCurveTo(10, 10, 50, 10, 50, 50)
	c.LineTo(90, 50)
	sub := c.tessellate()
	if len(sub) != 1 {
		t.Fatalf("got %d sub-paths, want 1", len(sub))
	}
	pts := sub[0]
	// The point after the MoveTo is the first point of the curve.
	if d := math.Hypot(float64(pts[1].X-10), float64(pts[1].Y-50)); d > 6 {
		t.Errorf("curve starts at %v, %.1f px from (10,50) where the path was", pts[1], d)
	}
	for i, p := range pts {
		if p.X > 50.5 && i < len(pts)-1 {
			t.Errorf("point %d = %v lies beyond the curve's end before the final LineTo", i, p)
			break
		}
	}
}

func TestRoundClipKeepsContentInsideTheCorner(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(100, 100, NewSharedCache())
		defer c.Unload()
		c.BeginFrame()
		rl.ClearBackground(rl.Color(navy))
		c.SetMaskBackground("#204060")
		c.Save()
		c.BeginPath()
		c.RoundRect(10, 10, 80, 80, 20)
		c.Clip()
		c.SetFillStyle("#ffffff")
		c.FillRect(0, 0, 100, 100)
		c.Restore()
		c.EndFrame()
		img := snapshot(c)
		wantPixel(t, img, 50, 50, white, 2, "centre")
		wantPixel(t, img, 5, 5, navy, 2, "outside the box")
		wantPixel(t, img, 11, 11, navy, 2, "inside the box, outside the round corner")
		// (18,18) is 17 px from the corner centre (30,30): inside the radius.
		wantPixel(t, img, 18, 18, white, 2, "inside the round corner")
		wantPixel(t, img, 82, 82, white, 2, "inside the opposite round corner")
	})
}

func TestSaveInsideRoundClipDoesNotCloseIt(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(100, 100, NewSharedCache())
		defer c.Unload()
		c.BeginFrame()
		rl.ClearBackground(rl.Color(navy))
		c.SetMaskBackground("#204060")
		c.Save()
		c.BeginPath()
		c.RoundRect(10, 10, 80, 80, 20)
		c.Clip()
		c.Save() // an ordinary nested save, as any widget does
		c.Translate(40, 40)
		c.Restore()
		c.SetFillStyle("#ffffff")
		c.FillRect(0, 0, 100, 100)
		c.Restore()
		c.EndFrame()
		img := snapshot(c)
		wantPixel(t, img, 11, 11, navy, 2, "corner after content drawn following a nested restore")
		wantPixel(t, img, 51, 51, white, 2, "nested restore must not paint a corner cap mid-content")
	})
}

func TestDrawImageHonoursGlobalAlpha(t *testing.T) {
	onGL(t, func() {
		cache := NewSharedCache()
		src := NewOffscreen(20, 20, cache)
		defer src.Unload()
		src.BeginFrame()
		rl.ClearBackground(rl.White)
		src.EndFrame()

		c := NewOffscreen(60, 60, cache)
		defer c.Unload()
		c.BeginFrame()
		rl.ClearBackground(rl.Black)
		c.SetGlobalAlpha(0.5)
		c.DrawImage(src.Texture(), 10, 10)
		c.EndFrame()
		got := snapshot(c).RGBAAt(20, 20)
		if got.R < 110 || got.R > 145 {
			t.Errorf("white image at globalAlpha 0.5 over black gave %v, want mid grey", got)
		}
	})
}

func TestDashedStrokeSurvivesRepeatedPoint(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(100, 100, NewSharedCache())
		defer c.Unload()
		c.BeginFrame()
		rl.ClearBackground(rl.Black)
		c.SetStrokeStyle("#ffffff")
		c.SetLineWidth(3)
		c.SetLineDash([]float32{200})
		c.BeginPath()
		c.MoveTo(10, 50)
		c.LineTo(40, 50)
		c.LineTo(40, 50) // the same point twice: a zero-length segment
		c.LineTo(90, 50)
		c.Stroke()
		c.EndFrame()
		img := snapshot(c)
		wantPixel(t, img, 25, 50, white, 40, "before the repeated point")
		wantPixel(t, img, 70, 50, white, 40, "after the repeated point")
	})
}

func TestConcaveFillLeavesTheNotchEmpty(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(100, 100, NewSharedCache())
		defer c.Unload()
		c.BeginFrame()
		rl.ClearBackground(rl.Black)
		c.SetFillStyle("#ffffff")
		// A U: the middle of the top is outside the shape, and the average
		// of its corners lies in that gap.
		c.BeginPath()
		c.MoveTo(10, 10)
		c.LineTo(30, 10)
		c.LineTo(30, 70)
		c.LineTo(70, 70)
		c.LineTo(70, 10)
		c.LineTo(90, 10)
		c.LineTo(90, 90)
		c.LineTo(10, 90)
		c.ClosePath()
		c.Fill()
		c.EndFrame()
		img := snapshot(c)
		wantPixel(t, img, 20, 40, white, 2, "left arm")
		wantPixel(t, img, 80, 40, white, 2, "right arm")
		wantPixel(t, img, 50, 80, white, 2, "base")
		wantPixel(t, img, 50, 40, black, 2, "the gap")
		wantPixel(t, img, 50, 5, black, 2, "above the shape")
	})
}

func TestTwoShapesInOnePathAreNotJoined(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(100, 100, NewSharedCache())
		defer c.Unload()
		c.BeginFrame()
		rl.ClearBackground(rl.Black)
		c.SetFillStyle("#ffffff")
		c.BeginPath()
		c.MoveTo(10, 10)
		c.LineTo(30, 10)
		c.LineTo(30, 30)
		c.LineTo(10, 30)
		c.ClosePath()
		c.MoveTo(70, 70)
		c.LineTo(90, 70)
		c.LineTo(90, 90)
		c.LineTo(70, 90)
		c.ClosePath()
		c.Fill()
		c.EndFrame()
		img := snapshot(c)
		wantPixel(t, img, 20, 20, white, 2, "first square")
		wantPixel(t, img, 80, 80, white, 2, "second square")
		wantPixel(t, img, 50, 50, black, 2, "between the squares")
	})
}

func TestFillsKeepTheirShapes(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(100, 100, NewSharedCache())
		defer c.Unload()
		c.BeginFrame()
		rl.ClearBackground(rl.Black)
		c.SetFillStyle("#ffffff")
		// A half disc: an arc closed by its chord, both ways round.
		c.BeginPath()
		c.Arc(30, 50, 20, 0, math.Pi, false) // lower half
		c.ClosePath()
		c.Fill()
		c.BeginPath()
		c.Arc(70, 50, 20, 0, math.Pi, true) // upper half
		c.ClosePath()
		c.Fill()
		c.EndFrame()
		img := snapshot(c)
		wantPixel(t, img, 30, 60, white, 2, "lower half disc")
		wantPixel(t, img, 30, 40, black, 2, "above the lower half disc")
		wantPixel(t, img, 70, 40, white, 2, "upper half disc")
		wantPixel(t, img, 70, 60, black, 2, "below the upper half disc")
	})
}

// checker is a 2x2 pattern of 10 px squares, used as a background that a
// solid-colour trick cannot imitate.
func checker(c *Context) {
	for y := 0; y < int(c.Height()); y += 10 {
		for x := 0; x < int(c.Width()); x += 10 {
			col := rl.Color{R: 200, A: 255}
			if (x/10+y/10)%2 == 0 {
				col = rl.Color{B: 200, A: 255}
			}
			rl.DrawRectangle(int32(x), int32(y), 10, 10, col)
		}
	}
}

func checkerAt(x, y int) color.RGBA {
	if (x/10+y/10)%2 == 0 {
		return color.RGBA{B: 200, A: 255}
	}
	return color.RGBA{R: 200, A: 255}
}

func TestRoundedImageLeavesWhatIsBehindItsCorners(t *testing.T) {
	for _, aa := range []bool{false, true} {
		onGL(t, func() {
			cache := NewSharedCache()
			src := NewOffscreen(40, 40, cache)
			defer src.Unload()
			src.BeginFrame()
			rl.ClearBackground(rl.White)
			rl.DrawRectangle(0, 0, 40, 20, rl.Green) // top half green: shows which way up
			src.EndFrame()

			c := NewOffscreen(100, 100, cache)
			defer c.Unload()
			c.SetAntialias(aa)
			c.BeginFrame()
			checker(c)
			c.DrawImageOffscreenRounded(src, rl.Rectangle{X: 10, Y: 10, Width: 80, Height: 80}, 20)
			c.EndFrame()
			img := snapshot(c)
			green := color.RGBA(rl.Green)
			wantPixel(t, img, 50, 30, green, 3, "top half of the image")
			wantPixel(t, img, 50, 70, white, 3, "bottom half of the image")
			wantPixel(t, img, 18, 18, green, 3, "inside the round corner")
			wantPixel(t, img, 82, 82, white, 3, "inside the opposite corner")
			for _, p := range [][2]int{{11, 11}, {88, 11}, {11, 88}, {88, 88}, {5, 50}} {
				wantPixel(t, img, p[0], p[1], checkerAt(p[0], p[1]), 3, "background kept outside the corner")
			}
		})
	}
}

func TestSoftEdgeBlendsOnlyAtTheRim(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(100, 100, NewSharedCache())
		defer c.Unload()
		c.SetAntialias(true)
		c.BeginFrame()
		rl.ClearBackground(rl.Black)
		c.SetFillStyle("#ffffff")
		c.FillRoundRect(10, 10, 80, 80, 30)
		c.EndFrame()
		img := snapshot(c)
		wantPixel(t, img, 50, 50, white, 0, "centre")
		wantPixel(t, img, 50, 11, white, 0, "one pixel inside the straight edge")
		wantPixel(t, img, 50, 8, black, 0, "outside the straight edge")
		// Along the corner's diagonal there must be partly covered pixels,
		// and they must lie within a pixel or so of the radius.
		partial := 0
		for i := 10; i < 40; i++ {
			v := img.RGBAAt(i, i).R
			d := math.Hypot(float64(i)+0.5-40, float64(i)+0.5-40) - 30
			if v > 0 && v < 255 {
				partial++
				if math.Abs(d) > 1.5 {
					t.Errorf("partly covered pixel (%d,%d)=%d lies %.2f px from the rim", i, i, v, d)
				}
			} else if v == 255 && d > 1 {
				t.Errorf("full pixel (%d,%d) outside the rim by %.2f px", i, i, d)
			} else if v == 0 && d < -1 {
				t.Errorf("empty pixel (%d,%d) inside the rim by %.2f px", i, i, -d)
			}
		}
		if partial == 0 {
			t.Error("no partly covered pixel on the corner: the edge is not soft")
		}
	})
}

func TestRoundRectMeshMatchesRaylibWithoutSoftEdge(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(120, 120, NewSharedCache())
		defer c.Unload()
		c.BeginFrame()
		rl.ClearBackground(rl.Black)
		c.fillRoundMesh(rl.Rectangle{X: 10, Y: 10, Width: 100, Height: 100}, 25, meshPaint{col: white})
		c.EndFrame()
		img := snapshot(c)
		bad := 0
		for y := 0; y < 120; y++ {
			for x := 0; x < 120; x++ {
				// distance outside the ideal rounded rectangle, at the pixel centre
				px, py := float64(x)+0.5, float64(y)+0.5
				dx := math.Max(math.Max(35-px, px-85), 0)
				dy := math.Max(math.Max(35-py, py-85), 0)
				d := math.Hypot(dx, dy) - 25
				if dx == 0 && dy == 0 {
					d = math.Max(math.Max(10-px, px-110), math.Max(10-py, py-110))
				} else if dx == 0 || dy == 0 {
					d = math.Max(dx, dy) - 25
				}
				on := img.RGBAAt(x, y).R > 127
				if (on && d > 1) || (!on && d < -1) {
					bad++
				}
			}
		}
		if bad > 0 {
			t.Errorf("%d pixels more than a pixel on the wrong side of the outline", bad)
		}
	})
}

func TestSoftRoundStrokeIsCentredOnTheOutline(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(100, 100, NewSharedCache())
		defer c.Unload()
		c.SetAntialias(true)
		c.BeginFrame()
		rl.ClearBackground(rl.Black)
		c.SetStrokeStyle("#ffffff")
		c.SetLineWidth(4)
		c.StrokeRoundRect(20, 20, 60, 60, 15)
		c.EndFrame()
		img := snapshot(c)
		// A 4 px line centred on the outline covers 18..22 and 78..82.
		for _, p := range [][2]int{{50, 19}, {50, 20}, {19, 50}, {20, 50}, {79, 50}, {80, 50}, {50, 79}, {50, 80}} {
			wantPixel(t, img, p[0], p[1], white, 2, "on the line")
		}
		for _, p := range [][2]int{{50, 50}, {50, 16}, {50, 24}, {16, 50}, {24, 50}, {84, 50}, {50, 84}, {21, 21}} {
			wantPixel(t, img, p[0], p[1], black, 2, "off the line")
		}
	})
}

func TestPlainRoundStrokeIsARing(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(100, 100, NewSharedCache())
		defer c.Unload()
		c.BeginFrame()
		rl.ClearBackground(rl.Black)
		c.SetStrokeStyle("#ffffff")
		c.SetLineWidth(4)
		c.StrokeRoundRect(20, 20, 60, 60, 15)
		c.EndFrame()
		img := snapshot(c)
		// Without the soft edge raylib draws the line, and puts it outside
		// the rectangle rather than centred on it: 16..20 and 80..84.
		for _, p := range [][2]int{{50, 18}, {18, 50}, {82, 50}, {50, 82}} {
			wantPixel(t, img, p[0], p[1], white, 2, "on the line")
		}
		for _, p := range [][2]int{{50, 50}, {50, 12}, {50, 24}, {12, 50}, {88, 50}} {
			wantPixel(t, img, p[0], p[1], black, 2, "off the line")
		}
	})
}

func TestArcToRoundsTheCorner(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(100, 100, NewSharedCache())
		defer c.Unload()
		c.BeginFrame()
		rl.ClearBackground(rl.Black)
		c.SetStrokeStyle("#ffffff")
		c.SetLineWidth(3)
		c.StrokeRoundRectTop(20, 20, 60, 60, 20)
		c.EndFrame()
		img := snapshot(c)
		wantPixel(t, img, 20, 60, white, 60, "left side")
		wantPixel(t, img, 50, 20, white, 60, "top")
		wantPixel(t, img, 80, 60, white, 60, "right side")
		// the corner arc passes 20 px from its centre (40,40), at 45 degrees
		wantPixel(t, img, 26, 26, white, 90, "on the corner arc")
		wantPixel(t, img, 21, 21, black, 2, "outside the corner arc")
		wantPixel(t, img, 50, 50, black, 2, "inside the shape")
		wantPixel(t, img, 10, 30, black, 2, "left of the shape")
		wantPixel(t, img, 50, 8, black, 2, "above the shape")
	})
}

func TestRoundClipTakesItsColourWhenOpened(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(100, 100, NewSharedCache())
		defer c.Unload()
		c.BeginFrame()
		rl.ClearBackground(rl.Color(navy))
		c.SetFillStyle("#204060") // the card's background, as the docs suggest
		c.Save()
		c.BeginPath()
		c.RoundRect(10, 10, 80, 80, 20)
		c.Clip()
		c.SetFillStyle("#ffffff") // the content's own colour
		c.FillRect(0, 0, 100, 100)
		c.Translate(30, 30) // and its own transform
		c.Restore()
		c.EndFrame()
		img := snapshot(c)
		wantPixel(t, img, 11, 11, navy, 2, "corner painted in the colour set before Clip")
		wantPixel(t, img, 88, 88, navy, 2, "corner painted where the clip was opened")
		wantPixel(t, img, 18, 18, white, 2, "inside the round corner")
		wantPixel(t, img, 50, 50, white, 2, "centre")
	})
}

func TestNestedRoundClipsEachCloseOnce(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(120, 120, NewSharedCache())
		defer c.Unload()
		c.BeginFrame()
		rl.ClearBackground(rl.Color(navy))
		c.SetMaskBackground("#204060")
		c.Save()
		c.BeginPath()
		c.RoundRect(10, 10, 100, 100, 24)
		c.Clip()
		c.SetFillStyle("#ffffff")
		c.FillRect(0, 0, 120, 120)

		c.SetMaskBackground("#ffffff")
		c.Save()
		c.BeginPath()
		c.RoundRect(40, 40, 40, 40, 12)
		c.Clip()
		c.SetFillStyle("#000000")
		c.FillRect(0, 0, 120, 120)
		c.Restore()

		if n := len(c.state.clip.rounds); n != 1 {
			t.Errorf("after the inner Restore %d rounded clips are open, want 1", n)
		}
		c.Restore()
		if n := len(c.state.clip.rounds); n != 0 {
			t.Errorf("after the outer Restore %d rounded clips are open, want 0", n)
		}
		c.EndFrame()
		img := snapshot(c)
		wantPixel(t, img, 60, 60, black, 2, "inner content")
		wantPixel(t, img, 41, 41, white, 2, "inner corner painted in the inner background")
		wantPixel(t, img, 30, 60, white, 2, "outer content")
		wantPixel(t, img, 11, 11, navy, 2, "outer corner painted in the outer background")
		wantPixel(t, img, 108, 108, navy, 2, "outer corner, bottom right")
		wantPixel(t, img, 22, 22, white, 2, "inside the outer round corner")
	})
}

func TestEvictedShadowTexturesAreFreed(t *testing.T) {
	onGL(t, func() {
		cache := NewSharedCache()
		defer cache.Unload()
		c := NewOffscreen(64, 64, cache)
		defer c.Unload()
		c.BeginFrame()
		c.SetShadowColor("rgba(0,0,0,0.5)")
		c.SetShadowBlur(2)
		const extra = 8
		for i := 0; i < ShadowCacheMax+extra; i++ {
			c.FillRect(4, 4, float32(4+i%40), float32(4+i/40))
		}
		c.EndFrame()
		if n := len(cache.shadows); n != ShadowCacheMax {
			t.Errorf("cache holds %d shadows, want %d", n, ShadowCacheMax)
		}
		if cache.freed != 0 || len(cache.retired) != extra {
			t.Errorf("during the frame: freed %d, retired %d; want 0 and %d", cache.freed, len(cache.retired), extra)
		}
		c.BeginFrame()
		c.EndFrame()
		if cache.freed != extra || len(cache.retired) != 0 {
			t.Errorf("after the next frame began: freed %d, retired %d; want %d and 0", cache.freed, len(cache.retired), extra)
		}
	})
}

func TestColourSettersAndAlpha(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(60, 60, NewSharedCache())
		defer c.Unload()
		c.SetAntialias(true)
		c.BeginFrame()
		rl.ClearBackground(rl.Black)
		c.SetFillColor(color.RGBA{R: 255, G: 255, B: 255, A: 128})
		c.FillRoundRect(5, 5, 50, 50, 10)
		c.SetStrokeColor(color.RGBA{G: 255, A: 255})
		c.SetLineWidth(2)
		c.SetGlobalAlpha(0.5)
		c.StrokeRoundRect(5, 5, 50, 50, 10)
		c.EndFrame()
		img := snapshot(c)
		if got := img.RGBAAt(30, 30); got.R < 120 || got.R > 136 {
			t.Errorf("half-transparent white over black gave %v, want mid grey", got)
		}
		if got := img.RGBAAt(30, 5); got.G < 100 || got.G < got.R {
			t.Errorf("half-alpha green line over grey gave %v, want green to lead", got)
		}
	})
}

func TestGradientFillRunsFromTopToBottom(t *testing.T) {
	for _, aa := range []bool{false, true} {
		onGL(t, func() {
			c := NewOffscreen(100, 120, NewSharedCache())
			defer c.Unload()
			c.SetAntialias(aa)
			c.BeginFrame()
			rl.ClearBackground(rl.Black)
			// White, half see-through at the top and solid at the bottom,
			// over black: the grey level is the alpha.
			c.FillRoundRectGradient(10, 10, 80, 100, 12,
				color.RGBA{R: 255, G: 255, B: 255, A: 128}, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			c.EndFrame()
			img := snapshot(c)
			for _, p := range []struct{ y, want int }{{12, 131}, {35, 160}, {60, 192}, {85, 224}, {107, 252}} {
				for _, x := range []int{14, 50, 86} {
					if x != 50 && (p.y < 22 || p.y > 98) {
						continue // the corners
					}
					if got := int(img.RGBAAt(x, p.y).R); got < p.want-4 || got > p.want+4 {
						t.Errorf("soft edge %v: grey at (%d,%d) = %d, want about %d", aa, x, p.y, got, p.want)
					}
				}
			}
			wantPixel(t, img, 11, 11, black, 2, "outside the round corner")
			wantPixel(t, img, 50, 5, black, 2, "above the rectangle")
			wantPixel(t, img, 50, 115, black, 2, "below the rectangle")
		})
	}
}

func TestGradientStrokeRunsFromTopToBottom(t *testing.T) {
	onGL(t, func() {
		c := NewOffscreen(100, 120, NewSharedCache())
		defer c.Unload()
		c.SetAntialias(true)
		c.BeginFrame()
		rl.ClearBackground(rl.Black)
		c.SetLineWidth(4)
		c.SetGlobalAlpha(0.5)
		c.StrokeRoundRectGradient(10, 10, 80, 100, 12,
			color.RGBA{R: 255, G: 255, B: 255, A: 100}, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		c.EndFrame()
		img := snapshot(c)
		// globalAlpha halves both ends: 50 at the top, 127 at the bottom.
		for _, p := range []struct{ x, y, want int }{{50, 10, 50}, {10, 60, 88}, {89, 60, 88}, {50, 109, 127}} {
			if got := int(img.RGBAAt(p.x, p.y).R); got < p.want-4 || got > p.want+4 {
				t.Errorf("grey at (%d,%d) = %d, want about %d", p.x, p.y, got, p.want)
			}
		}
		wantPixel(t, img, 50, 60, black, 2, "the middle is not filled")
	})
}
