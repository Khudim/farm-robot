package main

/*
import (
	"github.com/go-vgo/robotgo"
	"log"
	"os"
	"time"
)

type ClassicFisher struct {
	pause chan bool
	unPause chan bool
	isClamsTime chan bool
	isBaublesTime chan bool
	exit chan bool
}

func (f ClassicFisher) init()  {
	if f, err := os.OpenFile("fisherlogs.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
		log.Fatalf("error opening file: %v", err)
		return
	} else {
		defer f.Close()
	}
	_ = os.Mkdir("failed", os.FileMode(777))
	// templates := actions.LoadTemplates(c.TemplateDir)

	clams := NewElement("./templates/clams", "clams.png", 1, 1)
	meat := NewElement("./templates/meat", "meat.png", 1, 1)
	confirm := NewElement("./templates/confirm", "confirm.png", 1, 1)
	lootEl := NewElement("./templates/loot", "loot.png", 0.2, 1)

	f.pause = make(chan bool)
	f.unPause = make(chan bool)
	f.exit = make(chan bool)
	go func() {
		for {
			f.pause <- robotgo.AddEvent("f2")
			f.unPause <- robotgo.AddEvent("f3")
			f.exit <- robotgo.AddEvent("f4")
		}
	}()

	f.isClamsTime, f.isBaublesTime = runBackgroundBehavior()
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


func (f *ClassicFisher) start() {

	for {
		select {
		case <-f.pause:
			{
				log.Println("Pause")
				<-f.unPause
				log.Println("Continue")
			}
		case <-f.isClamsTime:
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
		case <-f.isBaublesTime:
			{
				log.Println("Baubles time.")
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
		case <-f.exit:
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

	}*/
