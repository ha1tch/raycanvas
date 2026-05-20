# Changelog — raycanvas

All notable changes to this project are documented here.
The top entry always matches the VERSION file and pkg/version/version.go.

## 0.2.3 — 2026-05-20

### Added

- **`icons` example** (`examples/icons`): demonstrates the SVG icon pipeline across four sections. (1) Shevo icons — the actual SVGs from joxel/quag/dekk shell files (chevron, align, upload, download, undo, redo, anchor, layout, link, globe, edit), rendered at 14px and 24px. (2) Original icons in the same geometric style (search, filter, settings, copy, trash, check, close, pin, tag, expand, collapse, star, bell, sort). (3) 40px showcase of all icons with per-icon accent tinting. (4) Light/Dark panel comparison — same registered textures, different tints, showing toolbar simulation and labelled button rows in both themes.
- **`ParseColor(css string) color.RGBA`** — public CSS colour parser for use outside a `Context`, e.g. when constructing tint arguments for `DrawIcon`.

---

## 0.2.2 — 2026-05-19

### Fixed

- **Pie slices, filled arcs, and arcTo polygons not rendering** (`draw.go`): `DrawTriangleFan` documentation states *"following vertex should be provided in counter-clockwise order"* — CCW in screen coordinates (Y-down). Our tessellation generates points in increasing-angle order, which is CW in screen coords (the wrong winding). Fixed by reversing the tessellated perimeter before building the fan, and prepending the centroid as the fan centre so the fan is well-defined for all polygon shapes including non-convex paths.
- **Shadow animation jumps discretely** (`shadow.go`, `color.go`): blur was formatted as `%.2f` in the cache key so every sub-integer change created a new cache entry. Changed to `%.0f` (integer). Additionally, `drawShadowTexture` now snaps `shadowBlur` to the nearest integer with `math.Round` before use, ensuring the `ImageBlurGaussian` kernel steps smoothly without discrete jumps.
- **Shadow eviction causes colour flash** (`cache.go`): `UnloadTexture` on eviction freed the GPU texture ID; raylib could immediately reuse that ID for a new upload, causing stale references to draw the wrong texture. Removed `UnloadTexture` from the eviction path. Increased shadow cache cap from 512 to 1024.
- **Makefile help text missing `zui` and `shadows`** (`Makefile`): both examples now listed in `make help` with descriptions.

---

## 0.2.1 — 2026-05-19

### Fixed

- **Drop shadow always appeared top-left, never bottom-right** (`shadow.go`): `buildShadowTexture` allocates a texture padded by `blur*2` on all sides so blur doesn't clip. The shape was being drawn at `(blur, blur)` inside the texture, but the blit assumed `(pad, pad)` = `(blur*2, blur*2)`. The shape was therefore offset by `blur` pixels up and left relative to where the blit expected it, making all shadows appear shifted toward the top-left regardless of the offset values. Fixed: shape now drawn at `(pad, pad)` inside the texture.
- **Baseline metric 0.8 → 0.75** (`draw.go`): empirical correction for Inter and Fira Code at 8–14px baked atlases. The alphabetic baseline now aligns closer to where browser canvas places it.
- **Shadow state bleeding between sections** (`examples/shadows`): added explicit `clearShadow()` resets after every shadow draw call. The vibrant colour flash on the third card was caused by residual shadow state from the glow section being active during the next card's draw.
- **`shadows` example rewritten** with light theme (`#e8e8f0` background), correct `SetShadowOffsetX/Y` usage, `clearShadow()` helper called after every shadow, text at legible sizes (9–13px), and proper z-ordering (shadow drawn before fill).

---

## 0.2.0 — 2026-05-19

### Fixed

