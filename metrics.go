package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	castsTotal  = promauto.NewCounter(prometheus.CounterOpts{Name: "fisher_casts_total", Help: "Забросов"})
	bitesTotal  = promauto.NewCounter(prometheus.CounterOpts{Name: "fisher_bites_total", Help: "Услышанных поклёвок"})
	lootTotal   = promauto.NewCounter(prometheus.CounterOpts{Name: "fisher_loot_total", Help: "Собранной добычи"})
	castRetries = promauto.NewCounter(prometheus.CounterOpts{Name: "fisher_cast_retries_total", Help: "Повторных забросов"})
	areaShifts  = promauto.NewCounter(prometheus.CounterOpts{Name: "fisher_area_shifts_total", Help: "Сдвигов области поиска"})
	missesTotal = promauto.NewCounterVec(prometheus.CounterOpts{Name: "fisher_misses_total", Help: "Неудач по причине"}, []string{"reason"})

	splashBaseline = promauto.NewGauge(prometheus.GaugeOpts{Name: "fisher_audio_baseline", Help: "Фоновый уровень звука (RMS)"})
	splashPeak     = promauto.NewGauge(prometheus.GaugeOpts{Name: "fisher_audio_peak", Help: "Пик звука за интервал (RMS)"})
	matchScore     = promauto.NewGauge(prometheus.GaugeOpts{Name: "fisher_match_score", Help: "Счёт последнего сопоставления"})

	aimOffset = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "fisher_aim_offset_px",
		Help:    "Доводка мыши до поплавка перед подсечкой, px",
		Buckets: []float64{0, 1, 2, 4, 8, 16, 32, 64},
	})
)
