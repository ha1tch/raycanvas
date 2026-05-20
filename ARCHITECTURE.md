# raycanvas — Architecture & Design Reference

**Module:** `github.com/ha1tch/raycanvas`  
**Purpose:** A Go package that mirrors the HTML5 Canvas 2D API, backed by raylib-go for GPU rendering and fogleman/gg for CPU-side precision work. Designed as the rendering substrate for porting Shevo perspectives (joxel, quag, dekk) from JavaScript to Go.

---

## 1. Motivation

Shevo perspectives are written against the HTML5 `CanvasRenderingContext2D` API. Porting them to Go without a canvas-compatible abstraction requires translating every draw call into raw raylib primitives — an impedance mismatch that multiplies friction. raycanvas provides that abstraction: ported code uses `ctx.FillRect(...)`, `ctx.BeginPath()`, `ctx.Arc(...)` etc., and the package handles the raylib mapping transparently.

---

## 2. Dependency Stack

| Package | Role | Used when |
|---|---|---|
| `github.com/gen2brain/raylib-go/raylib` v0.60.0 | GPU rendering, window, input | Hot path — every frame |
| `github.com/fogleman/gg` v1.3.0 | CPU-side precision rendering, alpha masks, shadow pipeline | Cold path — cached to texture |
| `github.com/srwiley/oksvg` | SVG path parsing | Startup only — icon rasterisation |
| `github.com/srwiley/rasterx` | SVG rasterisation backend | Startup only — icon rasterisation |

### Why gg alongside raylib?

raylib's GPU rasteriser is fast but limited: no arbitrary-path clip, no sub-pixel anti-aliasing for fine strokes, no even-odd fill rule. gg (backed by `golang/freetype/raster`) covers these cases precisely. Its output is always `*image.RGBA`, which maps directly to `rl.LoadTextureFromImage` → `Texture2D`. The two tiers connect through texture upload; gg never runs in the frame loop.

---

## 3. Two-Tier Rendering Architecture

### Tier 1 — raylib GPU (hot path, every frame)

- `FillRect`, `StrokeRect` → `rl.DrawRectangleRec`, `rl.DrawRectangleLinesEx`
- `FillText`, `StrokeText` → `rl.DrawTextEx`
- Rectangular clip → `rl.BeginScissorMode` / `rl.EndScissorMode`
- Rounded rect fill/stroke → `rl.DrawRectangleRounded`, `rl.DrawRectangleRoundedLinesEx`
- Full circles → `rl.DrawCircleV` (fill), `rl.DrawCircleLinesV` (stroke)
- Cubic Bézier (inter-perspective links) → `rl.DrawSplineSegmentBezierCubic`
- Image/texture blit → `rl.DrawTexturePro`
- Off-screen render → `rl.BeginTextureMode` / `rl.EndTextureMode`

### Tier 2 — gg CPU (cold path, result cached as Texture2D)

- **Arbitrary-path clip** (roundRect → clip): render clip shape in gg, extract `*image.Alpha` mask, upload as `Texture2D` stencil
- **Shadow blur**: render shadow shape in gg → `ImageBlurGaussian` → `LoadTextureFromImage` → cache
- **SVG icons**: oksvg parses SVG → rasterx renders into gg context → upload as `Texture2D`
- **Fine anti-aliased strokes** at small sizes where raylib polygon approximation is inadequate

---

## 4. Package Structure

```
raycanvas/
  context.go      — Context type, state stack, BeginFrame/EndFrame
  draw.go         — Fill/stroke primitives: FillRect, StrokeRect, FillText, etc.
  path.go         — Path accumulator: BeginPath, MoveTo, LineTo, Arc, ArcTo,
                    RoundRect, BezierCurveTo, ClosePath, Fill, Stroke, Clip
  transform.go    — Matrix stack: Save, Restore, Translate, Scale, ResetTransform, SetTransform
  color.go        — CSS color string parser + cache → color.RGBA
  font.go         — Font string parser + registry + cache → rl.Font per FontKey
  image.go        — DrawImage variants, off-screen Context (RenderTexture2D)
  shadow.go       — Shadow property handling + blur pipeline via gg + cache
  icon.go         — SVG icon registry: RegisterIcon, DrawIcon
  clip.go         — Clip region management: scissor stack + gg alpha-mask path
  cache.go        — Shared LRU/map caches for colors, fonts, shadows, icons
  tessellate.go   — Bézier/arc/arcTo tessellation to []rl.Vector2 polylines
```

