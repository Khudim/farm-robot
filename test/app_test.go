package test

import (
	"encoding/json"
	"farm-robot/detector"
	"farm-robot/utils"
	"github.com/go-vgo/robotgo"
	"gocv.io/x/gocv"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRate(t *testing.T) {
	var limit = 5
	lastCheck := time.Now()
	interval := time.Second / time.Duration(limit)
	for start := time.Now(); time.Since(start) < 25*time.Second; {
		log.Println("Scan for bite")
		if time.Since(lastCheck) < interval {
			time.Sleep(interval - time.Since(lastCheck))
		}
		lastCheck = time.Now()
	}
}

func TestShouldFindClams(t *testing.T) {
	var templates []gocv.Mat
	_ = filepath.Walk("./clams", func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".png" {
			templates = append(templates, gocv.IMRead(path, gocv.IMReadGrayScale))
		}
		return err
	})

	log.Println(len(templates))

	if point, err := detector.Detect("test-clam.png", templates, 0.70); err == nil && point != nil {
		log.Println(point)
	} else {
		log.Println(point)
	}
}

type Event struct {
	Kind    uint8 `json:"id"`
	When    time.Time
	Rawcode uint16 `json:"rawcode"`
}

func TestShouldRecord(t *testing.T) {

	EvChan := robotgo.Start()
	var events []Event
	var lastToggle uint16
	for ev := range EvChan {
		if ev.Kind != 9 && ev.Kind != 4 {
			if ev.Kind == 3 {
				if lastToggle == ev.Rawcode {
					continue
				} else {
					lastToggle = ev.Rawcode
				}
			}
			events = append(events, Event{
				Kind:    ev.Kind,
				Rawcode: ev.Rawcode,
				When:    ev.When,
			})
			if ev.Rawcode == 27 {
				robotgo.End()
				break
			}
		}
	}

	file, _ := json.MarshalIndent(events, "", " ")
	_ = ioutil.WriteFile("test.json", file, 0644)
}

func TestShouldPlay(t *testing.T) {
	data, _ := ioutil.ReadFile("test.json")
	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return
	}

	var lastTime time.Time
	for i, v := range events {
		if i != 0 {
			time.Sleep(v.When.Sub(lastTime))
		}
		lastTime = v.When
		switch v.Kind {
		case 3:
			robotgo.KeyToggle(utils.Raw2key[v.Rawcode], "down")
		case 5:
			robotgo.KeyToggle(utils.Raw2key[v.Rawcode], "up")
		default:
			robotgo.KeyTap(utils.Raw2key[v.Rawcode])
		}
	}
}
