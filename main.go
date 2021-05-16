package main

import (
	"bytes"
	"encoding/json"
	"github.com/kbinani/screenshot"
	"github.com/valyala/fasthttp"
	"image"
	"io/ioutil"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
)

var appConfig *FisherConfig
var screen image.Rectangle

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

	screen = screenshot.GetDisplayBounds(0)

	templateId := uploadTemplates()
	fisher := loadFisher(templateId)
	fisher.start()
}

func uploadTemplates() string {
	var strRequestURI = appConfig.TemplateMatcherUrl + "/template/upload"

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	_ = filepath.Walk(appConfig.TemplateDir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".png" {
			part, err := writer.CreateFormFile(info.Name(), path)
			if err != nil {
				return err
			}
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			_, _ = part.Write(b)
		}
		return err
	})
	_ = writer.Close()

	req.Header.SetMethodBytes([]byte("POST"))
	req.Header.Add("Content-Type", writer.FormDataContentType())
	req.SetRequestURI(strRequestURI)
	req.SetBody(buf.Bytes())

	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	if err := fasthttp.Do(req, res); err != nil {
		panic("handle error")
	}

	if res.StatusCode() != 200 {
		panic(res.Body())
	}
	var template Template
	err := json.Unmarshal(res.Body(), &template)
	if err != nil {
		panic(err)
	}
	log.Println(template)

	return template.Id
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
