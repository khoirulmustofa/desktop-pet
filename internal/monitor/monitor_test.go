package monitor

import "testing"

func testInfos() []Info {
	// Two side-by-side monitors: primary 1920x1200 at x=0, second at x=1920.
	return []Info{
		{Index: 0, WorkArea: WorkArea{Left: 0, Top: 0, Right: 1920, Bottom: 1152}, Primary: true},
		{Index: 1, WorkArea: WorkArea{Left: 1920, Top: 0, Right: 3840, Bottom: 1080}, Primary: false},
	}
}

func TestIndexAtInside(t *testing.T) {
	m := testInfos()
	if i := IndexAt(100, 600, m); i != 0 {
		t.Fatalf("point on monitor 1 resolved to %d", i)
	}
	if i := IndexAt(2000, 600, m); i != 1 {
		t.Fatalf("point on monitor 2 resolved to %d", i)
	}
	// bottom edge of monitor 2
	if i := IndexAt(3000, 1070, m); i != 1 {
		t.Fatalf("point near bottom of monitor 2 resolved to %d", i)
	}
}

func TestIndexAtNearestForOutsidePoints(t *testing.T) {
	m := testInfos()
	// far left of everything -> nearest is monitor 1
	if i := IndexAt(-5000, 600, m); i != 0 {
		t.Fatalf("left point resolved to %d, want 0", i)
	}
	// far right -> monitor 2
	if i := IndexAt(9000, 600, m); i != 1 {
		t.Fatalf("right point resolved to %d, want 1", i)
	}
}

func TestIndexAtEmpty(t *testing.T) {
	if i := IndexAt(100, 100, nil); i != -1 {
		t.Fatalf("empty list resolved to %d, want -1", i)
	}
}

func TestWorkAreaAccessors(t *testing.T) {
	w := WorkArea{Left: 0, Top: 0, Right: 1920, Bottom: 1152}
	if w.Width() != 1920 || w.Height() != 1152 {
		t.Fatalf("size mismatch: %dx%d", w.Width(), w.Height())
	}
}
