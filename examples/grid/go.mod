module github.com/ha1tch/raycanvas/examples/grid

go 1.22

require (
	github.com/gen2brain/raylib-go/raylib v0.60.0
	github.com/ha1tch/raycanvas v0.0.0
	github.com/ha1tch/raycanvas/examples/internal/fonts v0.0.0-00010101000000-000000000000
)

require (
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/jupiterrider/ffi v0.7.0 // indirect
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c // indirect
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	golang.org/x/image v0.0.0-20211028202545-6944b10bf410 // indirect
	golang.org/x/net v0.0.0-20211118161319-6a13c67c3ce4 // indirect
	golang.org/x/text v0.3.6 // indirect
)

replace (
	github.com/ha1tch/raycanvas => ../../
	github.com/ha1tch/raycanvas/examples/internal/fonts => ../internal/fonts
)
