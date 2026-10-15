package poly

import (
	"slices"
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

// monoVoice records what a voice is told: how often it starts, the pitch it is at, let go or not
type recVoice struct {
	*psalm.Module
	starts int
	pitch  float64
	held   bool
}

func newRecVoice() *recVoice {
	v := &recVoice{Module: psalm.NewModule(nil, []int{psalm.Stereo})}
	v.SetSelf(v)
	return v
}

func (v *recVoice) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("pitch", func(f float64) { v.pitch = f })
}
func (v *recVoice) NoteOn(_ float64, msg psalm.Message, _ psalm.CtrlTick, _ psalm.Config) {
	v.starts++
	v.held = true
	v.ReceiveMessage(msg)
}
func (v *recVoice) Note(_ float64, _ float64, msg psalm.Message, _ psalm.CtrlTick, _ psalm.Config) {
	v.starts++
	v.ReceiveMessage(msg)
}
func (v *recVoice) NoteOff()       { v.held = false }
func (v *recVoice) Clear()         {}
func (v *recVoice) IsActive() bool { return v.held }
func (v *recVoice) Synthesize()    {}

func on(p *Poly[*recVoice], id string, pitch float64) {
	p.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "noteOn": id, "amplitude": 1.0,
		"message": psalm.Message{"pitch": pitch}})
}

func off(p *Poly[*recVoice], id string) {
	p.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "noteOff": id})
}

func TestMonoLegato(t *testing.T) {
	voices := []*recVoice{newRecVoice(), newRecVoice(), newRecVoice()}
	p := New(psalm.Stereo, voices)
	p.ReceiveMessage(psalm.Message{"mono": 1.0, "legato": 1.0})
	on(p, "a", 60)
	on(p, "b", 64) // held over a: slides, no new start
	v := p.monoVoice.voice
	if v.starts != 1 || v.pitch != 64 || len(p.activePool) != 1 {
		t.Fatalf("legato: %d starts, pitch %g, %d voices", v.starts, v.pitch, len(p.activePool))
	}
	on(p, "c", 67)
	off(p, "c") // back to b, still held
	if v.pitch != 64 || !v.held {
		t.Errorf("let go of the top note: pitch %g, held %v, want back to 64", v.pitch, v.held)
	}
	off(p, "a") // not the sounding one: nothing changes
	if v.pitch != 64 || !v.held {
		t.Errorf("let go of a note under it: pitch %g", v.pitch)
	}
	off(p, "b")
	if v.held {
		t.Errorf("all let go, the voice still holds")
	}
	// a new note after all were let go starts the voice again
	on(p, "d", 62)
	if v.starts != 2 {
		t.Errorf("a new phrase: %d starts, want 2", v.starts)
	}
}

func TestMonoRetrigger(t *testing.T) {
	voices := []*recVoice{newRecVoice(), newRecVoice()}
	p := New(psalm.Stereo, voices).SetMono(true, false)
	on(p, "a", 60)
	on(p, "b", 64) // not legato: starts again, on the same voice
	v := p.monoVoice.voice
	if v.starts != 2 || v.pitch != 64 || voices[0].starts+voices[1].starts != 2 {
		t.Errorf("mono: %d starts on the voice, pitch %g", v.starts, v.pitch)
	}
	off(p, "b") // the held note starts again
	if v.starts != 3 || v.pitch != 60 {
		t.Errorf("back to the held note: %d starts, pitch %g", v.starts, v.pitch)
	}
	// polyphonic again
	p.SetMono(false, false)
	on(p, "x", 60)
	on(p, "y", 64)
	if len(p.activePool) != 2 {
		t.Errorf("poly again: %d voices", len(p.activePool))
	}
}

func TestMonoDurations(t *testing.T) {
	voices := []*recVoice{newRecVoice()}
	p := New(psalm.Stereo, voices).SetMono(true, true)
	sr := p.Config().SampleRate
	n := float64(p.Config().BufferSize)
	trigger := func(pitch, seconds float64) {
		p.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "duration": seconds, "amplitude": 1.0,
			"message": psalm.Message{"pitch": pitch}})
	}
	trigger(60, 3*n/sr) // three buffers
	p.Synthesize()
	trigger(62, 3*n/sr) // overlapping: slides
	v := voices[0]
	if v.starts != 1 || v.pitch != 62 {
		t.Errorf("overlapping notes of a duration: %d starts, pitch %g", v.starts, v.pitch)
	}
	for range 4 {
		p.Synthesize()
	}
	if v.held {
		t.Errorf("the notes of a duration never let go")
	}
}

func TestNoteMessage(t *testing.T) {
	voices := []*recVoice{newRecVoice(), newRecVoice(), newRecVoice()}
	p := New(psalm.Stereo, voices)
	on(p, "a", 60)
	on(p, "b", 64)
	p.ReceiveMessage(psalm.Message{"command": "note", "id": "b", "message": psalm.Message{"pitch": 99.0}})
	var pitches []float64
	for _, v := range voices {
		pitches = append(pitches, v.pitch)
	}
	if !slices.Contains(pitches, 99) || !slices.Contains(pitches, 60) {
		t.Errorf("the message for b reached the voices as %v", pitches)
	}
	// in mono only the note sounding takes it
	p.ReceiveMessage(psalm.Message{"mono": 1.0, "legato": 1.0})
	on(p, "c", 60)
	on(p, "d", 62)
	p.ReceiveMessage(psalm.Message{"command": "note", "id": "c", "message": psalm.Message{"pitch": 1.0}})
	if v := p.monoVoice.voice; v.pitch != 62 {
		t.Errorf("a note held under the one sounding moved it to %g", v.pitch)
	}
	p.ReceiveMessage(psalm.Message{"command": "note", "id": "d", "message": psalm.Message{"pitch": 63.0}})
	if v := p.monoVoice.voice; v.pitch != 63 {
		t.Errorf("the note sounding didn't take its message: %g", v.pitch)
	}
}

