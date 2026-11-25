package script

import (
	"strings"
	"testing"

	"github.com/almerlucke/genny"
)

func TestFormsFugue(t *testing.T) {
	genny.Seed(3)
	got, err := eval(t, `
		const key = genny.scale(62, "minor")
		const f = genny.fugue({degrees: [0, 2, 4, 2, 0, -1, 0], durations: [4, 2, 2, 4, 2, 1, 1]}, {scale: key, voices: 3})
		const bars = f.sections().map(s => s.name)
		const shift = f.episode({voices: [0, 1], repeats: 2})
		f.middle(0, "answer", 4, true)
		f.close({bars: 2})
		const v = f.voices()
		return {voices: v.length, notes: v.every(x => x.length > 0), end: f.end() > 40, sections: bars, last: f.sections().map(s => s.name).pop(), warning: f.warning || null, entries: f.entries().length >= 3}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"voices":3`, `"notes":true`, `"end":true`, `"last":"close"`, `"entries":true`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s: want %s", got, want)
		}
	}
}

func TestFormsFugueErrors(t *testing.T) {
	for _, c := range []struct{ code, want string }{
		{`return genny.fugue({degrees: [0, 1], durations: [4, 4]}, {})`, "the key"},
		{`return genny.fugue({durations: [4]}, {scale: genny.scale(62, "minor")})`, "a subject is"},
		{`return genny.fugue({degrees: [0], durations: [4]}, {scale: genny.scale(62, "minor"), answer: "sideways"})`, `no "sideways"`},
		{`return genny.harmony.diatonic("H", 1)`, ""},
		{`return genny.harmony.diatonic("C sharp", 1)`, "the mode"},
		{`return genny.harmony.realize([1, 2])`, "chords are those"},
		{`return genny.polyrhythm.layer({pulses: 3, span: "1/0"})`, "fraction"},
		{`return genny.tuning({})`, "edo"},
	} {
		_, err := eval(t, c.code)
		if err == nil {
			t.Errorf("%s: no error", c.code)
		} else if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %q, want %q in it", c.code, err, c.want)
		}
	}
}

func TestFormsHarmony(t *testing.T) {
	genny.Seed(5)
	got, err := eval(t, `
		const H = genny.harmony
		const I = H.diatonic("C", 0), IV = H.diatonic("C", 3), V = H.diatonic("C", 4, true)
		const r = H.realize([I, IV, V, I])
		const gen = H.generator({key: "A minor", phraseLength: 4})
		const phrase = gen.phrase(4, "authentic")
		const voices = H.voices(r, {length: 8})
		return {names: [I.name, IV.name, V.name], seventh: V.seventh, voicings: r.voicings.length,
			four: r.voicings.every(v => v.notes.length === 4), top: r.voicings.every(v => v.notes[0] > v.notes[3]),
			phrase: phrase.length, first: phrase[0].numeral, last: phrase[3].numeral,
			lines: voices.length, bass: voices[3].length, bassStart: voices[3][2].start,
			dom: H.secondaryDominant("C", 5).numeral, none: H.borrowed("C", 1), modulate: H.modulate("C", "G").length > 0}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"names":["C","F","G7"]`, `"seventh":true`, `"voicings":4`, `"four":true`, `"top":true`, `"phrase":4`, `"first":"i"`,
		`"lines":4`, `"bass":4`, `"bassStart":16`, `"modulate":true`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s: want %s", got, want)
		}
	}
}

func TestFormsPolyrhythm(t *testing.T) {
	got, err := eval(t, `
		const P = genny.polyrhythm
		const triplet = P.layer({pulses: 3, span: 2, pitch: 60})
		const e = triplet.events(0, 2, 12)
		const three = P.layer({tuplet: [3, 2], period: 1})
		const four = P.layer({every: "1/2"})
		const grid = P.grid([three, four])
		const meter = P.meter("1/2", [3, 3, 2])
		const m = P.ratio(3, 2)
		return {starts: e.map(x => x.start), cycle: P.cycle([three, P.layer({every: 1})]), period: triplet.period,
			steps: grid.steps, unit: grid.unit, onsets: meter.onsets, length: meter.length, sig: meter.signature,
			tempo: m.tempo(120), carried: m.carryEvents([{pitch: 60, start: 6, duration: 3}])[0].start,
			velocity: e[0].velocity > 0 && e[0].velocity <= 1, third: P.layer({pulses: 1, span: 1}).events("1/3", 2, 6).map(x => x.start)}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"starts":[0,8,16]`, `"cycle":2`, `"period":0.6666666666666666`, `"steps":12`, `"onsets":[0,0.5,1,1.5,2,2.5,3,3.5]`, `"length":4`, `"sig":[8,8]`,
		`"tempo":180`, `"velocity":true`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s: want %s", got, want)
		}
	}
}

func TestFormsTuning(t *testing.T) {
	got, err := eval(t, `
		const t = genny.tuning({edo: 31, root: 60})
		const j = genny.tuning({just: [1, 9/8, 5/4, 3/2], root: 60})
		const c = genny.tuning({cents: [0, 200, 400], period: 1200, root: 60})
		return {len: t.len(), step: Math.round(t.pitch(1) * 1000) / 1000, octave: t.note(0, 1), snap: Math.round(t.snap(60.2) * 1000) / 1000,
			justLen: j.len(), fifth: Math.round(j.pitch(3) * 1000) / 1000, c: c.pitch(2), near: t.nearest(61), inRange: t.pitches(60, 61).length}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"len":31`, `"step":60.387`, `"octave":72`, `"snap":60.387`, `"justLen":4`, `"fifth":67.02`, `"c":64`, `"inRange":`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s: want %s", got, want)
		}
	}
}
