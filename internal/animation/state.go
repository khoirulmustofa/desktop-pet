package animation

import "image"

// StateMachine tracks the current state and animates between states.
type StateMachine struct {
	state  State
	player Player
}

func (m *StateMachine) State() State { return m.state }

// Change sets a new state and its animation. Returns true on state change.
func (m *StateMachine) Change(s State, anims map[State]*Animation) bool {
	changed := m.state != s
	m.state = s
	if a, ok := anims[s]; ok {
		m.player.SetAnimation(a)
	}
	return changed
}

// Tick advances the active animation. Returns true when the visible frame
// changed.
func (m *StateMachine) Tick(dt float64) bool {
	return m.player.Advance(dt)
}

func (m *StateMachine) Frame() *image.RGBA { return m.player.Frame() }

// Finished reports whether a non-looping animation finished playing.
func (m *StateMachine) Finished() bool { return m.player.Finished() }
