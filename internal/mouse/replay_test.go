package mouse

import (
	"math"
	"testing"
	"time"
)

type sumMover struct{ x, y, steps int }

func (m *sumMover) Glide(dx, dy int, _, _ time.Duration) error { m.x += dx; m.y += dy; return nil }
func (m *sumMover) Step(dx, dy int) error                      { m.x += dx; m.y += dy; m.steps++; return nil }

// Записанный переход, подогнанный под цель, приходит в цель при любом
// направлении и длине.
func TestReplayTravelLandsOnTarget(t *testing.T) {
	var g Gesture
	for i := 0; i <= 30; i++ { // дуга вправо на 200 px за 300 мс
		u := float64(i) / 30
		g = append(g, Sample{T: u * 300, X: int(200 * u), Y: int(-40 * math.Sin(math.Pi*u))})
	}
	defer UseGestures(nil)
	if travel, _ := UseGestures([]Gesture{g}); travel != 1 {
		t.Fatalf("переход не распознан")
	}
	m := &sumMover{}
	mover = m
	defer func() { mover = nil }()

	for _, d := range [][2]int{{200, 0}, {-150, 90}, {0, -400}, {35, 20}} {
		m.x, m.y = 0, 0
		if !replayTravel(d[0], d[1]) {
			t.Fatalf("%v: не воспроизвёлся", d)
		}
		if abs(m.x-d[0]) > 1 || abs(m.y-d[1]) > 1 {
			t.Errorf("цель %v, пришли в (%d,%d)", d, m.x, m.y)
		}
	}
}

func TestLoopIsNotTravel(t *testing.T) {
	var g Gesture
	for i := 0; i <= 40; i++ { // круг: пришёл туда же, откуда начал
		th := 2 * math.Pi * float64(i) / 40
		g = append(g, Sample{T: float64(i) * 10, X: int(50 * (math.Cos(th) - 1)), Y: int(50 * math.Sin(th))})
	}
	defer UseGestures(nil)
	if travel, idle := UseGestures([]Gesture{g}); travel != 0 || idle != 1 {
		t.Fatalf("круг: переходов %d, шевелений %d; ожидалось 0 и 1", travel, idle)
	}
}
