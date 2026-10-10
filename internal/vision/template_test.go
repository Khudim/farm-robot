package vision

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// scene рисует шумный фон и вставляет в него «поплавок» — яркое пятно с тёмной
// каймой, похожее по структуре на настоящий.
func scene(w, h int, blobX, blobY, r int, seed int64) *image.Gray {
	rng := rand.New(rand.NewSource(seed))
	img := image.NewGray(image.Rect(0, 0, w, h))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetGray(x, y, color.Gray{Y: uint8(50 + rng.Intn(25))})
		}
	}
	if r > 0 {
		stampBlob(img, blobX, blobY, r)
	}
	return img
}

func stampBlob(img *image.Gray, cx, cy, r int) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			d2 := (x-cx)*(x-cx) + (y-cy)*(y-cy)
			if d2 > r*r {
				continue
			}
			if !(image.Point{X: x, Y: y}).In(img.Bounds()) {
				continue
			}
			v := uint8(240)
			if d2 > (r-2)*(r-2) {
				v = 20 // тёмная кайма
			}
			img.SetGray(x, y, color.Gray{Y: v})
		}
	}
}

func cut(src *image.Gray, x, y, w, h int) *image.Gray {
	out := image.NewGray(image.Rect(0, 0, w, h))
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			out.SetGray(i, j, src.GrayAt(x+i, y+j))
		}
	}
	return out
}

func mustTemplate(t *testing.T, img image.Image) Template {
	t.Helper()
	tmpl, err := NewTemplate("проверка", img)
	if err != nil {
		t.Fatalf("подготовка эталона: %v", err)
	}
	return tmpl
}

// Базовая проверка: вырезаем кусок картинки и ищем его в ней же.
func TestFindsExactCutout(t *testing.T) {
	const bx, by, r = 180, 90, 10
	img := scene(300, 200, bx, by, r, 1)
	tmpl := mustTemplate(t, cut(img, bx-12, by-12, 24, 24))

	hit, _, ok := FindBest(img, []Template{tmpl}, DefaultMatchOptions())
	if !ok {
		t.Fatalf("совпадение не найдено, лучший счёт %.3f", hit.Score)
	}
	if hit.Score < 0.99 {
		t.Errorf("точный вырез должен давать счёт около 1, получено %.3f", hit.Score)
	}
	if hit.X != bx-12 || hit.Y != by-12 {
		t.Errorf("найдено в (%d,%d), ожидалось (%d,%d)", hit.X, hit.Y, bx-12, by-12)
	}
}

// Эталон, снятый на одном фоне, должен находиться на другом: в игре вода
// всегда немного другая. Именно ради этого используется нормированная
// корреляция, а не разность пикселей.
func TestFindsOnDifferentBackground(t *testing.T) {
	const r = 10
	source := scene(300, 200, 150, 100, r, 2)
	tmpl := mustTemplate(t, cut(source, 138, 88, 24, 24))

	const bx, by = 70, 140
	other := scene(300, 200, bx, by, r, 99) // другой шум

	hit, _, ok := FindBest(other, []Template{tmpl}, DefaultMatchOptions())
	if !ok {
		t.Fatalf("на другом фоне не нашлось, лучший счёт %.3f", hit.Score)
	}

	cx, cy := hit.Center(tmpl)
	if d := math.Hypot(float64(cx-bx), float64(cy-by)); d > 4 {
		t.Errorf("центр найден в (%d,%d), это %.1f px от поплавка (%d,%d), счёт %.3f",
			cx, cy, d, bx, by, hit.Score)
	}
}

// Если поплавка в кадре нет, порог не должен быть пройден. Ложное срабатывание
// хуже промаха: бот будет ждать поклёвки, глядя в пустую воду.
func TestRejectsWhenAbsent(t *testing.T) {
	source := scene(300, 200, 150, 100, 10, 3)
	tmpl := mustTemplate(t, cut(source, 138, 88, 24, 24))

	empty := scene(300, 200, 0, 0, 0, 77) // только шум

	hit, _, ok := FindBest(empty, []Template{tmpl}, DefaultMatchOptions())
	if ok {
		t.Errorf("на пустой воде нашлось совпадение со счётом %.3f в (%d,%d)",
			hit.Score, hit.X, hit.Y)
	}
}

