package main

import (
	"fmt"
	"log"
	"time"

	"farm-robot/internal/mouse"
	"farm-robot/internal/vision"
	"farm-robot/internal/window"

	"github.com/go-vgo/robotgo"
)

type point struct{ X, Y int }

type Fisher struct {
	cfg       AppConfig
	castKey   hotkey
	baitKey   hotkey
	splash    *SplashDetector
	templates []vision.Template
	trace     *tracer

	anchor         *point // где показали поплавок, относительно окна
	misses         int
	warned         map[string]bool
	lastArea       *window.Rect
	lastBounds     window.Rect
	blockedMsg     string
	blockedAt      time.Time
	lastCastEffect string
	castsToBreak   int                  // забросов до перерыва
	lastHits       []vision.Hit         // счёт каждого эталона в последней проверке
	tstats         map[string]*tmplStat // счёт эталонов за сессию

	pause, unPause, exit, bait chan struct{}
	grab                       chan grabbed
	done                       chan struct{}
}

func newFisher(cfg AppConfig, splash *SplashDetector, templates []vision.Template, tr *tracer) *Fisher {
	cast, _ := parseHotkey(cfg.Keys.Cast) // проверено в validate
	bait, _ := parseHotkey(cfg.Keys.Bait)
	return &Fisher{
		cfg: cfg, castKey: cast, baitKey: bait, splash: splash, templates: templates, trace: tr,
		warned:  map[string]bool{},
		pause:   make(chan struct{}, 1),
		unPause: make(chan struct{}, 1),
		exit:    make(chan struct{}, 1),
		bait:    make(chan struct{}, 1),
		grab:    make(chan grabbed, 4),
		done:    make(chan struct{}),
	}
}

func (f *Fisher) Done() <-chan struct{} { return f.done }