---

## 5. Context Type

```go
type Context struct {
    // Backing surface — one of:
    //   screen: draws to framebuffer (BeginDrawing/EndDrawing)
    //   offscreen: draws to RenderTexture2D (BeginTextureMode/EndTextureMode)
    rt      *rl.RenderTexture2D // nil = screen context
    width   int32
    height  int32

    // State stack (Save/Restore)
    stack   []canvasState
    state   canvasState

    // Path accumulator
    path    pathBuffer

    // Shared caches (pointer; shared across contexts in the same app)
    cache   *SharedCache
}

type canvasState struct {
    fillStyle     color.RGBA
    strokeStyle   color.RGBA
    lineWidth     float32
    globalAlpha   float32
    font          resolvedFont
    textAlign     textAlign
    textBaseline  textBaseline
    lineDash      []float32
    lineDashOffset float32
    shadowColor   color.RGBA
    shadowBlur    float32
    shadowOffsetX float32
    shadowOffsetY float32
    lineCap       lineCap
    lineJoin      lineJoin
    transform     matrix3x2
    // Clip stack entry: the scissor rect active at this save level.
    // For roundRect clips, also holds a reference to the gg mask texture.
    clip          clipState
}
```

---

## 6. Type Strategy

All geometry parameters use `float32` throughout — matching raylib's native types and avoiding constant casting.

| Value | Type | Reason |
|---|---|---|
| Coordinates, dimensions | `float32` | Matches `rl.Rectangle`, `rl.Vector2` |
| Color (resolved) | `color.RGBA` | Raylib-go's native color type (standard library) |
| Color (CSS string) | `string` → cache → `color.RGBA` | Parsed once, cached |
| Alpha | `float32` | 0.0–1.0 |
| Font (resolved) | `rl.Font` | Per FontKey entry |
| Texture | `rl.Texture2D` | Never `RenderTexture2D` for drawing |
| Off-screen surface | `rl.RenderTexture2D` | Accessed via `.Texture` field for blitting |
| Window/texture dimensions | `int32` | Matches `rl.LoadRenderTexture`, `rl.InitWindow` |
| `Width()`, `Height()` | `int32` | Matches raylib's own dimension fields |
| Path points | `rl.Vector2` | Direct use in `DrawSplineLinear`, etc. |
| Image rects | `rl.Rectangle` | Direct use in `DrawTexturePro` |

### Key raylib type notes

- `rl.Rectangle{X, Y, Width, Height float32}` — use for all source/dest rects
- `rl.Vector2{X, Y float32}` — use for path points and positions
- `color.RGBA{R, G, B, A uint8}` — raylib's color; alpha is uint8 (0–255), not float
- `rl.BeginScissorMode(x, y, w, h int32)` — takes int32; cast internally, never exposed
- `rl.DrawRectangleRounded(rec, roundness float32, segments int32, col)` — roundness is ratio 0–1, NOT pixel radius; convert: `roundness = clamp((2*radius)/min(w,h), 0, 1)`
- `rl.DrawCircleV` / `rl.DrawCircleLinesV` — use V variants (float32 center), never int32 variants
- `rl.DrawTextEx(font, text, pos Vector2, fontSize, spacing float32, tint)` — fontSize is float32
- `rl.MeasureTextEx(font, text, fontSize, spacing float32) Vector2` — returns Vector2; use `.X` for width
- `rl.RenderTexture2D.Texture` — the `Texture2D` field; use this for drawing, never the wrapper
- `rl.LoadImageFromTexture` — GPU readback, expensive, never call per-frame
- `rl.ImageBlurGaussian(image *Image, blurSize int32)` — CPU-side, used in shadow pipeline

