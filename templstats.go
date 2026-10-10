package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"farm-robot/internal/vision"
)

// Счёт каждого эталона за сессию. Эталон, который за много находок поплавка ни
// разу не дотянул до порога, ничего не находит, а время на проход тратит и
// шанс ложного срабатывания добавляет — до перезапуска он больше не ищет.

// judgeAfter — после стольких найденных поплавков бесполезный эталон отключается.
const judgeAfter = 15

type tmplStat struct {
	finds int     // сколько раз поплавок был найден, пока эталон искал
	wins  int     // сколько раз нашёл именно он
	best  float64 // лучший его счёт за эти находки
}

// scoresLine — счёт каждого эталона в процентах, от лучшего к худшему.
func scoresLine(hits []vision.Hit) string {
	sorted := append([]vision.Hit(nil), hits...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Score > sorted[j].Score })
	parts := make([]string, 0, len(sorted))
	for _, h := range sorted {
		parts = append(parts, fmt.Sprintf("%s %.0f%%", strings.TrimSuffix(h.Template, ".png"), h.Score*100))
	}
	return strings.Join(parts, ", ")
}

// maxHits — лучший счёт каждого эталона за несколько проверок.
func maxHits(acc, hits []vision.Hit) []vision.Hit {
	if len(acc) != len(hits) {
		return append([]vision.Hit(nil), hits...)
	}
	for i := range hits {
		if hits[i].Score > acc[i].Score {
			acc[i] = hits[i]
		}
	}
	return acc
}

// judgeTemplates учитывает находку поплавка эталоном winner и отключает те,
// что за judgeAfter находок ни разу не прошли порог сами.
func (f *Fisher) judgeTemplates(hits []vision.Hit, winner int) {
	if f.tstats == nil {
		f.tstats = map[string]*tmplStat{}
	}
	var drop []string
	for i, h := range hits {
		st := f.tstats[h.Template]
		if st == nil {
			st = &tmplStat{best: -2}
			f.tstats[h.Template] = st
		}
		st.finds++
		st.best = max(st.best, h.Score)
		if i == winner {
			st.wins++
		}
		if st.finds >= judgeAfter && st.wins == 0 && st.best < f.cfg.Search.MinScore {
			drop = append(drop, h.Template)
		}
	}
	for _, name := range drop {
		if len(f.templates) <= 1 {
			return
		}
		for i, t := range f.templates {
			if t.Name == name {
				st := f.tstats[name]
				log.Printf("эталон %s не ищу до перезапуска: за %d находок поплавка лучший его счёт %.0f%% при пороге %.0f%% — можно удалить файл",
					name, st.finds, st.best*100, f.cfg.Search.MinScore*100)
				f.templates = append(f.templates[:i], f.templates[i+1:]...)
				break
			}
		}
	}
}
