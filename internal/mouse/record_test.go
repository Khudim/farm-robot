package mouse

import (
	"testing"
	"time"
)

func TestSegmenterSplitsOnPause(t *testing.T) {
	g := &Segmenter{Still: 300 * time.Millisecond, MinPath: 8}
	at := time.Unix(0, 0)
	x, y := 100, 100
	step := func(dx, dy int) {
		at = at.Add(8 * time.Millisecond)
		x, y = x+dx, y+dy
		g.Add(at, x, y)
	}
	g.Add(at, x, y)
	for range 50 { // покой
		step(0, 0)
	}
	for range 20 { // первое движение вправо на 100
		step(5, 0)
	}
	for range 50 { // пауза 400 мс
		step(0, 0)
	}
	for range 10 { // второе — вниз на 30
		step(0, 3)
	}
	step(1, 0) // дрожь сразу после — часть второго движения
	g.Flush()

	s := g.Gestures()
	if len(s) != 2 {
		t.Fatalf("движений %d, ожидалось 2", len(s))
	}
	if nx, ny := s[0].Net(); nx != 100 || ny != 0 {
		t.Errorf("первое пришло в (%d,%d), ожидалось (100,0)", nx, ny)
	}
	if nx, ny := s[1].Net(); nx != 1 || ny != 30 {
		t.Errorf("второе пришло в (%d,%d), ожидалось (1,30)", nx, ny)
	}
	if d := s[0].Duration(); d < 150*time.Millisecond || d > 200*time.Millisecond {
		t.Errorf("первое длилось %v, ожидалось ~160ms", d)
	}
}

func TestSegmenterDropsJitter(t *testing.T) {
	g := &Segmenter{Still: 300 * time.Millisecond, MinPath: 8}
	at := time.Unix(0, 0)
	g.Add(at, 0, 0)
	for i := range 6 {
		at = at.Add(8 * time.Millisecond)
		g.Add(at, i%2, 0)
	}
	g.Flush()
	if n := len(g.Gestures()); n != 0 {
		t.Fatalf("дрожь на 1 px записалась как %d движений", n)
	}
}