---

## 7. Color Cache

CSS color strings → `color.RGBA`, parsed once and cached by string key.

Formats to support (all appear in shevo source):
- `#rgb` — 3-digit hex
- `#rrggbb` — 6-digit hex  
- `rgba(r, g, b, a)` — a is float 0–1, converted to uint8
- Named colors: `currentColor` handled at call site (caller provides current stroke/fill)

```go
type ColorCache struct {
    mu    sync.RWMutex
    table map[string]color.RGBA
}
```

`globalAlpha` is applied by multiplying the resolved color's A channel at draw time, not stored in the cache.

---

## 8. Font System

### The problem

raylib bakes a font atlas at a fixed pixel size. The browser synthesises weight/style variants. These are fundamentally different models.

### Font registry

```go
type FontKey struct {
    Family string  // "fira", "inter", "georgia"
    Size   float32 // exact pixel size
    Weight int     // 400=regular, 500=medium, 600=semibold, 700=bold
    Italic bool
}
```

`RegisterFont(family string, weight int, italic bool, data []byte, sizes []float32)`  
— loads TTF from memory, bakes one atlas per requested size via `rl.LoadFontEx`, stores all in cache.

**Critical rule:** bake size == draw size, always. When `DrawTextEx` is called with `fontSize == font.BaseSize`, raylib renders directly from the atlas with no scaling — maximum sharpness. We therefore bake at every size actually used.

### Sizes used in shevo

8, 9, 10, 11, 12, 13, 14px. Bake all of these at registration time.

### Weights used in shevo

- Fira Code: Regular (400), Bold (700). No italic variant exists.
- Inter: Regular (400), Medium (500), SemiBold (600), Bold (700), Italic variants.
- Georgia: Regular (400), Bold (700) — fallback serif.

### Font string parsing

CSS font strings like `"italic 700 13px Inter"`, `"500 9px Fira Code"`, `"bold 12px system-ui"`.

Parse order: optional style (`italic`) → optional weight (`bold`/`700`) → size (`13px`) → family.

Weight keywords: `bold` = 700, `normal` = 400. Numeric weights taken directly.

Fallback: if `(family, weight, italic)` not registered, try `(family, 400, false)`. If family not registered, use raylib default font.

### Spacing

Always pass `spacing = 0` to `DrawTextEx` and `MeasureTextEx`. Matches browser default (normal letter-spacing). Consistent between measure and draw — essential for correct text layout.

---

## 9. Clip System

### Clip cases in shevo

Max nesting depth: **3** (joxel grid-draw.js and quag render.js).  
All clips are within `save()`/`restore()` pairs — restore always unwinds clip.

| Shape | Count | Locations |
|---|---|---|
| `rect` → `clip()` | 14 | All perspectives, common case |
| `roundRect` → `clip()` | 2 | quag line 2790 (card chrome), joxel line 382 (ghost cell) |

### Rectangular clip implementation

`BeginScissorMode` with intersection of accumulated scissor rects.  
State stack stores the active scissor rect at each save level.  
On `restore()`: pop and re-apply previous rect (`BeginScissorMode` again, or `EndScissorMode` at depth 0).

Intersection math in `float32`; cast to `int32` only at `rl.BeginScissorMode` call site.

### roundRect clip implementation

Cannot use `BeginScissorMode` (axis-aligned only).

Approach:
1. Render clip shape into a gg context (CPU, same dimensions as target)
2. Extract `*image.Alpha` mask via `gg.ClipPreserve()` + `gg.AsMask()`
3. Upload mask as `rl.Texture2D` (UNCOMPRESSED_GRAYSCALE format)
4. At draw time: render content to offscreen `RenderTexture2D`, composite onto main surface using mask texture as alpha

