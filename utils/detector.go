package utils

import (
	"gocv.io/x/gocv"
	"log"
	"image"
	"errors"
)

func Detect(filename string, templates []gocv.Mat) (*image.Point, error) {
	img := gocv.IMRead(filename, gocv.IMReadGrayScale)
	if img.Empty() {
		log.Printf("Invalid read of file %s when detect", filename)
	}
	defer img.Close()

	for _, template := range templates {
		point, err := findTemplate(template, img)
		if err == nil {
			return point, nil
		}

	}
	return nil, errors.New("can't detect pipe")
}

func findTemplate(template, img gocv.Mat) (*image.Point, error) {
	result := gocv.NewMat()
	defer result.Close()
	mask := gocv.NewMat()
	defer mask.Close()

	gocv.MatchTemplate(img, template, &result, 5, mask)

	_, maxConfidence, _, point := gocv.MinMaxLoc(result)
	if maxConfidence > 0.70 {
		return &point, nil
	}
	return nil, errors.New("can't find template")
}
