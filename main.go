package main

import (
	"bytes"
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

var matcherUrl string

func init() {
	if logsFile, err := os.OpenFile("fisherlogs.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
		log.Fatalf("error opening file: %v", err)
		return
	} else {
		defer logsFile.Close()
	}
}

var (
	lootProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "fisher_loot_total",
		Help: "The total number of fish looted",
	})
)

func main() {
	http.Handle("/metrics", promhttp.Handler())

	appConfig := fromProperties()
	log.Printf("%+v\n", appConfig)

	matcherUrl = appConfig.TemplateMatcherUrl

	fisher := newFisher()

	for _, t := range appConfig.Templates {
		id := uploadTemplates(t.Path, matcherUrl)
		if id == "" {
			continue
		}
		fisher.elements[t.Name] = fromTemplate(id, t)
	}
	if fisher.elements["float"] == nil {
		panic("float template not specified.")
	}

	go fisher.start()

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(appConfig.Port), nil))
}

func fromTemplate(id string, t *Template) *Element {
	return &Element{
		templateId: id,
		conf:       t.Conf,
		x:          t.X,
		y:          t.Y,
		width:      t.Width,
		height:     t.Height,
		name:       t.Name,
		isDebug:    t.Debug,
	}
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

	if err != nil || buf.Len() < 70 {
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
