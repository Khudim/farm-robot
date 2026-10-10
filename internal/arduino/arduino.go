// Package arduino шлёт ввод через Pro Micro со скетчем hid_emulator
// (github.com/mvandrew/arduino-hid-emulator): плата видна системе как USB-
// клавиатура и мышь, и игра принимает её нажатия как от живого устройства.
//
// Протокол скетча — строка-команда, в ответ "True" или "False":
//
//	key_down <клавиша>, key_up <клавиша>     — "ctrl", "a" или код "0xC2"
//	mouse_down left|right, mouse_up left|right
//	mouse_move_direct dx,dy                   — мгновенный относительный сдвиг
//	mouse_move dx,dy,kx,ky,min,max            — сдвиг кривой Безье за min..max мс;
//	                                            ответ приходит по завершении
//
// Скетч сам выжидает 150–300 мс перед каждым нажатием и отпусканием.
package arduino

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// replyTimeout перекрывает задержку скетча перед нажатием (до 300 мс).
const replyTimeout = 2 * time.Second

// Device — открытый порт платы. Безопасен для вызова из разных горутин.
type Device struct {
	mu   sync.Mutex
	port serial.Port
	name string
}

// Open подключается к плате; port "auto" — найти плату Arduino среди портов.
func Open(port string) (*Device, error) {
	if port == "auto" {
		found, err := Find()
		if err != nil {
			return nil, err
		}
		port = found
	}
	// Не 1200: открытие на 1200 бод перезагружает 32U4 в загрузчик.
	p, err := serial.Open(port, &serial.Mode{
		BaudRate:          9600,
		InitialStatusBits: &serial.ModemOutputBits{DTR: true, RTS: true},
	})
	if err != nil {
		return nil, fmt.Errorf("открытие %s: %w", port, err)
	}
	if err := p.SetReadTimeout(100 * time.Millisecond); err != nil {
		p.Close()
		return nil, fmt.Errorf("%s: %w", port, err)
	}
	d := &Device{port: p, name: port}

	// Нулевой сдвиг мыши ничего не делает, но проверяет, что на том конце скетч.
	if err := d.send("mouse_move_direct 0,0", replyTimeout); err != nil {
		p.Close()
		return nil, fmt.Errorf("%s не отвечает как hid_emulator: %w", port, err)
	}
	return d, nil
}

// Find ищет порт платы по USB VID: Arduino и SparkFun.
func Find() (string, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return "", fmt.Errorf("список портов: %w", err)
	}
	for _, p := range ports {
		if !p.IsUSB {
			continue
		}
		switch strings.ToUpper(p.VID) {
		case "2341", "2A03", "1B4F":
			// 0036 — загрузчик, он живёт 8 секунд после сброса.
			if strings.ToUpper(p.PID) != "0036" {
				return p.Name, nil
			}
		}
	}
	return "", fmt.Errorf("плата Arduino не найдена среди COM-портов")
}

func (d *Device) Port() string { return d.name }

// Key жмёт или отпускает клавишу по имени из конфига ("ctrl", "k", "f6").
func (d *Device) Key(name string, down bool) error {
	token, ok := keyToken(name)
	if !ok {
		return fmt.Errorf("клавиша %q не поддерживается платой", name)
	}
	cmd := "key_up "
	if down {
		cmd = "key_down "
	}
	return d.send(cmd+token, replyTimeout)
}

// Button жмёт или отпускает кнопку мыши: "left" или "right".
func (d *Device) Button(button string, down bool) error {
	if button != "left" && button != "right" {
		return fmt.Errorf("кнопка мыши %q не поддерживается платой", button)
	}
	cmd := "mouse_up "
	if down {
		cmd = "mouse_down "
	}
	return d.send(cmd+button, replyTimeout)
}

// Glide ведёт мышь на dx, dy отсчётов по кривой за случайное время из
// [min, max]. Скетч отвечает, только доехав, плюс 200 мс своей доводки.
func (d *Device) Glide(dx, dy int, min, max time.Duration) error {
	if dx == 0 && dy == 0 {
		return nil // на нулевой сдвиг скетч отвечает False
	}
	cmd := fmt.Sprintf("mouse_move %d,%d,1,1,%d,%d", dx, dy, min.Milliseconds(), max.Milliseconds())
	return d.send(cmd, max+200*time.Millisecond+replyTimeout)
}

// Step сдвигает мышь на dx, dy отсчётов мгновенно. HID-отчёт вмещает ±127 на
// ось, длинный сдвиг идёт несколькими командами.
func (d *Device) Step(dx, dy int) error {
	for dx != 0 || dy != 0 {
		sx, sy := clamp(dx), clamp(dy)
		if err := d.send(fmt.Sprintf("mouse_move_direct %d,%d", sx, sy), replyTimeout); err != nil {
			return err
		}
		dx -= sx
		dy -= sy
	}
	return nil
}

// Close отпускает всё, что могло остаться зажатым, и закрывает порт.
func (d *Device) Close() error {
	for _, k := range []string{"shift", "ctrl", "alt"} {
		_ = d.Key(k, false)
	}
	for _, b := range []string{"left", "right"} {
		_ = d.Button(b, false)
	}
	return d.port.Close()
}

func (d *Device) send(cmd string, timeout time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	// Ответ, опоздавший после прошлого таймаута, иначе засчитается этой команде.
	_ = d.port.ResetInputBuffer()
	if _, err := d.port.Write([]byte(cmd + "\n")); err != nil {
		return fmt.Errorf("%s: запись: %w", cmd, err)
	}
	reply, err := d.readLine(timeout)
	if err != nil {
		return fmt.Errorf("%s: %w", cmd, err)
	}
	switch reply {
	case "True":
		return nil
	case "False":
		return fmt.Errorf("%s: скетч ответил False", cmd)
	}
	return fmt.Errorf("%s: непонятный ответ %q", cmd, reply)
}

// readLine читает строку ответа. Read порта по таймауту возвращает 0 байт без
// ошибки, поэтому срок отсчитываем сами.
func (d *Device) readLine(timeout time.Duration) (string, error) {
	var line []byte
	buf := make([]byte, 64)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		n, err := d.port.Read(buf)
		if err != nil {
			return "", fmt.Errorf("чтение ответа: %w", err)
		}
		line = append(line, buf[:n]...)
		if i := bytes.IndexByte(line, '\n'); i >= 0 {
			return strings.TrimSpace(string(line[:i])), nil
		}
	}
	return "", fmt.Errorf("нет ответа за %v", timeout)
}

func clamp(v int) int {
	return max(-127, min(127, v))
}
