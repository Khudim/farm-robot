package main

import (
	"bytes"
	"encoding/json"
	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
	"github.com/valyala/fasthttp"
	"image/png"
	"log"
	"time"
)

type Element struct {
	templateId string
	x          float64
	y          float64
}

type point struct {
	Confidence float32
	X          int
	Y          int
}

/*func NewElement(templatesDir, screenShot string, X, Y float64) Element {
	templates := readTemplates(templatesDir)
	return Element{screenShot, templates, X, Y}
}*/

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
	image := createScreenshot(el.x, el.y)
	if point := detect(image, el.templateId, 0.70); point != nil {
		robotgo.MoveMouseSmooth(point.X+20, point.Y+20, 1.0, 1.0)
		return true
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
	sizeMod := appConfig.ScreenshotsSize

	image := createScreenshot(sizeMod, sizeMod)

	if point := detect(image, templateId, appConfig.ConfLevel); point == nil {
		log.Fatal("Can't find point")
		return false
	} else {
		robotgo.MoveMouseSmooth(point.X+20, point.Y+20, 1.0, 1.0)
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
	image := createScreenshot(appConfig.ScreenshotsSize, appConfig.ScreenshotsSize)
	p := detect(image, templateId, appConfig.ConfLevel)
	return p != nil
}

func detect(image []byte, templateId string, acceptableConfidence float32) *point {

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	url := appConfig.TemplateMatcherUrl + "/template/detect/" + templateId
	req.SetRequestURI(url)
	req.Header.SetMethodBytes([]byte("POST"))
	req.AppendBody(image)

	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	if err := fasthttp.Do(req, res); err != nil {
		log.Fatal(err)
		return nil
	}

	var response point
	if err := json.Unmarshal(res.Body(), &response); err == nil {
		log.Printf("%+v\n", response)
		if response.Confidence >= acceptableConfidence {
			return &response
		}
	} else {
		log.Fatal(err)
	}

	return nil
}

func createScreenshot(x, y float64) []byte {
	screen := screenshot.GetDisplayBounds(0)
	img, err := screenshot.Capture(0, 0, int(float64(screen.Max.X)*x), int(float64(screen.Max.Y)*y))
	if err != nil {
		log.Fatal(err)
		return nil
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
