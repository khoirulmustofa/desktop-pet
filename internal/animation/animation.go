package animation

import "image"

// State is the pet's behavior state. It maps 1:1 to an animation.
type State int

const (
	Idle State = iota
	WalkLeft
	WalkRight
	Jump
	Sleep
	Happy
)

var stateNames = [...]string{
	"IDLE", "WALK_LEFT", "WALK_RIGHT", "JUMP", "SLEEP", "HAPPY",
}

func (s State) String() string {
	if int(s) < len(stateNames) {
		return stateNames[s]
	}
	return "UNKNOWN"
}

// Animation is a named set of frames played at a fixed rate.
type Animation struct {
	Name   string
	Frames []*image.RGBA
	FPS    int
	Loop   bool // false = play once, then stay on the last frame
}

func (a *Animation) frameDuration() float64 {
	if a.FPS <= 0 {
		return 0.033
	}
	return 1.0 / float64(a.FPS)
}

// Player advances a single animation over time.
type Player struct {
	anim     *Animation
	index    int
	elapsed  float64
	finished bool
}

// SetAnimation switches the active animation and resets playback.
func (p *Player) SetAnimation(a *Animation) {
	p.anim = a
	p.index = 0
	p.elapsed = 0
	p.finished = false
}

// Advance moves playback by dt seconds. It reports whether the visible frame
// changed so the renderer can skip redraws.
func (p *Player) Advance(dt float64) bool {
	if p.anim == nil || len(p.anim.Frames) == 0 {
		return false
	}
	if p.finished {
		return false
	}
	old := p.index
	p.elapsed += dt
	n := len(p.anim.Frames)
	if p.anim.Loop {
		idx := int(p.elapsed / p.anim.frameDuration())
		p.index = idx % n
	} else {
		idx := int(p.elapsed / p.anim.frameDuration())
		if idx >= n {
			p.index = n - 1
			p.finished = true
		} else {
			p.index = idx
		}
	}
	return p.index != old
}

// Frame returns the current frame (nil when no animation is set).
func (p *Player) Frame() *image.RGBA {
	if p.anim == nil || len(p.anim.Frames) == 0 {
		return nil
	}
	return p.anim.Frames[p.index]
}

// Finished reports whether a non-looping animation has played to the end.
func (p *Player) Finished() bool { return p.finished }

// Index returns the current frame index.
func (p *Player) Index() int { return p.index }
