package animation

import (
	"image"
	"testing"
)

func testAnimation(name string, n, fps int, loop bool) *Animation {
	frames := make([]*image.RGBA, n)
	for i := range frames {
		frames[i] = image.NewRGBA(image.Rect(0, 0, 4, 4))
	}
	return &Animation{Name: name, Frames: frames, FPS: fps, Loop: loop}
}

func TestPlayerLoop(t *testing.T) {
	p := &Player{}
	p.SetAnimation(testAnimation("idle", 3, 3, true))
	for i := 0; i < 4; i++ {
		p.Advance(0.333)
	}
	if p.Index() != 0 {
		t.Fatalf("looped to index %d, want 0", p.Index())
	}
}

func TestPlayerPlayOnce(t *testing.T) {
	p := &Player{}
	p.SetAnimation(testAnimation("jump", 3, 10, false))
	// advance well past the end
	for i := 0; i < 10; i++ {
		p.Advance(0.1)
	}
	if !p.Finished() {
		t.Fatal("expected Finished")
	}
	if p.Index() != 2 {
		t.Fatalf("stuck at index %d, want last (2)", p.Index())
	}
}

func TestPlayerAdvanceNoChangeOnFinished(t *testing.T) {
	p := &Player{}
	p.SetAnimation(testAnimation("jump", 2, 5, false))
	for i := 0; i < 5; i++ {
		p.Advance(1)
	}
	if p.Advance(1) {
		t.Fatal("Advance should return false once finished")
	}
}

func TestStateMachineChangeSetsAnimation(t *testing.T) {
	anims := map[State]*Animation{
		Idle: testAnimation("idle", 1, 2, true),
	}
	m := &StateMachine{}
	// Initial state is Idle (zero value); Change must still bind the animation
	// even though the state value does not change.
	if m.Change(Idle, anims) {
		t.Fatal("expected changed=false when state is already Idle")
	}
	if m.Frame() == nil {
		t.Fatal("frame must be set after Change")
	}
}
