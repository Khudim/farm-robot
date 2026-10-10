package main

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"farm-robot/internal/mouse"
	"farm-robot/internal/vision"
	"farm-robot/internal/window"

	"github.com/go-vgo/robotgo"
)

// area — прямоугольник search.width×height с центром в якоре (где показали
// поплавок), в границах окна игры. Пересчитывается на каждом забросе.
func (f *Fisher) area() (window.Rect, error) {
	bounds := screenRect()
	origin := point{}
	if f.cfg.Window.Title != "" {
		r, err := window.Find(f.cfg.Window.Title)
		if err != nil {
			return window.Rect{}, err
		}
		origin, bounds = point{r.X, r.Y}, r
	}
	a := window.Centered(f.cfg.Search.Width, f.cfg.Search.Height,
		origin.X+f.anchor.X, origin.Y+f.anchor.Y, bounds)
	f.noteArea(a, bounds)
	return a, nil
}

func screenRect() window.Rect {
	w, h := robotgo.GetScreenSize()
	return window.Rect{W: w, H: h}
}

// noteArea замечает сдвиг области: при неподвижном окне это значит, что под
// window.title подходит несколько окон и выбирается разное.
func (f *Fisher) noteArea(a, bounds window.Rect) {
	if f.lastArea != nil && *f.lastArea == a && f.lastBounds == bounds {
		return
	}
	if f.lastArea == nil {
		log.Printf("область поиска %s, окно %s", a, bounds)
	} else {
		log.Printf("ВНИМАНИЕ: область поиска сдвинулась с (%d,%d) на (%d,%d), окно %s — "+
			"если окно не двигали, под window.title подходит несколько окон",
			f.lastArea.X, f.lastArea.Y, a.X, a.Y, bounds)
		areaShifts.Inc()
	}
	cp := a
	f.lastArea, f.lastBounds = &cp, bounds
}

func (f *Fisher) checkWindowAmbiguity() {
	if f.cfg.Window.Title == "" {
		return
	}
	matches := window.List(f.cfg.Window.Title)
	if len(matches) <= 1 {
		return
	}
	log.Printf("ВНИМАНИЕ: под window.title %q подходит %d окон, берётся первое:", f.cfg.Window.Title, len(matches))
	for i, m := range matches {
		mark := " "
		if i == 0 {
			mark = ">"
		}
		log.Printf("  %s %s", mark, m)
	}
	log.Printf("лаунчер Battle.net тоже подходит — закрой лишние окна или уточни window.title")
}

// setAnchor запоминает, где показали поплавок, относительно окна игры, и
// сохраняет это рядом с эталонами — чтобы после перезапуска не просить F6 снова.
func (f *Fisher) setAnchor(screenX, screenY int) {
	origin := point{}
	if f.cfg.Window.Title != "" {
		if r, err := window.Find(f.cfg.Window.Title); err == nil {
			origin = point{r.X, r.Y}
		} else {
			log.Printf("якорь: окно игры не найдено, координаты считаю экранными")
		}
	}
	f.anchor = &point{screenX - origin.X, screenY - origin.Y}
	if err := saveAnchor(f.anchorPath(), *f.anchor); err != nil {
		log.Printf("якорь не сохранён: %v", err)
	}
}

func (f *Fisher) anchorPath() string { return filepath.Join(f.cfg.Search.Templates, "anchor.json") }

// loadAnchor поднимает якорь с диска. Если камера или окно с тех пор изменились,
// поиск не найдёт поплавок — тогда F6 перепишет якорь.
func (f *Fisher) loadAnchor() {
	p, err := readAnchor(f.anchorPath())
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("якорь не прочитан: %v", err)
		}
		return
	}
	f.anchor = &p
	log.Printf("якорь загружен: (%d,%d) относительно окна; если поплавок садится в другом месте — нажми %s",
		p.X, p.Y, f.cfg.Search.GrabKey)
}

func saveAnchor(path string, p point) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readAnchor(path string) (point, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return point{}, err
	}
	var p point
	if err := json.Unmarshal(data, &p); err != nil {
		return point{}, fmt.Errorf("%s: %w", path, err)
	}
	return p, nil
}

func (f *Fisher) snapshot(a window.Rect) image.Image {
	img, err := robotgo.CaptureImg(a.X, a.Y, a.W, a.H)
	if err != nil {
		log.Printf("захват экрана: %v", err)
		return nil
	}
	return img
}

// match снимает область и сопоставляет с эталонами.
func (f *Fisher) match(a window.Rect) (vision.Hit, int, bool) {
	img := f.snapshot(a)
	if img == nil {
		return vision.Hit{}, 0, false
	}
	f.lastHits = vision.FindEach(img, f.templates, vision.MatchOptions{
		Stride: f.cfg.Search.Stride, MinScore: f.cfg.Search.MinScore,
	})
	hit, idx, ok := vision.Best(f.lastHits, f.cfg.Search.MinScore)
	matchScore.Set(hit.Score)
	f.trace.step("сопоставление: %.3f (%s), порог %.2f", hit.Score, hit.Template, f.cfg.Search.MinScore)
	if !ok {
		f.warnOnce("порог %.2f не берётся, лучший счёт %.2f (%s) — сними эталон под эту воду: %s на поплавке",
			f.cfg.Search.MinScore, hit.Score, hit.Template, f.cfg.Search.GrabKey)
	}
	return hit, idx, ok
}

