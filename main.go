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

var appConfig *AppConfig
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

	appConfig, elConfig := fromPropeties(mode)
	log.Printf("%+v\n", appConfig)

	screen = screenshot.GetDisplayBounds(0)

	fisher := newRetailFisher()

	floatId := uploadTemplates(elConfig.FloatTemplatesDir)
	fisher.floatEl = &Element{templateId: floatId}

	lootId := uploadTemplates(elConfig.LootTemplatesDir)
	fisher.lootEl = &Element{templateId: lootId}

	biteId := uploadTemplates(elConfig.LootTemplatesDir)
	fisher.biteEl = &Element{templateId: biteId}

	fisher.start()
}

func uploadTemplates(elDir string) string {
	if elDir == "" {
		panic("No pipe templates")
	}
	return upload(elDir)
}

func upload(templatesDir string) string {
	var strRequestURI = appConfig.TemplateMatcherUrl + "/template/upload"

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	_ = filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
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
