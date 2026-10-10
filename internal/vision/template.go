// Package vision ищет поплавок на снимке экрана сопоставлением с эталонами.
package vision

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "image/jpeg"
	_ "image/png"
)

// Template — эталон с вычтенным средним: тогда нормированная корреляция
// сводится к свёртке, а статистика окна берётся из интегральных сумм.
type Template struct {
	Name string
	W, H int

	dev  []float64
	norm float64
}

// Hit — совпадение: левый верхний угол в координатах кадра и счёт от -1 до 1.
type Hit struct {
	Template string
	X, Y     int
	Score    float64
}

func (h Hit) Center(t Template) (int, int) { return h.X + t.W/2, h.Y + t.H/2 }

// LoadTemplates читает картинки из каталога. Пустой каталог — не ошибка: снимки
// добавляются по ходу работы, и каталог создаётся здесь же. Негодные файлы
// пропускаются, причины возвращаются вторым значением.
func LoadTemplates(dir string) ([]Template, []error, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("каталог эталонов %s: %w", dir, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("каталог эталонов %s: %w", dir, err)
	}

	var out []Template
	var skipped []error
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".png", ".jpg", ".jpeg":
		default:
			continue
		}
		t, err := loadTemplate(filepath.Join(dir, e.Name()))
		if err != nil {
			skipped = append(skipped, err)
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, skipped, nil
}

func loadTemplate(path string) (Template, error) {
	f, err := os.Open(path)
	if err != nil {
		return Template{}, fmt.Errorf("эталон %s: %w", path, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return Template{}, fmt.Errorf("эталон %s: %w", path, err)
	}
	return NewTemplate(filepath.Base(path), img)
}

func NewTemplate(name string, img image.Image) (Template, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return Template{}, fmt.Errorf("эталон %s пуст", name)
	}

	vals := make([]float64, 0, w*h)
	var sum float64
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			v := luma(img.At(x, y))
			vals = append(vals, v)
			sum += v
		}
	}

	mean := sum / float64(len(vals))
	var sq float64
	for i := range vals {
		vals[i] -= mean
		sq += vals[i] * vals[i]
	}
	if contrast := math.Sqrt(sq / float64(len(vals))); contrast < MinContrast {
		return Template{}, fmt.Errorf("эталон %s почти однотонный (контраст %.1f, нужно от %.0f) — это вода без поплавка",
			name, contrast, MinContrast)
	}
	return Template{Name: name, W: w, H: h, dev: vals, norm: math.Sqrt(sq)}, nil
}

// MinContrast — наименьший разброс яркости (0..255) годного эталона. У снимков
// поплавка он около 20, у ровной воды — около 2; такой эталон корреляция
// «находит» где угодно со счётом 0.8.
const MinContrast = 8.0

// ScoreAt — счёт эталона в заданной позиции кадра; 0, если он не помещается.
// Нужен, чтобы отличить поплавок от того, что стояло на этом месте до заброса.
func ScoreAt(img image.Image, t Template, x, y int) float64 {
	f := newFrame(img)
	if x < 0 || y < 0 || x+t.W > f.w || y+t.H > f.h {
		return 0
	}
	return f.score(&t, x, y)
}

func luma(c interface{ RGBA() (r, g, b, a uint32) }) float64 {
	r, g, b, _ := c.RGBA()
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 257
}

// frame — кадр с интегральными суммами яркости и её квадратов.
type frame struct {
	w, h   int
	gray   []float64
	sum    []float64
	sumSq  []float64
	stride int
}

func newFrame(img image.Image) *frame {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	f := &frame{
		w: w, h: h,
		gray:   make([]float64, w*h),
		sum:    make([]float64, (w+1)*(h+1)),
		sumSq:  make([]float64, (w+1)*(h+1)),
		stride: w + 1,
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := luma(img.At(b.Min.X+x, b.Min.Y+y))
			f.gray[y*w+x] = v
			i := (y+1)*f.stride + (x + 1)
			f.sum[i] = v + f.sum[i-1] + f.sum[i-f.stride] - f.sum[i-f.stride-1]
			f.sumSq[i] = v*v + f.sumSq[i-1] + f.sumSq[i-f.stride] - f.sumSq[i-f.stride-1]
		}
	}
	return f
}

