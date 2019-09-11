package main

import (
	"farm-robot/utils"
	"fmt"
	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
	"gocv.io/x/gocv"
	"image/png"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var fileName = "screen.png"
var templateDir = "./templates"
var confLevel = float32(0.75)

func main() {
	if len(os.Args) > 3 {
		confLevel = getConfLvl()
		fileName = os.Args[2]
		templateDir = os.Args[3]
	} else if len(os.Args) > 1 {
		confLevel = getConfLvl()
	}
	f, err := os.OpenFile("fisherlogs.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
	os.Mkdir("failed", os.FileMode(777))
	templates := loadTemplates(templateDir)

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
				//time.Sleep(10 * time.Minute)
			}
		case <-pipe:
			{
				log.Println("Bright Baubles")
				robotgo.KeyTap("h")
				robotgo.Sleep(3)
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
				findPipe(fileName, templates)
			}
		}

	}

}

func getConfLvl() float32 {
	value, err := strconv.ParseFloat(os.Args[1], 32)
	if err != nil {
		value = 0.75
	}
	return float32(value)
}

func loadTemplates(templateDir string) []gocv.Mat {
	var templates []gocv.Mat
	filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
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

func findPipe(fileName string, templates []gocv.Mat) {
	if err := createScreenFile(fileName); err != nil {
		return
	}
	if point, err := utils.Detect(fileName, templates, confLevel); err != nil {
		name := fmt.Sprintf("./failed/%d.png", rand.Intn(10)+1)
		log.Println(err, name)
		_ = os.Rename(fileName, name)
		return
	} else {
		robotgo.MoveMouseSmooth(point.X+20, point.Y+20, 1.0, 1.0)
		robotgo.Sleep(2)

		for start := time.Now(); time.Since(start) < 20*time.Second; {
			if fishIsBiting(fileName, templates) {
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

func fishIsBiting(fileName string, templates []gocv.Mat) bool {
	if err := createScreenFile(fileName); err != nil {
		return false
	}
	if _, err := utils.Detect(fileName, templates, confLevel); err != nil {
		return true
	}
	return false
}

func createScreenFile(fileName string) error {
	img, err := screenshot.Capture(0, 0, 300, 300)
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
