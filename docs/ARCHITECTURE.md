# raycanvas — Architecture & Design Reference

**Module:** `github.com/ha1tch/raycanvas`  
**Purpose:** A Go package that mirrors the HTML5 Canvas 2D API, backed by raylib-go for GPU rendering and fogleman/gg for CPU-side precision work. Designed as the rendering substrate for porting canvas-based web applications from JavaScript to Go.

---

## 1. Motivation

Canvas-based web applications are written against the HTML5 `CanvasRenderingContext2D` API. Porting them to Go without a canvas-compatible abstraction requires translating every draw call into raw raylib primitives — an impedance mismatch that multiplies friction. raycanvas provides that abstraction: ported code uses `ctx.FillRect(...)`, `ctx.BeginPath()`, `ctx.Arc(...)` etc., and the package handles the raylib mapping transparently.

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
- `FillText` → `rl.DrawTextEx`
- Rectangular clip → `rl.BeginScissorMode` / `rl.EndScissorMode`
- Rounded rect fill/stroke → `rl.DrawRectangleRounded`, `rl.DrawRectangleRoundedLinesEx`
- Full circles → `rl.DrawCircleV` (fill), `rl.DrawCircleLinesV` (stroke)
- Cubic Bézier stroke → cached anti-aliased gg texture, blitted via `rl.DrawTexturePro`
- Image/texture blit → `rl.DrawTexturePro`
- Off-screen render → `rl.BeginTextureMode` / `rl.EndTextureMode`

### Tier 2 — gg CPU (cold path, result cached as Texture2D)

- **Arbitrary-path clip** (roundRect → clip): corner overdraw with background colour
- **Shadow blur**: render shadow shape in gg → `ImageBlurGaussian` → `LoadTextureFromImage` → cache
- **SVG icons**: oksvg parses SVG → rasterx renders into gg context → upload as `Texture2D`
- **Anti-aliased Bézier strokes**: rendered in gg, cached by geometry+lineWidth

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
  clip.go         — Clip region management: scissor stack + corner overdraw
  cache.go        — Shared caches for colors, fonts, shadows, icons
  path.go         — Bézier/arc/arcTo tessellation to []rl.Vector2 polylines

fonts/
  fonts.go        — Embeds Inter and Fira Code TTFs, public Register() call
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
    fillStyle      color.RGBA
    strokeStyle    color.RGBA
    lineWidth      float32
    globalAlpha    float32
    font           resolvedFont
    textAlign      textAlign
    textBaseline   textBaseline
    lineDash       []float32
    lineDashOffset float32
    shadowColor    color.RGBA
    shadowBlur     float32
    shadowOffsetX  float32
    shadowOffsetY  float32
    lineCap        lineCap
    lineJoin       lineJoin
    transform      matrix3x2
    clip           clipState
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

CSS color strings → `color.RGBA`, parsed once and cached by string key. Cap: 256 entries, FIFO eviction.

Supported formats:
- `#rgb` — 3-digit hex
- `#rrggbb` — 6-digit hex
- `rgba(r, g, b, a)` — a is float 0–1, converted to uint8
- `rgb(r, g, b)`
- Named colors: `transparent`, `white`, `black`, and common web colors

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

**Critical rule:** bake size == draw size, always. When `DrawTextEx` is called with `fontSize == font.BaseSize`, raylib renders directly from the atlas with no scaling — maximum sharpness.

### Default sizes

8, 9, 10, 11, 12, 13, 14px. Covers typical UI chrome text at standard DPI.

### Font string parsing

CSS font strings like `"italic 700 13px Inter"`, `"500 9px Fira Code"`, `"bold 12px system-ui"`.

Parse order: optional style (`italic`) → optional weight (`bold`/`700`) → size (`13px`) → family.

Fallback: if `(family, weight, italic)` not registered, try `(family, 400, false)`. If family not registered, use raylib default font.

### Spacing

Always pass `spacing = 0` to `DrawTextEx` and `MeasureTextEx`. Matches browser default (normal letter-spacing). Consistent between measure and draw — essential for correct text layout.

---

## 9. Clip System

### Clip shapes supported

| Shape | Implementation |
|---|---|
| `rect` → `clip()` | `rl.BeginScissorMode` with intersection tracking |
| `roundRect` → `clip()` | Scissor for bounding box + corner overdraw on `Restore()` |

