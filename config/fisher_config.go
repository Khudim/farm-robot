package config

import (
	"github.com/olebedev/config"
	"io/ioutil"
)

type FisherConfig struct {
	FileName        string
	TemplateDir     string
	TemplateClams   string
	TemplateMeat    string
	TemplateConfirm string
	TemplateLoot    string
	ConfLevel       float32
	FailTolerance   int
	RefreshRate     int
	ScreenshotsSize float64
}

func Parse(mode string) FisherConfig {
	appConfig := FisherConfig{
		FileName:        "screen.png",
		TemplateDir:     "./templates/hinterlands",
		TemplateClams:   "./templates/clams",
		TemplateMeat:    "./templates/meat",
		TemplateConfirm: "./templates/confirm",
		TemplateLoot:    "./templates/loot",
		ConfLevel:       0.75,
		RefreshRate:     4,
		ScreenshotsSize: 0.5,
	}

	if file, err := ioutil.ReadFile("./fisher.properties"); err == nil {
		if cfg, er := config.ParseYaml(string(file)); er == nil {
			if v, e := cfg.Float64("detector.ConfLevel"); e == nil {
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
		}
	} else {
		panic(err)
	}
	return appConfig
}
