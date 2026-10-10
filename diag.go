package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"farm-robot/internal/vision"
	"farm-robot/internal/window"
)

// tracer печатает цикл по шагам с временем: «не дошло до заброса», «не нашёлся
// поплавок» и «не было поклёвки» снаружи выглядят одинаково.
type tracer struct {
	on    bool
	n     int
	begin time.Time
	last  time.Time
}

func newTracer(on bool) *tracer { return &tracer{on: on} }

func (t *tracer) cycle() {
	if !t.on {
		return
	}
	t.n++
	t.begin = time.Now()
	t.last = t.begin
	log.Printf("---------- цикл %d ----------", t.n)
}

func (t *tracer) step(format string, args ...any) {
	if !t.on {
		return
	}
	now := time.Now()
	prefix := fmt.Sprintf("  [%6s / %6s] ",
		now.Sub(t.last).Round(time.Millisecond), now.Sub(t.begin).Round(time.Millisecond))
	log.Printf(prefix+format, args...)
	t.last = now
}

func (t *tracer) done(outcome string) {
	if !t.on {
		return
	}
	log.Printf("  итог: %s, цикл занял %v", outcome, time.Since(t.begin).Round(time.Millisecond))
}

// logBite пишет, на каком уровне сработала поклёвка и какой запас над
// порогами — по этим числам подкручиваются splash.threshold и splash.minLevel.
func (f *Fisher) logBite() {
	level, base := f.splash.LastHit()
	cfg := f.cfg.Splash
	ratio := 0.0
	if base > 0 {
		ratio = level / base
	}
	log.Printf("поклёвка: уровень %.4f при фоне %.4f — громче фона в %.1f раза; пороги: threshold %.1f, minLevel %.4f",
		level, base, ratio, cfg.Threshold, cfg.MinLevel)
}

// explainSilence — почему окно ожидания поклёвки прошло впустую.
func (f *Fisher) explainSilence() {
	peak, ratio, base := f.splash.Window()
	cfg := f.cfg.Splash
	switch {
	case peak == 0:
		f.warnOnce("звука нет: устройство захвата молчит — проверь устройство вывода игры")
	case peak < 0.005:
		f.warnOnce("звук почти на нуле (пик %.4f): прибавь громкость игры и системы", peak)
	case peak <= cfg.MinLevel:
		f.warnOnce("пик %.4f ниже splash.minLevel %.4f — опусти его до ~%.4f", peak, cfg.MinLevel, peak/2)
	case ratio < cfg.Threshold:
		f.warnOnce("пик %.4f при фоне %.4f (x%.1f), а splash.threshold %.1f — опусти до ~%.1f или приглуши музыку",
			peak, base, ratio, cfg.Threshold, ratio*0.7)
	default:
		log.Printf("звук проходил пороги (x%.1f), но событие съел refractory %v", ratio, cfg.Refractory.D())
	}
}

// reportCastEffect — произошло ли после нажатия хоть что-то. Поплавок, падая в
// воду, слышен и виден; несработавшая клавиша не даёт ни того, ни другого.
func (f *Fisher) reportCastEffect(a window.Rect, before image.Image) {
	sound := "звук не слушали"
	if f.splash != nil {
		peak, ratio, _ := f.splash.Window()
		f.splash.Arm(false)
		switch {
		case peak == 0:
			sound = "звука нет"
		case ratio < 1.5:
			sound = fmt.Sprintf("звук на уровне фона (x%.1f)", ratio)
		default:
			sound = fmt.Sprintf("звук x%.1f к фону", ratio)
		}
	}
	water := "кадр не сравнить"
	if ch := vision.FrameChange(before, f.snapshot(a)); ch >= 0 {
		water = fmt.Sprintf("вода изменилась на %.2f%%", ch*100)
	}
	f.lastCastEffect = sound + ", " + water
	f.trace.step("после заброса: %s", f.lastCastEffect)
}

// dumpBite в режиме трассировки сохраняет область с крестом в точке клика.
func (f *Fisher) dumpBite(a window.Rect, p point) {
	if !f.trace.on {
		return
	}
	img := f.snapshot(a)
	if img == nil {
		return
	}
	out := image.NewRGBA(img.Bounds())
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Src)
	red := color.RGBA{R: 255, G: 40, B: 40, A: 255}
	cx, cy := p.X-a.X, p.Y-a.Y
	for d := -12; d <= 12; d++ {
		out.Set(cx+d, cy, red)
		out.Set(cx, cy+d, red)
	}
	if err := os.MkdirAll("trace", 0o755); err != nil {
		return
	}
	name := filepath.Join("trace", fmt.Sprintf("bite-%03d.png", f.trace.n))
	if err := savePNG(name, out); err == nil {
		f.trace.step("снимок клика: %s", name)
	}
}

func savePNG(path string, img image.Image) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, img)
}
