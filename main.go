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

var screen image.Rectangle
var matcherUrl string

func init() {
	if logsFile, err := os.OpenFile("fisherlogs.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
		log.Fatalf("error opening file: %v", err)
		return
	} else {
		defer logsFile.Close()
	}
}

func main() {
	appConfig := fromProperties()
	log.Printf("%+v\n", appConfig)

	screen = screenshot.GetDisplayBounds(appConfig.Display)
	matcherUrl = appConfig.TemplateMatcherUrl

	fisher := newFisher()

	for _, t := range appConfig.Templates {
		id := uploadTemplates(t.Path, matcherUrl)
		if id == "" {
			continue
		}
		fisher.elements[t.Name] = &Element{templateId: id, conf: t.Conf}
	}
	if fisher.elements["float"] == nil {
		panic("float template not specified.")
	}
	fisher.start()
}

func uploadTemplates(elDir, url string) string {
	if elDir == "" {
		return ""
	}
	return upload(elDir, url)
}

func upload(templatesDir, url string) string {
	var strRequestURI = url + "/template/upload"

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	err := filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
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

	if err != nil || buf.Len() < 100 {
		panic("No templates were found.")
	}

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
	var response MatcherResponse
	err = json.Unmarshal(res.Body(), &response)
	if err != nil {
		panic(err)
	}
	log.Println(response)

	return response.TemplateId
}

type MatcherResponse struct {
	TemplateId string `json:"templateId"`
}
