package arduino

import "testing"

func TestKeyToken(t *testing.T) {
	for name, want := range map[string]string{
		"ctrl": "0x80", "rctrl": "0x84", "shift": "0x81", "k": "0x6B", "0": "0x30",
		"f1": "0xC2", "f12": "0xCD", "f13": "0xF0", "num0": "0xEA", "num1": "0xE1",
		"space": "0x20", "esc": "0xB1", "-": "0x2D",
	} {
		got, ok := keyToken(name)
		if !ok || got != want {
			t.Errorf("keyToken(%q) = %q, %v; want %q", name, got, ok, want)
		}
	}
	for _, name := range []string{"", "K", "f0", "f25", "foo", "!"} {
		if Known(name) {
			t.Errorf("Known(%q) = true, want false", name)
		}
	}
}
