package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
	"github.com/valyala/fasthttp"
	"image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type Element struct {
	screenFile string
	templateId string
	x          float64
	y          float64
}

type point struct {
	confidence float32
	x          int
	y          int
}

/*func NewElement(templatesDir, screenShot string, x, y float64) Element {
	templates := readTemplates(templatesDir)
	return Element{screenShot, templates, x, y}
}*/

func readTemplates(templateDir string) []byte {
	var templates []byte
	_ = filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".png" {
			if image, e := ioutil.ReadFile(path); e != nil {
				templates = append(templates, image...)
			} else {
				fmt.Println(e)
			}
		}
		return err
	})

	return templates
}

func drop(confirmEl Element) {
	screen := screenshot.GetDisplayBounds(0)
	robotgo.MouseClick("left")
	robotgo.MoveMouseSmooth(screen.Max.X/2, screen.Max.Y/2)
	robotgo.MouseClick("left")
	robotgo.MicroSleep(1000)
	find(confirmEl)
	robotgo.MouseClick("left")
}

func find(el Element) bool {
	if err := createScreenshot(el.x, el.y, el.screenFile); err == nil {
		if point := detect(el.screenFile, el.templateId, 0.70); point != nil {
			robotgo.MoveMouseSmooth(point.x+20, point.y+20, 1.0, 1.0)
			return true
		}
	}
	return false
}

func useBait() {
	robotgo.KeyTap("r", "control")
	robotgo.MicroSleep(500)
}

func useFishingRod() {
	robotgo.KeyTap("k", "control")
	robotgo.Sleep(3)
}

func isFishBiting(templateId string) bool {
	fileName := appConfig.FileName
	sizeMod := appConfig.ScreenshotsSize

	if err := createScreenshot(sizeMod, sizeMod, fileName); err != nil {
		return false
	}

	if point := detect(fileName, templateId, appConfig.ConfLevel); point == nil {
		name := fmt.Sprintf("./failed/%d.png", rand.Intn(10)+1)
		_ = os.Rename(fileName, name)
		return false
	} else {
		robotgo.MoveMouseSmooth(point.x+20, point.y+20, 1.0, 1.0)
		robotgo.Sleep(2)

		lastCheck := time.Now()
		interval := time.Second / time.Duration(appConfig.RefreshRate)

		for start := time.Now(); time.Since(start) < 25*time.Second; {
			if fishIsBiting(templateId) {
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

/*func lootWithFilter() {
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
}*/

func fishIsBiting(templateId string) bool {
	if err := createScreenshot(appConfig.ScreenshotsSize, appConfig.ScreenshotsSize, appConfig.FileName); err != nil {
		return false
	}

	p := detect(appConfig.FileName, templateId, appConfig.ConfLevel)
	return p != nil
}

func detect(fileName, templateId string, confLevel float32) *point {
	image, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	status, body, err := fasthttp.Post(image, appConfig.TemplateMatcherUrl+"/template/detect"+templateId, nil)
	if status != 200 {
		log.Fatal(err)
		return nil
	}

	var response point
	if err := json.Unmarshal(body, &response); err == nil {
		log.Printf("%+v\n", response)
		if response.confidence >= confLevel {
			return &response
		}
	}

	return nil
}

func createScreenshot(x, y float64, fileName string) error {
	screen := screenshot.GetDisplayBounds(0)
	img, err := screenshot.Capture(
		0,
		0,
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