Cache key: `"roundrect|x|y|w|h|r"` — reuse mask if geometry is identical.

---

## 10. Shadow Pipeline

Used once in joxel (chevron glow) — decorative, so v1 quality can be moderate.

**Per unique shadow configuration** (color + blur + shape):
1. Render shape to gg context (CPU)
2. Apply `rl.ImageBlurGaussian` (blurSize = shadowBlur/2, clamped)
3. `rl.LoadTextureFromImage` → `rl.Texture2D`
4. Unload intermediate `rl.Image`
5. Cache `Texture2D` keyed by `"shadowColor|blur|w|h|shape-hash"`

At draw time: `DrawTexturePro` the cached texture at offset `(shadowOffsetX, shadowOffsetY)`, then draw main content on top.

**Never recompute per frame.**

### Bézier stroke cache

Anti-aliased cubic Bézier strokes are also cached as `Texture2D` values in the shadow cache slot, keyed by `bezierCacheKey(p0, cp1, cp2, p1, lineWidth)`. Color and alpha are applied as a tint at blit time so animated opacity doesn't bust the cache.

**Limitation:** the cache is unbounded. If curve endpoints move continuously (e.g. drag interaction), stale textures accumulate. Before implementing draggable links in joxel, add LRU eviction to the bezier cache entries — cap at ~64 entries and evict LRU on miss.

---

## 11. Path Tessellation

The path buffer accumulates segments. At `Fill()` or `Stroke()` time, the buffer is tessellated to `[]rl.Vector2` and dispatched.

### Tessellation targets

| Path segment | Tessellation | Raylib call |
|---|---|---|
| `lineTo` | Direct point | `DrawSplineLinear` |
| `arc` (full circle, `0..2π`) | `DrawCircleV` / `DrawCircleLinesV` — no tessellation | Direct |
| `arc` (partial) | N points on arc | `DrawSplineLinear` |
| `arcTo` | Compute tangent points + arc geometry | Points added to buffer |
| `bezierCurveTo` (cubic) | `rl.DrawSplineSegmentBezierCubic` for stroke | Direct |
| `roundRect` | Decompose to lines + arc segments | Points added to buffer |

### Tessellation quality

Package-level constant `TessellationSteps = 32` — sufficient for 8–100px arcs at screen resolution. Configurable.

### arcTo geometry

Standard construction:
1. Compute tangent lengths from current point and the two control points
2. Find the two tangent points T1, T2
3. Add `LineTo(T1)`, then arc from T1 to T2 around the computed center
4. Pure float32 geometry, no external library needed

### bezierCurveTo

The shevo integration layer uses this for inter-perspective link curves (two calls per frame, not in the cell render loop). Stroke only, never filled.  
→ `rl.DrawSplineSegmentBezierCubic(p1, c2, c3, p4 Vector2, thick float32, col)`  
Direct mapping, no tessellation needed.

---

## 12. DrawImage Variants

The JS canvas `drawImage` has three call signatures. In Go, three distinct methods using `rl.Rectangle` directly:

```go
// Draw entire texture at position
func (c *Context) DrawImage(src rl.Texture2D, dx, dy float32)

// Draw entire texture scaled to dest rect
func (c *Context) DrawImageScaled(src rl.Texture2D, dst rl.Rectangle)

// Draw source sub-rect of texture scaled to dest rect
func (c *Context) DrawImageCropped(src rl.Texture2D, src rl.Rectangle, dst rl.Rectangle)
```

Off-screen contexts expose their texture for use as a source:
```go
func (c *Context) Texture() rl.Texture2D // returns c.rt.Texture
```

---

## 13. SVG Icon System

All icons in shevo are compile-time constants (≈35 total, 12×12 to 16×16 viewBox).  
No dynamic SVG generation. No per-frame SVG rendering.

### Path complexity

SVG path commands used: `M`, `L`, `H`, `V`, `Z` (simple), `A` (elliptical arc), `C` (cubic Bézier), `S` (smooth cubic). Also `<circle>`, `<rect>`, `<line>`, `<polygon>` elements. `fill-rule="evenodd"` used in file icon paths.