- **Shadow pipeline not connected** (`draw.go`): `SetShadowColor`/`SetShadowBlur` stored state but `FillRect`, `FillRoundRect`, `FillCircle`, and `FillText` never checked it. Shadow was silently a no-op. Now wired: `FillRect` and `FillRoundRect` call `drawShadowForRect/RoundRect` (gg blur pipeline, cached); `FillCircle` approximates as a rounded square; `FillText` draws the text a second time at the shadow offset in the shadow colour (offset-only, no blur — matches browser behaviour for text-shadow).
- **`shadows` example rewritten**: fixed wrong coordinate arguments, z-ordering (shadow drawn before fill, not after), font size issues, and layout. Four clean panels: drop shadow on cards, coloured glow on circles and panels, blur depth comparison, text shadow legibility demo.

---

## 0.1.9 — 2026-05-19

### Fixed

- **Severe card distortion at high zoom** (`draw.go`, `examples/zui`, `examples/shadows`): the centroid prepend approach from 0.1.8 made the `DrawTriangleFan` distortion dramatically worse at zoom levels above ~200%. Reverted. The real fix: the rrTop title bar path (custom `ArcTo` path for a top-rounded rect) was going through `DrawTriangleFan` which is wrong for perimeter-order points. Replaced with two stacked raylib primitives: `FillRoundRect` gives the rounded top corners, `FillRect` squares off the bottom — no tessellation, no fan, correct at all zoom levels.

### Added

- **`FillRoundRectTop(x,y,w,h,r)`** and **`StrokeRoundRectTop(x,y,w,h,r)`** — rounded only at the top two corners, flat at the bottom. The rrTop primitive from quag used for title bars and similar chrome. `FillRoundRectTop` uses two stacked raylib calls (no tessellation). `StrokeRoundRectTop` uses a path with explicit arc segments.

---

## 0.1.8 — 2026-05-19

### Fixed

- **Card top edge skewed / geometry above card top** (`draw.go`): `DrawTriangleFan` treats `pts[0]` as the fan centre — for perimeter-order tessellation points this produces incorrect triangles from a corner point, extending geometry outside the intended polygon (visibly tilting the card top edge ~16px across card width). Fixed by computing the centroid of the tessellated points and prepending it as the fan centre, so all triangles radiate from an interior point.

### Added

- **`shadows` example** (`examples/shadows`): demonstrates the gg-backed blur/shadow pipeline — drop shadows with animated elevation, coloured glow halos, text shadow for legibility over noise, and layered blur depth panels. All four techniques drawn from quag's visual design vocabulary.

---

## 0.1.7 — 2026-05-19

### Added

- **`FillRoundRect(x,y,w,h,r)`** and **`StrokeRoundRect(x,y,w,h,r)`** — single-call convenience methods equivalent to `BeginPath(); RoundRect(...); Fill/Stroke()`. Eliminates the three-step boilerplate that appears constantly in card chrome code.
- **`FillCircle(x,y,r)`** and **`StrokeCircle(x,y,r)`** — single-call circle primitives.
- **`RegisteredFamilies() []string`** — returns canonical family names currently in the font registry, making font name mismatches debuggable without reading source.
- **`autoMaskBackground()`** — `Restore()` now falls back to the current `fillStyle` as the roundRect corner overdraw colour when `SetMaskBackground` has not been called explicitly. Since fill style is usually set to the card background before building a clip path, `SetMaskBackground` is now optional in the common case.

### Fixed

- **Font size does not scale with zoom in zui example**: all `SetFont` calls now use `zoomedFont()` which computes `worldSize * zoomVal`, snaps to the nearest pre-baked atlas size (8–14px), and returns the correct CSS font string. Text now changes atlas at zoom thresholds rather than staying at a fixed size regardless of zoom.
- **`SetMaskBackground` removed from examples** — no longer required due to `autoMaskBackground()` fallback.

---

## 0.1.6 — 2026-05-19

### Added

- **`zui` example** (`examples/zui`): a faithful Quag-style ZUI with pan/zoom infinite canvas, card chrome (title bar, drag dots, close/minimise/mode buttons), card drag, drop shadow, and the four Quag themes (Kaputccino, Light, Polykai, Dark). Theme colour tables match `quag/src/constants.js` exactly. Grid matches quag's two-level fine/major grid with zoom-based fade. Controls: scroll to zoom, space+drag or middle-mouse to pan, drag title bar to move cards, T to cycle themes, N to create a card at cursor.

