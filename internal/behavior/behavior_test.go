package behavior

import (
	"math/rand"
	"testing"
)

func TestWeightsTotalAndPick(t *testing.T) {
	w := Weights{Idle: 30, Walk: 30, Jump: 20, Happy: 10, Sleep: 10}
	if w.Total() != 100 {
		t.Fatalf("total=%d want 100", w.Total())
	}
	rng := rand.New(rand.NewSource(1))
	seen := map[Action]int{}
	for i := 0; i < 10000; i++ {
		seen[w.Pick(rng)]++
	}
	// all five actions should be reachable
	for _, a := range []Action{StayIdle, Walk, Jump, Happy, Sleep} {
		if seen[a] == 0 {
			t.Fatalf("action %v never picked", a)
		}
	}
}

func TestWeightsZeroFallsBack(t *testing.T) {
	w := Weights{}
	if w.Pick(rand.New(rand.NewSource(1))) != StayIdle {
		t.Fatal("zero weights must fall back to StayIdle")
	}
}

func TestRandomIdleDurationWithinRange(t *testing.T) {
	e := NewEngine(5, 15, rand.New(rand.NewSource(2)))
	for i := 0; i < 1000; i++ {
		d := e.RandomIdleDuration()
		if d < 5 || d > 15 {
			t.Fatalf("idle duration %v out of range", d)
		}
	}
}
