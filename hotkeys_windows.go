//go:build windows

package main

import (
	"fmt"
	"strconv"
	"syscall"
	"time"
)

// Горячие клавиши на Windows опрашиваются, а не ловятся хуком. Глобальный хук
// пропускает через себя каждое событие ввода в системе: стоит процессу
// подвиснуть или встать в отладчике, и тормозит весь компьютер. Опрос
// состояния клавиш системе ничего не стоит.

var procGetAsyncKeyState = syscall.NewLazyDLL("user32.dll").NewProc("GetAsyncKeyState")

const keyPollInterval = 30 * time.Millisecond

// watchKeys следит за клавишами. Неизвестная клавиша не мешает остальным:
// stop годен всегда, err перечисляет, что опросить не удалось.
func watchKeys(handlers map[string]func()) (stop func(), err error) {
	type key struct {
		vk   uintptr
		on   func()
		down bool
	}
	var keys []*key
	var unknown []string
	for name, on := range handlers {
		vk, ok := hotkeyVK(name)
		if !ok {
			unknown = append(unknown, name)
			continue
		}
		keys = append(keys, &key{vk: vk, on: on})
	}
	if len(unknown) > 0 {
		err = fmt.Errorf("клавиши %v опросить не умею — годятся F1–F24, буквы и цифры", unknown)
	}

	done := make(chan struct{})
	go func() {
		t := time.NewTicker(keyPollInterval)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
			}
			for _, k := range keys {
				state, _, _ := procGetAsyncKeyState.Call(k.vk)
				down := state&0x8000 != 0
				if down && !k.down {
					k.on()
				}
				k.down = down
			}
		}
	}()
	return func() { close(done) }, err
}

// hotkeyVK — виртуальный код горячей клавиши: F1–F24, буквы и цифры.
func hotkeyVK(name string) (uintptr, bool) {
	switch {
	case len(name) == 1 && name[0] >= 'a' && name[0] <= 'z':
		return uintptr(name[0] - 'a' + 'A'), true
	case len(name) == 1 && name[0] >= '0' && name[0] <= '9':
		return uintptr(name[0]), true
	case len(name) >= 2 && name[0] == 'f':
		if n, err := strconv.Atoi(name[1:]); err == nil && n >= 1 && n <= 24 {
			return uintptr(0x70 + n - 1), true
		}
	}
	return 0, false
}
