package mouse

import "testing"

// Логическая → физическая → логическая должна возвращать исходное значение при
// любом реальном масштабе. Именно эта пара преобразований стоит между тем, где
// поплавок найден на снимке, и тем, куда поедет мышь.
func TestRoundTripAcrossScales(t *testing.T) {
	for _, f := range []float64{1.0, 1.25, 1.5, 1.75, 2.0} {
		for _, v := range []int{0, 1, 7, 100, 1177, 1919, 2559} {
			got := ToLogical(ToPhysical(v, f), f)
			if got != v {
				t.Errorf("масштаб %.2f: %d → %d → %d", f, v, ToPhysical(v, f), got)
			}
		}
	}
}

// При 100% преобразования не должны трогать значение вообще.
func TestIdentityAtHundredPercent(t *testing.T) {
	for _, v := range []int{0, 1, 555, 2560} {
		if ToPhysical(v, 1) != v || ToLogical(v, 1) != v {
			t.Errorf("при масштабе 1.0 значение %d изменилось", v)
		}
	}
}

// При 125% расхождение между системами — ровно множитель. Это та ошибка,
// которую шим и должен закрывать: без него мышь уезжала бы в ToLogical(v).
func TestScaleGapIsTheBug(t *testing.T) {
	const f, v = 1.25, 1000
	if got := ToLogical(v, f); got != 800 {
		t.Errorf("без компенсации мышь уехала бы в %d вместо %d", got, v)
	}
}
