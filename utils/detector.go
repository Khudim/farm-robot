package utils

import (
	"gocv.io/x/gocv"
	"log"
	"image"
	"errors"
)

func Detect(filename string, templates []gocv.Mat, confLevel float32) (*image.Point, error) {
	img := gocv.IMRead(filename, gocv.IMReadGrayScale)
	if img.Empty() {
		log.Printf("Invalid read of file %s when detect", filename)
		return nil, errors.New("can't detect pipe")
	}
	defer img.Close()
	result := make(chan image.Point, len(templates))

	for _, template := range templates {
		go findTemplate(template, img, confLevel, result)
	}

	for i := 0; i < len(templates); i++ {
		select {
		case point := <-result:
			if point.X != 0 {
				return &point, nil
			}
		}
	}
	return nil, errors.New("can't detect pipe")
}

func findTemplate(template, img gocv.Mat, confLevel float32, pointChannel chan image.Point) {
	result := gocv.NewMat()
	defer result.Close()
	mask := gocv.NewMat()
	defer mask.Close()

	gocv.MatchTemplate(img, template, &result, 5, mask)

	_, maxConfidence, _, point := gocv.MinMaxLoc(result)
	if maxConfidence > confLevel {
		pointChannel <- point
	} else {
		log.Print("not found ", maxConfidence)
		pointChannel <- image.Point{0, 0}
	}
}
