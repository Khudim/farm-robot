package arduino

import (
	"fmt"
	"strconv"
)

// Коды Keyboard.h. Скетч понимает по имени лишь несколько клавиш, а у прочих
// берёт первую букву («rctrl» превратится в «r»), поэтому всё шлём кодом.
var namedCodes = map[string]byte{
	"ctrl": 0x80, "lctrl": 0x80, "control": 0x80, "rctrl": 0x84,
	"shift": 0x81, "lshift": 0x81, "rshift": 0x85,
	"alt": 0x82, "lalt": 0x82, "ralt": 0x86,
	"cmd": 0x83, "win": 0x83, "super": 0x83, "meta": 0x83,
	"up": 0xDA, "down": 0xD9, "left": 0xD8, "right": 0xD7,
	"backspace": 0xB2, "tab": 0xB3, "enter": 0xB0, "esc": 0xB1, "escape": 0xB1,
	"insert": 0xD1, "delete": 0xD4, "pageup": 0xD3, "pagedown": 0xD6,
	"home": 0xD2, "end": 0xD5, "capslock": 0xC1, "numlock": 0xDB,
	"space": ' ',
}

// Code — код Keyboard.h для имени клавиши из конфига.
func Code(name string) (byte, bool) {
	if c, ok := namedCodes[name]; ok {
		return c, true
	}
	switch {
	case len(name) == 1 && (name[0] >= 'a' && name[0] <= 'z' || name[0] >= '0' && name[0] <= '9'):
		return name[0], true
	case len(name) == 1 && isPunct(name[0]):
		return name[0], true
	case len(name) == 4 && name[:3] == "num" && name[3] >= '0' && name[3] <= '9':
		if name[3] == '0' {
			return 0xEA, true
		}
		return 0xE1 + name[3] - '1', true
	case len(name) >= 2 && name[0] == 'f':
		n, err := strconv.Atoi(name[1:])
		switch {
		case err != nil:
		case n >= 1 && n <= 12:
			return byte(0xC2 + n - 1), true
		case n >= 13 && n <= 24:
			return byte(0xF0 + n - 13), true
		}
	}
	return 0, false
}

// Known — может ли плата нажать такую клавишу.
func Known(name string) bool {
	_, ok := Code(name)
	return ok
}

func keyToken(name string) (string, bool) {
	c, ok := Code(name)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("0x%02X", c), true
}

// Знаки без Shift на US-раскладке: Keyboard.h с ними жмёт голую клавишу.
func isPunct(c byte) bool {
	switch c {
	case '-', '=', ',', '.', '/', ';', '\'', '[', ']', '\\', '`':
		return true
	}
	return false
}
