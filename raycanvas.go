// Package raycanvas provides a 2D drawing API that mirrors the HTML5
// CanvasRenderingContext2D interface, backed by raylib-go for GPU rendering.
//
// It is designed as the rendering substrate for porting Shevo perspectives
// (joxel, quag, dekk) from JavaScript to Go, allowing ported code to use
// familiar canvas idioms (BeginPath, FillRect, Arc, BezierCurveTo, etc.)
// while raylib handles the GPU layer.
//
// # Architecture
//
// Rendering is split into two tiers:
//
//   - Hot path (GPU, every frame): FillRect, StrokeRect, FillText, path
//     fill/stroke, image blit. All dispatched to raylib primitives.
//
//   - Cold path (CPU, cached): Shadow blur, arbitrary-path clip masks,
//     SVG icon rasterisation. Computed once via fogleman/gg and cached
//     as Texture2D values. Never executed inside the frame loop.
//
// # Typical usage
//
//	cache := raycanvas.NewSharedCache()
//	defer cache.Unload()
//
//	// Register fonts (call after rl.InitWindow)
//	firaData, _ := os.ReadFile("FiraCode-Regular.ttf")
//	raycanvas.RegisterFont(cache, "fira", 400, false, firaData, nil)
//
//	// Register icons
//	raycanvas.RegisterIcon(cache, "undo", undoSVG, 14)
//
//	ctx := raycanvas.NewContext(int32(w), int32(h), cache)
//
//	for !rl.WindowShouldClose() {
//	    ctx.BeginFrame()
//	    ctx.SetFillStyle("#1e1e2e")
//	    ctx.FillRect(0, 0, float32(w), float32(h))
//	    // ... draw calls ...
//	    ctx.EndFrame()
//	}
//
// # Type conventions
//
// All geometry parameters use float32. Color parameters use color.RGBA
// (the standard library type, also used natively by raylib-go). CSS color
// strings and font strings are accepted as Go strings and cached after the
// first parse.
//
// See ARCHITECTURE.md for the full design rationale and constraint documentation.
package raycanvas

import rl "github.com/gen2brain/raylib-go/raylib"

// SetupWindow configures raylib flags for best visual quality and then
// initialises the window. Call this instead of rl.InitWindow directly.
//
// Currently enables:
//   - FlagMsaa4xHint: 4× multisample antialiasing for all GPU-rendered
//     shapes (DrawRectangleRounded, DrawCircleV, DrawSplineLinear, etc.).
//     This flag must be set before the OpenGL context is created, so it
//     cannot be applied after InitWindow. SetupWindow handles the ordering.
//
// Required call order — GPU resources cannot be allocated before InitWindow:
//
//	cache := raycanvas.NewSharedCache()
//	ctx   := raycanvas.SetupWindow(800, 600, "My App", cache)  // InitWindow happens here
//	defer rl.CloseWindow()
//	defer cache.Unload()
//	rl.SetTargetFPS(60)
//	raycanvas.RegisterFont(cache, ...)  // safe: window is open
//	raycanvas.RegisterIcon(cache, ...)  // safe: window is open
func SetupWindow(width, height int32, title string, cache *SharedCache) *Context {
	rl.SetConfigFlags(rl.FlagMsaa4xHint)
	rl.InitWindow(width, height, title)
	return NewContext(width, height, cache)
}