func trigger(p *Poly[*mockVoice], id string, duration float64) {
	msg := psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "amplitude": 1.0, "message": psalm.Message{}}
	if id != "" {
		msg["noteOn"] = id
	} else {
		msg["duration"] = duration
	}
	p.ReceiveMessage(msg)
}

func mockPoly(n int) (*Poly[*mockVoice], []*mockVoice) {
	vs := make([]*mockVoice, n)
	for i := range vs {
		vs[i] = newMockVoice()
	}
	return New(psalm.Stereo, vs), vs
}

// the voices of another Poly replace its own: the notes sounding are cut (with a fade, no click),
// the new voices play, the other is left without
func TestAdoptVoices(t *testing.T) {
	p, old := mockPoly(3)
	trigger(p, "a", 0)
	trigger(p, "b", 0)
	other, fresh := mockPoly(5)
	if !p.AdoptVoices(other) {
		t.Fatal("a Poly of the same voices should be adopted")
	}
	if len(p.activePool) != 0 || len(p.freePool) != 5 || len(other.freePool)+len(other.activePool) != 0 {
		t.Fatalf("pools after: %d active, %d free (other %d)", len(p.activePool), len(p.freePool), len(other.freePool))
	}
	for _, v := range old {
		if slices.Contains(fresh, v) {
			t.Fatal("an old voice is still there")
		}
	}
	if !p.hasFadeOut {
		t.Error("the notes that sounded should fade out, not stop")
	}
	p.Synthesize() // plays the fade
	if p.hasFadeOut {
		t.Error("the fade should be played once")
	}
	trigger(p, "c", 0)
	if len(p.activePool) != 1 || !slices.ContainsFunc(fresh, func(v *mockVoice) bool { return v.IsActive() }) {
		t.Error("the adopted voices should play")
	}
	p.CallVoices(func(v *mockVoice) {
		if slices.Contains(old, v) {
			t.Error("a message reaches an old voice")
		}
	})
	// a Poly of another kind of voice, or of another number of channels, is not taken
	if p.AdoptVoices(New[*mockVoice](psalm.Mono, []*mockVoice{newMockVoice()})) {
		t.Error("another number of channels should not be adopted")
	}
	if p.AdoptVoices(p) {
		t.Error("itself should not be adopted")
	}
}

// mono and legato stay as they were, the notes held are let go, and the one voice is one of the new
func TestAdoptVoicesMono(t *testing.T) {
	p, old := mockPoly(3)
	p.SetMono(true, true)
	trigger(p, "a", 0)
	trigger(p, "b", 0)
	if len(p.held) != 2 {
		t.Fatalf("held %d", len(p.held))
	}
	other, fresh := mockPoly(2)
	if !p.AdoptVoices(other) {
		t.Fatal("not adopted")
	}
	if m, l := p.Mono(); !m || !l {
		t.Errorf("mono %v legato %v, want both kept", m, l)
	}
	if len(p.held) != 0 || len(p.timers) != 0 {
		t.Errorf("held %d, timers %d", len(p.held), len(p.timers))
	}
	if p.monoVoice == nil || !slices.Contains(fresh, p.monoVoice.voice) || slices.Contains(old, p.monoVoice.voice) {
		t.Fatal("the mono voice should be one of the adopted")
	}
	trigger(p, "c", 0)
	if !p.monoVoice.voice.IsActive() || len(p.held) != 1 {
		t.Error("it should play mono on the new voice")
	}
	// not mono, it stays so
	q, _ := mockPoly(2)
	r, _ := mockPoly(2)
	q.AdoptVoices(r)
	if m, _ := q.Mono(); m || q.monoVoice != nil {
		t.Error("a polyphonic synth should stay so")
	}
}

// the swap makes the synth off the calling goroutine, and the last one asked for wins
func TestSwap(t *testing.T) {
	p, _ := mockPoly(2)
	built := map[string]*Poly[*mockVoice]{}
	sw := NewSwap(func(f func()) { f() }, p, func(name string) psalm.Modular {
		m, _ := mockPoly(3 + len(name))
		built[name] = m
		return m
	})
	if !sw.Can() {
		t.Fatal("a Poly takes voices")
	}
	sw.Load("ab")
	sw.Wait()
	if n := len(p.freePool); n != 5 {
		t.Errorf("%d voices after the swap, want 5", n)
	}
	// something that is not an Adopter does nothing
	plain := NewSwap(nil, newMockVoice(), func(string) psalm.Modular { t.Error("built"); return nil })
	if plain.Can() {
		t.Error("a voice takes no voices")
	}
	plain.Load("x")
	plain.Wait()
}