Max observed nesting depth in typical canvas applications: 3. All clips are within `save()`/`restore()` pairs.

### Rectangular clip

`BeginScissorMode` with intersection of accumulated scissor rects. State stack stores the active scissor rect at each save level. On `restore()`: pop and re-apply previous rect.

Intersection math in `float32`; cast to `int32` only at `rl.BeginScissorMode` call site.

### Rounded-rect clip

Uses a corner overdraw technique: content is drawn normally inside the scissor bounding box, then four quarter-circle caps in the background colour are painted over the corners on `Restore()`. Call `SetMaskBackground(css)` before the clip if the background is not the current fill style.

**Limitation:** requires a solid known background colour. Semi-transparent backgrounds produce artefacts. True GPU alpha masking is a planned improvement.

---

## 10. Shadow Pipeline

**Per unique shadow configuration** (color + blur + shape + geometry):
1. Render shape to gg context (CPU)
2. Apply `rl.ImageBlurGaussian` (blurSize snapped to nearest integer)
3. `rl.LoadTextureFromImage` → `rl.Texture2D`
4. Unload intermediate `rl.Image`
5. Cache `Texture2D` — keyed by `color|blur|w×h|shape`

At draw time: `DrawTexturePro` the cached texture at `(shadowOffsetX, shadowOffsetY)`, then draw main content on top.

Cache cap: 1024 entries, FIFO eviction. Eviction does **not** call `UnloadTexture` — freeing the GPU ID causes raylib to reuse it immediately, leading to stale references drawing the wrong texture.

**Never recompute per frame.** Blur is snapped to integer values so sub-integer animation changes share cache entries.

### Bézier stroke cache

Anti-aliased cubic Bézier strokes are cached in the same pool, keyed by `(p0, cp1, cp2, p1, lineWidth)`. Color and alpha are applied as a `DrawTexturePro` tint so animated opacity doesn't bust the cache.

---

## 11. Path Tessellation

The path buffer accumulates segments. At `Fill()` or `Stroke()` time, the buffer is tessellated to `[][]rl.Vector2` (one sub-path per `MoveTo`) and dispatched.

### Tessellation targets

| Path segment | Fill dispatch | Stroke dispatch |
|---|---|---|
| `rect` | `DrawRectangleRec` | `DrawRectangleLinesEx` |
| `roundRect` | `DrawRectangleRounded` | `DrawRectangleRoundedLinesEx` |
| `arc` full circle | `DrawCircleV` | `DrawRing` |
| `arc` partial | `DrawTriangleFan` (centroid) | `DrawRing` |
| `arcTo` | tessellate → `DrawTriangleFan` | `DrawSplineLinear` |
| `bezierCurveTo` | tessellate → `DrawTriangleFan` | gg cached texture |
| multi-segment | tessellate → `DrawTriangleFan` | `DrawSplineLinear` per sub-path |

### Fill winding

`DrawTriangleFan` requires CCW winding in screen coordinates (Y-down). Tessellated points are in increasing-angle order (CW in screen), so the perimeter is reversed before fan construction. The centroid is prepended as the explicit fan centre.

**Limitation:** non-convex polygons may render incorrectly. Ear-clipping triangulation is not yet implemented.

### Stroke sub-paths

`tessellate()` returns `[][]rl.Vector2` — one slice per `MoveTo`. Each sub-path is drawn independently, preventing spurious connecting lines across discontinuities. Two-point sub-paths use `DrawLineEx` directly (sharper than `DrawSplineLinear` for hairlines).

### Tessellation quality

Package-level constant `TessellationSteps = 32`. Arc strokes use adaptive segment counts: `max(16, min(128, radius * π / 4))`.

---

## 12. DrawImage Variants

The JS canvas `drawImage` has three call signatures. In Go, three distinct methods:

```go
func (c *Context) DrawImage(src rl.Texture2D, dx, dy float32)
func (c *Context) DrawImageScaled(src rl.Texture2D, dst rl.Rectangle)
func (c *Context) DrawImageCropped(src rl.Texture2D, srcRect, dst rl.Rectangle)
```

Off-screen contexts expose their texture for use as a source:
```go
func (c *Context) Texture() rl.Texture2D // returns c.rt.Texture
```

**Important:** `RenderTexture2D` has Y-axis flipped relative to screen. `DrawImageOffscreen` / `DrawImageOffscreenCropped` handle the flip automatically.

---

