package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDurationParsing(t *testing.T) {
	var d Duration
	set := func(v interface{}) error { *(v.(*string)) = "1500ms"; return nil }
	if err := d.UnmarshalYAML(set); err != nil || d.D() != 1500*time.Millisecond {
		t.Fatalf("получено %v, %v", d.D(), err)
	}
	bad := func(v interface{}) error { *(v.(*string)) = "полторы секунды"; return nil }
	if err := d.UnmarshalYAML(bad); err == nil {
		t.Fatal("ожидалась ошибка разбора")
	}
}

// Не заданные в файле значения берутся из дефолтов, а не обнуляются.
func TestConfigFallsBackToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "props.yaml")
	if err := os.WriteFile(path, []byte("search:\n  minScore: 0.7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	def := defaultConfig()
	if cfg.Search.MinScore != 0.7 {
		t.Errorf("minScore не прочитан: %v", cfg.Search.MinScore)
	}
	if cfg.Search.Width != def.Search.Width ||
		cfg.Timing.BiteTimeout != def.Timing.BiteTimeout ||
		cfg.Splash.Refractory != def.Splash.Refractory {
		t.Errorf("дефолты потеряны: %+v", cfg)
	}
}

func TestValidateRejectsBadValues(t *testing.T) {
	cases := map[string]func(*AppConfig){
		"опечатка в бинде":      func(c *AppConfig) { c.Keys.Cast = "crtl+k" },
		"нет каталога эталонов": func(c *AppConfig) { c.Search.Templates = "" },
		"minScore вне (0,1]":    func(c *AppConfig) { c.Search.MinScore = 1.5 },
		"нулевой stride":        func(c *AppConfig) { c.Search.Stride = 0 },
		"радиус снимка":         func(c *AppConfig) { c.Search.GrabRadius = 1 },
		"нулевая область":       func(c *AppConfig) { c.Search.Width = 0 },
		"порог звука <= 1":      func(c *AppConfig) { c.Splash.Threshold = 1 },
		"нулевое удержание":     func(c *AppConfig) { c.Input.Hold = 0 },
		"нулевая пауза":         func(c *AppConfig) { c.Input.Settle = 0 },
		"нет попыток заброса":   func(c *AppConfig) { c.Timing.CastRetries = 0 },
		"нулевой castTimeout":   func(c *AppConfig) { c.Timing.CastTimeout = 0 },
		"нулевой maxMisses":     func(c *AppConfig) { c.Timing.MaxMisses = 0 },
	}
	for name, mutate := range cases {
		cfg := defaultConfig()
		mutate(&cfg)
		if err := cfg.validate(); err == nil {
			t.Errorf("%s: ожидалась ошибка валидации", name)
		}
	}
	if err := defaultConfig().validate(); err != nil {
		t.Errorf("дефолтный конфиг должен быть валиден: %v", err)
	}
}

// Неизвестный модификатор не должен молча превратиться в голую клавишу.
// Ловим на разборе.
func TestParseHotkey(t *testing.T) {
	ok := map[string]string{
		"ctrl+k": "ctrl+k", "control+k": "ctrl+k", "CTRL+K": "ctrl+k", " ctrl + k ": "ctrl+k",
		"ctrl+shift+f1": "ctrl+shift+f1", "win+e": "cmd+e", "k": "k",
	}
	for in, want := range ok {
		h, err := parseHotkey(in)
		if err != nil || h.String() != want {
			t.Errorf("%q → %v, %v; ожидалось %q", in, h, err, want)
		}
	}
	for _, in := range []string{"crtl+k", "opt+k", "ctrl+", ""} {
		if _, err := parseHotkey(in); err == nil {
			t.Errorf("%q: ожидалась ошибка", in)
		}
	}
}
