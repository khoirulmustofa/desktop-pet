// Package spritegen procedurally draws the built-in chibi cat sprites.
// It is used by cmd/genassets to produce the committed assets/pet PNG files
// and as a runtime fallback when the embedded sprites are unavailable.
package spritegen

import (
	"image"
	"image/color"
	"math"
)

// Size is the natural sprite size in pixels.
const Size = 96

// Params parameterize a single cat drawing.
type Params struct {
	BodyScaleY float64 // breathing (1.0 neutral)
	BobY       float64 // vertical bob offset for walking
	EyeOpen    bool
	Smile      bool
	Sleeping   bool
	LegPhase   float64 // 0..1 walk cycle
	TailSwing  float64 // -1..1 tail sway
	Tuck       bool    // jump tuck
	ArmsUp     bool    // happy arms raised
	Zzz        bool
	Heart      bool
}

var (
	colOrange = color.RGBA{246, 165, 35, 255}
	colDark   = color.RGBA{74, 53, 32, 255}
	colBelly  = color.RGBA{255, 227, 179, 255}
	colInner  = color.RGBA{255, 159, 159, 255}
	colNose   = color.RGBA{232, 99, 60, 255}
	colEyeW   = color.RGBA{255, 255, 255, 255}
	colPupil  = color.RGBA{30, 30, 30, 255}
	colBlush  = color.RGBA{255, 170, 170, 255}
	colHeart  = color.RGBA{255, 80, 120, 255}
)

// ---------------------------------------------------------------------------
// Rendering primitives (SDF coverage + 4x4 supersampling)
// ---------------------------------------------------------------------------

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func smoothstep(e0, e1, x float64) float64 {
	t := clamp01((x - e0) / (e1 - e0))
	return t * t * (3 - 2*t)
}

// paint fills pixels with the coverage function sampled 4x4 per pixel,
// using proper straight-alpha source-over compositing.
func paint(dst *image.RGBA, x0, y0, x1, y1 int, cov func(x, y float64) float64, col color.RGBA) {
	ca := float64(col.A) / 255.0
	if ca <= 0 {
		return
	}
	cr := float64(col.R) / 255.0
	cg := float64(col.G) / 255.0
	cb := float64(col.B) / 255.0
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if x < 0 || y < 0 || x >= dst.Rect.Dx() || y >= dst.Rect.Dy() {
				continue
			}
			var sum float64
			for sy := 0; sy < 4; sy++ {
				for sx := 0; sx < 4; sx++ {
					sum += cov(float64(x)+float64(sx)*0.25+0.125, float64(y)+float64(sy)*0.25+0.125)
				}
			}
			covv := sum / 16.0
			sa := covv * ca
			if sa <= 0 {
				continue
			}
			o := y*dst.Stride + x*4
			dr := float64(dst.Pix[o]) / 255.0
			dg := float64(dst.Pix[o+1]) / 255.0
			db := float64(dst.Pix[o+2]) / 255.0
			da := float64(dst.Pix[o+3]) / 255.0
			outA := sa + da*(1-sa)
			if outA <= 0 {
				continue
			}
			invA := 1 - sa
			dst.Pix[o] = uint8(((cr*sa+dr*da*invA)/outA)*255 + 0.5)
			dst.Pix[o+1] = uint8(((cg*sa+dg*da*invA)/outA)*255 + 0.5)
			dst.Pix[o+2] = uint8(((cb*sa+db*da*invA)/outA)*255 + 0.5)
			dst.Pix[o+3] = uint8(outA*255 + 0.5)
		}
	}
}

func ellipse(dst *image.RGBA, cx, cy, rx, ry float64, col color.RGBA) {
	x0 := int(math.Floor(cx - rx - 1))
	x1 := int(math.Ceil(cx + rx + 1))
	y0 := int(math.Floor(cy - ry - 1))
	y1 := int(math.Ceil(cy + ry + 1))
	paint(dst, x0, y0, x1, y1, func(x, y float64) float64 {
		dx := (x - cx) / rx
		dy := (y - cy) / ry
		d := dx*dx + dy*dy
		if d >= 1 {
			return 0
		}
		return smoothstep(0.72, 1, 1-d)
	}, col)
}