## 13. SVG Icon System

Icons are compile-time assets registered at startup. No per-frame SVG rendering.

### Path complexity supported

SVG path commands: `M`, `L`, `H`, `V`, `Z`, `A` (elliptical arc), `C` (cubic Bézier), `S` (smooth cubic), `Q` (quadratic). Also `<circle>`, `<rect>`, `<line>`, `<polygon>`.

**Note:** oksvg requires explicit spaces between arc flag parameters. The compact form `a4 4 0 014 4` (valid per spec) is not parsed correctly — write as `a4 4 0 0 1 4 4`.

### Icon pipeline

1. `RegisterIcon(name string, svgData []byte, size float32)` — at startup, after `SetupWindow`
2. Parse with oksvg → render into gg context at requested size
3. Convert to white-on-transparent (RGB channels set to 255, alpha preserved)
4. `rl.LoadTextureFromImage` → `rl.Texture2D`
5. Cache by `(name, size)`

### Drawing

```go
func (c *Context) DrawIcon(name string, x, y, size float32, tint color.RGBA)
```

Tint applied via `rl.DrawTexturePro` — multiplies the white pixels to the desired colour. Pass `rc.ParseColor(css)` to convert a CSS string to `color.RGBA` outside a Context.

---

## 14. Off-Screen Contexts

```go
offscreen := rc.NewOffscreen(width, height int32, cache *SharedCache)
defer offscreen.Unload()
```

Backed by `rl.LoadRenderTexture(width, height)`. Draw into it between `BeginFrame`/`EndFrame`. Blit to another context via `DrawImageOffscreen`.

**Y-flip:** textures rendered off-screen are vertically inverted. `DrawImageOffscreen` and `DrawImageOffscreenCropped` correct for this automatically using a negative height in the source rectangle.

---

## 15. Cache Architecture

```
SharedCache
  colors      map[string]color.RGBA        cap: 256,  FIFO, no GPU resource
  fonts       map[FontKey]rl.Font          unbounded, registered at startup
  shadows     map[string]rl.Texture2D      cap: 1024, FIFO, no UnloadTexture on eviction
  icons       map[iconKey]rl.Texture2D     unbounded, registered at startup
  maskImages  map[string]*image.RGBA       unbounded, CPU-side, roundRect clip masks
```

All methods are safe for concurrent use during asset loading. Frame-loop access is read-only after startup and requires no locking.

---

## 16. Known Constraints & Gotchas

### raylib-go specific

- **`BeginScissorMode` is not nestable** — raycanvas maintains a rect stack and re-issues the intersected rect on each clip change.
- **`RenderTexture2D` Y-flip** — textures rendered off-screen are vertically flipped. Always negate height in source `rl.Rectangle` when blitting with `DrawTexturePro`.
- **`LoadImageFromTexture` is a GPU readback** — stalls pipeline. Only call during asset preparation, never in the frame loop.
- **`DrawRectangleRounded` roundness is a ratio** — convert pixel radius: `roundness = clamp(2*r/min(w,h), 0, 1)`.
- **`DrawCircle` / `DrawCircleLines` take int32 center** — use `DrawCircleV` / `DrawCircleLinesV` instead.
- **Font atlas baked at fixed size** — `DrawTextEx` with `fontSize != font.BaseSize` causes bilinear scaling. Always bake at exact sizes needed.
- **`color.RGBA` alpha is uint8** — when applying `globalAlpha`, multiply: `a = uint8(float32(color.A) * globalAlpha)`.
- **No weight synthesis** — bold/italic require separate TTF files loaded as separate `rl.Font` values.
- **`DrawTriangleFan` winding** — requires CCW in screen coordinates. Perimeter points must be reversed before passing to the fan.

### gg specific

- **gg uses `float64`** — convert from float32 at the gg call boundary only.
- **gg is CPU-only** — never call gg in the frame loop. All gg work is cached to `Texture2D` before the loop starts or on first use.

### General

- **Font spacing = 0** always — consistent between `MeasureTextEx` and `DrawTextEx`.
- **`MeasureText()` returns width only** — exposed as `float32` (`.X` of `MeasureTextEx` result).
- **SVG arc flags need spaces** — oksvg does not parse compact arc flag syntax. Use explicit spaces between flag parameters.
- **Shadow cache eviction does not free VRAM** — by design; freeing GPU IDs causes stale reference artifacts.