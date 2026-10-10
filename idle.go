package main

import (
	"math"
	"time"

	"farm-robot/internal/mouse"
)

// waitBite ждёт поклёвку и между делом шевелит мышью, как человек, которому
// скучно смотреть на поплавок: восьмёрка, покачивание, круг, сдвиг в сторону.
// p — где поплавок: мышь далеко от него не уходит.
func (f *Fisher) waitBite(p point) bool {
	timeout := f.cfg.Timing.BiteTimeout.D()
	if f.cfg.Human.IdleEvery.Max.D() == 0 {
		return f.splash.Wait(timeout)
	}
	deadline := time.Now().Add(timeout)
	for {
		left := time.Until(deadline)
		if left <= 0 || f.stopping() {
			return false
		}
		if f.splash.Wait(min(left, f.cfg.Human.IdleEvery.Pick())) {
			return true
		}
		if time.Until(deadline) > 0 && f.fidget(p) {
			return true
		}
	}
}

// fidget делает одно случайное шевеление; true — во время него клюнуло.
func (f *Fisher) fidget(p point) bool {
	size := float64(f.cfg.Human.IdleSize.Pick())
	if size == 0 {
		return false
	}
	// Шевеления копят снос; ушли далеко — сперва обратно к поплавку.
	if x, y := mouse.Pos(); math.Hypot(float64(x-p.X), float64(y-p.Y)) > 2*size {
		mouse.MoveTo(p.X+jitter(int(size/2)), p.Y+jitter(int(size/2)))
		f.trace.step("шевеление: обратно к поплавку")
		return f.splash.Poll()
	}

	if mouse.HasGestures() {
		aborted, took := mouse.PlayIdle(size, f.splash.Poll)
		f.trace.step("шевеление: из записи, размах до %.0f px, %v", size, took.Round(10*time.Millisecond))
		return aborted
	}

	name, curve, dur := mouse.RandomFigure(size)
	if curve == nil {
		x, y := mouse.Pos()
		mouse.MoveTo(x+jitter(int(size)), y+jitter(int(size)))
		f.trace.step("шевеление: сдвиг")
		return f.splash.Poll()
	}
	f.trace.step("шевеление: %s, размах %.0f px, %v", name, size, dur.Round(10*time.Millisecond))
	return mouse.Stroke(curve, dur, f.splash.Poll)
}

// backToBobber возвращает мышь на поплавок, если шевеление её увело.
func (f *Fisher) backToBobber(p point) {
	j := f.cfg.Human.AimJitter
	if x, y := mouse.Pos(); abs(x-p.X) > j || abs(y-p.Y) > j {
		mouse.MoveTo(p.X+jitter(j), p.Y+jitter(j))
	}
}
