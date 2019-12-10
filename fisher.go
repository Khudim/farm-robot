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
	templates := loadTemplates(appConfig.TemplateDir)
	clamsTemplates := loadTemplates(appConfig.TemplateClamsDir)

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

	clams := runBackgroundBehavior()

	for {
		select {
		case <-pause:
			{
				log.Println("Pause")
				<-unPause
				log.Println("Continue")
			}
		case <-clams:
			{
				log.Println("Clams time.")
				fileName := "clam.png"
				c := 0
				ok := true
				for c < 10 && ok {
					ok = findClam(fileName, clamsTemplates)
					c++
				}
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

func findClam(fileName string, clamsTemplates []gocv.Mat) bool {
	if err := createScreenFile(1, 1, fileName); err == nil {
		if point, err := detector.Detect(fileName, clamsTemplates, 0.80); err == nil && point != nil {
			robotgo.MoveMouseSmooth(point.X+20, point.Y+20, 1.0, 1.0)
			loot()
			return true
		}
	}
	return false
}

func loadTemplates(templateDir string) []gocv.Mat {
	var templates []gocv.Mat
	_ = filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".png" {
			templates = append(templates, gocv.IMRead(path, gocv.IMReadGrayScale))
		}
		return err
	})

	return templates
}
func runBackgroundBehavior() chan bool {

	clams := make(chan bool)

	go func() {
		for {
			time.Sleep(30 * time.Minute)
			clams <- true
		}
	}()

	return clams
}

var errorCount = 0

func findPipe(config config.FisherConfig, templates []gocv.Mat) {
	fileName := config.FileName
	sizeMod := config.ScreenshotsSize
	if err := createScreenFile(sizeMod, sizeMod, fileName); err != nil {
		return
	}
	if point, err := detector.Detect(fileName, templates, config.ConfLevel); err != nil {
		name := fmt.Sprintf("./failed/%d.png", rand.Intn(10)+1)
		log.Println(err, name)
		_ = os.Rename(fileName, name)
		errorCount++
		if errorCount > 100 {
			os.Exit(0)
		}
		return
	} else {
		errorCount = 0
		robotgo.MoveMouseSmooth(point.X+20, point.Y+20, 1.0, 1.0)
		robotgo.Sleep(2)

		lastCheck := time.Now()
		checkEveryMS := 1000 / config.RefreshRate

		for start := time.Now(); time.Since(start) < 25*time.Second; {
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
	if err := createScreenFile(config.ScreenshotsSize, config.ScreenshotsSize, config.FileName); err != nil {
		return false
	}
	if _, err := detector.Detect(config.FileName, templates, config.ConfLevel); err != nil {
		return true
	}
	return false
}

func createScreenFile(x float64, y float64, fileName string) error {
	screen := screenshot.GetDisplayBounds(0)
	img, err := screenshot.Capture(0, 0,
		int(float64(screen.Max.X)*x),
		int(float64(screen.Max.Y)*y))
	if err != nil {
		return err
	}
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}
