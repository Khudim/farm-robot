package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"farm-robot/internal/mouse"
	"farm-robot/internal/vision"
	"farm-robot/internal/window"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	configPath := flag.String("config", "props.yaml", "путь к конфигу")
	traceFlag := flag.Bool("trace", false, "пошаговый разбор каждого цикла")
	tuneFlag := flag.Bool("tune-audio", false, "показать уровни звука для подбора порогов и выйти")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("fisher: ")

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("конфигурация: %v", err)
	}

	splash, err := NewSplashDetector(cfg.Splash)
	if err != nil {
		log.Fatalf("детектор всплеска: %v", err)
	}
	defer splash.Close()

	if *tuneFlag {
		tuneAudio(splash, cfg.Splash)
		return
	}
	defer window.Close()

	closeBoard, err := useBoard(cfg.Input)
	if err != nil {
		log.Fatalf("плата: %v", err)
	}
	defer closeBoard()

	useRecording(cfg.Human.Recording)
	mouse.LogScale()
	log.Printf("ввод: удержание %v, после модификаторов %v", cfg.Input.Hold.D(), cfg.Input.Settle.D())

	templates, skipped, err := vision.LoadTemplates(cfg.Search.Templates)
	if err != nil {
		log.Fatalf("эталоны: %v", err)
	}
	for _, err := range skipped {
		log.Printf("пропускаю: %v — удали файл", err)
	}
	log.Printf("эталонов: %d из %s", len(templates), cfg.Search.Templates)

	f := newFisher(cfg, splash, templates, newTracer(*traceFlag))
	go f.Start()
	go pollLevels(splash, f.Done())

	srv := &http.Server{Addr: ":" + strconv.Itoa(cfg.Port), Handler: promhttp.Handler()}
	go func() {
		log.Printf("метрики: http://localhost:%d/metrics", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-f.Done():
		log.Println("рыбак остановился")
	case s := <-sig:
		log.Printf("сигнал %s", s)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func pollLevels(d *SplashDetector, done <-chan struct{}) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			base, peak := d.Levels()
			splashBaseline.Set(base)
			splashPeak.Set(peak)
		case <-done:
			return
		}
	}
}
