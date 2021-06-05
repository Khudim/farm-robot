package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type AppConfig struct {
	RefreshRate        int         `yaml:"refreshRate"`
	TemplateMatcherUrl string      `yaml:"matcherUrl"`
	Display            int         `yaml:"display"`
	Templates          []*Template `yaml:"templates"`
}

type Template struct {
	Name   string  `json:"name"`
	Path   string  `json:"path"`
	Conf   float32 `json:"conf"`
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Debug  bool    `json:"debug"`
}

func fromProperties() AppConfig {
	var appConfig AppConfig

	if file, err := ioutil.ReadFile("./props.yaml"); err == nil {
		if err := yaml.Unmarshal(file, &appConfig); err != nil {
			panic(err)
		}
	} else {
		panic(err)
	}
	return appConfig
}
