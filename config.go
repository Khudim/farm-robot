package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"farm-robot/internal/arduino"

	"gopkg.in/yaml.v2"
)

// Duration разбирается из строк вида "1500ms" — yaml.v2 сам их не понимает.
type Duration time.Duration

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("некорректная длительность %q: %w", s, err)
	}
	*d = Duration(v)
	return nil
}

func (d Duration) D() time.Duration { return time.Duration(d) }

type AppConfig struct {
	Port   int          `yaml:"port"`
	Window WindowConfig `yaml:"window"`
	Keys   KeysConfig   `yaml:"keys"`
	Loot   LootConfig   `yaml:"loot"`
	Search SearchConfig `yaml:"search"`
	Splash SplashConfig `yaml:"splash"`
	Input  InputConfig  `yaml:"input"`
	Timing TimingConfig `yaml:"timing"`
	Human  HumanConfig  `yaml:"human"`
}

type WindowConfig struct {
	Title        string `yaml:"title"`        // подстрока заголовка; пусто — экранные координаты
	RequireFocus bool   `yaml:"requireFocus"` // не жать клавиши, пока окно игры не активно
}

type KeysConfig struct {
	Cast string `yaml:"cast"`
	Bait string `yaml:"bait"`
}

// Shift в WoW инвертирует автолут: значение должно быть противоположно игровой настройке.
type LootConfig struct {
	Shift bool `yaml:"shift"`
}

type SearchConfig struct {
	Templates  string  `yaml:"templates"`  // каталог эталонов
	MinScore   float64 `yaml:"minScore"`   // порог совпадения, (0, 1]
	Stride     int     `yaml:"stride"`     // шаг грубого прохода
	GrabKey    string  `yaml:"grabKey"`    // снять эталон из-под курсора
	GrabRadius int     `yaml:"grabRadius"` // пикселей от курсора в каждую сторону
	Width      int     `yaml:"width"`      // область поиска вокруг поплавка
	Height     int     `yaml:"height"`
}

type SplashConfig struct {
	Threshold  float64  `yaml:"threshold"`  // во сколько раз пик громче фона
	MinLevel   float64  `yaml:"minLevel"`   // абсолютный порог RMS
	Refractory Duration `yaml:"refractory"` // зазор между срабатываниями
}

// Игра опрашивает клавиатуру и мышь раз в кадр; события короче кадра теряются.
type InputConfig struct {
	Hold    Duration `yaml:"hold"`    // удержание клавиши и кнопки мыши
	Settle  Duration `yaml:"settle"`  // пауза после отпускания модификаторов
	Arduino string   `yaml:"arduino"` // порт платы с hid_emulator: "auto" или "COM4"
	MoveMin Duration `yaml:"moveMin"` // мышь доезжает до точки за случайное время
	MoveMax Duration `yaml:"moveMax"` // из [moveMin, moveMax]
}

type TimingConfig struct {
	CastSettle   Duration `yaml:"castSettle"`   // до первой проверки после заброса
	CastTimeout  Duration `yaml:"castTimeout"`  // сколько ждать появления поплавка
	CastRetries  int      `yaml:"castRetries"`  // повторов заброса
	BiteGrace    Duration `yaml:"biteGrace"`    // после появления поплавка не слушать: его приводнение тоже всплеск
	BiteTimeout  Duration `yaml:"biteTimeout"`  // сколько ждать поклёвку
	LootSettle   Duration `yaml:"lootSettle"`   // пауза после лута
	BaitInterval Duration `yaml:"baitInterval"` // период обновления наживки
	MaxMisses    int      `yaml:"maxMisses"`    // неудач подряд до остановки
}

func defaultConfig() AppConfig {
	return AppConfig{
		Port:   9100,
		Window: WindowConfig{Title: "World of Warcraft", RequireFocus: true},
		Keys:   KeysConfig{Cast: "ctrl+k", Bait: "ctrl+r"},
		Loot:   LootConfig{Shift: true},
		Search: SearchConfig{
			Templates: "./templates/bobber", MinScore: 0.6, Stride: 2,
			GrabKey: "f6", GrabRadius: 20, Width: 320, Height: 220,
		},
		Splash: SplashConfig{Threshold: 4, MinLevel: 0.02, Refractory: Duration(1500 * time.Millisecond)},
		Input: InputConfig{
			Hold:    Duration(80 * time.Millisecond),
			Settle:  Duration(150 * time.Millisecond),
			Arduino: "auto",
			MoveMin: Duration(300 * time.Millisecond),
			MoveMax: Duration(600 * time.Millisecond),
		},
		Human: defaultHuman(),
		Timing: TimingConfig{
			CastSettle:   Duration(1200 * time.Millisecond),
			CastTimeout:  Duration(5 * time.Second),
			CastRetries:  2,
			BiteGrace:    Duration(1500 * time.Millisecond),
			BiteTimeout:  Duration(22 * time.Second),
			LootSettle:   Duration(1500 * time.Millisecond),
			BaitInterval: Duration(10 * time.Minute),
			MaxMisses:    20,
		},
	}
}

