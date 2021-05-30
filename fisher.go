package main

import (
	"github.com/go-vgo/robotgo"
	"log"
	"time"
)

type Fisher struct {
	pause           chan bool
	unPause         chan bool
	exit            chan bool
	isBaitTime      chan bool
	isFilterEnabled bool
	floatEl         *Element
	biteEl          *Element
	poleEl          *Element
	lootEl          *Element
	errorCount      int
}

func newRetailFisher() *Fisher {
	f := &Fisher{}

	f.pause = make(chan bool)
	f.unPause = make(chan bool)
	f.exit = make(chan bool)
	f.isBaitTime = make(chan bool)
	f.isFilterEnabled = false
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
			time.Sleep(30 * time.Minute)
			f.isBaitTime <- true
		}
	}()
}

func (f *Fisher) start() {
	f.init()
	f.run()
}

func (f *Fisher) run() {
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
				if f.biteEl != nil && f.poleEl != nil {
					useClassicBait(f.biteEl, f.poleEl)
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

				point := findFloat(f.floatEl)
				if point == nil {
					continue
				}
				robotgo.Sleep(2)

				if isCaught(f) {
					loot(f.lootEl)
				}
			}
		}

	}

}

func isCaught(f *Fisher) bool {
	if catch(f.floatEl) {
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
