// Package window находит окно игры и отдаёт его клиентскую область в экранных
// координатах, чтобы область поиска задавалась относительно окна.
package window

import "fmt"

type Rect struct{ X, Y, W, H int }

func (r Rect) String() string { return fmt.Sprintf("%dx%d в (%d,%d)", r.W, r.H, r.X, r.Y) }

// Centered — прямоугольник w×h с центром в (cx, cy), прижатый к границам bounds.
func Centered(w, h, cx, cy int, bounds Rect) Rect {
	r := Rect{X: cx - w/2, Y: cy - h/2, W: w, H: h}
	if r.X < bounds.X {
		r.X = bounds.X
	}
	if r.Y < bounds.Y {
		r.Y = bounds.Y
	}
	if right := bounds.X + bounds.W; r.X+r.W > right {
		r.X = max(right-r.W, bounds.X)
	}
	if bottom := bounds.Y + bounds.H; r.Y+r.H > bottom {
		r.Y = max(bottom-r.H, bounds.Y)
	}
	return r
}

type Match struct {
	Title string
	Rect  Rect
}

func (m Match) String() string { return fmt.Sprintf("%q — %s", m.Title, m.Rect) }

type ErrNotFound struct{ Title string }

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("окно с заголовком %q не найдено", e.Title)
}
