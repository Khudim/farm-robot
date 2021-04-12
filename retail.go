package main

import (
	"github.com/go-vgo/robotgo"
	"log"
	"time"
)

type Fisher interface {
	init()
	start()
}

type RetailFisher struct {
	pause           chan bool
	unPause         chan bool
	exit            chan bool
	isBaitTime      chan bool
	templateId      string
	isFilterEnabled bool
}

type RetailElements struct {
	confirm Element
	loot    Element
}

/*func InitElements() {
	var elements = RetailElements{}
	elements.confirm = NewElement("./templateId/confirm", "confirm.png", 1, 1)
	elements.loot = NewElement("./templateId/loot", "loot.png", 0.2, 1)
}*/

func newRetailFisher(templateId string) *RetailFisher {
	f := &RetailFisher{}

	f.templateId = templateId
	f.pause = make(chan bool)
	f.unPause = make(chan bool)
	f.exit = make(chan bool)
	f.isBaitTime = make(chan bool)
	f.isFilterEnabled = false
	return f
}

func (f RetailFisher) init() {

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

func (f *RetailFisher) start() {
	var errorCount = 0
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
				log.Println("Bobber time.")
				useBait()
			}
		case <-f.exit:
			{
				log.Println("Exit.")
				return
			}
		default:
			{
				useFishingRod()

				if isFishBiting(f.templateId) {
					if errorCount > 0 {
						errorCount--
					}
					loot()

				} else {
					if errorCount++; errorCount > 50 {
						go func() { f.exit <- true }()
					}
				}
				robotgo.MicroSleep(500)
			}
		}

	}

}
