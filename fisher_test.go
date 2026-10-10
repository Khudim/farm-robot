package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Состояние простоя должно напоминать о себе, пока длится, а не мелькнуть однажды.
func TestBlockedRepeatsWhileItLasts(t *testing.T) {
	f := &Fisher{}
	if !f.blocked("причина А") {
		t.Fatal("первое сообщение должно печататься")
	}
	if f.blocked("причина А") {
		t.Error("повтор той же причины сразу печататься не должен")
	}
	if !f.blocked("причина Б") {
		t.Error("смена причины печатается сразу")
	}
	f.blockedAt = time.Now().Add(-20 * time.Second)
	if !f.blocked("причина Б") {
		t.Error("после интервала причина должна напомнить о себе")
	}
	f.unblocked()
	if f.blockedMsg != "" {
		t.Error("unblocked должен сбрасывать состояние")
	}
}

// Область поиска встаёт вокруг якоря — точки, где показали поплавок. Без этого
// она садилась в угол окна и сопоставляла эталон с интерфейсом вместо воды.
func TestAreaCentersOnAnchor(t *testing.T) {
	cfg := defaultConfig()
	cfg.Window.Title = ""
	cfg.Search.Templates = t.TempDir()
	f := newFisher(cfg, nil, nil, newTracer(false))
	f.setAnchor(1000, 600)

	a, err := f.area()
	if err != nil {
		t.Fatal(err)
	}
	if cx, cy := a.X+a.W/2, a.Y+a.H/2; cx != 1000 || cy != 600 {
		t.Errorf("центр (%d,%d), ожидался (1000,600)", cx, cy)
	}
	if a.W != cfg.Search.Width || a.H != cfg.Search.Height {
		t.Errorf("размер %dx%d, ожидался %dx%d", a.W, a.H, cfg.Search.Width, cfg.Search.Height)
	}
}

func TestMissStopsAfterMaxMisses(t *testing.T) {
	cfg := defaultConfig()
	cfg.Timing.MaxMisses = 3
	f := newFisher(cfg, nil, nil, newTracer(false))
	for i := 0; i < 3; i++ {
		f.miss("тест")
	}
	select {
	case <-f.exit:
	default:
		t.Fatal("после maxMisses должен прийти сигнал выхода")
	}
}

// Якорь переживает перезапуск: иначе каждый старт требует F6, хотя эталоны
// на диске уже есть, и бот молча стоит.
func TestAnchorPersistsAcrossRestarts(t *testing.T) {
	cfg := defaultConfig()
	cfg.Window.Title = ""
	cfg.Search.Templates = t.TempDir()

	first := newFisher(cfg, nil, nil, newTracer(false))
	first.setAnchor(1000, 600)

	second := newFisher(cfg, nil, nil, newTracer(false))
	if second.anchor != nil {
		t.Fatal("новый Fisher не должен знать якорь до загрузки")
	}
	second.loadAnchor()
	if second.anchor == nil || *second.anchor != (point{1000, 600}) {
		t.Fatalf("якорь не загрузился: %+v", second.anchor)
	}
}

func TestLoadAnchorWithoutFileIsSilent(t *testing.T) {
	cfg := defaultConfig()
	cfg.Search.Templates = t.TempDir()
	f := newFisher(cfg, nil, nil, newTracer(false))
	f.loadAnchor()
	if f.anchor != nil {
		t.Error("без файла якоря быть не должно")
	}
}

// Снимки не затирают друг друга между запусками.
func TestNextTemplateNameSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	if got := nextTemplateName(dir); got != "bobber-001.png" {
		t.Errorf("в пустом каталоге ожидалось bobber-001.png, получено %q", got)
	}
	for _, n := range []string{"bobber-001.png", "bobber-002.png", "bobber-004.png"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := nextTemplateName(dir); got != "bobber-003.png" {
		t.Errorf("должна занять первую дырку, получено %q", got)
	}
}
