package mouse

import (
	"math"
	"math/rand/v2"
	"time"
)

// RandomFigure выбирает фигуру размахом size под случайным углом. Все фигуры
// замкнуты — мышь возвращается туда, откуда начала. nil — просто сдвинуться.
func RandomFigure(size float64) (string, func(t float64) (float64, float64), time.Duration) {
	turn := rand.Float64() * 2 * math.Pi
	pick := func(lo, hi time.Duration) time.Duration {
		return lo + rand.N(hi-lo)
	}

	var name string
	var base func(t float64) (float64, float64)
	var dur time.Duration
	switch rand.IntN(4) {
	case 0: // восьмёрка — лемниската Жероно
		loops := float64(1 + rand.IntN(2))
		flat := 0.4 + 0.3*rand.Float64()
		name, dur = "восьмёрка", time.Duration(loops*float64(pick(1500*time.Millisecond, 3*time.Second)))
		base = func(t float64) (float64, float64) {
			th := 2 * math.Pi * loops * t
			return size * math.Sin(th), size * flat * math.Sin(th) * math.Cos(th)
		}
	case 1: // влево-вправо, почти горизонтально
		k := float64(1 + rand.IntN(3))
		turn = (rand.Float64() - 0.5) * 0.6
		name, dur = "влево-вправо", time.Duration(k*float64(pick(600*time.Millisecond, time.Second)))
		base = func(t float64) (float64, float64) {
			th := 2 * math.Pi * k * t
			return size * math.Sin(th), size * 0.1 * math.Sin(th/2)
		}
	case 2: // круг в случайную сторону
		dir := 1.0
		if rand.IntN(2) == 0 {
			dir = -1
		}
		r := size / 2
		name, dur = "круг", pick(time.Second, 2*time.Second)
		base = func(t float64) (float64, float64) {
			th := 2 * math.Pi * t * dir
			return r * (math.Cos(th) - 1), r * math.Sin(th)
		}
	default:
		return "сдвиг", nil, 0
	}
	return name, humanize(base, turn), dur
}

// humanize портит геометрию, как рука: фигура растянута по осям, размах
// плавно «дышит» по ходу, скорость чуть плавает. Множитель берётся к смещению
// от начала, поэтому замкнутая фигура остаётся замкнутой.
func humanize(base func(t float64) (float64, float64), turn float64) func(t float64) (float64, float64) {
	sx, sy := 0.75+0.35*rand.Float64(), 0.75+0.35*rand.Float64()
	f1, f2 := 1+rand.Float64()*2, 3+rand.Float64()*3
	p1, p2 := rand.Float64()*2*math.Pi, rand.Float64()*2*math.Pi
	a1, a2 := 0.08+0.1*rand.Float64(), 0.03+0.05*rand.Float64()
	speed := 0.02 * rand.Float64()
	s, c := math.Sincos(turn)
	return func(t float64) (float64, float64) {
		e := Ease(t)
		e += speed * math.Sin(2*math.Pi*e) // не ломает монотонность: производная > 0
		x, y := base(e)
		k := 1 + a1*math.Sin(2*math.Pi*f1*t+p1) + a2*math.Sin(2*math.Pi*f2*t+p2)
		x, y = x*sx*k, y*sy*k
		return x*c - y*s, x*s + y*c
	}
}

// Ease — профиль минимального рывка: фигура начинается и кончается плавно.
func Ease(t float64) float64 {
	return t * t * t * (10 - 15*t + 6*t*t)
}
