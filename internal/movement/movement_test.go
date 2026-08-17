package movement

import (
	"math"
	"testing"
)

func TestWalkRight(t *testing.T) {
	m := Mover{X: 0, Width: 10, Dir: Right, Speed: 100, RightMax: 1000}
	if m.Step(0.5) {
		t.Fatal("should not hit boundary")
	}
	if m.X != 50 {
		t.Fatalf("x=%v want 50", m.X)
	}
}

func TestBoundaryLeft(t *testing.T) {
	m := Mover{X: 20, Width: 10, Dir: Left, Speed: 100, LeftMin: 0}
	if !m.Step(1.0) {
		t.Fatal("should hit left boundary")
	}
	if m.X != 0 {
		t.Fatalf("clamped to %v want 0", m.X)
	}
	if m.Dir != Right {
		t.Fatal("direction should flip to Right")
	}
}

func TestBoundaryRight(t *testing.T) {
	m := Mover{X: 995, Width: 10, Dir: Right, Speed: 100, RightMax: 1000}
	if !m.Step(1.0) {
		t.Fatal("should hit right boundary")
	}
	if m.X != 990 {
		t.Fatalf("clamped to %v want 990", m.X)
	}
	if m.Dir != Left {
		t.Fatal("direction should flip to Left")
	}
}

func TestSpeedIndependentOfFrameRate(t *testing.T) {
	// distance over 1 second at 60fps must match distance at 30fps
	d1 := 0.0
	m1 := Mover{X: 0, Width: 10, Dir: Right, Speed: 50, RightMax: 1e9}
	for i := 0; i < 60; i++ {
		m1.Step(1.0 / 60)
	}
	d1 = m1.X

	d2 := 0.0
	m2 := Mover{X: 0, Width: 10, Dir: Right, Speed: 50, RightMax: 1e9}
	for i := 0; i < 30; i++ {
		m2.Step(1.0 / 30)
	}
	d2 = m2.X

	if math.Abs(d1-d2) > 0.001 {
		t.Fatalf("distance differs: %v vs %v", d1, d2)
	}
}
