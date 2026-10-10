//go:build !windows

package main

import hook "github.com/robotn/gohook"

func watchKeys(handlers map[string]func()) (stop func(), err error) {
	for name, on := range handlers {
		hook.Register(hook.KeyDown, []string{name}, func(hook.Event) { on() })
	}
	events := hook.Start()
	go func() { <-hook.Process(events) }()
	return hook.End, nil
}
