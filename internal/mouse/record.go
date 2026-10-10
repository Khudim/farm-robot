package mouse

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"
)

// Записанные движения живой руки. Координаты — физические пиксели экрана,
// они же отсчёты USB-мыши платы (ускорение указателя выключено, 1:1).

// Sample — точка записи: миллисекунды и смещение от начала движения.
type Sample struct {
	T    float64 `json:"t"`
	X, Y int
}

// Gesture — одно движение: от остановки до остановки.
type Gesture []Sample

// Recording — файл записи.
type Recording struct {
	Gestures []Gesture `json:"gestures"`
}

func (s Gesture) Duration() time.Duration {
	if len(s) == 0 {
		return 0
	}
	return time.Duration(s[len(s)-1].T * float64(time.Millisecond))
}

// Net — куда движение пришло от начала.
func (s Gesture) Net() (int, int) {
	if len(s) == 0 {
		return 0, 0
	}
	return s[len(s)-1].X, s[len(s)-1].Y
}

// Length — длина пути.
func (s Gesture) Length() float64 {
	var l float64
	for i := 1; i < len(s); i++ {
		l += math.Hypot(float64(s[i].X-s[i-1].X), float64(s[i].Y-s[i-1].Y))
	}
	return l
}

// Extent — самое дальнее отклонение от начала.
func (s Gesture) Extent() float64 {
	var e float64
	for _, p := range s {
		e = max(e, math.Hypot(float64(p.X), float64(p.Y)))
	}
	return e
}

// Segmenter режет поток положений курсора на движения: движение начинается
// со сдвига и кончается, когда курсор простоял still.
type Segmenter struct {
	Still   time.Duration
	MinPath float64 // короче — дрожь руки, не движение

	cur       Gesture
	start     time.Time
	ox, oy    int
	lastX     int
	lastY     int
	lastMove  time.Time
	have      bool
	completed []Gesture
}

// Add принимает положение курсора в момент at.
func (g *Segmenter) Add(at time.Time, x, y int) {
	if !g.have {
		g.lastX, g.lastY, g.lastMove, g.have = x, y, at, true
		return
	}
	moved := x != g.lastX || y != g.lastY
	switch {
	case moved && g.cur == nil:
		// Начало — с последней точки покоя, чтобы первый сдвиг не потерялся.
		g.start, g.ox, g.oy = g.lastMove, g.lastX, g.lastY
		if at.Sub(g.start) > 50*time.Millisecond {
			g.start = at.Add(-8 * time.Millisecond)
		}
		g.cur = Gesture{{T: 0, X: 0, Y: 0}}
		fallthrough
	case moved:
		g.cur = append(g.cur, Sample{T: ms(at.Sub(g.start)), X: x - g.ox, Y: y - g.oy})
		g.lastMove = at
	case g.cur != nil && at.Sub(g.lastMove) >= g.Still:
		g.finish()
	}
	g.lastX, g.lastY = x, y
	if !moved && g.cur == nil {
		g.lastMove = at
	}
}

// Flush закрывает незаконченное движение.
func (g *Segmenter) Flush() {
	if g.cur != nil {
		g.finish()
	}
}

func (g *Segmenter) finish() {
	if len(g.cur) >= 4 && g.cur.Length() >= g.MinPath {
		g.completed = append(g.completed, g.cur)
	}
	g.cur = nil
}

// Gestures — готовые движения.
func (g *Segmenter) Gestures() []Gesture { return g.completed }

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func LoadRecording(path string) (Recording, error) {
	var r Recording
	raw, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return r, fmt.Errorf("разбор %s: %w", path, err)
	}
	return r, nil
}

func SaveRecording(path string, r Recording) error {
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}
