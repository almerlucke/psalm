package score

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// triggers runs a sequencer for loops and returns the triggers per loop
func triggers(s *Sequencer, loops int) [][]psalm.Message {
	rec := newRecorder()
	rec.ValueIn(s)
	var out [][]psalm.Message
	n := s.length
	for l := range loops {
		rec.got = nil
		for t := int64(0); t < n; t++ {
			s.CtrlTick(psalm.CtrlTick{TickTimestamp: int64(l)*n + t})
		}
		out = append(out, rec.got)
	}
	return out
}

func beatOf(m psalm.Message, loop int, s *Sequencer) float64 {
	start := int64(0)
	if s != nil {
		start = int64(loop) * s.length
	}
	return float64(tickOf(m)-start) / float64(psalm.PPQ())
}

func msToBeats(ms float64) float64 { return ms * psalm.BPM() / 60000 }

func TestSwing(t *testing.T) {
	for _, tc := range []struct {
		swing, unit float64
		want        []float64
	}{
		{0.66, 0, []float64{0, 0.66, 1, 1.66}},        // 8ths: every second 8th late
		{0.75, 0.25, []float64{0, 0.375, 0.5, 0.875}}, // 16ths
		{0.5, 0, []float64{0, 0.5, 1, 1.5}},           // straight
	} {
		unit := tc.unit
		if unit == 0 {
			unit = 0.5
		}
		s := NewSequencer(Notes(unit, 60, 62, 64, 65), Config{Loop: true, Feel: Feel{Swing: tc.swing, SwingUnit: tc.unit}})
		got := triggers(s, 1)[0]
		for i, w := range tc.want {
			if b := beatOf(got[i], 0, s); math.Abs(b-w) > 1.0/float64(psalm.PPQ()) {
				t.Errorf("swing %v unit %v: note %d at beat %v, want %v", tc.swing, unit, i, b, w)
			}
		}
	}
}

func TestSwingStretchesLengths(t *testing.T) {
	// a swung pair: the first 8th lasts longer, the second shorter
	s := NewSequencer(Notes(0.5, 60, 62), Config{Legato: 1, Feel: Feel{Swing: 0.66}})
	got := triggers(s, 1)[0]
	beatSec := 60 / psalm.BPM()
	d1, d2 := got[0]["duration"].(float64)/beatSec, got[1]["duration"].(float64)/beatSec
	if math.Abs(d1-0.66) > 1e-6 || math.Abs(d2-0.34) > 1e-6 {
		t.Fatalf("expected lengths 0.66 and 0.34 beats, got %v and %v", d1, d2)
	}
}

func TestTimingHumanize(t *testing.T) {
	notes := Repeat(Join(Notes(1, 60), Chords(1, []float64{60, 64, 67})), 16)
	s := NewSequencer(notes, Config{Loop: true, Feel: Feel{Timing: 20, Seed: 7}})
	loops := triggers(s, 2)
	maxOff, moved := 0.0, 0
	for _, m := range loops[0] {
		b := beatOf(m, 0, s)
		off := b - math.Round(b)
		maxOff = math.Max(maxOff, math.Abs(off))
		if math.Abs(off) > 1.5/float64(psalm.PPQ()) {
			moved++
		}
	}
	if maxOff > msToBeats(20)+1.0/float64(psalm.PPQ()) || moved < len(loops[0])/2 {
		t.Fatalf("expected notes within ±20 ms, most of them moved: max %v beats, %d of %d moved", maxOff, moved, len(loops[0]))
	}
	// the notes of a chord move together
	for i := 0; i+3 < len(loops[0]); i++ {
		m := loops[0][i]
		if m["message"].(psalm.Message)["pitch"] == 64.0 {
			if tickOf(loops[0][i-1]) != tickOf(m) || tickOf(loops[0][i+1]) != tickOf(m) {
				t.Fatal("the notes of a chord should move together")
			}
		}
	}
	// every loop is played differently, and the same seed plays the same
	same := true
	for i := range loops[0] {
		if beatOf(loops[0][i], 0, s) != beatOf(loops[1][i], 1, s) {
			same = false
		}
	}
	if same {
		t.Fatal("expected the second loop to be played differently")
	}
	again := triggers(NewSequencer(notes, Config{Loop: true, Feel: Feel{Timing: 20, Seed: 7}}), 1)[0]
	for i := range again {
		if tickOf(again[i]) != tickOf(loops[0][i]) {
			t.Fatal("the same seed should give the same feel")
		}
	}
}

