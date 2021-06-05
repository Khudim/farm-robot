package main

import (
	"github.com/go-vgo/robotgo"
	"log"
	"time"
)

type Fisher struct {
	pause      chan bool
	unPause    chan bool
	exit       chan bool
	isBaitTime chan bool
	errorCount int
	elements   map[string]*Element
}

func newFisher() *Fisher {
	f := &Fisher{}
	f.elements = make(map[string]*Element)
	f.pause = make(chan bool)
	f.unPause = make(chan bool)
	f.exit = make(chan bool)
	f.isBaitTime = make(chan bool)
	return f
}

func (f Fisher) init() {

	go func() {
		for {
			f.pause <- robotgo.AddEvent("f2")
			f.unPause <- robotgo.AddEvent("f3")
			f.exit <- robotgo.AddEvent("f4")
		}
	}()

	go func() {
		for {
			time.Sleep(625 * time.Second)
			f.isBaitTime <- true
		}
	}()
}

func (f *Fisher) start() {
	f.init()
	for {
		select {
		case <-f.pause:
			{
				log.Println("Pause")
				<-f.unPause
				log.Println("Continue")
			}
		case <-f.isBaitTime:
			{
				log.Println("Bait time.")
				pole := f.elements["pole"]
				if pole != nil {
					useClassicBait(pole)
				} else {
					useBait()
				}
			}
		case <-f.exit:
			{
				log.Println("Exit.")
				return
			}
		default:
			{
				robotgo.MicroSleep(500)

				useFishingRod()

				float := f.elements["float"]
				point := findFloat(float)
				if point == nil {
					continue
				}
				el := &Element{
					float.templateId,
					float.conf,
					float.x + point.X - 25,
					float.y + point.Y - 25,
					150,
					150,
					"catch",
					float.isDebug,
				}

				if isCaught(f, el) {
					loot(f.elements["loot"])
				}
			}
		}

	}

}

func isCaught(f *Fisher, float *Element) bool {
	if catch(float) {
		if f.errorCount > 0 {
			f.errorCount--
		}
		return true
	} else {
		if f.errorCount++; f.errorCount > 50 {
			go func() { f.exit <- true }()
		}
		return false
	}
}