func segDist(px, py, ax, ay, bx, by float64) float64 {
	dx := bx - ax
	dy := by - ay
	if dx == 0 && dy == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	t := ((px-ax)*dx + (py-ay)*dy) / (dx*dx + dy*dy)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}

func line(dst *image.RGBA, ax, ay, bx, by, width float64, col color.RGBA) {
	r := width / 2
	x0 := int(math.Floor(math.Min(ax, bx) - r - 1))
	x1 := int(math.Ceil(math.Max(ax, bx) + r + 1))
	y0 := int(math.Floor(math.Min(ay, by) - r - 1))
	y1 := int(math.Ceil(math.Max(ay, by) + r + 1))
	paint(dst, x0, y0, x1, y1, func(x, y float64) float64 {
		return smoothstep(r, r-1.2, segDist(x, y, ax, ay, bx, by))
	}, col)
}

func triInside(px, py, ax, ay, bx, by, cx, cy float64) bool {
	sign := func(p1x, p1y, p2x, p2y, p3x, p3y float64) float64 {
		return (p1x-p3x)*(p2y-p3y) - (p2x-p3x)*(p1y-p3y)
	}
	d1 := sign(px, py, ax, ay, bx, by)
	d2 := sign(px, py, bx, by, cx, cy)
	d3 := sign(px, py, cx, cy, ax, ay)
	hasNeg := d1 < 0 || d2 < 0 || d3 < 0
	hasPos := d1 > 0 || d2 > 0 || d3 > 0
	return !(hasNeg && hasPos)
}

func triangle(dst *image.RGBA, ax, ay, bx, by, cx, cy float64, col color.RGBA) {
	minx := int(math.Floor(math.Min(ax, math.Min(bx, cx)) - 1))
	maxx := int(math.Ceil(math.Max(ax, math.Max(bx, cx)) + 1))
	miny := int(math.Floor(math.Min(ay, math.Min(by, cy)) - 1))
	maxy := int(math.Ceil(math.Max(ay, math.Max(by, cy)) + 1))
	paint(dst, minx, miny, maxx, maxy, func(x, y float64) float64 {
		if triInside(x, y, ax, ay, bx, by, cx, cy) {
			return 1
		}
		return 0
	}, col)
}

func heart(dst *image.RGBA, cx, cy, s float64, col color.RGBA) {
	x0 := int(math.Floor(cx - 1.3*s))
	x1 := int(math.Ceil(cx + 1.3*s))
	y0 := int(math.Floor(cy - 1.3*s))
	y1 := int(math.Ceil(cy + 1.2*s))
	paint(dst, x0, y0, x1, y1, func(x, y float64) float64 {
		hx := (x - cx) / s
		hy := (y - cy) / s
		// classic heart implicit curve
		f := (hx*hx+hy*hy-1)*(hx*hx+hy*hy-1)*(hx*hx+hy*hy-1) - hx*hx*hy*hy*hy
		if f <= 0 {
			return 1
		}
		return 0
	}, col)
}

// ---------------------------------------------------------------------------
// The cat
// ---------------------------------------------------------------------------

func Draw(p Params) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, Size, Size))
	if p.Sleeping {
		drawSleeping(img, p)
	} else {
		drawStanding(img, p)
	}
	return img
}

