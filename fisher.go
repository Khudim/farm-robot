package main

import (
	"github.com/go-vgo/robotgo"
	"log"
	"time"
)

type RetailFisher struct {
	pause           chan bool
	unPause         chan bool
	exit            chan bool
	isBaitTime      chan bool
	templateId      string
	isFilterEnabled bool
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
	f.init()
	f.run()
}

func (f *RetailFisher) run() {
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
				log.Println("Bait time.")
				//useBait()
				useClassicBait()
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

				float := findFloat(f.templateId)
				if float == nil {
					continue
				}
				robotgo.Sleep(2)

				if catch(float) {
					if appConfig.AllowLootFilter {
						lootWithFilter()
					} else {
						loot()
					}
					if errorCount > 0 {
						errorCount--
					}
				} else {
					if errorCount++; errorCount > 50 {
						go func() { f.exit <- true }()
					}
				}
			}
		}

	}

}