oksvg handles all of these. rasterx provides the anti-aliased rasterisation backend.

### Icon pipeline

1. `RegisterIcon(name string, svgData []byte, size float32)` — at startup
2. Parse with oksvg → render into gg context at requested size
3. `rl.LoadTextureFromImage` → `rl.Texture2D`
4. Cache by `(name, size)`

### Drawing icons

```go
func (c *Context) DrawIcon(name string, x, y, size float32, tint color.RGBA)
```

`currentColor` icons: pass current `strokeStyle` or `fillStyle` as tint.  
Tint applied via `rl.DrawTexturePro`'s tint parameter — modulates the white/grey pixels in the pre-rasterised texture.

**Note:** icons rasterised with white fill, black outline on transparent background. Tinting then colours them correctly.

---

## 14. Off-Screen Contexts

```go
func NewOffscreen(width, height int32, cache *SharedCache) *Context
```

Backed by `rl.LoadRenderTexture(width, height)`.  
Draw into it with `BeginFrame`/`EndFrame` (which call `rl.BeginTextureMode`/`rl.EndTextureMode`).  
Blit to another context via `DrawImageCropped` using `.Texture()`.

**Important:** `RenderTexture2D` has Y-axis flipped relative to screen. `DrawTexturePro` source rect must use `Height: -height` to flip. This is handled internally.

---

## 15. Canvas API Surface Implemented

Derived from audit of joxel, quag, dekk, and shevo integration layer source.

### State
`Save()`, `Restore()`, `ResetTransform()`, `SetTransform(a,b,c,d,e,f float32)`

### Transform
`Translate(x, y float32)`, `Scale(x, y float32)`

### Style setters
`SetFillStyle(s string)`, `SetStrokeStyle(s string)`, `SetLineWidth(w float32)`,  
`SetGlobalAlpha(a float32)`, `SetFont(s string)`, `SetTextAlign(s string)`,  
`SetTextBaseline(s string)`, `SetLineDash(segments []float32)`,  
`SetLineDashOffset(o float32)`, `SetLineCap(s string)`, `SetLineJoin(s string)`,  
`SetShadowColor(s string)`, `SetShadowBlur(b float32)`,  
`SetShadowOffsetX(x float32)`, `SetShadowOffsetY(y float32)`,  
`SetImageSmoothingEnabled(b bool)`, `SetDirection(s string)`

### Path
`BeginPath()`, `ClosePath()`, `MoveTo(x, y float32)`, `LineTo(x, y float32)`,  
`Arc(x, y, r, startAngle, endAngle float32, anticlockwise bool)`,  
`ArcTo(x1, y1, x2, y2, r float32)`,  
`Rect(x, y, w, h float32)`,  
`RoundRect(x, y, w, h, r float32)`,  
`BezierCurveTo(cp1x, cp1y, cp2x, cp2y, x, y float32)`,  
`Fill()`, `Stroke()`, `Clip()`

### Drawing
`FillRect(x, y, w, h float32)`, `StrokeRect(x, y, w, h float32)`,  
`ClearRect(x, y, w, h float32)`,  
`FillText(text string, x, y float32)`,  
`MeasureText(text string) float32`

### Images
`DrawImage(src rl.Texture2D, dx, dy float32)`,  
`DrawImageScaled(src rl.Texture2D, dst rl.Rectangle)`,  
`DrawImageCropped(src rl.Texture2D, srcRect, dst rl.Rectangle)`

### Canvas info
`Width() int32`, `Height() int32`

### NOT implemented (not used in shevo)
`createLinearGradient`, `createRadialGradient`, `createPattern`,  
`quadraticCurveTo`, `getImageData`, `putImageData`, `bezierCurveTo` (filled path — stroke only)

---

## 16. Shevo Canvas API Audit Summary

Source: shevo-0.2.3, all three perspectives + integration layer.

