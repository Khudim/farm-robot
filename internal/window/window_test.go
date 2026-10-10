package window

import "testing"

func TestCentered(t *testing.T) {
	bounds := Rect{X: 100, Y: 50, W: 1280, H: 720}
	cases := map[string]struct{ cx, cy, wantX, wantY int }{
		"центр":      {740, 410, 610, 320},
		"левый верх": {100, 50, 100, 50},
		"правый низ": {1380, 770, 1120, 590},
	}
	for name, c := range cases {
		got := Centered(260, 180, c.cx, c.cy, bounds)
		if got.X != c.wantX || got.Y != c.wantY {
			t.Errorf("%s: (%d,%d), ожидалось (%d,%d)", name, got.X, got.Y, c.wantX, c.wantY)
		}
		if got.X < bounds.X || got.Y < bounds.Y ||
			got.X+got.W > bounds.X+bounds.W || got.Y+got.H > bounds.Y+bounds.H {
			t.Errorf("%s: %v вышел за %v", name, got, bounds)
		}
	}
}

// Область больше окна прижимается к началу, а не уезжает в отрицательные координаты.
func TestCenteredOversized(t *testing.T) {
	if got := Centered(500, 500, 100, 100, Rect{W: 200, H: 200}); got.X != 0 || got.Y != 0 {
		t.Errorf("ожидалось (0,0), получено (%d,%d)", got.X, got.Y)
	}
}
