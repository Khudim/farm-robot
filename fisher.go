package main

import (
	"fmt"
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
	searchGrid *Grid
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

				var point *point

				float := f.elements["float"]

				if float == nil {
					floatText := f.elements["floatText"]
					grid := f.searchGrid
					if floatText == nil {
						fmt.Println("Can't find anything")
						return
					}
					point = searchForFloat(floatText, grid)
					if point == nil {
						continue
					}
					robotgo.MoveMouseSmooth(point.X, point.Y+33, 0.9, 0.9)
					robotgo.Sleep(5)
					if f.isCaught(floatText, false) {
						robotgo.MoveMouseSmooth(point.X, point.Y, 0.9, 0.9)
						loot(f.elements["loot"])
					}
				} else {
					point = findFloat(float)
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

					if f.isCaught(el, true) {
						loot(f.elements["loot"])
					}
				}
			}
		}

	}

}

func searchForFloat(floatText *Element, grid *Grid) *point {
	stepX := grid.Width / 10
	stepY := grid.Height / 10
	positionX := grid.X
	positionY := grid.Y

	for i := 0; i < 100; i++ {
		robotgo.MoveMouseSmooth(positionX, positionY, 0.9, 0.9)
		p := find(floatText)
		if p != nil {
			return &point{X: positionX, Y: positionY}
		}
		if positionX != grid.X+grid.Width {
			positionX += stepX
		} else {
			positionX = grid.X
			positionY += stepY
		}
	}
	return nil
}

func (f *Fisher) isCaught(float *Element, reversed bool) bool {
	if catch(float, reversed) {
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
