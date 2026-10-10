package main

import (
	"errors"
	"fmt"
	"log"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/malgo"
)

// SplashDetector слушает вывод звуковой карты и ловит всплеск поклёвки —
// короткий пик громкости над фоном. Музыку и ambience в игре лучше выключить.
type SplashDetector struct {
	ctx    *malgo.AllocatedContext
	device *malgo.Device
	cfg    SplashConfig
	events chan time.Time

	mu       sync.Mutex
	baseline float64
	peak     float64
	lastHit  time.Time
	armed    bool
	blocks   int
	ring     []float64
	ringPos  int
	ringFull bool

	// статистика окна ожидания — чтобы промах мог объяснить себя
	winPeak, winRatio, winBase float64
	hitLevel, hitBase          float64 // уровень и фон в момент последнего срабатывания
}

const (
	sampleRate = 44100
	periodMs   = 10

	warmupBlocks       = 100 // секунда молчаливого прогрева
	baselineWindow     = 300 // три секунды
	baselinePercentile = 0.25
	baselineEvery      = 10
)

func newDetector(cfg SplashConfig) *SplashDetector {
	return &SplashDetector{cfg: cfg, events: make(chan time.Time, 4), ring: make([]float64, baselineWindow)}
}

func NewSplashDetector(cfg SplashConfig) (*SplashDetector, error) {
	d := newDetector(cfg)

	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("инициализация звука: %w", err)
	}
	d.ctx = ctx

	devCfg, err := loopbackConfig(ctx)
	if err != nil {
		d.Close()
		return nil, err
	}
	device, err := malgo.InitDevice(ctx.Context, devCfg, malgo.DeviceCallbacks{Data: d.onFrames})
	if err != nil {
		d.Close()
		return nil, fmt.Errorf("устройство захвата: %w", err)
	}
	d.device = device
	if err := device.Start(); err != nil {
		d.Close()
		return nil, fmt.Errorf("запуск захвата: %w", err)
	}
	return d, nil
}

// loopbackConfig: WASAPI умеет loopback штатно; на PulseAudio/PipeWire вывод
// карты виден как capture-устройство с суффиксом .monitor.
func loopbackConfig(ctx *malgo.AllocatedContext) (malgo.DeviceConfig, error) {
	if runtime.GOOS == "windows" {
		cfg := malgo.DefaultDeviceConfig(malgo.Loopback)
		applyFormat(&cfg)
		return cfg, nil
	}
	infos, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return malgo.DeviceConfig{}, fmt.Errorf("устройства захвата: %w", err)
	}
	for i := range infos {
		if !strings.Contains(strings.ToLower(infos[i].Name()), "monitor") {
			continue
		}
		cfg := malgo.DefaultDeviceConfig(malgo.Capture)
		cfg.Capture.DeviceID = infos[i].ID.Pointer()
		applyFormat(&cfg)
		log.Printf("звук: слушаю %q", infos[i].Name())
		return cfg, nil
	}
	return malgo.DeviceConfig{}, errors.New("не найдено monitor-устройство (pavucontrol → Recording или `pactl list sources short`)")
}

func applyFormat(cfg *malgo.DeviceConfig) {
	cfg.Capture.Format = malgo.FormatS16
	cfg.Capture.Channels = 1
	cfg.SampleRate = sampleRate
	cfg.PeriodSizeInMilliseconds = periodMs
}

func (d *SplashDetector) onFrames(_, in []byte, _ uint32) {
	if rms := blockRMS(in); rms >= 0 && d.observe(rms) {
		select {
		case d.events <- time.Now():
		default:
		}
	}
}

// blockRMS — среднеквадратичный уровень блока S16LE моно; -1 для пустого.
func blockRMS(in []byte) float64 {
	n := len(in) / 2
	if n == 0 {
		return -1
	}
	var sum float64
	for i := 0; i < n; i++ {
		s := float64(int16(uint16(in[2*i])|uint16(in[2*i+1])<<8)) / 32768
		sum += s * s
	}
	return math.Sqrt(sum / float64(n))
}

// observe прогоняет блок через детектор. Фон — нижний перцентиль скользящего
// окна: всплески в него не попадают, а при смене обстановки окно вытесняется.
func (d *SplashDetector) observe(rms float64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if rms > d.peak {
		d.peak = rms
	}
	d.ring[d.ringPos] = rms
	d.ringPos = (d.ringPos + 1) % len(d.ring)
	if d.ringPos == 0 {
		d.ringFull = true
	}
	if d.blocks++; d.blocks%baselineEvery == 0 {
		d.baseline = d.percentile(baselinePercentile)
	}
	if !d.ringFull && d.blocks < warmupBlocks {
		return false
	}

	loud := rms > d.baseline*d.cfg.Threshold
	if d.armed && rms > d.winPeak {
		d.winPeak, d.winBase = rms, d.baseline
		if d.baseline > 0 {
			d.winRatio = rms / d.baseline
		}
	}
	if !loud || rms <= d.cfg.MinLevel || !d.armed || time.Since(d.lastHit) < d.cfg.Refractory.D() {
		return false
	}
	d.lastHit = time.Now()
	d.hitLevel, d.hitBase = rms, d.baseline
	return true
}

// LastHit — уровень звука и фон, на которых сработал последний всплеск.
func (d *SplashDetector) LastHit() (level, baseline float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.hitLevel, d.hitBase
}

func (d *SplashDetector) percentile(p float64) float64 {
	n := len(d.ring)
	if !d.ringFull {
		n = d.ringPos
	}
	if n == 0 {
		return 0
	}
	buf := make([]float64, n)
	copy(buf, d.ring[:n])
	sort.Float64s(buf)
	return buf[min(int(p*float64(n)), n-1)]
}

// Ready — фон набран, детектор отличает всплеск от тишины.
func (d *SplashDetector) Ready() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.ringFull || d.blocks >= warmupBlocks
}

// Arm включает выдачу событий — только в окне ожидания поклёвки.
func (d *SplashDetector) Arm(on bool) {
	d.mu.Lock()
	if on {
		d.winPeak, d.winRatio, d.winBase = 0, 0, 0
	}
	d.armed = on
	d.mu.Unlock()
}

// Window — статистика последнего окна: пик, отношение к фону, фон.
func (d *SplashDetector) Window() (peak, ratio, baseline float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.winPeak, d.winRatio, d.winBase
}

func (d *SplashDetector) Drain() {
	for {
		select {
		case <-d.events:
		default:
			return
		}
	}
}

func (d *SplashDetector) Wait(timeout time.Duration) bool {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-d.events:
		return true
	case <-t.C:
		return false
	}
}

// Poll — был ли всплеск, не ожидая.
func (d *SplashDetector) Poll() bool {
	select {
	case <-d.events:
		return true
	default:
		return false
	}
}

// Levels — фон и пик с прошлого вызова.
func (d *SplashDetector) Levels() (baseline, peak float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	baseline, peak = d.baseline, d.peak
	d.peak = 0
	return
}

func (d *SplashDetector) Close() {
	if d.device != nil {
		d.device.Uninit()
		d.device = nil
	}
	if d.ctx != nil {
		_ = d.ctx.Uninit()
		d.ctx.Free()
		d.ctx = nil
	}
}
