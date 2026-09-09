package poly

import (
	"testing"

	"github.com/almerlucke/psalm"
)

type mockVoice struct {
	*psalm.Module
	active   bool
	lifeSpan int
}

func newMockVoice() *mockVoice {
	v := &mockVoice{
		Module: psalm.NewModule(nil, []int{psalm.Stereo}),
	}
	v.SetSelf(v)
	return v
}

func (v *mockVoice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, config psalm.Config) {
	v.active = true
	v.lifeSpan = 2
}

func (v *mockVoice) NoteOff() {
	v.active = false
}

func (v *mockVoice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, config psalm.Config) {
	v.active = true
	v.lifeSpan = int(duration)
}

func (v *mockVoice) Clear() {
	v.Output(0).Channels.Clear()
}

func (v *mockVoice) IsActive() bool {
	return v.active
}

func (v *mockVoice) Synthesize() {
	if v.lifeSpan > 0 {
		v.lifeSpan--
		if v.lifeSpan == 0 {
			v.active = false
		}
	}
}

func TestPolySynthesizeSwapAndPop(t *testing.T) {
	voices := make([]*mockVoice, 4)
	for i := range voices {
		voices[i] = newMockVoice()
	}

	p := New(psalm.Stereo, voices)

	// Trigger all 4 voices with duration 1
	for i := 0; i < 4; i++ {
		msg := psalm.Message{
			"command":   "trigger",
			"tick":      psalm.CtrlTick{},
			"duration":  float64(1),
			"amplitude": 1.0,
			"message":   psalm.Message{},
		}
		p.ReceiveMessage(msg)
	}

	if len(p.activePool) != 4 {
		t.Fatalf("expected 4 active voices, got %d", len(p.activePool))
	}

	// First synthesize: all 4 voices run, active becomes false at end of synthesize
	p.Synthesize()

	// Second synthesize: all 4 should be deactivated and popped back to freePool without crashing
	p.Synthesize()

	if len(p.activePool) != 0 {
		t.Fatalf("expected 0 active voices, got %d", len(p.activePool))
	}

	if len(p.freePool) != 4 {
		t.Fatalf("expected 4 free voices, got %d", len(p.freePool))
	}
}

func TestPolyPartialDeactivation(t *testing.T) {
	voices := make([]*mockVoice, 4)
	for i := range voices {
		voices[i] = newMockVoice()
	}

	p := New(psalm.Stereo, voices)

	// Trigger voices with different durations (voices 0 and 2 duration 1; voices 1 and 3 duration 3)
	for i := 0; i < 4; i++ {
		dur := 1
		if i%2 == 1 {
			dur = 3
		}
		msg := psalm.Message{
			"command":   "trigger",
			"tick":      psalm.CtrlTick{},
			"duration":  float64(dur),
			"amplitude": 1.0,
			"message":   psalm.Message{},
		}
		p.ReceiveMessage(msg)
	}

	p.Synthesize() // voices 0 and 2 reach lifespan 0 and become inactive
	p.Synthesize() // voices 0 and 2 get removed; voices 1 and 3 remain active

	if len(p.activePool) != 2 {
		t.Fatalf("expected 2 active voices, got %d", len(p.activePool))
	}

	p.Synthesize() // voices 1 and 3 reach lifespan 0
	p.Synthesize() // all voices removed

	if len(p.activePool) != 0 {
		t.Fatalf("expected 0 active voices, got %d", len(p.activePool))
	}
}

func TestPolyVoiceStealing(t *testing.T) {
	voices := make([]*mockVoice, 2)
	for i := range voices {
		voices[i] = newMockVoice()
	}

	p := New(psalm.Stereo, voices)

	// Trigger 2 voices
	for i := 0; i < 2; i++ {
		msg := psalm.Message{
			"command":   "trigger",
			"tick":      psalm.CtrlTick{},
			"duration":  float64(10),
			"amplitude": 1.0,
			"message":   psalm.Message{},
		}
		p.ReceiveMessage(msg)
	}

	// Trigger 3rd voice (must steal oldest)
	msg := psalm.Message{
		"command":   "trigger",
		"tick":      psalm.CtrlTick{},
		"duration":  float64(10),
		"amplitude": 1.0,
		"message":   psalm.Message{},
	}
	p.ReceiveMessage(msg)

	if len(p.activePool) != 2 {
		t.Fatalf("expected 2 active voices, got %d", len(p.activePool))
	}

	p.Synthesize()
}
