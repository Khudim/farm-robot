package test

import (
	"farm-robot/detector"
	"github.com/go-vgo/robotgo"
	"gocv.io/x/gocv"
	"log"
	"os"
	"path/filepath"
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

func TestShouldFindClams(t *testing.T) {
	var templates []gocv.Mat
	_ = filepath.Walk("C:/Users/Beaver/go/src/farm-robot/clams", func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".png" {
			templates = append(templates, gocv.IMRead(path, gocv.IMReadGrayScale))
		}
		return err
	})

	log.Println(len(templates))

	if point, err := detector.Detect("C:\\Users\\Beaver\\go\\src\\farm-robot\\test-clam.png", templates, 0.70); err == nil && point != nil {
		log.Println(point)
	} else {
		log.Println(point)
	}
}
