package vision

import "testing"

// FrameChange отличает несостоявшийся заброс от нераспознанного поплавка:
// без изменений на воде — ноль, с приводнившимся поплавком — заметно больше.
func TestFrameChangeDistinguishesStillFromEvent(t *testing.T) {
	water := scene(200, 150, 0, 0, 0, 11)
	if still := FrameChange(water, water); still != 0 {
		t.Errorf("одинаковые кадры должны давать 0, получено %.4f", still)
	}

	after := scene(200, 150, 0, 0, 0, 11)
	stampBlob(after, 100, 75, 14)
	if event := FrameChange(water, after); event < 0.005 {
		t.Errorf("поплавок размером с настоящий должен быть заметен, получено %.4f", event)
	}
}

func TestFrameChangeRejectsMismatch(t *testing.T) {
	a, b := scene(100, 100, 0, 0, 0, 1), scene(120, 100, 0, 0, 0, 1)
	if FrameChange(a, b) != -1 || FrameChange(nil, a) != -1 {
		t.Error("несравнимые кадры должны давать -1")
	}
}
