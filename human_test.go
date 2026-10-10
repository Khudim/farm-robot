package main

import (
	"testing"
	"time"
)

func TestRandomStaysInBounds(t *testing.T) {
	r := Range{Duration(100 * time.Millisecond), Duration(300 * time.Millisecond)}
	ir := IntRange{25, 45}
	for range 1000 {
		if d := r.Pick(); d < 100*time.Millisecond || d >= 300*time.Millisecond {
			t.Fatalf("Range.Pick вне диапазона: %v", d)
		}
		if n := ir.Pick(); n < 25 || n > 45 {
			t.Fatalf("IntRange.Pick вне диапазона: %d", n)
		}
		if j := jitter(4); j < -4 || j > 4 {
			t.Fatalf("jitter вне диапазона: %d", j)
		}
		if d := vary(time.Second, 0.2); d < 800*time.Millisecond || d > 1200*time.Millisecond {
			t.Fatalf("vary вне диапазона: %v", d)
		}
	}
	if d := (Range{Duration(time.Second), Duration(time.Second)}).Pick(); d != time.Second {
		t.Fatalf("min == max: %v", d)
	}
}

func TestHumanValidate(t *testing.T) {
	if err := defaultHuman().validate(); err != nil {
		t.Fatalf("дефолт должен быть валиден: %v", err)
	}
	h := defaultHuman()
	h.Reaction = Range{Duration(time.Second), Duration(time.Millisecond)}
	if h.validate() == nil {
		t.Fatal("min > max должен быть ошибкой")
	}
	h = defaultHuman()
	h.Wander = 1.5
	if h.validate() == nil {
		t.Fatal("wander > 1 должен быть ошибкой")
	}
}
