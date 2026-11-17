package raycanvas

import (
	"image"
	"image/color"
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Cache size constants.
const (
	// ShadowCacheMax is the maximum number of entries in the shadow+bezier
	// texture cache. With integer-snapped blur values, the set of distinct
	// animated shadow textures is bounded by the blur range (typically 1–30),
	// so 1024 is large enough that eviction is rare in practice.
	ShadowCacheMax = 1024

	// ColorCacheMax is the maximum number of CSS color strings cached.
	// Shevo has ~60 distinct strings per theme; 256 is a generous ceiling.
	ColorCacheMax = 256
)

// SharedCache holds all pre-computed GPU resources. A single instance is
// shared across all Context values within an application. All methods are
// safe for concurrent use during asset loading; frame-loop access is
// read-only after startup and requires no locking.
type SharedCache struct {
	mu sync.RWMutex

	colors     map[string]color.RGBA
	colorQueue []string // FIFO eviction queue for colors

	fonts map[FontKey]rl.Font

	shadows     map[string]rl.Texture2D // shadow + bezier textures
	shadowQueue []string                // FIFO eviction queue for shadows

	// retired holds textures that have left the shadow cache and are waiting
	// to be freed; freed counts those that have been. See collect.
	retired []rl.Texture2D
	freed   int

	icons      map[iconKey]rl.Texture2D
	maskImages map[string]*image.RGBA
}

// NewSharedCache allocates an empty cache.
func NewSharedCache() *SharedCache {
	return &SharedCache{
		colors:      make(map[string]color.RGBA, ColorCacheMax),
		colorQueue:  make([]string, 0, ColorCacheMax),
		fonts:       make(map[FontKey]rl.Font),
		shadows:     make(map[string]rl.Texture2D, ShadowCacheMax),
		shadowQueue: make([]string, 0, ShadowCacheMax),
		icons:       make(map[iconKey]rl.Texture2D),
		maskImages:  make(map[string]*image.RGBA),
	}
}

// Unload releases all GPU resources held by the cache.
// Call once, after the raylib window is still open but before CloseWindow.
func (sc *SharedCache) Unload() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	for _, f := range sc.fonts {
		rl.UnloadFont(f)
	}
	for _, t := range sc.shadows {
		rl.UnloadTexture(t)
	}
	for _, t := range sc.icons {
		rl.UnloadTexture(t)
	}
	for _, t := range sc.retired {
		rl.UnloadTexture(t)
	}
	sc.retired = sc.retired[:0]
	sc.fonts = make(map[FontKey]rl.Font)
	sc.shadows = make(map[string]rl.Texture2D, ShadowCacheMax)
	sc.shadowQueue = sc.shadowQueue[:0]
	sc.icons = make(map[iconKey]rl.Texture2D)
}

// --- color cache -------------------------------------------------------------

func (sc *SharedCache) lookupColor(css string) (color.RGBA, bool) {
	sc.mu.RLock()
	c, ok := sc.colors[css]
	sc.mu.RUnlock()
	return c, ok
}

func (sc *SharedCache) storeColor(css string, c color.RGBA) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if _, exists := sc.colors[css]; exists {
		sc.colors[css] = c
		return
	}
	if len(sc.colorQueue) >= ColorCacheMax {
		// Evict oldest entry — no GPU resource to unload, just drop the map entry.
		oldest := sc.colorQueue[0]
		sc.colorQueue = sc.colorQueue[1:]
		delete(sc.colors, oldest)
	}
	sc.colors[css] = c
	sc.colorQueue = append(sc.colorQueue, css)
}

// --- font cache --------------------------------------------------------------

// Fonts are registered at startup and never evicted.

func (sc *SharedCache) lookupFont(key FontKey) (rl.Font, bool) {
	sc.mu.RLock()
	f, ok := sc.fonts[key]
	sc.mu.RUnlock()
	return f, ok
}

func (sc *SharedCache) storeFont(key FontKey, f rl.Font) {
	sc.mu.Lock()
	sc.fonts[key] = f
	sc.mu.Unlock()
}

// --- shadow + bezier cache ---------------------------------------------------
//
// Both shadow textures and anti-aliased Bézier stroke textures share this
// cache. Keys are prefixed ("shadow|...", "bezier|...", "rrmask|...") to
// avoid collisions. Total capacity: ShadowCacheMax entries.
//
// Eviction policy: FIFO. When the cache is full, the oldest entry is unloaded
// from GPU memory and removed. FIFO is appropriate here because:
//   - The working set is small and stable during a session.
//   - True LRU requires O(1) move-to-front which adds complexity for no
//     measurable benefit at this cache size.

func (sc *SharedCache) lookupShadow(key string) (rl.Texture2D, bool) {
	sc.mu.RLock()
	t, ok := sc.shadows[key]
	sc.mu.RUnlock()
	return t, ok
}

func (sc *SharedCache) storeShadow(key string, t rl.Texture2D) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if old, exists := sc.shadows[key]; exists {
		// Update in place without touching the eviction queue.
		if old.ID != t.ID {
			sc.retired = append(sc.retired, old)
		}
		sc.shadows[key] = t
		return
	}
	if len(sc.shadowQueue) >= ShadowCacheMax {
		// Evict the oldest. Its texture is not freed here: a draw later in
		// this frame may have looked it up already, and a freed ID can be
		// handed straight to the next upload, which would then be drawn in
		// its place. It is retired instead and freed by collect at the
		// start of the next frame, when nothing can still be holding it.
		oldest := sc.shadowQueue[0]
		sc.shadowQueue = sc.shadowQueue[1:]
		sc.retired = append(sc.retired, sc.shadows[oldest])
		delete(sc.shadows, oldest)
	}
	sc.shadows[key] = t
	sc.shadowQueue = append(sc.shadowQueue, key)
}

// collect frees the textures retired from the shadow cache. BeginFrame
// calls it, so a texture evicted during one frame is freed at the start of
// the next.
func (sc *SharedCache) collect() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	for _, t := range sc.retired {
		if t.ID != 0 {
			rl.UnloadTexture(t)
			sc.freed++
		}
	}
	sc.retired = sc.retired[:0]
}

// --- icon cache -------------------------------------------------------------
//
// Icons are registered at startup and never evicted.

type iconKey struct {
	name string
	size float32
}

func (sc *SharedCache) lookupIcon(name string, size float32) (rl.Texture2D, bool) {
	sc.mu.RLock()
	t, ok := sc.icons[iconKey{name, size}]
	sc.mu.RUnlock()
	return t, ok
}

func (sc *SharedCache) storeIcon(name string, size float32, t rl.Texture2D) {
	sc.mu.Lock()
	sc.icons[iconKey{name, size}] = t
	sc.mu.Unlock()
}

// --- mask image cache (CPU-resident, never evicted) -------------------------

func (sc *SharedCache) lookupMaskImage(key string) (*image.RGBA, bool) {
	sc.mu.RLock()
	m, ok := sc.maskImages[key]
	sc.mu.RUnlock()
	return m, ok
}

func (sc *SharedCache) storeMaskImage(key string, m *image.RGBA) {
	sc.mu.Lock()
	sc.maskImages[key] = m
	sc.mu.Unlock()
}
