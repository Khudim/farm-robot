package mouse

import (
	"math"
	"math/rand/v2"
	"sort"
	"time"
)

// Воспроизведение записанных движений руки: переходы между точками берутся из
// записи и подгоняются под нужный сдвиг, шевеления проигрываются как есть.

var (
	travels []Gesture // движения «откуда-то куда-то», годятся для переходов
	idles   []Gesture // все движения, для шевелений
)

// UseGestures подключает запись. Без неё MoveTo ведёт синтетической кривой, а
// шевеления рисуются фигурами.
func UseGestures(gs []Gesture) (travel, idle int) {
	travels, idles = nil, nil
	for _, g := range gs {
		if len(g) < 4 {
			continue
		}
		idles = append(idles, g)
		nx, ny := g.Net()
		net := math.Hypot(float64(nx), float64(ny))
		// Переход — когда рука в основном шла к цели, а не кружила: иначе при
		// подгонке под цель петля раздуется во много раз.
		if net >= 30 && net >= 0.5*g.Length() {
			travels = append(travels, g)
		}
	}
	return len(travels), len(idles)
}

// HasGestures — есть ли запись для шевелений.
func HasGestures() bool { return len(idles) > 0 }

// replayTravel ведёт мышь на dx, dy физических пикселей записанным движением
// похожей длины, повёрнутым и растянутым под цель. false — подходящего нет.
func replayTravel(dx, dy int) bool {
	want := math.Hypot(float64(dx), float64(dy))
	if len(travels) == 0 || want < 20 {
		return false
	}
	// Ближайшие по длине — их меньше всего растягивать; из трёх лучших случайное,
	// чтобы на одну и ту же дистанцию не шло одно и то же движение.
	type cand struct {
		g    Gesture
		diff float64
	}
	cs := make([]cand, 0, len(travels))
	for _, g := range travels {
		nx, ny := g.Net()
		cs = append(cs, cand{g, math.Abs(math.Log(math.Hypot(float64(nx), float64(ny)) / want))})
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].diff < cs[j].diff })
	g := cs[rand.IntN(min(3, len(cs)))].g

	nx, ny := g.Net()
	have := math.Hypot(float64(nx), float64(ny))
	scale := want / have
	turn := math.Atan2(float64(dy), float64(dx)) - math.Atan2(float64(ny), float64(nx))
	s, c := math.Sincos(turn)
	// Длинное движение рука делает дольше, но не пропорционально длине.
	timeScale := min(1.6, max(0.6, math.Sqrt(scale)))
	play(g, func(x, y float64) (float64, float64) {
		x, y = x*scale, y*scale
		return x*c - y*s, x*s + y*c
	}, timeScale, nil)
	return true
}

// PlayIdle проигрывает случайное записанное движение, уменьшенное до размаха
// size, если оно больше. abort проверяется по ходу; true — прервано.
func PlayIdle(size float64, abort func() bool) (aborted bool, took time.Duration) {
	if len(idles) == 0 || mover == nil {
		return false, 0
	}
	// Предпочитаем движения своего масштаба: сильно ужатый дальний переход
	// выглядит как рывок.
	var fit []Gesture
	for _, g := range idles {
		if e := g.Extent(); e >= size/3 && e <= size*2.5 {
			fit = append(fit, g)
		}
	}
	if len(fit) == 0 {
		fit = idles
	}
	g := fit[rand.IntN(len(fit))]
	k := min(1, size/max(1, g.Extent()))
	start := time.Now()
	aborted = play(g, func(x, y float64) (float64, float64) { return x * k, y * k }, 1, abort)
	return aborted, time.Since(start)
}

// play ведёт мышь по записи в её темпе. Если плата отвечает медленнее, чем
// шла запись, промежуточные точки сливаются: важнее не отстать по времени.
func play(g Gesture, at func(x, y float64) (float64, float64), timeScale float64, abort func() bool) bool {
	start := time.Now()
	px, py := 0, 0
	for i := 1; i < len(g); {
		elapsed := float64(time.Since(start)) / float64(time.Millisecond) / timeScale
		if wait := g[i].T - elapsed; wait > 0 {
			time.Sleep(time.Duration(wait * timeScale * float64(time.Millisecond)))
			continue
		}
		for i+1 < len(g) && g[i+1].T <= elapsed {
			i++
		}
		x, y := at(float64(g[i].X), float64(g[i].Y))
		ix, iy := int(math.Round(x)), int(math.Round(y))
		if ix != px || iy != py {
			if err := mover.Step(ix-px, iy-py); err != nil {
				return false
			}
			px, py = ix, iy
		}
		i++
		if abort != nil && abort() {
			return true
		}
	}
	return false
}
