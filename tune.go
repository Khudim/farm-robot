package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// tuneAudio показывает уровни вживую, чтобы выставить пороги по факту.
func tuneAudio(d *SplashDetector, cfg SplashConfig) {
	log.Printf("сейчас threshold=%.1f, minLevel=%.4f; забрасывай и смотри на всплеск, Ctrl+C — выход",
		cfg.Threshold, cfg.MinLevel)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()

	var loudest, loudestRatio float64
	for {
		select {
		case <-t.C:
			base, peak := d.Levels()
			var ratio float64
			if base > 0 {
				ratio = peak / base
			}
			if ratio > loudestRatio {
				loudest, loudestRatio = peak, ratio
			}
			mark := "  "
			if peak > cfg.MinLevel && ratio >= cfg.Threshold {
				mark = "<<"
			}
			fmt.Printf("\rфон %7.1f дБ   пик %7.1f дБ (%.4f)   x%5.1f %s   рекорд x%.1f   ",
				dbfs(base), dbfs(peak), peak, ratio, mark, loudestRatio)

		case <-sig:
			fmt.Println()
			switch {
			case loudest == 0:
				log.Printf("звука не было — устройство захвата отдаёт тишину")
			case loudest < 0.005:
				log.Printf("самое громкое %.1f дБ — почти тишина, прибавь громкость игры и системы", dbfs(loudest))
			default:
				log.Printf("самое громкое %.1f дБ при x%.1f к фону", dbfs(loudest), loudestRatio)
				log.Printf("рекомендую splash.threshold: %.1f, splash.minLevel: %.4f",
					max(loudestRatio*0.6, 1.5), loudest/3)
			}
			return
		}
	}
}

func dbfs(rms float64) float64 {
	if rms <= 0 {
		return -120
	}
	return 20 * math.Log10(rms)
}
