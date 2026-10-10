// Package mouse — движение мыши через плату (Mover) и единая система
// координат: положение курсора читается через robotgo.
//
// robotgo на Windows делит координату на DPI-множитель в Move и умножает в
// Location, а снимок экрана и геометрия окна идут в сырых пикселях. При 100%
// это совпадает; при 125% найденный на снимке поплавок и точка, куда поедет
// мышь, расходятся на множитель. Здесь всё приводится к логическим пикселям.
package mouse

import (
	"log"
	"math"
	"time"

	"github.com/go-vgo/robotgo"
)

// Mover сдвигает мышь относительно: так ходит USB-мышь, у неё нет абсолютных
// координат.
type Mover interface {
	// Glide ведёт мышь плавной кривой за случайное время из [min, max].
	Glide(dx, dy int, min, max time.Duration) error
	// Step сдвигает мгновенно.
	Step(dx, dy int) error
}

var (
	mover              Mover
	glideMin, glideMax time.Duration
)

// correctGlide — длительность доводки: промах после основного движения мал, и
// вести его полсекунды незачем.
const correctGlide = 120 * time.Millisecond

// UseMover задаёт, чем двигать мышь и за сколько проходить путь; без него
// MoveTo не сработает.
func UseMover(m Mover, min, max time.Duration) {
	mover, glideMin, glideMax = m, min, max
}

// Scale — множитель robotgo, 1 при 100%.
func Scale() float64 {
	if f := robotgo.ScaleF(); f > 0 {
		return f
	}
	return 1
}

func ToPhysical(v int, f float64) int { return int(float64(v)*f + 0.5) }
func ToLogical(v int, f float64) int  { return int(float64(v)/f + 0.5) }

// MoveTo ставит мышь в логическую точку экрана.
func MoveTo(x, y int) {
	if mover == nil {
		log.Printf("мышь: не задано, чем двигать — UseMover")
		return
	}
	moveRelative(x, y)
}

// moveRelative ведёт мышь сдвигами с обратной связью: отсчёт мыши — не пиксель,
// если в системе включено «Повышенная точность установки указателя» или
// скорость не 6/11, поэтому сдвигаемся, смотрим, где курсор, и доводим.
func moveRelative(x, y int) {
	f := Scale()
	for i := range 12 {
		cx, cy := Pos()
		dx, dy := x-cx, y-cy
		if abs(dx) <= 1 && abs(dy) <= 1 {
			return
		}
		var err error
		switch {
		case abs(dx) <= 3 && abs(dy) <= 3:
			// Ускорение указателя раздувает даже мелкий сдвиг, и курсор
			// прыгает через точку; вблизи доводим по одному отсчёту.
			err = mover.Step(sign(dx), sign(dy))
		case i == 0 && replayTravel(ToPhysical(dx, f), ToPhysical(dy, f)):
			// Записанное движение руки; дальше доводка, как и после кривой.
		case i == 0:
			// Дальнее движение у человека дольше ближнего (закон Фиттса).
			k := min(1.5, max(0.5, 0.5+math.Hypot(float64(dx), float64(dy))/600))
			lo, hi := time.Duration(float64(glideMin)*k), time.Duration(float64(glideMax)*k)
			err = mover.Glide(ToPhysical(dx, f), ToPhysical(dy, f), lo, hi)
		default:
			err = mover.Glide(ToPhysical(dx, f), ToPhysical(dy, f), correctGlide, correctGlide)
		}
		if err != nil {
			log.Printf("мышь: %v", err)
			return
		}
		time.Sleep(15 * time.Millisecond) // курсор догоняет отчёт не сразу
	}
	cx, cy := Pos()
	log.Printf("мышь: не довёл до (%d, %d), стоит в (%d, %d)", x, y, cx, cy)
}

// Stroke ведёт мышь по кривой за dur: curve(t) при t от 0 до 1 даёт смещение от
// начальной точки в логических пикселях, curve(0) = (0, 0). Идёт мелкими
// мгновенными сдвигами, чтобы фигура была плавной без пауз между отрезками.
// abort проверяется на каждом шаге; true — фигура прервана.
func Stroke(curve func(t float64) (x, y float64), dur time.Duration, abort func() bool) bool {
	if mover == nil {
		log.Printf("мышь: не задано, чем двигать — UseMover")
		return false
	}
	f := Scale()
	px, py := 0, 0
	start := time.Now()
	for {
		t := min(1, float64(time.Since(start))/float64(dur))
		cx, cy := curve(t)
		ix, iy := int(math.Round(cx*f)), int(math.Round(cy*f))
		if ix != px || iy != py {
			if err := mover.Step(ix-px, iy-py); err != nil {
				log.Printf("мышь: %v", err)
				return false
			}
			px, py = ix, iy
		}
		if t >= 1 {
			return false
		}
		if abort() {
			return true
		}
		time.Sleep(8 * time.Millisecond)
	}
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// Pos — положение мыши в логических пикселях.
func Pos() (int, int) {
	f := Scale()
	x, y := robotgo.Location()
	return ToLogical(x, f), ToLogical(y, f)
}

func LogScale() {
	if f := Scale(); f != 1 {
		log.Printf("масштаб экрана %.0f%% — координаты мыши и снимков приводятся к одной системе", f*100)
	}
}
