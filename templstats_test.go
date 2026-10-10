package main

import (
	"strings"
	"testing"

	"farm-robot/internal/vision"
)

func TestUselessTemplateIsDropped(t *testing.T) {
	f := &Fisher{cfg: defaultConfig()}
	f.templates = []vision.Template{{Name: "good.png"}, {Name: "close.png"}, {Name: "bad.png"}}
	hits := []vision.Hit{{Template: "good.png", Score: 0.8}, {Template: "close.png", Score: 0.65}, {Template: "bad.png", Score: 0.3}}
	for i := 1; i < judgeAfter; i++ {
		f.judgeTemplates(hits, 0)
	}
	if len(f.templates) != 3 {
		t.Fatalf("до %d находок отключать рано, осталось %d", judgeAfter, len(f.templates))
	}
	f.judgeTemplates(hits, 0)
	var names []string
	for _, tm := range f.templates {
		names = append(names, tm.Name)
	}
	// close.png сам проходил порог, хоть и не побеждал, — остаётся.
	if got := strings.Join(names, ","); got != "good.png,close.png" {
		t.Fatalf("осталось %s, ожидалось good.png,close.png", got)
	}
}

func TestLastTemplateStays(t *testing.T) {
	f := &Fisher{cfg: defaultConfig()}
	f.templates = []vision.Template{{Name: "only.png"}}
	for range judgeAfter * 2 {
		f.judgeTemplates([]vision.Hit{{Template: "only.png", Score: 0.1}}, -1)
	}
	if len(f.templates) != 1 {
		t.Fatal("последний эталон отключать нельзя")
	}
}

func TestScoresLine(t *testing.T) {
	got := scoresLine([]vision.Hit{{Template: "a.png", Score: 0.412}, {Template: "b.png", Score: 0.734}})
	if got != "b 73%, a 41%" {
		t.Fatalf("получено %q", got)
	}
}
