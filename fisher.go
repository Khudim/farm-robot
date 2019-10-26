package main

import (
	"farm-robot/utils"
	"fmt"
	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
	"github.com/olebedev/config"
	"gocv.io/x/gocv"
	"image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type AppConfig struct {
	fileName       string
	templateDir    string
	confLevel      float32
	failTolerance  int
	refreshRate    int
	screenShotSize float64
}

func main() {
	mode := "default"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	appConfig := parseConfig(mode)
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

func parseConfig(mode string) AppConfig {
	appConfig := AppConfig{fileName: "screen.png", templateDir: "./templates", confLevel: 0.75, refreshRate: 4, screenShotSize: 0.5}

	if file, err := ioutil.ReadFile("./fisher.properties"); err == nil {
		if cfg, er := config.ParseYaml(string(file)); er == nil {
			if v, e := cfg.Float64("detector.confLevel"); e == nil {
				appConfig.confLevel = float32(v)
			}
			if v, e := cfg.String("templates." + mode); e == nil {
				appConfig.templateDir = v
			}
		}
	} else {
		panic(err)
	}
	return appConfig
}

func getConfLvl() float32 {
	value, err := strconv.ParseFloat(os.Args[1], 32)
	if err != nil {
		value = 0.75
	}
	return float32(value)
}

func loadTemplates(config AppConfig) []gocv.Mat {
	var templates []gocv.Mat
	_ = filepath.Walk(config.templateDir, func(path string, info os.FileInfo, err error) error {
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

func findPipe(config *AppConfig, templates []gocv.Mat) {
	fileName := config.fileName
	if err := createScreenFile(config); err != nil {
		return
	}
	if point, err := utils.Detect(fileName, templates, config.confLevel); err != nil {
		name := fmt.Sprintf("./failed/%d.png", rand.Intn(10)+1)
		log.Println(err, name)
		_ = os.Rename(fileName, name)
		return
	} else {
		robotgo.MoveMouseSmooth(point.X+20, point.Y+20, 1.0, 1.0)
		robotgo.Sleep(2)

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

func fishIsBiting(config *AppConfig, templates []gocv.Mat) bool {
	if err := createScreenFile(config); err != nil {
		return false
	}
	if _, err := utils.Detect(config.fileName, templates, config.confLevel); err != nil {
		return true
	}
	return false
}

func createScreenFile(appConfig *AppConfig) error {
	screen := screenshot.GetDisplayBounds(0)
	img, err := screenshot.Capture(0, 0,
		int(float64(screen.Max.X)*appConfig.screenShotSize),
		int(float64(screen.Max.Y)*appConfig.screenShotSize))
	if err != nil {
		return err
	}
	file, err := os.Create(appConfig.fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}
