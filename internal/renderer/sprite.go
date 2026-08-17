package renderer

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"math"
	"sort"

	"desktop-pet/internal/animation"
)

// LoadAnimation reads <name>/NN.png frames from fsys under the given prefix
// and returns a ready-to-play animation scaled by scale.
func LoadAnimation(fsys fs.FS, prefix, name string, fps int, loop bool, scale float64) (*animation.Animation, error) {
	dir := prefix + "/" + name
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var frames []*image.RGBA
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := fs.ReadFile(fsys, dir+"/"+e.Name())
		if err != nil {
			return nil, err
		}
		img, err := decodeRGBA(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if scale != 1 {
			img = ScaleRGBA(img, scale)
		}
		frames = append(frames, img)
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("animation %q has no frames", name)
	}
	return &animation.Animation{Name: name, Frames: frames, FPS: fps, Loop: loop}, nil
}

// ScaleRGBA rescales a frame with bilinear sampling.
func ScaleRGBA(src *image.RGBA, s float64) *image.RGBA { return scaleRGBA(src, s) }

func decodeRGBA(data []byte) (*image.RGBA, error) {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst, nil
}

// scaleRGBA rescales a frame with bilinear sampling.
func scaleRGBA(src *image.RGBA, s float64) *image.RGBA {
	sw := src.Rect.Dx()
	sh := src.Rect.Dy()
	if sw == 0 || sh == 0 {
		return src
	}
	dw := int(math.Round(float64(sw) * s))
	dh := int(math.Round(float64(sh) * s))
	if dw <= 0 {
		dw = 1
	}
	if dh <= 0 {
		dh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for dy := 0; dy < dh; dy++ {
		sy := float64(dy) * float64(sh-1) / float64(dh-1)
		y0 := int(sy)
		y1 := y0 + 1
		if y1 > sh-1 {
			y1 = sh - 1
		}
		fy := sy - float64(y0)
		for dx := 0; dx < dw; dx++ {
			sx := float64(dx) * float64(sw-1) / float64(dw-1)
			x0 := int(sx)
			x1 := x0 + 1
			if x1 > sw-1 {
				x1 = sw - 1
			}
			fx := sx - float64(x0)

			c00 := pixAt(src, x0, y0)
			c10 := pixAt(src, x1, y0)
			c01 := pixAt(src, x0, y1)
			c11 := pixAt(src, x1, y1)

			dst.SetRGBA(dx, dy, blend2x2(c00, c10, c01, c11, fx, fy))
		}
	}
	return dst
}

func pixAt(img *image.RGBA, x, y int) [4]float64 {
	o := y*img.Stride + x*4
	return [4]float64{float64(img.Pix[o]), float64(img.Pix[o+1]), float64(img.Pix[o+2]), float64(img.Pix[o+3])}
}

func blend2x2(c00, c10, c01, c11 [4]float64, fx, fy float64) color.RGBA {
	var out [4]float64
	for i := 0; i < 4; i++ {
		top := c00[i]*(1-fx) + c10[i]*fx
		bot := c01[i]*(1-fx) + c11[i]*fx
		out[i] = top*(1-fy) + bot*fy
	}
	return color.RGBA{uint8(out[0] + 0.5), uint8(out[1] + 0.5), uint8(out[2] + 0.5), uint8(out[3] + 0.5)}
}
