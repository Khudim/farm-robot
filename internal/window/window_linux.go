//go:build linux

package window

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

var (
	mu   sync.Mutex
	conn *xgb.Conn
	root xproto.Window

	atomClientList, atomActiveWindow, atomNetWMName, atomUTF8String xproto.Atom
)

func connect() error {
	if conn != nil {
		return nil
	}
	c, err := xgb.NewConn()
	if err != nil {
		return fmt.Errorf("подключение к X11 (нужна сессия X11, не Wayland): %w", err)
	}
	root = xproto.Setup(c).DefaultScreen(c).Root

	for _, a := range []struct {
		name string
		dst  *xproto.Atom
	}{
		{"_NET_CLIENT_LIST", &atomClientList},
		{"_NET_ACTIVE_WINDOW", &atomActiveWindow},
		{"_NET_WM_NAME", &atomNetWMName},
		{"UTF8_STRING", &atomUTF8String},
	} {
		reply, err := xproto.InternAtom(c, true, uint16(len(a.name)), a.name).Reply()
		if err != nil {
			c.Close()
			return fmt.Errorf("атом %s: %w", a.name, err)
		}
		*a.dst = reply.Atom
	}
	conn = c
	return nil
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if conn != nil {
		conn.Close()
		conn = nil
	}
}

func windowName(w xproto.Window) string {
	if r, err := xproto.GetProperty(conn, false, w, atomNetWMName, atomUTF8String, 0, 1024).Reply(); err == nil && len(r.Value) > 0 {
		return string(r.Value)
	}
	if r, err := xproto.GetProperty(conn, false, w, xproto.AtomWmName, xproto.AtomString, 0, 1024).Reply(); err == nil && len(r.Value) > 0 {
		return string(r.Value)
	}
	return ""
}

func windowList() ([]xproto.Window, error) {
	r, err := xproto.GetProperty(conn, false, root, atomClientList, xproto.AtomWindow, 0, 4096).Reply()
	if err != nil {
		return nil, fmt.Errorf("_NET_CLIENT_LIST: %w", err)
	}
	out := make([]xproto.Window, 0, len(r.Value)/4)
	for i := 0; i+3 < len(r.Value); i += 4 {
		out = append(out, xproto.Window(readU32(r.Value[i:])))
	}
	return out, nil
}

func readU32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// matching — окна, чей заголовок содержит title. Вызывать под mu.
func matching(title string) ([]xproto.Window, error) {
	if err := connect(); err != nil {
		return nil, err
	}
	wins, err := windowList()
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(title)
	var out []xproto.Window
	for _, w := range wins {
		if strings.Contains(strings.ToLower(windowName(w)), needle) {
			out = append(out, w)
		}
	}
	return out, nil
}

// geometry — область окна в экранных координатах. GetGeometry отсчитывает от
// родителя (рамки оконного менеджера), поэтому начало переводится через root.
func geometry(w xproto.Window) (Rect, error) {
	g, err := xproto.GetGeometry(conn, xproto.Drawable(w)).Reply()
	if err != nil {
		return Rect{}, err
	}
	abs, err := xproto.TranslateCoordinates(conn, w, root, 0, 0).Reply()
	if err != nil {
		return Rect{}, err
	}
	return Rect{X: int(abs.DstX), Y: int(abs.DstY), W: int(g.Width), H: int(g.Height)}, nil
}

func Find(title string) (Rect, error) {
	mu.Lock()
	defer mu.Unlock()
	wins, err := matching(title)
	if err != nil {
		return Rect{}, err
	}
	if len(wins) == 0 {
		return Rect{}, ErrNotFound{Title: title}
	}
	return geometry(wins[0])
}

func List(title string) []Match {
	mu.Lock()
	defer mu.Unlock()
	wins, err := matching(title)
	if err != nil {
		return nil
	}
	var out []Match
	for _, w := range wins {
		if r, err := geometry(w); err == nil {
			out = append(out, Match{Title: windowName(w), Rect: r})
		}
	}
	return out
}

// Focused — активно ли любое из подходящих окон.
func Focused(title string) (bool, error) {
	mu.Lock()
	defer mu.Unlock()
	wins, err := matching(title)
	if err != nil {
		return false, err
	}
	if len(wins) == 0 {
		return false, ErrNotFound{Title: title}
	}
	r, err := xproto.GetProperty(conn, false, root, atomActiveWindow, xproto.AtomWindow, 0, 1).Reply()
	if err != nil || len(r.Value) < 4 {
		return false, fmt.Errorf("_NET_ACTIVE_WINDOW недоступен: %w", err)
	}
	active := xproto.Window(readU32(r.Value))
	for _, w := range wins {
		if w == active {
			return true, nil
		}
	}
	return false, nil
}
