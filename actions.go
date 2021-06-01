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
	conf       float32
	x          int
	y          int
	width      int
	height     int
}

type point struct {
	Confidence float32
	X          int
	Y          int
}

func find(el *Element) bool {
	img := makeScreenshot(el.x, el.y, el.width, el.height)
	if point := detect(img, el); point != nil {
		robotgo.MoveMouseSmooth(point.X+20, point.Y+20, 1.0, 1.0)
		return true
	}
	return false
}

func useBait() {
	robotgo.KeyTap("r", "control")
	robotgo.MicroSleep(500)
}

func useClassicBait(bait, pole *Element) {
	robotgo.KeyTap("r", "control")
	robotgo.MicroSleep(500)
	robotgo.KeyTap("0")
	robotgo.KeyTap("space")
	if find(bait) {
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

func findFloat(element *Element) *point {
	image := makeScreenshot(0, 0, screen.Max.X-200, screen.Max.Y-200)
	point := detect(image, element)
	if point != nil {
		robotgo.MoveMouseSmooth(point.X+20, point.Y+20, 0.9, 0.9)
	}
	return point
}

func catch(float *Element) bool {
	lastCheck := time.Now()
	interval := time.Second / time.Duration(4)

	for start := time.Now(); time.Since(start) < 25*time.Second; {
		image := makeScreenshot(float.x, float.y, 100, 100)

		p := detect(image, float)
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

func loot(lootEl *Element) {
	if lootEl == nil {
		lootAll()
	} else {
		lootWithFilter(lootEl)
	}
}

func lootAll() {
	robotgo.KeyToggle("shift", "down")
	robotgo.MicroSleep(500)
	robotgo.MouseClick("right")
	robotgo.KeyToggle("shift", "up")
	robotgo.MicroSleep(1000)
}

func lootWithFilter(lootEl *Element) {
	robotgo.MouseClick("right")
	robotgo.MicroSleep(1000)
	for i := 0; i < 3; i++ {
		if find(lootEl) {
			robotgo.MouseClick("right")
			robotgo.MicroSleep(200)
		}
	}
}

func detect(image []byte, element *Element) *point {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	url := matcherUrl + "/template/detect/" + element.templateId
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
		if response.Confidence >= element.conf {
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
