package main

import (
	"github.com/olebedev/config"
	"io/ioutil"
)

type FisherConfig struct {
	FileName           string
	TemplateDir        string
	ConfLevel          float32
	FailTolerance      int
	RefreshRate        int
	ScreenshotsSize    float64
	AllowLootFilter    bool
	IsClassic          bool
	TemplateMatcherUrl string
}

func Parse(mode string) *FisherConfig {
	appConfig := &FisherConfig{
		FileName:           "screen.png",
		TemplateDir:        "./templates/hinterlands",
		ConfLevel:          0.75,
		RefreshRate:        4,
		ScreenshotsSize:    0.5,
		AllowLootFilter:    false,
		IsClassic:          false,
		TemplateMatcherUrl: "http://localhost:8080",
	}

	if file, err := ioutil.ReadFile("./.properties"); err == nil {
		if cfg, er := config.ParseYaml(string(file)); er == nil {
			if v, e := cfg.Float64("detector.confLevel"); e == nil {
				appConfig.ConfLevel = float32(v)
			}
			if v, e := cfg.String("templates." + mode); e == nil {
				appConfig.TemplateDir = "./templates/" + v
			}
			if v, e := cfg.Int("detector.refreshRate"); e == nil {
				appConfig.RefreshRate = v
			}
			if v, e := cfg.Float64("screenshots.size"); e == nil {
				appConfig.ScreenshotsSize = v
			}
			if v, e := cfg.Bool("detector.allowLootFilter"); e == nil {
				appConfig.AllowLootFilter = v
			}
			if v, e := cfg.String("matcher.url"); e == nil {
				appConfig.TemplateMatcherUrl = v
			}
		}
	} else {
		panic(err)
	}
	return appConfig
}
