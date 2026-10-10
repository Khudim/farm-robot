//go:build windows

package main

import "testing"

func TestHotkeyVK(t *testing.T) {
	for name, want := range map[string]uintptr{"f1": 0x70, "f6": 0x75, "f24": 0x87, "g": 0x47, "5": 0x35} {
		if got, ok := hotkeyVK(name); !ok || got != want {
			t.Errorf("hotkeyVK(%q) = %#x, %v; want %#x", name, got, ok, want)
		}
	}
	for _, name := range []string{"f0", "f25", "ctrl", ""} {
		if _, ok := hotkeyVK(name); ok {
			t.Errorf("hotkeyVK(%q) должен быть неизвестен", name)
		}
	}
}

// Неизвестная клавиша отключает только себя: F4 для выхода должен работать.
func TestUnknownHotkeyKeepsOthers(t *testing.T) {
	stop, err := watchKeys(map[string]func(){"f4": func() {}, "space": func() {}})
	if err == nil {
		t.Fatal("про space ожидалась ошибка")
	}
	if stop == nil {
		t.Fatal("остальные клавиши должны опрашиваться")
	}
	stop()
}