func drawStanding(img *image.RGBA, p Params) {
	bob := p.BobY
	ry := 14 * p.BodyScaleY
	bodyCy := 70 + bob
	legShift := 0.0
	if p.LegPhase > 0 {
		// simple alternating leg stride
		legShift = math.Sin(p.LegPhase * 2 * math.Pi)
	}

	// tail (behind body)
	tailBaseX, tailBaseY := 60.0, 66+bob
	swing := p.TailSwing * 10
	tipX := tailBaseX + 18 + swing
	tipY := tailBaseY - 26 + math.Abs(swing)*0.5
	line(img, tailBaseX, tailBaseY, tipX, tipY, 8, colDark)
	line(img, tailBaseX, tailBaseY, tipX, tipY, 5.5, colOrange)

	// back legs
	if p.Tuck {
		ellipse(img, 39, 78+bob, 6, 4, colDark)
		ellipse(img, 57, 78+bob, 6, 4, colDark)
	} else {
		ellipse(img, 38+legShift*3, 80+bob, 6, 9, colDark)
		ellipse(img, 58-legShift*3, 80+bob, 6, 9, colDark)
	}

	// body
	ellipse(img, 48, bodyCy, 18.5, ry+1.5, colDark)
	ellipse(img, 48, bodyCy, 17, ry, colOrange)
	// belly
	ellipse(img, 48, bodyCy+6, 11, ry*0.55, colBelly)

	// front legs
	if p.Tuck {
		ellipse(img, 42, 78+bob, 5, 5, colDark)
		ellipse(img, 54, 78+bob, 5, 5, colDark)
	} else {
		front := legShift * 4
		ellipse(img, 41+front, 82+bob, 6, 8, colDark)
		ellipse(img, 55-front, 82+bob, 6, 8, colDark)
	}

	// ears (behind head)
	drawEars(img, 48, 40+bob, p.Sleeping)

	// head
	ellipse(img, 48, 42+bob, 22.5, 20.5, colDark)
	ellipse(img, 48, 42+bob, 21, 19, colOrange)

	// arms for happy
	if p.ArmsUp {
		ellipse(img, 28, 56+bob, 5, 8, colDark)
		ellipse(img, 28, 56+bob, 3.5, 6.5, colOrange)
		ellipse(img, 68, 56+bob, 5, 8, colDark)
		ellipse(img, 68, 56+bob, 3.5, 6.5, colOrange)
	}

	drawFace(img, 48, 42+bob, p)

	// heart when happy
	if p.Heart {
		heart(img, 70, 20, 9, colHeart)
	}
}

func drawEars(img *image.RGBA, hx, hy float64, _ bool) {
	// left ear
	triangle(img, hx-16, hy-14, hx-6, hy-30, hx+2, hy-12, colDark)
	triangle(img, hx-14, hy-13, hx-7, hy-26, hx+1, hy-12, colOrange)
	triangle(img, hx-11, hy-14, hx-7, hy-22, hx-1, hy-13, colInner)
	// right ear
	triangle(img, hx+2, hy-12, hx+6, hy-30, hx+16, hy-14, colDark)
	triangle(img, hx+1, hy-12, hx+7, hy-26, hx+14, hy-13, colOrange)
	triangle(img, hx+1, hy-13, hx+7, hy-22, hx+11, hy-14, colInner)
}

func drawFace(img *image.RGBA, hx, hy float64, p Params) {
	// eyes
	if p.EyeOpen {
		ellipse(img, hx-9, hy-2, 4.5, 5, colEyeW)
		ellipse(img, hx+9, hy-2, 4.5, 5, colEyeW)
		ellipse(img, hx-8.2, hy-1, 2, 2.6, colPupil)
		ellipse(img, hx+9.8, hy-1, 2, 2.6, colPupil)
	} else {
		line(img, hx-12, hy-1, hx-6, hy-1, 2, colDark)
		line(img, hx+6, hy-1, hx+12, hy-1, 2, colDark)
	}

	// blush
	ellipse(img, hx-15, hy+5, 3.5, 2, colBlush)
	ellipse(img, hx+15, hy+5, 3.5, 2, colBlush)

	// nose
	triangle(img, hx-2.5, hy+6, hx+2.5, hy+6, hx, hy+9, colNose)

	// mouth
	if p.Smile {
		// open smiling mouth
		ellipse(img, hx, hy+13, 5, 3.5, colDark)
		ellipse(img, hx, hy+12, 4, 2.5, colEyeW)
		ellipse(img, hx, hy+13.5, 2.5, 1.2, colInner)
	} else {
		line(img, hx-5, hy+11, hx, hy+13, 1.6, colDark)
		line(img, hx+5, hy+11, hx, hy+13, 1.6, colDark)
	}

	// whiskers
	line(img, hx-17, hy+2, hx-28, hy, 1.2, colDark)
	line(img, hx-17, hy+5, hx-28, hy+6, 1.2, colDark)
	line(img, hx+17, hy+2, hx+28, hy, 1.2, colDark)
	line(img, hx+17, hy+5, hx+28, hy+6, 1.2, colDark)
}

