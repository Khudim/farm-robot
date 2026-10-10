package mouse

import (
	"math"
	"testing"
)

// Фигуры замкнуты: иначе шевеления копят снос и мышь уползает с поплавка.
func TestFiguresAreClosed(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		name, curve, dur := RandomFigure(50)
		seen[name] = true
		if curve == nil {
			continue
		}
		if dur <= 0 {
			t.Fatalf("%s: длительность %v", name, dur)
		}
		for _, at := range []float64{0, 1} {
			if x, y := curve(at); math.Hypot(x, y) > 0.5 {
				t.Fatalf("%s: в t=%v смещение (%.1f, %.1f), фигура не замкнута", name, at, x, y)
			}
		}
		for i := 0; i <= 100; i++ {
			if x, y := curve(float64(i) / 100); math.Hypot(x, y) > 100 {
				t.Fatalf("%s: ушла на (%.0f, %.0f) при размахе 50", name, x, y)
			}
		}
	}
	for _, name := range []string{"восьмёрка", "влево-вправо", "круг", "сдвиг"} {
		if !seen[name] {
			t.Errorf("фигура %q ни разу не выпала", name)
		}
	}
}