### Fixed

- **Circular arc stroke jagged** (continued from 0.1.5): full-circle strokes also now use `DrawRing` for consistency with partial arcs.

---

## 0.1.5 — 2026-05-19

### Fixed

- **Entire top row of panels missing in paths example**: `openMaskedRegion` was calling `BeginTextureMode` inside the frame loop, redirecting all subsequent rendering — including next-frame panel draws — into the offscreen RT indefinitely. Replaced the offscreen RT compositing approach with a corner overdraw technique: after the clipped content is drawn, four quarter-circle caps in the background colour are painted over the corner regions, producing a correct rounded clip without GPU readback or nested render targets. New `SetMaskBackground(css)` API for callers to specify the background colour.
- **Circular arc stroke jagged** (`draw.go`): partial and full arc strokes were going through `DrawSplineLinear` on a 32-step tessellated polyline. Replaced with `DrawRing` which uses raylib's native arc renderer. Segment count scales with radius (~1 segment per 4px of arc length, clamped 16–128).

---

## 0.1.4 — 2026-05-19

### Fixed

- **Bézier endpoint dots rendered below curves** (`examples/curves`): anchor dots were drawn inside the panel loops before `drawCurve`, placing them underneath the curve stroke. Extracted into a dedicated third pass after all curves are drawn so dots always appear on top.
- **roundRect clip missing bottom-right corner** (`clip.go`, `context.go`, `shadow.go`, `cache.go`): the alpha mask texture was built correctly but never applied — `hasMask` was set but nothing read it. Implemented a proper offscreen compositing pipeline: `Clip()` after `roundRect` opens a `RenderTexture2D`; all subsequent draw calls go there; `Restore()` pulls the result to CPU, applies the roundRect alpha mask via `rl.ImageAlphaMask`, uploads, and blits back. The CPU-side mask image is cached as `*image.RGBA` in `SharedCache` so it is built at most once per unique geometry.

---

## 0.1.3 — 2026-05-19

### Fixed

- **SIGSEGV on startup** (all examples): `fonts.Register` and `SetTargetFPS` were called before `rc.SetupWindow`, i.e. before `InitWindow`. GPU resource allocation before the OpenGL context exists causes a null-pointer crash in raylib's `SetTextureFilter`. All examples corrected to call `SetupWindow` first. `SetupWindow` doc updated with the required call order.

---

## 0.1.2 — 2026-05-19

### Fixed

- **`EnableSmoothLines` had no effect on shapes** (`context.go`): `GL_LINE_SMOOTH` only applies to raw GL line primitives, not raylib's polygon-based shape renderer. Removed misleading call.
- **MSAA must be set before `InitWindow`** (`raycanvas.go`, examples): added `SetupWindow(width, height, title, cache)` helper that calls `rl.SetConfigFlags(rl.FlagMsaa4xHint)` before `rl.InitWindow`, ensuring 4× multisampling is active for all GPU-rendered shapes. All examples updated to use `SetupWindow`.

---

## 0.1.1 — 2026-05-19

### Fixed

- **Diagonal artefacts in stroked paths** (`draw.go`, `path.go`): `tessellate()` returned a flat `[]rl.Vector2`, causing `DrawSplineLinear` to connect `MoveTo` discontinuities with spurious lines. Now returns `[][]rl.Vector2` (one sub-path per `MoveTo`); each sub-path is drawn independently.
- **Aliased lines** (`draw.go`): all strokes were dispatched to `DrawSplineLinear` (thick polygon renderer). Two-point sub-paths now use `DrawLineEx` directly — sharper, sub-pixel-width capable.
- **Aliased Bézier curves** (`draw.go`, `shadow.go`): `DrawSplineSegmentBezierCubic` replaced by a gg-rendered anti-aliased texture cached by geometry+lineWidth. Color and alpha applied as tint at blit time so animated opacity doesn't bust the cache.
- **Line and shape antialiasing** (`context.go`): `rl.EnableSmoothLines()` called in `BeginFrame`, enabling `GL_LINE_SMOOTH` globally for all line primitives at no per-draw cost.
- **Font registration silently failing in examples**: all examples now use the embedded `internal/fonts` package (Inter + Fira Code via `//go:embed`), guaranteed present on any platform.
- **Shadow cache unbounded**: shadow and Bézier texture cache capped at 512 entries with FIFO eviction and `rl.UnloadTexture` on eviction. Color cache capped at 256 entries.