func (f *frame) window(x, y, w, h int) (sum, sumSq float64) {
	a := y*f.stride + x
	b := y*f.stride + (x + w)
	c := (y+h)*f.stride + x
	d := (y+h)*f.stride + (x + w)
	return f.sum[d] - f.sum[b] - f.sum[c] + f.sum[a],
		f.sumSq[d] - f.sumSq[b] - f.sumSq[c] + f.sumSq[a]
}

// score — нормированная кросс-корреляция эталона в позиции (x, y).
func (f *frame) score(t *Template, x, y int) float64 {
	sum, sumSq := f.window(x, y, t.W, t.H)
	variance := sumSq - sum*sum/float64(t.W*t.H)
	if variance <= 0 {
		return 0
	}
	var num float64
	k := 0
	for j := 0; j < t.H; j++ {
		row := (y+j)*f.w + x
		for i := 0; i < t.W; i++ {
			num += f.gray[row+i] * t.dev[k]
			k++
		}
	}
	return num / (math.Sqrt(variance) * t.norm)
}

type MatchOptions struct {
	Stride   int     // шаг грубого прохода; лучшее место затем уточняется попиксельно
	MinScore float64 // порог совпадения
}

func DefaultMatchOptions() MatchOptions { return MatchOptions{Stride: 2, MinScore: 0.6} }

// FindEach ищет каждым эталоном и возвращает лучшее место каждого, по
// порядку эталонов. Эталон крупнее снимка получает счёт -2.
func FindEach(img image.Image, tmpls []Template, opt MatchOptions) []Hit {
	f := newFrame(img)
	stride := max(opt.Stride, 1)
	hits := make([]Hit, len(tmpls))
	for i := range tmpls {
		t := &tmpls[i]
		hits[i] = Hit{Template: t.Name, Score: -2}
		if t.W > f.w || t.H > f.h {
			continue
		}
		hits[i].X, hits[i].Y, hits[i].Score = f.bestFor(t, stride)
	}
	return hits
}

// Best — лучшее из попаданий FindEach: попадание, индекс эталона и прошёл ли порог.
func Best(hits []Hit, minScore float64) (Hit, int, bool) {
	best, bestIdx := Hit{Score: -2}, -1
	for i, h := range hits {
		if h.Score > best.Score {
			best, bestIdx = h, i
		}
	}
	if bestIdx < 0 {
		return Hit{}, 0, false
	}
	return best, bestIdx, best.Score >= minScore
}

// FindBest ищет лучшее совпадение среди эталонов. Возвращает попадание,
// индекс эталона и прошёл ли порог.
func FindBest(img image.Image, tmpls []Template, opt MatchOptions) (Hit, int, bool) {
	return Best(FindEach(img, tmpls, opt), opt.MinScore)
}

func (f *frame) bestFor(t *Template, stride int) (bestX, bestY int, best float64) {
	maxX, maxY := f.w-t.W, f.h-t.H
	best = -2
	for y := 0; y <= maxY; y += stride {
		for x := 0; x <= maxX; x += stride {
			if s := f.score(t, x, y); s > best {
				best, bestX, bestY = s, x, y
			}
		}
	}
	if stride == 1 {
		return
	}
	fromX, toX := max(bestX-stride, 0), min(bestX+stride, maxX)
	fromY, toY := max(bestY-stride, 0), min(bestY+stride, maxY)
	for y := fromY; y <= toY; y++ {
		for x := fromX; x <= toX; x++ {
			if s := f.score(t, x, y); s > best {
				best, bestX, bestY = s, x, y
			}
		}
	}
	return
}

// FrameChange — средняя разница яркости двух кадров одного размера, 0..1;
// -1, если сравнить нельзя. Отличает несостоявшийся заброс от нераспознанного
// поплавка: в первом случае на воде ничего не меняется.
func FrameChange(a, b image.Image) float64 {
	if a == nil || b == nil {
		return -1
	}
	ba, bb := a.Bounds(), b.Bounds()
	if ba.Dx() != bb.Dx() || ba.Dy() != bb.Dy() || ba.Dx() == 0 || ba.Dy() == 0 {
		return -1
	}
	var sum float64
	for y := 0; y < ba.Dy(); y++ {
		for x := 0; x < ba.Dx(); x++ {
			d := luma(a.At(ba.Min.X+x, ba.Min.Y+y)) - luma(b.At(bb.Min.X+x, bb.Min.Y+y))
			if d < 0 {
				d = -d
			}
			sum += d
		}
	}
	return sum / float64(ba.Dx()*ba.Dy()) / 255
}