func (f *Fisher) hitPoint(a window.Rect, hit vision.Hit, idx int) *point {
	cx, cy := hit.Center(f.templates[idx])
	return &point{a.X + cx, a.Y + cy}
}

// static — совпадение стояло на том же месте и до заброса: это интерфейс или
// камень, а не поплавок. Иначе бот ждёт поклёвку, глядя на кнопку.
func (f *Fisher) static(before image.Image, hit vision.Hit, idx int) bool {
	return before != nil && vision.ScoreAt(before, f.templates[idx], hit.X, hit.Y) >= f.cfg.Search.MinScore
}

// awaitBobber ждёт появления поплавка: время полёта плавает, константой не задаётся.
func (f *Fisher) awaitBobber(a window.Rect, before image.Image) *point {
	robotgo.MilliSleep(ms(f.cfg.Timing.CastSettle.D()))
	f.reportCastEffect(a, before)

	deadline := time.Now().Add(f.cfg.Timing.CastTimeout.D())
	var castBest []vision.Hit
	for attempt := 1; ; attempt++ {
		hit, idx, ok := f.match(a)
		castBest = maxHits(castBest, f.lastHits)
		if ok && f.static(before, hit, idx) {
			f.trace.step("в (%d,%d) до заброса было то же самое — не поплавок", a.X+hit.X, a.Y+hit.Y)
			ok = false
		}
		if ok {
			p := f.hitPoint(a, hit, idx)
			f.trace.step("поплавок в (%d,%d) с %d-й проверки", p.X, p.Y, attempt)
			log.Printf("поплавок найден, эталоны: %s", scoresLine(f.lastHits))
			f.judgeTemplates(f.lastHits, idx)
			return p
		}
		if f.stopping() {
			return nil
		}
		if time.Now().After(deadline) {
			f.trace.step("поплавок не появился за %v", f.cfg.Timing.CastTimeout.D())
			log.Printf("поплавок не найден, лучшее за заброс: %s", scoresLine(castBest))
			return nil
		}
		robotgo.MilliSleep(300)
	}
}

// aimAtBobber смотрит, что реально под курсором, и доводит мышь на поплавок.
// Смещение уходит в метрику: стабильно большое означает эталон не по центру.
func (f *Fisher) aimAtBobber(p *point) *point {
	mx, my := mouse.Pos()
	r := f.cfg.Search.GrabRadius * 2
	img, err := robotgo.CaptureImg(mx-r, my-r, 2*r+1, 2*r+1)
	if err != nil {
		return p
	}
	hit, idx, ok := vision.FindBest(img, f.templates, vision.MatchOptions{Stride: 1, MinScore: f.cfg.Search.MinScore})
	if !ok {
		log.Printf("под курсором поплавка нет (счёт %.2f): просили (%d,%d), мышь в (%d,%d)", hit.Score, p.X, p.Y, mx, my)
		f.trace.step("под курсором поплавка нет, счёт %.2f", hit.Score)
		return p
	}
	cx, cy := hit.Center(f.templates[idx])
	aim := &point{mx - r + cx, my - r + cy}
	dx, dy := aim.X-mx, aim.Y-my
	// В пределах разброса клика мышь и так на поплавке: человек бы не поправлял.
	if j := f.cfg.Human.AimJitter; abs(dx) > j || abs(dy) > j {
		aim.X += jitter(j)
		aim.Y += jitter(j)
		mouse.MoveTo(aim.X, aim.Y)
		time.Sleep(vary(f.cfg.Input.Settle.D(), 0.4))
	}
	aimOffset.Observe(math.Hypot(float64(dx), float64(dy)))
	f.trace.step("под курсором поплавок, счёт %.2f, доводка (%+d,%+d)", hit.Score, dx, dy)
	return aim
}

// grabbed — снятый эталон и где стояла мышь: туда встанет якорь области.
type grabbed struct {
	tmpl vision.Template
	x, y int
}

// grabTemplate снимает эталон из-под курсора и сохраняет его. В поиск его
// пускает главный цикл: шаблоны и якорь трогает только он.
func (f *Fisher) grabTemplate() (grabbed, bool) {
	r := f.cfg.Search.GrabRadius
	size := 2*r + 1
	mx, my := mouse.Pos()

	img, err := robotgo.CaptureImg(mx-r, my-r, size, size)
	if err != nil {
		log.Printf("снимок эталона: %v", err)
		return grabbed{}, false
	}
	dir := f.cfg.Search.Templates
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("снимок эталона: %v", err)
		return grabbed{}, false
	}
	name := nextTemplateName(dir)
	tmpl, err := vision.NewTemplate(name, img)
	if err != nil {
		log.Printf("снимок не годится: %v — наведись точнее на поплавок", err)
		return grabbed{}, false
	}
	path := filepath.Join(dir, name)
	if err := savePNG(path, img); err != nil {
		log.Printf("снимок эталона: %v", err)
		return grabbed{}, false
	}
	log.Printf("эталон %s снят в (%d,%d)", path, mx, my)
	return grabbed{tmpl, mx, my}, true
}

func nextTemplateName(dir string) string {
	for n := 1; n < 1000; n++ {
		name := fmt.Sprintf("bobber-%03d.png", n)
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			return name
		}
	}
	return fmt.Sprintf("bobber-%d.png", time.Now().Unix())
}
