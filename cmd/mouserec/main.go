//go:build windows

// mouserec записывает, как вы двигаете мышью, чтобы бот повторял ваши
// движения, а не синтетические кривые. Плата не нужна.
//
//	go run ./cmd/mouserec -d 3m
//
// Двигайте мышью как обычно в игре: переходы между точками разной длины и в
// разные стороны, покачивания, круги, восьмёрки, мелкие сдвиги — с паузами
// между ними. Пауза от 0,3 с отделяет одно движение от другого. Остановить —
// по времени, Ctrl+C или F4. Запись дописывается в файл, если он уже есть.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"farm-robot/internal/mouse"

	"github.com/go-vgo/robotgo"
)

var procGetAsyncKeyState = syscall.NewLazyDLL("user32.dll").NewProc("GetAsyncKeyState")

func main() {
	out := flag.String("o", "mouse.json", "файл записи")
	dur := flag.Duration("d", 3*time.Minute, "сколько писать")
	still := flag.Duration("still", 300*time.Millisecond, "пауза, которая разделяет движения")
	flag.Parse()

	rec, err := mouse.LoadRecording(*out)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("запись: %v", err)
	}
	had := len(rec.Gestures)

	seg := &mouse.Segmenter{Still: *still, MinPath: 8}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	fmt.Printf("пишу %v в %s (уже есть %d движений); стоп — F4 или Ctrl+C\n", *dur, *out, had)
	fmt.Println("двигайте мышью как в игре: переходы разной длины, покачивания, круги, с паузами")
	end := time.Now().Add(*dur)
	tick := time.NewTicker(4 * time.Millisecond)
	defer tick.Stop()
	lastReport := time.Now()
loop:
	for time.Now().Before(end) {
		select {
		case <-sig:
			break loop
		case now := <-tick.C:
			if state, _, _ := procGetAsyncKeyState.Call(0x73); state&0x8000 != 0 { // F4
				break loop
			}
			x, y := robotgo.Location()
			seg.Add(now, x, y)
			if time.Since(lastReport) > 10*time.Second {
				fmt.Printf("  движений: %d, осталось %v\n", len(seg.Gestures()), time.Until(end).Round(time.Second))
				lastReport = time.Now()
			}
		}
	}
	seg.Flush()

	got := seg.Gestures()
	rec.Gestures = append(rec.Gestures, got...)
	if err := mouse.SaveRecording(*out, rec); err != nil {
		log.Fatalf("сохранение: %v", err)
	}
	var short, long int
	for _, s := range got {
		if x, y := s.Net(); x*x+y*y >= 100*100 {
			long++
		} else {
			short++
		}
	}
	fmt.Printf("записано %d движений (%d дальних от 100 px, %d коротких), всего в файле %d\n",
		len(got), long, short, len(rec.Gestures))
}