---

## 0.1.0 — 2026-05-18

Initial scaffold release.

### Library

- `Context` type with full save/restore state stack
- CSS color string parser with cache (`color.go`)
- Font registry and CSS font string parser with cache (`font.go`)
  - Per-size atlas baking via `rl.LoadFontFromMemory`
  - Weight/italic variant support; graceful fallback chain
- Path accumulator (`path.go`)
  - `MoveTo`, `LineTo`, `Arc`, `ArcTo`, `BezierCurveTo`, `Rect`, `RoundRect`, `ClosePath`
  - Tessellation to `[]rl.Vector2` for fill/stroke dispatch
  - `arcTo` geometry (tangent-point construction)
  - Cubic Bézier tessellation
  - Rounded rect tessellation
- Draw primitives (`draw.go`)
  - `FillRect`, `StrokeRect`, `ClearRect`
  - `Fill`, `Stroke` with single-segment fast paths
  - `FillText`, `MeasureText` with textAlign/textBaseline
  - Dashed stroke via polyline segmentation
  - `BezierCurveTo` stroke dispatched to `rl.DrawSplineSegmentBezierCubic`
- Transform stack (`transform.go`)
  - `Translate`, `Scale`, `ResetTransform`, `SetTransform`
  - `matrix3x2` with `transformPoint`
- Clip system (`clip.go`)
  - Rectangular clips via `rl.BeginScissorMode` with intersection stack
  - `roundRect` → clip via gg alpha-mask pipeline
- Shadow pipeline (`shadow.go`)
  - CPU-side blur via `rl.ImageBlurGaussian` + gg
  - Cached `rl.Texture2D` per unique shadow configuration
- SVG icon registry (`icon.go`)
  - oksvg + rasterx rasterisation at startup
  - White-on-transparent pre-bake for tint-at-draw-time
- Off-screen contexts (`image.go`)
  - `NewOffscreen` backed by `rl.RenderTexture2D`
  - `DrawImage`, `DrawImageScaled`, `DrawImageCropped`
  - `DrawImageOffscreen` / `DrawImageOffscreenCropped` with Y-flip correction
- Shared cache (`cache.go`) with RW-mutex for thread-safe startup loading
- Style setters: `SetFillStyle`, `SetStrokeStyle`, `SetLineWidth`,
  `SetGlobalAlpha`, `SetFont`, `SetTextAlign`, `SetTextBaseline`,
  `SetLineDash`, `SetLineDashOffset`, `SetShadowColor`, `SetShadowBlur`,
  `SetShadowOffsetX`, `SetShadowOffsetY`, `SetLineCap`, `SetLineJoin`,
  `SetDirection`, `SetImageSmoothingEnabled`

### Examples

- `basic` — FillRect, StrokeRect, globalAlpha, save/restore, RoundRect
- `text` — Font variants, MeasureText, textAlign/textBaseline, word-wrap
- `paths` — Arc, ArcTo, LineDash, nested clip, animated sweep
- `curves` — BezierCurveTo with shevo-style glow+stroke+pulse pattern
- `grid` — Full joxel-style spreadsheet: zoom transform, nested clip, headers

### Build system

- `Makefile` with `build`, `examples`, per-example run targets, `lint`, `fmt`, `clean`
- `release.sh` — version bump, consistency check, zip packaging
- `pkg/version/version.go` synced to `VERSION` by `release.sh`