func loadConfig(path string) (AppConfig, error) {
	cfg := defaultConfig()
	raw, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("чтение %s: %w", path, err)
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("разбор %s: %w", path, err)
	}
	return cfg, cfg.validate()
}

func (c AppConfig) validate() error {
	for name, bind := range map[string]string{
		"keys.cast": c.Keys.Cast, "keys.bait": c.Keys.Bait, "search.grabKey": c.Search.GrabKey,
	} {
		if _, err := parseHotkey(bind); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}

	if c.Input.Arduino == "" {
		return fmt.Errorf("input.arduino: не задан порт платы, \"auto\" — найти самому")
	}
	if c.Input.MoveMin.D() <= 0 || c.Input.MoveMax.D() < c.Input.MoveMin.D() {
		return fmt.Errorf("input.moveMin должен быть больше нуля и не больше input.moveMax")
	}

	s := c.Search
	switch {
	case s.Templates == "":
		return fmt.Errorf("search.templates: не задан каталог эталонов")
	case s.MinScore <= 0 || s.MinScore > 1:
		return fmt.Errorf("search.minScore должен быть в (0, 1], задано %v", s.MinScore)
	case s.Stride < 1:
		return fmt.Errorf("search.stride должен быть не меньше 1")
	case s.GrabRadius < 2:
		return fmt.Errorf("search.grabRadius должен быть не меньше 2")
	case s.Width <= 0 || s.Height <= 0:
		return fmt.Errorf("search.width и search.height должны быть больше нуля")
	}

	if c.Splash.Threshold <= 1 {
		return fmt.Errorf("splash.threshold должен быть больше 1, иначе сработает на фоне")
	}
	for name, d := range map[string]Duration{
		"input.hold": c.Input.Hold, "input.settle": c.Input.Settle,
		"timing.castTimeout": c.Timing.CastTimeout, "timing.biteTimeout": c.Timing.BiteTimeout,
	} {
		if d.D() <= 0 {
			return fmt.Errorf("%s должен быть больше нуля", name)
		}
	}
	if err := c.Human.validate(); err != nil {
		return err
	}
	if c.Timing.CastRetries < 1 {
		return fmt.Errorf("timing.castRetries должен быть не меньше 1")
	}
	if c.Timing.MaxMisses < 1 {
		return fmt.Errorf("timing.maxMisses должен быть больше нуля")
	}
	return nil
}

// hotkey — разобранный бинд вида "ctrl+k".
type hotkey struct {
	Key  string
	Mods []string
}

// Модификаторы бинда. Неизвестное имя не должно молча превратиться в голую
// клавишу, поэтому проверяем сами.
var knownMods = map[string]string{
	"ctrl": "ctrl", "lctrl": "lctrl", "rctrl": "rctrl", "control": "ctrl",
	"alt": "alt", "lalt": "lalt", "ralt": "ralt",
	"shift": "shift", "lshift": "lshift", "rshift": "rshift",
	"cmd": "cmd", "win": "cmd", "super": "cmd", "meta": "cmd",
}

func parseHotkey(s string) (hotkey, error) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(s)), "+")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if parts[len(parts)-1] == "" {
		return hotkey{}, fmt.Errorf("пустой бинд %q", s)
	}
	h := hotkey{Key: parts[len(parts)-1]}
	if !arduino.Known(h.Key) {
		return hotkey{}, fmt.Errorf("неизвестная клавиша %q в бинде %q", h.Key, s)
	}
	for _, m := range parts[:len(parts)-1] {
		canonical, ok := knownMods[m]
		if !ok {
			return hotkey{}, fmt.Errorf("неизвестный модификатор %q в бинде %q", m, s)
		}
		h.Mods = append(h.Mods, canonical)
	}
	return h, nil
}

func (h hotkey) String() string {
	if len(h.Mods) == 0 {
		return h.Key
	}
	return strings.Join(h.Mods, "+") + "+" + h.Key
}
