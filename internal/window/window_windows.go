//go:build windows

package window

import (
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")

	procEnumWindows         = user32.NewProc("EnumWindows")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procGetClientRect       = user32.NewProc("GetClientRect")
	procClientToScreen      = user32.NewProc("ClientToScreen")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
)

type winRect struct{ Left, Top, Right, Bottom int32 }
type winPoint struct{ X, Y int32 }

type winMatch struct {
	hwnd syscall.Handle
	Match
}

// Колбэк перебора создаётся один раз: слоты syscall.NewCallback не освобождаются,
// а окно ищется на каждом забросе.
var (
	enumMu       sync.Mutex
	enumOnce     sync.Once
	enumCallback uintptr
	enumNeedle   string
	enumFirst    bool
	enumResult   []winMatch
)

func enumProc(hwnd syscall.Handle, _ uintptr) uintptr {
	const keepGoing, stop = 1, 0

	if v, _, _ := procIsWindowVisible.Call(uintptr(hwnd)); v == 0 {
		return keepGoing
	}
	var buf [512]uint16
	n, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return keepGoing
	}
	title := syscall.UTF16ToString(buf[:n])
	if !strings.Contains(strings.ToLower(title), enumNeedle) {
		return keepGoing
	}

	var rc winRect
	if r, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc))); r == 0 {
		return keepGoing
	}
	origin := winPoint{}
	if r, _, _ := procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&origin))); r == 0 {
		return keepGoing
	}

	enumResult = append(enumResult, winMatch{hwnd: hwnd, Match: Match{
		Title: title,
		Rect:  Rect{X: int(origin.X), Y: int(origin.Y), W: int(rc.Right - rc.Left), H: int(rc.Bottom - rc.Top)},
	}})
	if enumFirst {
		return stop
	}
	return keepGoing
}

func enumWindows(title string, firstOnly bool) []winMatch {
	enumMu.Lock()
	defer enumMu.Unlock()

	enumOnce.Do(func() { enumCallback = syscall.NewCallback(enumProc) })
	enumNeedle, enumFirst, enumResult = strings.ToLower(title), firstOnly, nil
	procEnumWindows.Call(enumCallback, 0)

	out := enumResult
	enumResult = nil
	return out
}

// Find — клиентская область первого окна с таким заголовком, без рамки и шапки.
func Find(title string) (Rect, error) {
	m := enumWindows(title, true)
	if len(m) == 0 {
		return Rect{}, ErrNotFound{Title: title}
	}
	return m[0].Rect, nil
}

// List — все видимые окна с таким заголовком: лаунчер Battle.net тоже подходит.
func List(title string) []Match {
	found := enumWindows(title, false)
	out := make([]Match, 0, len(found))
	for _, m := range found {
		out = append(out, m.Match)
	}
	return out
}

// Focused — активно ли любое из подходящих окон.
func Focused(title string) (bool, error) {
	matches := enumWindows(title, false)
	if len(matches) == 0 {
		return false, ErrNotFound{Title: title}
	}
	fg, _, _ := procGetForegroundWindow.Call()
	for _, m := range matches {
		if syscall.Handle(fg) == m.hwnd {
			return true, nil
		}
	}
	return false, nil
}

func Close() {}
