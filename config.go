package main

import (
	"github.com/olebedev/config"
	"io/ioutil"
)

type AppConfig struct {
	ConfLevel          float32
	RefreshRate        int
	ScreenshotsSize    float64
	TemplateMatcherUrl string
}

type ElConfig struct {
	FloatTemplatesDir string
	LootTemplatesDir  string
	BiteTemplatesDir  string
}

func fromPropeties(mode string) (*AppConfig, *ElConfig) {
	appConfig := &AppConfig{
		ConfLevel:          0.75,
		RefreshRate:        4,
		ScreenshotsSize:    0.5,
		TemplateMatcherUrl: "http://localhost:8080",
	}

	elConfig := &ElConfig{
		FloatTemplatesDir: "./templates/hinterlands",
		LootTemplatesDir:  "",
		BiteTemplatesDir:  "",
	}

	if file, err := ioutil.ReadFile("./.properties"); err == nil {
		if cfg, er := config.ParseYaml(string(file)); er == nil {
			if v, e := cfg.Float64("detector.confLevel"); e == nil {
				appConfig.ConfLevel = float32(v)
			}
			if v, e := cfg.String("templates." + mode + ".loot"); e == nil {
				elConfig.LootTemplatesDir = v
			}
			if v, e := cfg.String("templates." + mode + ".float"); e == nil {
				elConfig.FloatTemplatesDir = v
			}
			if v, e := cfg.String("templates." + mode + ".bite"); e == nil {
				elConfig.BiteTemplatesDir = v
			}
			if v, e := cfg.String("templates." + mode + ".pole"); e == nil {
				elConfig.BiteTemplatesDir = v
			}
			if v, e := cfg.Int("detector.refreshRate"); e == nil {
				appConfig.RefreshRate = v
			}
			if v, e := cfg.Float64("screenshots.size"); e == nil {
				appConfig.ScreenshotsSize = v
			}
			if v, e := cfg.String("matcher.url"); e == nil {
				appConfig.TemplateMatcherUrl = v
			}
		}
	} else {
		panic(err)
	}
	return appConfig, elConfig
}
