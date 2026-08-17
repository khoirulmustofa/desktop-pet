package movement

// Direction is the horizontal movement direction.
type Direction int

const (
	Left Direction = iota
	Right
)

func (d Direction) Other() Direction {
	if d == Left {
		return Right
	}
	return Left
}

// Mover moves a pet horizontally with a per-second speed, clamped to bounds.
type Mover struct {
	X        float64
	Width    float64
	Dir      Direction
	Speed    float64 // pixels per second
	LeftMin  float64
	RightMax float64
}

// Step advances by dt seconds. Returns true when a screen boundary was hit
// (direction was flipped at the boundary).
func (m *Mover) Step(dt float64) bool {
	dx := m.Speed * dt
	if m.Dir == Left {
		dx = -dx
	}
	m.X += dx

	if m.X <= m.LeftMin {
		m.X = m.LeftMin
		m.Dir = Right
		return true
	}
	if m.X+m.Width >= m.RightMax {
		m.X = m.RightMax - m.Width
		m.Dir = Left
		return true
	}
	return false
}
