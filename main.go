package main

import (
	"github.com/valyala/fasthttp"
	"log"
	"os"
)

var appConfig *FisherConfig

func init() {
	if logsFile, err := os.OpenFile("fisherlogs.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
		log.Fatalf("error opening file: %v", err)
		return
	} else {
		defer logsFile.Close()
	}
	_ = os.Mkdir("failed", os.FileMode(777))
}

func main() {
	mode := "default"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	appConfig = Parse(mode)
	log.Printf("%+v\n", appConfig)

	templates := readTemplates(appConfig.TemplateDir)
	status, templateId, err := fasthttp.Post(templates, appConfig.TemplateMatcherUrl+"/template/upload", nil)
	if status != 200 {
		log.Fatalln("Can't upload templates", err)
		return
	}

	var fisherBot = loadFisher(string(templateId))

	fisherBot.start()
}

func loadFisher(templateId string) Fisher {
	var f Fisher
	if appConfig.IsClassic {
		// f = &ClassicFisher{}
		panic("Unsupported mode")
	} else {
		f = newRetailFisher(templateId)
	}
	f.init()
	return f
}
