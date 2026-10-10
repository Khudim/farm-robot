package main

import (
	"math"
	"testing"
	"time"
)

func testCfg() SplashConfig {
	return SplashConfig{Threshold: 3, MinLevel: 0.005, Refractory: Duration(100 * time.Millisecond)}
}

func feed(d *SplashDetector, rms float64, n int) int {
	fired := 0
	for i := 0; i < n; i++ {
		if d.observe(rms) {
			fired++
		}
	}
	return fired
}

// Тишина при старте, потом звук: фон должен подтянуться, а не залипнуть на нуле.
func TestBaselineRecoversAfterSilentStart(t *testing.T) {
	d := newDetector(testCfg())
	d.Arm(true)
	feed(d, 0, baselineWindow)
	feed(d, 0.02, baselineWindow*2)
	if base, _ := d.Levels(); base < 0.015 || base > 0.025 {
		t.Fatalf("фон должен был подтянуться к 0.02, получено %.4f", base)
	}
}

// Всплески не должны затягивать сами себя в фон.
func TestBaselineIgnoresSpikes(t *testing.T) {
	d := newDetector(testCfg())
	d.Arm(true)
	feed(d, 0.02, baselineWindow)
	before, _ := d.Levels()
	for i := 0; i < 20; i++ {
		feed(d, 0.02, 40)
		feed(d, 0.30, 10)
	}
	if after, _ := d.Levels(); after > before*1.5 {
		t.Errorf("фон уполз с %.4f до %.4f", before, after)
	}
}

func TestDetectsSpikeAboveThreshold(t *testing.T) {
	d := newDetector(testCfg())
	feed(d, 0.02, baselineWindow)
	d.Arm(true)
	if fired := feed(d, 0.02, 50); fired != 0 {
		t.Errorf("ровный фон сработал %d раз", fired)
	}
	if fired := feed(d, 0.20, 1); fired != 1 {
		t.Errorf("всплеск x10 дал %d событий", fired)
	}
}

func TestNoEventsWhenDisarmed(t *testing.T) {
	d := newDetector(testCfg())
	feed(d, 0.02, baselineWindow)
	d.Arm(false)
	if fired := feed(d, 0.50, 10); fired != 0 {
		t.Errorf("невзведённый детектор сработал %d раз", fired)
	}
}

func TestRefractorySuppressesRepeats(t *testing.T) {
	d := newDetector(testCfg())
	feed(d, 0.02, baselineWindow)
	d.Arm(true)
	if fired := feed(d, 0.30, 20); fired != 1 {
		t.Errorf("подряд громкие блоки дали %d событий, ожидалось 1", fired)
	}
}

func TestMinLevelBlocksQuietSpikes(t *testing.T) {
	d := newDetector(testCfg())
	feed(d, 0.0001, baselineWindow)
	d.Arm(true)
	if fired := feed(d, 0.001, 5); fired != 0 {
		t.Errorf("всплеск ниже minLevel сработал %d раз", fired)
	}
}

func TestWindowReportsPeakAndRatio(t *testing.T) {
	d := newDetector(testCfg())
	feed(d, 0.02, baselineWindow)
	d.Arm(true)
	feed(d, 0.02, 10)
	feed(d, 0.06, 1)
	d.Arm(false)

	peak, ratio, base := d.Window()
	if math.Abs(peak-0.06) > 1e-9 || base <= 0 || math.Abs(ratio-peak/base) > 1e-6 {
		t.Errorf("пик %.4f, фон %.4f, отношение %.2f", peak, base, ratio)
	}
	d.Arm(true)
	if p, _, _ := d.Window(); p != 0 {
		t.Errorf("Arm(true) должен сбрасывать статистику, пик %.4f", p)
	}
}

func TestBlockRMS(t *testing.T) {
	if got := blockRMS(nil); got != -1 {
		t.Errorf("пустой блок: %v", got)
	}
	buf := []byte{0x00, 0x40, 0x00, 0x40, 0x00, 0x40, 0x00, 0x40} // 4 семпла по 0.5
	if got := blockRMS(buf); math.Abs(got-0.5) > 1e-4 {
		t.Errorf("RMS %.4f, ожидалось 0.5", got)
	}
	if got := blockRMS(make([]byte, 64)); got != 0 {
		t.Errorf("тишина: %v", got)
	}
}

// До прогрева детектор не готов: первый заброс иначе уходит вглухую.
func TestReadyAfterWarmup(t *testing.T) {
	d := newDetector(testCfg())
	if d.Ready() {
		t.Fatal("новый детектор не должен быть готов")
	}
	for i := 0; i < warmupBlocks; i++ {
		d.observe(0.001)
	}
	if !d.Ready() {
		t.Error("после прогрева детектор должен быть готов")
	}
}