func TestStrumSpreadAndPush(t *testing.T) {
	ppqBeat := 1.0 / float64(psalm.PPQ())
	chord := []Note{Rest(1), Chord([]float64{67, 60, 64}, 4)} // not on the very first beat, which can't play early

	got := triggers(NewSequencer(chord, Config{Feel: Feel{Strum: 30}}), 1)[0]
	for k, want := range []float64{60, 64, 67} {
		m := got[k]
		if m["message"].(psalm.Message)["pitch"] != want || math.Abs(beatOf(m, 0, nil)-1-msToBeats(30)*float64(k)) > ppqBeat {
			t.Fatalf("strum: note %d is %v at %v", k, m["message"], beatOf(m, 0, nil))
		}
	}
	got = triggers(NewSequencer(chord, Config{Feel: Feel{Strum: -30}}), 1)[0]
	if got[0]["message"].(psalm.Message)["pitch"] != 67.0 {
		t.Fatal("a negative strum plays high to low")
	}

	got = triggers(NewSequencer(chord, Config{Feel: Feel{Spread: 15, Seed: 3}}), 1)[0]
	if tickOf(got[0]) == tickOf(got[1]) && tickOf(got[1]) == tickOf(got[2]) {
		t.Fatal("spread should start the notes of a chord at different moments")
	}

	got = triggers(NewSequencer(Notes(1, 0, 60, 0, 0), Config{Feel: Feel{Push: 25}}), 1)[0]
	if b := beatOf(got[0], 0, nil); math.Abs(b-1-msToBeats(25)) > ppqBeat {
		t.Fatalf("laid back: expected beat %v, got %v", 1+msToBeats(25), b)
	}
}

func TestVelocityAndAccents(t *testing.T) {
	notes := Notes(1, 60, 60, 60, 60, 60, 60, 60, 60)
	got := triggers(NewSequencer(notes, Config{Amp: 0.5, Feel: Feel{Accents: []float64{1, 0.6, 0.8, 0.6}}}), 1)[0]
	for i, want := range []float64{0.5, 0.3, 0.4, 0.3, 0.5, 0.3, 0.4, 0.3} {
		if a := got[i]["amplitude"].(float64); math.Abs(a-want) > 1e-9 {
			t.Fatalf("accent %d: amplitude %v, want %v", i, a, want)
		}
	}

	got = triggers(NewSequencer(Repeat(notes, 8), Config{Amp: 0.5, Velocity: 0.8, Feel: Feel{Velocity: 0.2, Seed: 5}}), 1)[0]
	lo, hi := math.Inf(1), 0.0
	for _, m := range got {
		a := m["amplitude"].(float64)
		v := m["message"].(psalm.Message)["velocity"].(float64)
		if math.Abs(a/0.5-v/0.8) > 1e-9 {
			t.Fatal("velocity and amplitude should vary together")
		}
		lo, hi = math.Min(lo, a), math.Max(hi, a)
	}
	if lo < 0.4-1e-9 || hi > 0.6+1e-9 || hi-lo < 0.05 {
		t.Fatalf("expected amplitudes varying within ±20%% of 0.5, got %v - %v", lo, hi)
	}
}

// Changing the feel during a tick, after the notes of that tick played, doesn't play them again
func TestSetFeelDoesntReplay(t *testing.T) {
	s := NewSequencer(Notes(1, 60, 62), Config{Loop: true})
	rec := newRecorder()
	rec.ValueIn(s)
	s.CtrlTick(psalm.CtrlTick{TickTimestamp: 0}) // plays the first note
	s.SetFeel(Feel{Swing: 0.66})                 // during the same tick
	for t := int64(1); t < int64(psalm.PPQ())*2; t++ {
		s.CtrlTick(psalm.CtrlTick{TickTimestamp: t})
	}
	if len(rec.got) != 2 {
		t.Fatalf("expected each note once, got %d triggers", len(rec.got))
	}
}
