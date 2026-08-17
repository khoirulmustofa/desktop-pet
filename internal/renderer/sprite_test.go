package renderer

import (
	"image"
	"image/color"
	"testing"

	"desktop-pet"
)

func newImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	return img
}

func TestEmbeddedAnimationsHaveAlpha(t *testing.T) {
	cases := []struct {
		name string
		fps  int
		loop bool
		min  int // minimum frame count
	}{
		{"idle", 3, true, 3},
		{"walk", 10, true, 4},
		{"jump", 12, false, 3},
		{"sleep", 2, true, 2},
		{"happy", 8, false, 3},
	}
	for _, c := range cases {
		a, err := LoadAnimation(assets.Pet, "assets/pet", c.name, c.fps, c.loop, 1.0)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if len(a.Frames) < c.min {
			t.Fatalf("%s: got %d frames, want >= %d", c.name, len(a.Frames), c.min)
		}
		f := a.Frames[0]
		b := f.Bounds()
		if b.Dx() <= 0 || b.Dy() <= 0 {
			t.Fatalf("%s: bad bounds %v", c.name, b)
		}
		// The sprite body must contain opaque pixels.
		var opaque int
		for y := 0; y < b.Dy(); y++ {
			for x := 0; x < b.Dx(); x++ {
				if f.RGBAAt(x, y).A == 255 {
					opaque++
				}
			}
		}
		if opaque < 100 {
			t.Fatalf("%s: only %d opaque pixels, sprite looks empty", c.name, opaque)
		}
		// WalkLeft and WalkRight share the walk frames.
		if a.Frames[0].Rect.Dx() != a.Frames[0].Rect.Dy() {
			t.Fatalf("%s: non-square frame", c.name)
		}
	}
}

func TestScaleRGBA(t *testing.T) {
	src := scaleRGBA(newImage(10, 8), 2.0)
	b := src.Bounds()
	if b.Dx() != 20 || b.Dy() != 16 {
		t.Fatalf("scaled to %dx%d, want 20x16", b.Dx(), b.Dy())
	}
	src2 := scaleRGBA(newImage(10, 8), 1.0)
	if src2.Bounds() != newImage(10, 8).Bounds() {
		t.Fatalf("identity scale changed bounds")
	}
}
