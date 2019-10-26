package test

import (
	"github.com/go-vgo/robotgo"
	"log"
	"testing"
	"time"
)

func TestRate(t *testing.T) {
	limit := 5
	lastCheck := time.Now()
	checkEveryMS := 1000 / limit
	for start := time.Now(); time.Since(start) < 25*time.Second; {
		log.Println("Scan for bite")
		t := (time.Millisecond * time.Duration(checkEveryMS)) - time.Since(lastCheck)
		if t > 0 {
			robotgo.MicroSleep(float64(t / (1000 * 1000)))
		}
		lastCheck = time.Now()
	}
}