// Грубый проход с шагом не должен терять цель: ради этого после него идёт
// уточнение по каждому пикселю.
func TestStrideDoesNotLoseTarget(t *testing.T) {
	const bx, by, r = 111, 77, 10
	img := scene(300, 200, bx, by, r, 4)
	tmpl := mustTemplate(t, cut(img, bx-12, by-12, 24, 24))

	exact, _, _ := FindBest(img, []Template{tmpl}, MatchOptions{Stride: 1, MinScore: 0.5})

	for _, stride := range []int{2, 3, 5} {
		got, _, ok := FindBest(img, []Template{tmpl}, MatchOptions{Stride: stride, MinScore: 0.5})
		if !ok {
			t.Errorf("шаг %d: совпадение потеряно", stride)
			continue
		}
		if got.X != exact.X || got.Y != exact.Y {
			t.Errorf("шаг %d: найдено (%d,%d) вместо (%d,%d), счёт %.3f против %.3f",
				stride, got.X, got.Y, exact.X, exact.Y, got.Score, exact.Score)
		}
	}
}

// Из нескольких эталонов должен выбираться тот, что совпал лучше.
func TestPicksBestTemplate(t *testing.T) {
	const bx, by, r = 200, 120, 10
	img := scene(300, 200, bx, by, r, 5)

	right := mustTemplate(t, cut(img, bx-12, by-12, 24, 24))
	wrong := mustTemplate(t, cut(scene(300, 200, 40, 40, 4, 6), 28, 28, 24, 24))
	wrong.Name = "не тот"
	right.Name = "тот"

	hit, idx, ok := FindBest(img, []Template{wrong, right}, DefaultMatchOptions())
	if !ok {
		t.Fatalf("не нашлось, лучший счёт %.3f", hit.Score)
	}
	if hit.Template != "тот" || idx != 1 {
		t.Errorf("выбран эталон %q (индекс %d), ожидался «тот»", hit.Template, idx)
	}
}

// Яркость кадра в игре гуляет от времени суток и погоды. Нормированная
// корреляция к этому нечувствительна — на этом и держится весь подход.
func TestSurvivesBrightnessShift(t *testing.T) {
	const bx, by, r = 150, 100, 10
	img := scene(300, 200, bx, by, r, 7)
	tmpl := mustTemplate(t, cut(img, bx-12, by-12, 24, 24))

	dimmed := image.NewGray(img.Bounds())
	for y := 0; y < 200; y++ {
		for x := 0; x < 300; x++ {
			dimmed.SetGray(x, y, color.Gray{Y: uint8(float64(img.GrayAt(x, y).Y) * 0.6)})
		}
	}

	hit, _, ok := FindBest(dimmed, []Template{tmpl}, DefaultMatchOptions())
	if !ok {
		t.Fatalf("на затемнённом кадре не нашлось, счёт %.3f", hit.Score)
	}
	if hit.X != bx-12 || hit.Y != by-12 {
		t.Errorf("найдено в (%d,%d), ожидалось (%d,%d)", hit.X, hit.Y, bx-12, by-12)
	}
}

// Одноцветный эталон бессмысленен: корреляция с ним не определена.
func TestRejectsFlatTemplate(t *testing.T) {
	flat := image.NewGray(image.Rect(0, 0, 10, 10))
	if _, err := NewTemplate("одноцветный", flat); err == nil {
		t.Error("ожидалась ошибка на одноцветном эталоне")
	}
}

