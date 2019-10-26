package main

import (
	"farm-robot/config"
	"farm-robot/detector"
	"fmt"
	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
	"gocv.io/x/gocv"
	"image/png"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

func main() {
	mode := "default"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	appConfig := config.Parse(mode)
	f, err := os.OpenFile("fisherlogs.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	//log.SetOutput(f)
	os.Mkdir("failed", os.FileMode(777))
	templates := loadTemplates(appConfig)

	pause := make(chan bool)
	unPause := make(chan bool)
	exit := make(chan bool)
	go func() {
		for {
			pause <- robotgo.AddEvent("f2")
			unPause <- robotgo.AddEvent("f3")
			exit <- robotgo.AddEvent("f4")
		}
	}()

	afk, pipe := runBackgroundBehavior()

	for {
		select {
		case <-pause:
			{
				log.Println("Pause")
				<-unPause
				log.Println("Continue")
			}
		case <-afk:
			{
				log.Println("Afk")
				robotgo.KeyTap("w")
				robotgo.Sleep(3)
				robotgo.KeyTap("s")
				//time.Sleep(10 * time.Minute)
			}
		case <-pipe:
			{
				/*log.Println("Bright Baubles")
				robotgo.KeyTap("h")
				robotgo.Sleep(3)*/
			}
		case <-exit:
			{
				log.Println("Exit.")
				return
			}
		default:
			{
				robotgo.KeyTap("k", "control")
				robotgo.Sleep(3)
				findPipe(appConfig, templates)
			}
		}

	}

}

func loadTemplates(conf config.FisherConfig) []gocv.Mat {
	var templates []gocv.Mat
	_ = filepath.Walk(conf.TemplateDir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".png" {
			templates = append(templates, gocv.IMRead(path, gocv.IMReadGrayScale))
		}
		return err
	})

	return templates
}

func runBackgroundBehavior() (chan bool, chan bool) {

	afk := make(chan bool)
	pipe := make(chan bool)

	go func() {
		for {
			timeToWait := time.Duration(rand.Intn(100) + 20)
			time.Sleep(timeToWait * time.Second)
			afk <- true
		}
	}()
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			pipe <- true
		}
	}()

	return afk, pipe
}

func findPipe(config config.FisherConfig, templates []gocv.Mat) {
	fileName := config.FileName
	if err := createScreenFile(config); err != nil {
		return
	}
	if point, err := detector.Detect(fileName, templates, config.ConfLevel); err != nil {
		name := fmt.Sprintf("./failed/%d.png", rand.Intn(10)+1)
		log.Println(err, name)
		_ = os.Rename(fileName, name)
		return
	} else {
		robotgo.MoveMouseSmooth(point.X+20, point.Y+20, 1.0, 1.0)
		robotgo.Sleep(2)

		lastCheck := time.Now()
		checkEveryMS := 1000 / config.RefreshRate

		for start := time.Now(); time.Since(start) < 25*time.Second; {
			log.Println("Scan for bite")
			if fishIsBiting(config, templates) {
				log.Println("get the signal")
				loot()
				robotgo.MicroSleep(500)
				return
			}
			t := (time.Millisecond * time.Duration(checkEveryMS)) - time.Since(lastCheck)
			if t > 0 {
				robotgo.MicroSleep(float64(t / (1000 * 1000)))
			}
			lastCheck = time.Now()
		}
		for start := time.Now(); time.Since(start) < 25*time.Second; {
			if fishIsBiting(config, templates) {
				log.Println("get the signal")
				loot()
				robotgo.MicroSleep(500)
				return
			}
			log.Println("Scan for bite")
		}
		log.Println("lost the fish")
	}
}

func loot() {
	robotgo.KeyToggle("shift", "down")
	robotgo.MouseClick("right", true)
	robotgo.MicroSleep(500)
	robotgo.KeyToggle("shift", "up")
}

func fishIsBiting(config config.FisherConfig, templates []gocv.Mat) bool {
	if err := createScreenFile(config); err != nil {
		return false
	}
	if _, err := detector.Detect(config.FileName, templates, config.ConfLevel); err != nil {
		return true
	}
	return false
}

func createScreenFile(conf config.FisherConfig) error {
	screen := screenshot.GetDisplayBounds(0)
	img, err := screenshot.Capture(0, 0,
		int(float64(screen.Max.X)*conf.ScreenshotsSize),
		int(float64(screen.Max.Y)*conf.ScreenshotsSize))
	if err != nil {
		return err
	}
	file, err := os.Create(conf.FileName)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}
