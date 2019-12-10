package detector

import (
	"errors"
	"gocv.io/x/gocv"
	"image"
	"log"
)

func Detect(filename string, templates []gocv.Mat, confLevel float32) (*image.Point, error) {
	img := gocv.IMRead(filename, gocv.IMReadGrayScale)
	if img.Empty() {
		log.Printf("Invalid read of file %s when detect", filename)
		return nil, errors.New("can't detect pipe")
	}
	defer closeResource(img)
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
	defer closeResource(result)
	mask := gocv.NewMat()
	defer closeResource(mask)

	gocv.MatchTemplate(img, template, &result, 5, mask)

	_, maxConfidence, _, point := gocv.MinMaxLoc(result)
	if maxConfidence > confLevel {
		pointChannel <- point
	} else {
		pointChannel <- image.Point{}
	}
}

func closeResource(resource gocv.Mat) {
	if err := resource.Close(); err != nil {
		log.Println(err)
	}
}