// Снимок ровной воды без поплавка — почти однотонный, с лёгкой рябью. На
// живом запуске такой эталон совпадал с пустой водой со счётом 0.8, и бот
// «ловил» там, где ничего нет. Настоящий поплавок контрастнее в разы.
func TestRejectsLowContrastTemplate(t *testing.T) {
	water := image.NewGray(image.Rect(0, 0, 41, 41))
	rng := rand.New(rand.NewSource(1))
	for y := 0; y < 41; y++ {
		for x := 0; x < 41; x++ {
			water.SetGray(x, y, color.Gray{Y: uint8(60 + rng.Intn(4))})
		}
	}
	if _, err := NewTemplate("вода", water); err == nil {
		t.Error("рябь без поплавка должна отклоняться")
	}
	if _, err := NewTemplate("поплавок", cut(scene(60, 60, 30, 30, 10, 2), 10, 10, 41, 41)); err != nil {
		t.Errorf("настоящий поплавок должен проходить: %v", err)
	}
}

// LoadTemplates не роняет запуск из-за одного негодного файла: остальные
// работают, а негодный называется по имени.
func TestLoadTemplatesSkipsBadFiles(t *testing.T) {
	dir := t.TempDir()
	good := cut(scene(60, 60, 30, 30, 10, 3), 10, 10, 41, 41)
	writePNG(t, filepath.Join(dir, "good.png"), good)
	writePNG(t, filepath.Join(dir, "flat.png"), image.NewGray(image.Rect(0, 0, 41, 41)))

	got, skipped, err := LoadTemplates(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "good.png" {
		t.Errorf("ожидался один годный эталон good.png, получено %+v", got)
	}
	if len(skipped) != 1 || !strings.Contains(skipped[0].Error(), "flat.png") {
		t.Errorf("негодный файл должен быть назван: %v", skipped)
	}
}

// ScoreAt в точке, откуда вырезан эталон, даёт около 1, а на пустом шуме — мало.
func TestScoreAtMatchesFindBest(t *testing.T) {
	const bx, by = 150, 100
	img := scene(300, 200, bx, by, 10, 4)
	tmpl := mustTemplate(t, cut(img, bx-12, by-12, 24, 24))

	if s := ScoreAt(img, tmpl, bx-12, by-12); s < 0.99 {
		t.Errorf("на своём месте счёт %.3f, ожидался около 1", s)
	}
	if s := ScoreAt(scene(300, 200, 0, 0, 0, 5), tmpl, bx-12, by-12); s > 0.5 {
		t.Errorf("на пустой воде счёт %.3f, слишком высок", s)
	}
	if s := ScoreAt(img, tmpl, 290, 190); s != 0 {
		t.Errorf("за границей кадра ожидался 0, получено %.3f", s)
	}
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// Эталон крупнее кадра не должен ронять поиск.
func TestTemplateLargerThanFrame(t *testing.T) {
	img := scene(20, 20, 10, 10, 4, 8)
	big := mustTemplate(t, scene(50, 50, 25, 25, 8, 9))

	if _, _, ok := FindBest(img, []Template{big}, DefaultMatchOptions()); ok {
		t.Error("эталон крупнее кадра не может совпасть")
	}
}

// Пустой каталог не должен быть ошибкой: это тот же каталог, куда складываются
// снимки, и на новой машине он законно пуст. Иначе бот нельзя запустить, пока
// не подготовишь файлы, — а готовить их удобнее из запущенного бота.
func TestLoadTemplatesAcceptsEmptyDir(t *testing.T) {
	dir := t.TempDir()

	got, _, err := LoadTemplates(dir)
	if err != nil {
		t.Fatalf("пустой каталог не должен быть ошибкой: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ожидался пустой набор, получено %d", len(got))
	}
}

// Каталога может не быть вовсе — создаём его сами, чтобы снимкам было куда лечь.
func TestLoadTemplatesCreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "нет", "такого")

	if _, _, err := LoadTemplates(dir); err != nil {
		t.Fatalf("отсутствующий каталог должен создаваться: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("каталог не создан: %v", err)
	}
}

// Пустой набор эталонов не должен ронять поиск.
func TestFindBestWithNoTemplates(t *testing.T) {
	img := scene(100, 100, 50, 50, 8, 1)

	if _, _, ok := FindBest(img, nil, DefaultMatchOptions()); ok {
		t.Error("без эталонов совпадений быть не может")
	}
}
