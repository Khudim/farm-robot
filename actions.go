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
	x          int
	y          int
}

type Template struct {
	Id string `json:"templateId"`
}

type point struct {
	Confidence float32
	X          int
	Y          int
}

func drop(confirmEl Element) {
	screen := screenshot.GetDisplayBounds(0)
	robotgo.MouseClick("left")
	robotgo.MoveMouseSmooth(screen.Max.X/2, screen.Max.Y/2)
	robotgo.MouseClick("left")
	robotgo.MicroSleep(1000)
	//find(confirmEl)
	robotgo.MouseClick("left")
}

func find(el Element) bool {
	img := makeScreenshot(el.x, el.y)
	if point := detect(img, el.templateId, 0.70); point != nil {
		robotgo.MoveMouseSmooth(point.X+20, point.Y+20, 1.0, 1.0)
		return true
	}
	return false
}

func useBait() {
	robotgo.KeyTap("r", "control")
	robotgo.MicroSleep(500)
}

func useClassicBait() {
	robotgo.KeyTap("r", "control")
	robotgo.MicroSleep(500)
	robotgo.KeyTap("0")
	robotgo.KeyTap("space")
	if find(baubles) {
		robotgo.MouseClick("right")
		robotgo.MicroSleep(500)
		if find(pole) {
			robotgo.MouseClick("left")
			robotgo.MicroSleep(7500)
		}
	}
	robotgo.KeyTap("0")
}

func useFishingRod() {
	robotgo.KeyTap("k", "control")
	robotgo.Sleep(3)
}

func findFloat(templateId string) *Element {
	image := makeScreenshot(0, 0, screen.Max.X-200, screen.Max.Y-200)
	point := detect(image, templateId, appConfig.ConfLevel)
	if point == nil {
		log.Fatal("Can't find point")
		return nil
	}
	robotgo.MoveMouseSmooth(point.X+20, point.Y+20, 1.0, 1.0)
	return &Element{templateId, point.X, point.Y}
}

func catch(float *Element) bool {
	lastCheck := time.Now()
	interval := time.Second / time.Duration(appConfig.RefreshRate)

	for start := time.Now(); time.Since(start) < 25*time.Second; {
		image := makeScreenshot(float.x, float.y, 100, 100)
		p := detect(image, float.templateId, appConfig.ConfLevel)
		if p == nil {
			log.Println("Fish bite")
			return true
		}
		if time.Since(lastCheck) < interval {
			time.Sleep(interval - time.Since(lastCheck))
		}
		lastCheck = time.Now()
	}
	log.Println("Lost fish")
	return false
}

func loot() {
	robotgo.KeyToggle("shift", "down")
	robotgo.MouseClick("right")
	robotgo.MicroSleep(500)
	robotgo.KeyToggle("shift", "up")
}

func lootWithFilter() {
	robotgo.MouseClick("right")
	robotgo.MicroSleep(1000)
	for i := 0; i < 3; i++ {
		if find(lootEl) {
			robotgo.MouseClick("right")
			robotgo.MicroSleep(200)
		}
	}
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

func makeScreenshot(x, y, width, height int) []byte {
	img, _ := screenshot.Capture(x, y, width, height)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return []byte{}
	}
	return buf.Bytes()
}
