package main

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

var appConfig *FisherConfig

func init() {
	if logsFile, err := os.OpenFile("fisherlogs.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
		log.Fatalf("error opening file: %v", err)
		return
	} else {
		defer logsFile.Close()
	}
}

func main() {
	mode := "default"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	appConfig = Parse(mode)
	log.Printf("%+v\n", appConfig)

	templateId := uploadTemplates()
	fisher := loadFisher(templateId)
	fisher.start()
}

func uploadTemplates() string {
	var strRequestURI = []byte(appConfig.TemplateMatcherUrl + "/template/upload")

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.Header.SetMethodBytes([]byte("POST"))

	_ = filepath.Walk(appConfig.TemplateDir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".png" {
			if image, e := ioutil.ReadFile(path); e == nil {
				req.Header.Add("file_"+info.Name(), strconv.Itoa(len(image)))
				req.AppendBody(image)
			} else {
				fmt.Println(e)
			}
		}
		return err
	})

	req.SetRequestURIBytes(strRequestURI)

	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	if err := fasthttp.Do(req, res); err != nil {
		panic("handle error")
	}

	templateId := res.Body()
	log.Println(string(templateId))

	return string(templateId)
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