func (f *Fisher) hotkeys() (stop func()) {
	signal := func(ch chan struct{}) func() {
		return func() {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}
	stop, err := watchKeys(map[string]func(){
		"f2": signal(f.pause),
		"f3": signal(f.unPause),
		"f4": signal(f.exit),
		// Эталон снимается сразу по нажатию: главный цикл занят забросом до
		// полуминуты, за это время бот сам уведёт мышь с поплавка.
		f.cfg.Search.GrabKey: func() {
			g, ok := f.grabTemplate()
			if !ok {
				return
			}
			select {
			case f.grab <- g:
			default:
				log.Printf("эталон сохранён, но в поиск попадёт после перезапуска: очередь занята")
			}
		},
	})
	if err != nil {
		log.Printf("горячие клавиши: %v", err)
	}

	log.Printf("управление: F2 — пауза, F3 — продолжить, F4 — выход, %s — снять эталон из-под курсора",
		f.cfg.Search.GrabKey)
	log.Printf("бинды в игре: заброс %s, наживка %s", f.castKey, f.baitKey)
	return stop
}

func (f *Fisher) baitTimer() {
	t := time.NewTicker(f.cfg.Timing.BaitInterval.D())
	go func() {
		defer t.Stop()
		for {
			select {
			case <-t.C:
				select {
				case f.bait <- struct{}{}:
				default:
				}
			case <-f.done:
				return
			}
		}
	}()
}

func (f *Fisher) Start() {
	defer close(f.done)

	defer f.hotkeys()()
	f.checkWindowAmbiguity()
	f.loadAnchor()
	f.baitTimer()
	f.awaitSound()

	switch {
	case len(f.templates) == 0:
		log.Printf("эталонов нет — заброси удочку руками, наведи мышь на поплавок и нажми %s", f.cfg.Search.GrabKey)
	case f.anchor == nil:
		log.Printf("эталоны есть, но не знаю, где вода — наведи мышь на поплавок и нажми %s", f.cfg.Search.GrabKey)
	}

	for {
		select {
		case <-f.exit:
			log.Println("выход")
			return
		case <-f.pause:
			log.Println("пауза")
			select {
			case <-f.unPause:
				log.Println("продолжаю")
			case <-f.exit:
				log.Println("выход")
				return
			}
		case g := <-f.grab:
			f.templates = append(f.templates, g.tmpl)
			f.setAnchor(g.x, g.y)
			log.Printf("эталон в поиске, всего %d", len(f.templates))
		case <-f.bait:
			log.Println("обновляю наживку")
			time.Sleep(f.cfg.Human.BeforeCast.Pick())
			tap(f.baitKey, f.cfg.Input.Hold.D())
			time.Sleep(vary(time.Second, 0.5))
		default:
			f.cycle()
		}
	}
}

// awaitSound даёт детектору набрать фон: иначе первый заброс идёт вглухую и
// его звук не с чем сравнить.
func (f *Fisher) awaitSound() {
	if f.splash == nil {
		return
	}
	for i := 0; i < 30 && !f.splash.Ready(); i++ {
		robotgo.MilliSleep(100)
	}
	if !f.splash.Ready() {
		log.Printf("звук не набрал фон за 3 с — проверь устройство вывода")
	}
}

// cycle — один заброс: кинуть, найти поплавок, дождаться всплеска, подсечь.
func (f *Fisher) cycle() {
	f.trace.cycle()

	if len(f.templates) == 0 || f.anchor == nil {
		f.blocked("наведи мышь на поплавок и нажми %s — дальше поеду сам", f.cfg.Search.GrabKey)
		f.trace.done("нет эталона или якоря")
		robotgo.MilliSleep(1000)
		return
	}
	a, err := f.area()
	if err != nil {
		f.blocked("окно игры не найдено: %v", err)
		f.trace.done("окно не найдено")
		robotgo.MilliSleep(1000)
		return
	}
	if f.cfg.Window.RequireFocus {
		if ok, err := window.Focused(f.cfg.Window.Title); err == nil && !ok {
			f.blocked("окно игры не активно — нажатия ушли бы в чужое окно (window.requireFocus)")
			f.trace.done("окно не в фокусе")
			robotgo.MilliSleep(1000)
			return
		}
	}
	f.unblocked()
	f.trace.step("область %s, эталонов %d", a, len(f.templates))

	f.maybeBreak()
	time.Sleep(f.cfg.Human.BeforeCast.Pick())
	if f.stopping() {
		f.trace.done("выход")
		return
	}

	p := f.castAndFind(a)
	if p == nil {
		if !f.stopping() {
			if f.lastCastEffect != "" {
				log.Printf("после нажатия заброса: %s", f.lastCastEffect)
			}
			f.miss("поплавок не найден")
		}
		f.trace.done("поплавок не найден")
		return
	}

	// Наводимся сразу: к поклёвке мышь уже стоит на поплавке, остаётся кликнуть.
	// Точка клика каждый раз немного своя.
	j := f.cfg.Human.AimJitter
	mouse.MoveTo(p.X+jitter(j), p.Y+jitter(j))
	time.Sleep(vary(f.cfg.Input.Settle.D(), 0.4))
	p = f.aimAtBobber(p)

	// Приводнение поплавка — тоже всплеск, и громкий; ждём, пока он затихнет.
	robotgo.MilliSleep(ms(f.cfg.Timing.BiteGrace.D()))
	f.splash.Drain()
	f.splash.Arm(true)
	f.trace.step("жду всплеск, до %v", f.cfg.Timing.BiteTimeout.D())
	bite := f.waitBite(*p)
	f.splash.Arm(false)
	if !bite {
		f.explainSilence()
		f.miss("поклёвки не было")
		f.trace.done("поклёвки не было")
		return
	}
	bitesTotal.Inc()
	f.trace.step("всплеск")
	f.logBite()
	f.dumpBite(a, *p)

	react := f.cfg.Human.Reaction.Pick()
	f.trace.step("реакция %v", react)
	time.Sleep(react)
	f.backToBobber(*p)
	lootAll(f.cfg.Input, f.cfg.Loot.Shift)
	f.trace.step("подсечка (shift=%v)", f.cfg.Loot.Shift)
	lootTotal.Inc()
	f.misses = 0
	log.Printf("добыча забрана")
	f.trace.done("добыча забрана")

	time.Sleep(vary(f.cfg.Timing.LootSettle.D(), 0.3))
	f.wander()
}

// castAndFind забрасывает и ждёт поплавок, при неудаче повторяет: обратной
// связи от игры нет, появившийся поплавок — единственный признак заброса.
func (f *Fisher) castAndFind(a window.Rect) *point {
	for i := 1; i <= f.cfg.Timing.CastRetries; i++ {
		releaseModifiers(f.cfg.Input.Settle.D())
		before := f.snapshot(a)
		f.splash.Drain()
		f.splash.Arm(true)

		tap(f.castKey, f.cfg.Input.Hold.D())
		castsTotal.Inc()
		f.trace.step("заброс %s", f.castKey)

		if p := f.awaitBobber(a, before); p != nil {
			return p
		}
		if f.stopping() {
			return nil
		}
		if i < f.cfg.Timing.CastRetries {
			log.Printf("поплавка нет, повторяю заброс (%d из %d)", i, f.cfg.Timing.CastRetries)
			castRetries.Inc()
		}
	}
	return nil
}

// miss считает неудачи подряд; длинная серия означает системную поломку.
func (f *Fisher) miss(reason string) {
	missesTotal.WithLabelValues(reason).Inc()
	f.misses++
	log.Printf("промах (%d/%d): %s", f.misses, f.cfg.Timing.MaxMisses, reason)
	if f.misses >= f.cfg.Timing.MaxMisses {
		log.Printf("%d неудач подряд, останавливаюсь", f.misses)
		select {
		case f.exit <- struct{}{}:
		default:
		}
	}
}

// stopping проверяет запрос на выход, не съедая сигнал.
func (f *Fisher) stopping() bool {
	select {
	case <-f.exit:
		f.exit <- struct{}{}
		return true
	default:
		return false
	}
}

func (f *Fisher) warnOnce(format string, args ...any) {
	if f.warned == nil {
		f.warned = map[string]bool{}
	}
	msg := fmt.Sprintf(format, args...)
	if !f.warned[msg] {
		f.warned[msg] = true
		log.Print(msg)
	}
}

// blocked сообщает о простое не чаще раза в 15 с, пока он длится — в отличие
// от warnOnce, после которого бот молча крутится вхолостую.
func (f *Fisher) blocked(format string, args ...any) bool {
	msg := fmt.Sprintf(format, args...)
	same := msg == f.blockedMsg
	if same && time.Since(f.blockedAt) < 15*time.Second {
		return false
	}
	if same {
		log.Printf("всё ещё стою: %s", msg)
	} else {
		log.Printf("стою: %s", msg)
	}
	f.blockedMsg, f.blockedAt = msg, time.Now()
	return true
}

func (f *Fisher) unblocked() {
	if f.blockedMsg != "" {
		log.Printf("продолжаю работу")
		f.blockedMsg = ""
	}
}
