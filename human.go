package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"farm-robot/internal/mouse"
)

// Человек не жмёт с одинаковыми паузами и не попадает в один пиксель: всё,
// что бот делает с постоянным шагом, здесь размывается.

// Range — случайная длительность из [min, max].
type Range struct {
	Min Duration `yaml:"min"`
	Max Duration `yaml:"max"`
}

func (r Range) Pick() time.Duration {
	lo, hi := r.Min.D(), r.Max.D()
	if hi <= lo {
		return lo
	}
	return lo + rand.N(hi-lo)
}

func (r Range) validate(name string) error {
	if r.Min.D() < 0 || r.Max.D() < r.Min.D() {
		return fmt.Errorf("%s: нужно 0 <= min <= max, задано %v..%v", name, r.Min.D(), r.Max.D())
	}
	return nil
}

// IntRange — случайное целое из [min, max].
type IntRange struct {
	Min int `yaml:"min"`
	Max int `yaml:"max"`
}

func (r IntRange) Pick() int {
	if r.Max <= r.Min {
		return r.Min
	}
	return r.Min + rand.IntN(r.Max-r.Min+1)
}

type HumanConfig struct {
	Reaction   Range    `yaml:"reaction"`   // от всплеска до клика
	BeforeCast Range    `yaml:"beforeCast"` // пауза перед забросом
	AimJitter  int      `yaml:"aimJitter"`  // разброс точки клика по поплавку, px
	Wander     float64  `yaml:"wander"`     // доля циклов, после которых мышь уходит в сторону
	WanderDist IntRange `yaml:"wanderDist"` // на сколько px
	BreakEvery IntRange `yaml:"breakEvery"` // перерыв раз в столько забросов
	BreakFor   Range    `yaml:"breakFor"`   // длительность перерыва
	IdleEvery  Range    `yaml:"idleEvery"`  // пока ждём поклёвку, мышь шевелится раз в столько
	IdleSize   IntRange `yaml:"idleSize"`   // размах шевеления, px
	Recording  string   `yaml:"recording"`  // запись движений руки (cmd/mouserec); пусто или нет файла — синтетика
}

func defaultHuman() HumanConfig {
	return HumanConfig{
		Reaction:   Range{Duration(250 * time.Millisecond), Duration(700 * time.Millisecond)},
		BeforeCast: Range{Duration(400 * time.Millisecond), Duration(2 * time.Second)},
		AimJitter:  4,
		Wander:     0.35,
		WanderDist: IntRange{60, 250},
		BreakEvery: IntRange{25, 45},
		BreakFor:   Range{Duration(20 * time.Second), Duration(90 * time.Second)},
		IdleEvery:  Range{Duration(2 * time.Second), Duration(6 * time.Second)},
		IdleSize:   IntRange{20, 70},
		Recording:  "mouse.json",
	}
}

func (h HumanConfig) validate() error {
	for name, r := range map[string]Range{
		"human.reaction": h.Reaction, "human.beforeCast": h.BeforeCast, "human.breakFor": h.BreakFor,
		"human.idleEvery": h.IdleEvery,
	} {
		if err := r.validate(name); err != nil {
			return err
		}
	}
	switch {
	case h.AimJitter < 0:
		return fmt.Errorf("human.aimJitter не может быть отрицательным")
	case h.Wander < 0 || h.Wander > 1:
		return fmt.Errorf("human.wander должен быть в [0, 1]")
	case h.WanderDist.Min < 0 || h.WanderDist.Max < h.WanderDist.Min:
		return fmt.Errorf("human.wanderDist: нужно 0 <= min <= max")
	case h.BreakEvery.Min < 0 || h.BreakEvery.Max < h.BreakEvery.Min:
		return fmt.Errorf("human.breakEvery: нужно 0 <= min <= max, 0 — без перерывов")
	case h.IdleSize.Min < 0 || h.IdleSize.Max < h.IdleSize.Min:
		return fmt.Errorf("human.idleSize: нужно 0 <= min <= max")
	case h.IdleEvery.Max.D() > 0 && h.IdleEvery.Min.D() < 500*time.Millisecond:
		return fmt.Errorf("human.idleEvery.min не меньше 500ms, иначе мышь не стоит ни секунды")
	}
	return nil
}

// useRecording подключает запись движений руки, если она есть.
func useRecording(path string) {
	if path == "" {
		return
	}
	rec, err := mouse.LoadRecording(path)
	if err != nil {
		log.Printf("запись движений %s: %v — двигаю синтетикой; записать: go run ./cmd/mouserec", path, err)
		return
	}
	travel, idle := mouse.UseGestures(rec.Gestures)
	log.Printf("движения из %s: %d переходов, %d для шевелений", path, travel, idle)
	if travel == 0 {
		log.Printf("в записи нет переходов от 30 px — переходы пойдут синтетикой")
	}
}

// jitter — случайный сдвиг в [-n, n].
func jitter(n int) int {
	if n <= 0 {
		return 0
	}
	return rand.IntN(2*n+1) - n
}

// vary размывает паузу на ±frac: 1500ms при 0.2 — от 1200 до 1800.
func vary(d time.Duration, frac float64) time.Duration {
	return time.Duration(float64(d) * (1 + frac*(2*rand.Float64()-1)))
}

// rest спит d, но выходит раньше по F4: перерыв длится до полутора минут.
func (f *Fisher) rest(d time.Duration) {
	end := time.Now().Add(d)
	for time.Now().Before(end) && !f.stopping() {
		time.Sleep(min(200*time.Millisecond, time.Until(end)))
	}
}

// maybeBreak устраивает перерыв, когда набралось нужное число забросов.
func (f *Fisher) maybeBreak() {
	if f.cfg.Human.BreakEvery.Max == 0 {
		return
	}
	if f.castsToBreak <= 0 {
		f.castsToBreak = f.cfg.Human.BreakEvery.Pick()
	}
	f.castsToBreak--
	if f.castsToBreak > 0 {
		return
	}
	d := f.cfg.Human.BreakFor.Pick()
	log.Printf("перерыв %v", d.Round(time.Second))
	f.trace.step("перерыв %v", d.Round(time.Second))
	f.rest(d)
}

// wander иногда уводит мышь в сторону от поплавка, не выходя за окно игры.
func (f *Fisher) wander() {
	a := f.lastBounds
	if rand.Float64() >= f.cfg.Human.Wander {
		return
	}
	x, y := mouse.Pos()
	dist := f.cfg.Human.WanderDist.Pick()
	x += jitter(dist)
	y += jitter(dist)
	x = max(a.X+10, min(a.X+a.W-10, x))
	y = max(a.Y+10, min(a.Y+a.H-10, y))
	mouse.MoveTo(x, y)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
