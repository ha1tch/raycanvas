package raycanvas

// Test harness for the tests that need a real OpenGL context.
//
// raylib's context belongs to the thread that opened the window, and Go runs
// each test on its own goroutine, so TestMain keeps the main thread for
// OpenGL and the tests hand it closures through onGL. Without a display
// (DISPLAY and WAYLAND_DISPLAY both unset on Linux and the BSDs) no window
// is opened and every test that calls onGL is skipped; the pure tests still
// run. RAYCANVAS_TEST_NOGL=1 forces that mode anywhere.

import (
	"image"
	"image/color"
	"os"
	"runtime"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	glJobs  chan func()
	glReady bool
)

func init() { runtime.LockOSThread() }

func haveDisplay() bool {
	if os.Getenv("RAYCANVAS_TEST_NOGL") != "" {
		return false
	}
	switch runtime.GOOS {
	case "darwin", "windows":
		return true
	}
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

func TestMain(m *testing.M) {
	if !haveDisplay() {
		os.Exit(m.Run())
	}
	rl.SetTraceLogLevel(rl.LogWarning)
	rl.SetConfigFlags(rl.FlagWindowHidden)
	rl.InitWindow(320, 240, "raycanvas tests")
	glReady = rl.IsWindowReady()
	glJobs = make(chan func())
	done := make(chan int)
	go func() { done <- m.Run() }()
	for {
		select {
		case job := <-glJobs:
			job()
		case code := <-done:
			if glReady {
				rl.CloseWindow()
			}
			os.Exit(code)
		}
	}
}

// onGL runs fn on the thread that owns the OpenGL context and waits for it.
func onGL(t *testing.T, fn func()) {
	t.Helper()
	if !glReady {
		t.Skip("no display: OpenGL tests skipped")
	}
	finished := make(chan struct{})
	glJobs <- func() {
		defer close(finished)
		fn()
	}
	<-finished
}

// snapshot reads an off-screen context back as an image with the origin at
// the top left, the way the context was drawn.
func snapshot(c *Context) *image.RGBA {
	img := rl.LoadImageFromTexture(c.Texture())
	defer rl.UnloadImage(img)
	return flipped(img.ToImage())
}

func snapshotTexture(tex rl.Texture2D) *image.RGBA {
	img := rl.LoadImageFromTexture(tex)
	defer rl.UnloadImage(img)
	return flipped(img.ToImage())
}

func flipped(src image.Image) *image.RGBA {
	b := src.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out.Set(x, b.Dy()-1-y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}

func near(a, b color.RGBA, tol int) bool {
	d := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return d(a.R, b.R) <= tol && d(a.G, b.G) <= tol && d(a.B, b.B) <= tol && d(a.A, b.A) <= tol
}

// wantPixel fails unless the pixel at (x, y) is within tol of want.
func wantPixel(t *testing.T, img *image.RGBA, x, y int, want color.RGBA, tol int, what string) {
	t.Helper()
	got := img.RGBAAt(x, y)
	if !near(got, want, tol) {
		t.Errorf("%s: pixel (%d,%d) = %v, want %v", what, x, y, got, want)
	}
}

var (
	white = color.RGBA{255, 255, 255, 255}
	black = color.RGBA{0, 0, 0, 255}
	navy  = color.RGBA{0x20, 0x40, 0x60, 255}
	clear = color.RGBA{}
)
