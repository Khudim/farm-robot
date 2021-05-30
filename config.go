package main

import (
	"github.com/olebedev/config"
	"io/ioutil"
)

type FisherConfig struct {
	PipeTemplatesDir   string
	ConfLevel          float32
	RefreshRate        int
	ScreenshotsSize    float64
	LootTemplatesDir   string
	TemplateMatcherUrl string
}

func Parse(mode string) *FisherConfig {
	appConfig := &FisherConfig{
		PipeTemplatesDir:   "./templates/hinterlands",
		ConfLevel:          0.75,
		RefreshRate:        4,
		ScreenshotsSize:    0.5,
		LootTemplatesDir:   "",
		TemplateMatcherUrl: "http://localhost:8080",
	}

	if file, err := ioutil.ReadFile("./.properties"); err == nil {
		if cfg, er := config.ParseYaml(string(file)); er == nil {
			if v, e := cfg.Float64("detector.confLevel"); e == nil {
				appConfig.ConfLevel = float32(v)
			}
			if v, e := cfg.String("templates." + mode); e == nil {
				appConfig.PipeTemplatesDir = "./templates/" + v
			}
			if v, e := cfg.Int("detector.refreshRate"); e == nil {
				appConfig.RefreshRate = v
			}
			if v, e := cfg.Float64("screenshots.size"); e == nil {
				appConfig.ScreenshotsSize = v
			}
			if v, e := cfg.String("templates.loot"); e == nil {
				appConfig.LootTemplatesDir = v
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