| API call | Frequency | Notes |
|---|---|---|
| `fillStyle =` | 237 | Most common; CSS strings |
| `font =` | 135 | CSS font strings |
| `beginPath()` | 130 | |
| `fillText()` | 114 | |
| `lineTo()` | 87 | |
| `textAlign =` | 75 | |
| `fillRect()` | 71 | |
| `strokeStyle =` | 70 | |
| `lineWidth =` | 70 | |
| `stroke()` | 68 | |
| `textBaseline =` | 65 | |
| `moveTo()` | 62 | |
| `fill()` | 60 | |
| `globalAlpha =` | 59 | |
| `measureText()` | 51 | `.width` only |
| `roundRect()` | 43 | Pixel radius, not ratio |
| `save()` / `restore()` | 33/34 | Max nesting depth: 3 |
| `arcTo()` | 20 | dekk rounded corners |
| `arc()` | 20 | All full circles (0..2π) except chevron |
| `clip()` | 16 | All rect-based except 2 roundRect cases |
| `scale()` | 14 | Zoom transform |
| `rect()` | 14 | Used before clip() |
| `translate()` | 12 | |
| `setLineDash()` | 10 | |
| `strokeRect()` | 8 | |
| `shadowColor =` | 8 | |
| `shadowBlur =` | 8 | |
| `drawImage()` | 5 | quag mipmap system |
| `closePath()` | 6 | |
| `clearRect()` | 3 | |
| `shadowOffsetY =` | 2 | |
| `resetTransform()` | 2 | |
| `bezierCurveTo()` | 2 | shevo integration layer only |
| `setTransform()` | 1 | |
| `lineJoin =` | 1 | |
| `lineCap =` | 1 | |
| `direction =` | 1 | RTL markdown |
| `imageSmoothingEnabled =` | 6 | |

---

## 17. Known Constraints & Gotchas

### raylib-go specific

- **`BeginScissorMode` is not nestable** — takes int32 params, replaces (doesn't intersect) previous scissor. raycanvas maintains a rect stack and re-issues the intersected rect on each clip change.
- **`RenderTexture2D` Y-flip** — textures rendered off-screen are vertically flipped. Always negate height in source `rl.Rectangle` when blitting with `DrawTexturePro`.
- **`LoadImageFromTexture` is a GPU readback** — stalls pipeline. Only call during asset preparation, never in frame loop.
- **`DrawRectangleRounded` roundness is a ratio** — convert pixel radius: `roundness = clamp(2*r/min(w,h), 0, 1)`.
- **`DrawCircle` / `DrawCircleLines` take int32 center** — use `DrawCircleV` / `DrawCircleLinesV` instead.
- **Font atlas baked at fixed size** — `DrawTextEx` with `fontSize != font.BaseSize` causes bilinear scaling. Always bake at exact sizes needed (8–14px for shevo).
- **`color.RGBA` alpha is uint8** — when applying `globalAlpha`, multiply: `a = uint8(float32(color.A) * globalAlpha)`.
- **No weight synthesis** — bold/italic require separate TTF files loaded as separate `rl.Font` values.

### gg specific

- **gg uses `float64`** — all gg API calls take float64. Convert from our float32 at the gg call boundary. No propagation.
- **gg mask size must match context size** — `SetMask` returns error if dimensions differ.
- **gg is CPU-only** — never call gg in the frame loop. All gg work is cached to `Texture2D` before the loop starts or on first use.

### General

- **Quag mipmap tiers** become `rl.Texture2D` uploaded once. Never convert back to `image.Image` during rendering.
- **Font spacing = 0** always — consistent between `MeasureTextEx` and `DrawTextEx`.
- **`measureText()` returns width only** — expose as `float32` (`.X` of `MeasureTextEx` result).
- **Fira Code has no italic** — fall back to regular if italic Fira Code requested.
- **`currentColor` in SVG icons** — icons pre-rasterised with white-on-transparent; tinted at draw time via `DrawTexturePro` tint parameter.
