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

type element struct {
	screenFile string
	templates  []gocv.Mat
	x          float64
	y          float64
}

func newElement(templatesDir, screenShot string, x, y float64) element {
	templates := loadTemplates(templatesDir)
	return element{screenShot, templates, x, y}
}

func main() {
	mode := "default"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	appConfig := config.Parse(mode)

	if f, err := os.OpenFile("fisherlogs.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
		log.Fatalf("error opening file: %v", err)
	} else {
		defer f.Close()
		//log.SetOutput(f)
	}
	_ = os.Mkdir("failed", os.FileMode(777))
	templates := loadTemplates(appConfig.TemplateDir)

	clams := newElement("./templates/clams", "clams.png", 1, 1)
	meat := newElement("./templates/meat", "meat.png", 1, 1)
	confirm := newElement("./templates/confirm", "confirm.png", 1, 1)
	lootEl := newElement("./templates/loot", "loot.png", 0.2, 1)

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

	isClamsTime, isBaublesTime := runBackgroundBehavior()

	var errorCount = 0

	log.Printf("%+v\n", appConfig)
	for {
		select {
		case <-pause:
			{
				log.Println("Pause")
				<-unPause
				log.Println("Continue")
			}
		case <-isClamsTime:
			{
				log.Println("Clams time.")

				for i := 0; i < 15 && find(clams); {
					loot()
					i++
				}
				if find(meat) {
					drop(confirm)
				}
			}
		case <-isBaublesTime:
			{
				log.Println("Baubles time.")
				robotgo.KeyTap("r", "control")
				robotgo.MicroSleep(500)
				/*				robotgo.KeyTap("0")
								robotgo.KeyTap("space")
								if find(baubles) {
									robotgo.MouseClick("right")
									robotgo.MicroSleep(500)
									if find(pole) {
										robotgo.MouseClick("left")
										robotgo.MicroSleep(7500)
									}
								}
								robotgo.KeyTap("0")*/
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
				if isFishBiting(appConfig, templates) {
					if errorCount > 0 {
						errorCount--
					}
					if appConfig.AllowLootFilter {
						robotgo.MouseClick("right")
						robotgo.MicroSleep(1000)
						for i := 0; i < 3; i++ {
							if find(lootEl) {
								robotgo.MouseClick("right")
								robotgo.MicroSleep(200)
							}
						}
					} else {
						loot()
					}
				} else {
					if errorCount++; errorCount > 50 {
						go func() { exit <- true }()
					}
				}
				robotgo.MicroSleep(500)
			}
		}

	}

}

func drop(confirmEl element) {
	screen := screenshot.GetDisplayBounds(0)
	robotgo.MouseClick("left")
	robotgo.MoveMouseSmooth(screen.Max.X/2, screen.Max.Y/2)
	robotgo.MouseClick("left")
	robotgo.MicroSleep(1000)
	find(confirmEl)
	robotgo.MouseClick("left")
}

func find(el element) bool {
	if err := createScreenFile(el.x, el.y, el.screenFile); err == nil {
		if point, err := detector.Detect(el.screenFile, el.templates, 0.70); err == nil && point != nil {
			robotgo.MoveMouseSmooth(point.X+20, point.Y+20, 1.0, 1.0)
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
func runBackgroundBehavior() (chan bool, chan bool) {

	clams := make(chan bool)
	pipe := make(chan bool)

	go func() {
		for {
			time.Sleep(11 * time.Minute)
			// clams <- true
		}
	}()
	go func() {
		for {
			time.Sleep(30 * time.Minute)
			pipe <- true
		}
	}()
	return clams, pipe
}

func isFishBiting(config config.FisherConfig, templates []gocv.Mat) bool {
	fileName := config.FileName
	sizeMod := config.ScreenshotsSize
	if err := createScreenFile(sizeMod, sizeMod, fileName); err != nil {
		return false
	}
	if point, err := detector.Detect(fileName, templates, config.ConfLevel); err != nil {
		name := fmt.Sprintf("./failed/%d.png", rand.Intn(10)+1)
		log.Println(err, name)
		_ = os.Rename(fileName, name)
		return false
	} else {
		robotgo.MoveMouseSmooth(point.X+20, point.Y+20, 1.0, 1.0)
		robotgo.Sleep(2)

		lastCheck := time.Now()
		interval := time.Second / time.Duration(config.RefreshRate)

		for start := time.Now(); time.Since(start) < 25*time.Second; {
			if fishIsBiting(config, templates) {
				log.Println("get the signal")
				return true
			}
			if time.Since(lastCheck) < interval {
				time.Sleep(interval - time.Since(lastCheck))
			}
			lastCheck = time.Now()
		}
		log.Println("lost the fish")
		return false
	}
}

func loot() {
	robotgo.KeyToggle("shift", "down")
	robotgo.MouseClick("right")
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

func createScreenFile(x, y float64, fileName string) error {
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
