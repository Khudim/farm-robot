// mousedemo гоняет мышь через плату по случайным точкам экрана, чтобы глазами
// посмотреть на движение: кривую, скорость, доводку, точность.
//
//	go run ./cmd/mousedemo -n 10
//	go run ./cmd/mousedemo -idle 8     — шевеления, как при ожидании поклёвки
//
// Остановить — Ctrl+C или увести мышь в левый верхний угол экрана.
package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"farm-robot/internal/arduino"
	"farm-robot/internal/mouse"

	"github.com/go-vgo/robotgo"
)

func main() {
	port := flag.String("port", "auto", "порт платы")
	n := flag.Int("n", 10, "сколько точек")
	minD := flag.Duration("min", 300*time.Millisecond, "минимальная длительность движения")
	maxD := flag.Duration("max", 600*time.Millisecond, "максимальная длительность движения")
	pause := flag.Duration("pause", 700*time.Millisecond, "пауза в каждой точке")
	idle := flag.Int("idle", 0, "вместо точек показать столько шевелений, как при ожидании поклёвки")
	size := flag.Int("size", 50, "размах шевелений, px")
	recPath := flag.String("rec", "mouse.json", "запись движений руки (cmd/mouserec); пусто — синтетика")
	flag.Parse()

	d, err := arduino.Open(*port)
	if err != nil {
		fmt.Println("плата:", err)
		return
	}
	defer d.Close()
	mouse.UseMover(d, *minD, *maxD)
	if *recPath != "" {
		if rec, err := mouse.LoadRecording(*recPath); err == nil {
			travel, idles := mouse.UseGestures(rec.Gestures)
			fmt.Printf("движения из %s: %d переходов, %d шевелений\n", *recPath, travel, idles)
		} else {
			fmt.Printf("записи нет (%v) — синтетика\n", err)
		}
	}

	sw, sh := robotgo.GetScreenSize()
	f := mouse.Scale()
	w, h := mouse.ToLogical(sw, f), mouse.ToLogical(sh, f)
	fmt.Printf("плата %s, экран %dx%d, движение %v..%v\n", d.Port(), w, h, *minD, *maxD)
	fmt.Println("через 2 с поехали; стоп — мышь в левый верхний угол")
	time.Sleep(2 * time.Second)

	if *idle > 0 {
		for i := 1; i <= *idle; i++ {
			if mouse.HasGestures() {
				_, took := mouse.PlayIdle(float64(*size), func() bool { return false })
				fmt.Printf("%2d: из записи %v\n", i, took.Round(10*time.Millisecond))
				time.Sleep(*pause)
				continue
			}
			name, curve, dur := mouse.RandomFigure(float64(*size))
			if curve == nil {
				x, y := mouse.Pos()
				mouse.MoveTo(x+rand.IntN(2**size+1)-*size, y+rand.IntN(2**size+1)-*size)
			} else {
				mouse.Stroke(curve, dur, func() bool { return false })
			}
			fmt.Printf("%2d: %s %v\n", i, name, dur.Round(10*time.Millisecond))
			time.Sleep(*pause)
		}
		return
	}

	var misses []float64
	for i := 1; i <= *n; i++ {
		if x, y := mouse.Pos(); x < 5 && y < 5 {
			fmt.Println("мышь в углу — стоп")
			break
		}
		// Точки берём из средней части экрана, чтобы не упираться в края.
		tx := w/6 + rand.IntN(w*2/3)
		ty := h/6 + rand.IntN(h*2/3)
		x0, y0 := mouse.Pos()
		start := time.Now()
		mouse.MoveTo(tx, ty)
		took := time.Since(start)
		x, y := mouse.Pos()
		miss := math.Hypot(float64(x-tx), float64(y-ty))
		misses = append(misses, miss)
		fmt.Printf("%2d: (%4d,%4d) → (%4d,%4d)  путь %4.0f px  за %4dms  промах %.1f px\n",
			i, x0, y0, tx, ty, math.Hypot(float64(tx-x0), float64(ty-y0)), took.Milliseconds(), miss)
		time.Sleep(*pause)
	}

	if len(misses) > 0 {
		var sum, worst float64
		for _, m := range misses {
			sum += m
			worst = max(worst, m)
		}
		fmt.Printf("точек %d, средний промах %.1f px, худший %.1f px\n", len(misses), sum/float64(len(misses)), worst)
	}
}
