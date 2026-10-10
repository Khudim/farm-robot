package main

import (
	"log"
	"time"

	"farm-robot/internal/arduino"
	"farm-robot/internal/mouse"

	"github.com/go-vgo/robotgo"
)

// board — плата, через которую идёт весь ввод: программные нажатия WoW
// игнорирует, а USB-клавиатуру и мышь платы принимает как живые.
var board *arduino.Device

// useBoard подключает плату и переводит на неё клавиатуру, кнопки и движение мыши.
func useBoard(in InputConfig) (func(), error) {
	d, err := arduino.Open(in.Arduino)
	if err != nil {
		return nil, err
	}
	board = d
	mouse.UseMover(d, in.MoveMin.D(), in.MoveMax.D())
	log.Printf("ввод через плату на %s", d.Port())
	return func() {
		mouse.UseMover(nil, 0, 0)
		board = nil
		if err := d.Close(); err != nil {
			log.Printf("плата: %v", err)
		}
	}, nil
}

// tap жмёт бинд с удержанием: игра опрашивает клавиатуру раз в кадр, и
// мгновенное нажатие может не попасть ни в один опрос. Модификаторы идут
// отдельными событиями, как с живой клавиатуры.
func tap(h hotkey, hold time.Duration) {
	for _, m := range h.Mods {
		keyToggle(m, "down")
	}
	keyToggle(h.Key, "down")
	time.Sleep(vary(hold, 0.3))
	keyToggle(h.Key, "up")
	for i := len(h.Mods) - 1; i >= 0; i-- {
		keyToggle(h.Mods[i], "up")
	}
}

func keyToggle(key, dir string) {
	if err := board.Key(key, dir == "down"); err != nil {
		log.Printf("клавиша %s %s: %v", key, dir, err)
	}
}

// click держит кнопку мыши дольше кадра, иначе игра клик пропустит.
func click(button string, hold time.Duration) {
	defer func() {
		if err := board.Button(button, false); err != nil {
			log.Printf("кнопка %s up: %v", button, err)
		}
	}()
	if err := board.Button(button, true); err != nil {
		log.Printf("кнопка %s down: %v", button, err)
		return
	}
	time.Sleep(vary(hold, 0.3))
}

// releaseModifiers снимает модификаторы и даёт игре кадр-другой это заметить:
// иначе отпускание Shift и следующее нажатие попадут в один опрос.
func releaseModifiers(settle time.Duration) {
	for _, k := range []string{"shift", "ctrl", "alt"} {
		keyToggle(k, "up")
	}
	robotgo.MilliSleep(ms(settle))
}

// lootAll подсекает и забирает добычу. Shift держится до появления окна лута:
// автолут срабатывает при его открытии, через секунду после клика.
func lootAll(in InputConfig, shift bool) {
	defer releaseModifiers(in.Settle.D())
	if shift {
		keyToggle("shift", "down")
		time.Sleep(vary(100*time.Millisecond, 0.5))
	}
	click("right", in.Hold.D())
	time.Sleep(vary(1500*time.Millisecond, 0.2))
}

func ms(d time.Duration) int { return int(d.Milliseconds()) }
