package renderer

import (
	"image"

	"desktop-pet/internal/window"
)

// Renderer blits RGBA frames to the layered window. It only draws when
// Draw is called (frame changed / position moved), so an idle pet causes
// almost no CPU usage.
type Renderer struct {
	w   *window.Window
	buf []byte
	wpx int
	hpx int
}

func New(w *window.Window) *Renderer { return &Renderer{w: w} }

// Ensure allocates the scratch buffer for the given pixel size.
func (r *Renderer) Ensure(width, height int) {
	n := width * height * 4
	if r.buf == nil || len(r.buf) < n {
		r.buf = make([]byte, n)
		r.wpx = width
		r.hpx = height
	}
}

// Draw renders img (screen position x, y) with per-pixel alpha and the given
// global opacity (0..255).
func (r *Renderer) Draw(img *image.RGBA, x, y int, opacity byte) {
	if img == nil {
		return
	}
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	r.Ensure(w, h)
	convertPremultipliedBGRA(img, r.buf)
	r.w.UpdateLayered(x, y, w, h, r.buf, opacity)
}

// convertPremultipliedBGRA copies a Go RGBA image into a Windows-compatible
// premultiplied BGRA byte buffer.
func convertPremultipliedBGRA(src *image.RGBA, dst []byte) {
	s := src.Pix
	si := 0
	di := 0
	for i := 0; i < len(src.Pix); i += 4 {
		r := s[si]
		g := s[si+1]
		b := s[si+2]
		a := s[si+3]
		if a == 0 {
			dst[di] = 0
			dst[di+1] = 0
			dst[di+2] = 0
			dst[di+3] = 0
		} else if a == 255 {
			dst[di] = b
			dst[di+1] = g
			dst[di+2] = r
			dst[di+3] = 255
		} else {
			ar := int(a)
			dst[di] = byte(int(b) * ar / 255)
			dst[di+1] = byte(int(g) * ar / 255)
			dst[di+2] = byte(int(r) * ar / 255)
			dst[di+3] = a
		}
		si += 4
		di += 4
	}
}
