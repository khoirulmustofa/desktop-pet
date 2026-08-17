package behavior

import (
	"math/rand"
)

// Action is what the pet chooses to do next after an idle period.
type Action int

const (
	StayIdle Action = iota
	Walk
	Jump
	Sleep
	Happy
)

func (a Action) String() string {
	switch a {
	case StayIdle:
		return "STAY_IDLE"
	case Walk:
		return "WALK"
	case Jump:
		return "JUMP"
	case Sleep:
		return "SLEEP"
	case Happy:
		return "HAPPY"
	}
	return "UNKNOWN"
}

// Weights holds the weighted probabilities for the five actions.
type Weights struct {
	Idle  int
	Walk  int
	Jump  int
	Happy int
	Sleep int
}

func (w Weights) Total() int { return w.Idle + w.Walk + w.Jump + w.Happy + w.Sleep }

// Pick returns a random action weighted by the given weights. A zero/negative
// total falls back to StayIdle.
func (w Weights) Pick(r *rand.Rand) Action {
	total := w.Total()
	if total <= 0 {
		return StayIdle
	}
	n := r.Intn(total)
	switch {
	case n < w.Idle:
		return StayIdle
	case n < w.Idle+w.Walk:
		return Walk
	case n < w.Idle+w.Walk+w.Jump:
		return Jump
	case n < w.Idle+w.Walk+w.Jump+w.Happy:
		return Happy
	default:
		return Sleep
	}
}

// Engine drives the behavior loop: produce random idle durations and pick
// weighted next actions. The caller (pet loop) owns the actual timing.
type Engine struct {
	rng     *rand.Rand
	idleMin float64
	idleMax float64
}

func NewEngine(idleMin, idleMax float64, rng *rand.Rand) *Engine {
	if rng == nil {
		rng = rand.New(rand.NewSource(42))
	}
	return &Engine{rng: rng, idleMin: idleMin, idleMax: idleMax}
}

// RandomIdleDuration returns a random idle wait in [min, max].
func (e *Engine) RandomIdleDuration() float64 {
	if e.idleMax <= e.idleMin {
		return e.idleMin
	}
	return e.idleMin + e.rng.Float64()*(e.idleMax-e.idleMin)
}

// PickAction chooses the next action with the given weights.
func (e *Engine) PickAction(w Weights) Action {
	return w.Pick(e.rng)
}