func drawSleeping(img *image.RGBA, p Params) {
	breath := p.BodyScaleY

	// curled tail at right
	line(img, 64, 66, 78, 72, 7, colDark)
	line(img, 64, 66, 78, 72, 5, colOrange)

	// body
	ry := 12 * breath
	ellipse(img, 46, 74, 28, ry+1.5, colDark)
	ellipse(img, 46, 74, 26, ry, colOrange)
	ellipse(img, 40, 78, 14, ry*0.6, colBelly)

	// ears flat on head
	triangle(img, 22, 60, 28, 52, 34, 62, colDark)
	triangle(img, 23, 61, 28, 55, 33, 62, colOrange)
	triangle(img, 26, 61, 28, 57, 32, 62, colInner)

	// head
	ellipse(img, 27, 68, 15, 13, colDark)
	ellipse(img, 27, 68, 13.5, 11.5, colOrange)

	// closed eyes
	line(img, 21, 66, 27, 66, 2, colDark)
	line(img, 29, 66, 35, 66, 2, colDark)
	// nose
	triangle(img, 26, 71, 29, 71, 27.5, 73, colNose)
	// whiskers
	line(img, 15, 70, 7, 69, 1.2, colDark)
	line(img, 15, 72, 7, 73, 1.2, colDark)

	// zzz
	if p.Zzz {
		line(img, 62, 30, 70, 30, 2.2, colDark)
		line(img, 70, 30, 62, 38, 2.2, colDark)
		line(img, 62, 38, 70, 38, 2.2, colDark)

		line(img, 74, 18, 80, 18, 1.8, colDark)
		line(img, 80, 18, 74, 24, 1.8, colDark)
		line(img, 74, 24, 80, 24, 1.8, colDark)

		line(img, 84, 8, 89, 8, 1.4, colDark)
		line(img, 89, 8, 84, 13, 1.4, colDark)
		line(img, 84, 13, 89, 13, 1.4, colDark)
	}
}

// ---------------------------------------------------------------------------
// Frame presets
// ---------------------------------------------------------------------------

func IdleFrames() []Params {
	return []Params{
		{BodyScaleY: 1.0, EyeOpen: true, TailSwing: 0.2},
		{BodyScaleY: 1.04, EyeOpen: true, TailSwing: 0.6},
		{BodyScaleY: 1.0, EyeOpen: false, TailSwing: -0.2},
	}
}

func WalkFrames() []Params {
	frames := make([]Params, 4)
	for i := 0; i < 4; i++ {
		phase := float64(i) / 4
		frames[i] = Params{
			BodyScaleY: 1.0,
			BobY:       math.Abs(math.Sin(phase*2*math.Pi)) * 2,
			EyeOpen:    true,
			LegPhase:   phase,
			TailSwing:  math.Sin(phase * 2 * math.Pi),
		}
	}
	return frames
}

func JumpFrames() []Params {
	return []Params{
		{BodyScaleY: 0.92, EyeOpen: true, Tuck: false, TailSwing: -0.5}, // crouch
		{BodyScaleY: 1.08, EyeOpen: true, Tuck: true, TailSwing: 0},     // stretch
		{BodyScaleY: 1.05, EyeOpen: true, Tuck: true, TailSwing: 0.5},   // mid-air
	}
}

func SleepFrames() []Params {
	return []Params{
		{Sleeping: true, BodyScaleY: 1.0, Zzz: false},
		{Sleeping: true, BodyScaleY: 1.04, Zzz: true},
	}
}

func HappyFrames() []Params {
	return []Params{
		{BodyScaleY: 1.0, EyeOpen: true, Smile: true, ArmsUp: true, Heart: true},
		{BodyScaleY: 1.06, EyeOpen: true, Smile: true, ArmsUp: true, Heart: true, BobY: -3},
		{BodyScaleY: 1.0, EyeOpen: true, Smile: true, Heart: true},
	}
}
