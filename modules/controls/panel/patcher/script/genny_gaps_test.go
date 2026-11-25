package script

import (
	"strings"
	"testing"

	"github.com/almerlucke/genny"
)

func TestGapsLists(t *testing.T) {
	genny.Seed(11)
	for _, c := range []struct{ name, code, want string }{
		{"group", `return genny.group(genny.sequence([1, 2]), 2).take(6)`, `[1,1,2,2,1,1]`},
		{"group of a generator", `return genny.group(genny.sequence(["a", "b"]), genny.sequence([1, 3])).take(6)`, `["a","b","b","b","a","b"]`},
		{"grow geometric", `return genny.grow(1, {amount: 2, lo: 1, hi: 8}).take(6)`, `[1,2,4,8,1,2]`},
		{"grow arithmetic clip", `return genny.grow(1, {mode: "arithmetic", amount: 1, lo: 1, hi: 3, limit: "clip"}).take(5)`, `[1,2,3,3,3]`},
		{"plusMin none", `return genny.plusMin(genny.sequence([1, 2]), 0).take(4)`, `[1,2,1,2]`},
		{"plusMin all", `return genny.plusMin(genny.sequence([1, 2]), 1).take(4)`, `[-1,-2,-1,-2]`},
		{"positiveNegative", `return genny.positiveNegative(genny.sequence([1, 2, 3]), 2, 1).take(6)`, `[1,2,-3,1,2,-3]`},
		{"gatherUntil", `return genny.gatherUntil(genny.sequence([3, 3, 3]), 8)`, `[3,3,2]`},
		{"splitEvenly", `return genny.splitEvenly(8, 4)`, `[2,2,2,2]`},
		{"permutations", `return genny.permutations([1, 2, 3], 3, 4)`, `[[1,2,3],[1,3,2],[2,1,3],[2,3,1]]`},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := eval(t, c.code)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("%s: %s, want %s", c.code, got, c.want)
			}
		})
	}
}

func TestGapsRowsAndMeter(t *testing.T) {
	got, err := eval(t, `
		const rows = genny.allIntervalRows(12, 3)
		const steps = rows[0].slice(1).map((p, i) => ((p - rows[0][i]) % 12 + 12) % 12)
		return {count: rows.length, first: rows[0][0], allIntervals: new Set(steps).size, ind: genny.pulseHierarchy([2, 2]),
			vel: genny.pulseVelocity([2, 2], 0.4, 1).map(v => Math.round(v * 100) / 100), thin: genny.pulseThin([2, 2], 2),
			morph: genny.pulseMorph([2, 2], [4], 0.5).length, weight: [0, 4, 2, 8].map(s => genny.metricWeight(s, 16, 4))}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"count":3`, `"first":0`, `"allIntervals":11`, `"ind":[`, `"morph":4`, `"weight":[3,2,1,2]`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s: want %s", got, want)
		}
	}
}

func TestGapsPitch(t *testing.T) {
	got, err := eval(t, `
		const key = genny.scale(60, "major")
		return {hz: Math.round(genny.midiToHz(69)), midi: Math.round(genny.hzToMidi(261.6256) * 100) / 100, degree: key.degreeOf(64.2), back: key.degree(key.degreeOf(67))}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"hz":440`, `"midi":60`, `"degree":2`, `"back":67`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s: want %s", got, want)
		}
	}
	if _, err := eval(t, `return genny.hzToMidi(0)`); err == nil {
		t.Error("hzToMidi(0) gave no error")
	}
}

func TestGapsSieve(t *testing.T) {
	got, err := eval(t, `
		const s = genny.sieve(3, 0, 0, 11)
		const picked = s.select(["a", "b", "c", "d", "e", "f", "g"])
		const f = s.filter(genny.sequence([1, 2, 3, 4, 5, 6, 7]), 7).take(7)
		return {picked, f, interpret: s.interpret([1, 1, 1, 1, 1, 1], 2)}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"picked":["a","d","g"]`, `"f":[1,null,null,4,null,null,7]`, `"interpret":[3,3]`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s: want %s", got, want)
		}
	}
}

func TestGapsFugueAndHarmony(t *testing.T) {
	genny.Seed(3)
	got, err := eval(t, `
		const key = genny.scale(62, "minor")
		const subject = {degrees: [0, 2, 4, 2, 0, -1, 0], durations: [4, 2, 2, 4, 2, 1, 1]}
		const ans = genny.fugueAnswer(subject, {scale: key})
		const cs = genny.countersubject(subject, {scale: key})
		const f = genny.fugue(subject, {scale: key, voices: 2})
		const H = genny.harmony
		const r = H.realize([H.diatonic("C", 0), H.diatonic("C", 3), H.diatonic("C", 4), H.diatonic("C", 0)])
		const tonic = H.dominant("C")
		return {answer: ans.degrees.length === subject.degrees.length && ans.degrees[0] !== subject.degrees[0], cs: cs.degrees.length > 0 && cs.length > 0,
			bar: f.bar(0) === 1 && f.bar(20) === 2, dom: tonic.name, rel: H.relative("A minor").name, check: H.check(r).length === r.violations.length,
			chain: H.relative(H.dominant("D minor")).minor}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"answer":true`, `"cs":true`, `"bar":true`, `"dom":"G major"`, `"rel":"C major"`, `"check":true`, `"chain":false`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s: want %s", got, want)
		}
	}
}

func TestGapsPolyrhythm(t *testing.T) {
	got, err := eval(t, `
		const P = genny.polyrhythm
		const accented = P.layer({pulses: 8, span: 4, euclidAccent: [8, 3, 0, 0.2]}).events(0, 4)
		const up = P.ratio(3, 2), down = up.inverse()
		const both = up.then(down)
		return {accented: accented.length, velocities: new Set(accented.map(e => e.velocity)).size > 1, carry: up.carry(3), inverse: Math.round(down.factor() * 1000) / 1000,
			identity: Math.round(both.factor() * 1000) / 1000}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"accented":8`, `"velocities":true`, `"carry":4.5`, `"inverse":0.667`, `"identity":1`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s: want %s", got, want)
		}
	}
}

func TestGapsErrors(t *testing.T) {
	for _, c := range []struct{ code, want string }{
		{`return genny.grow(1, {lo: 5, hi: 2})`, "lo is above hi"},
		{`return genny.grow(1, {mode: "sideways"})`, `no "sideways"`},
		{`return genny.positiveNegative(genny.sequence([1]), 0, 0)`, "not both 0"},
		{`return genny.allIntervalRows(13)`, "n from 1 to 12"},
		{`return genny.permutations([1, 2, 3, 4, 5, 6, 7, 8, 9])`, "at most 8"},
		{`return genny.pulseMorph([2, 2], [3])`, "same"},
		{`return genny.countersubject({degrees: [], durations: []}, {scale: genny.scale(62, "minor")})`, "a subject is"},
		{`return genny.sieve(3, 0, 0, 11).filter(genny.sequence([1]), 0)`, "length"},
		{`return genny.polyrhythm.ratio(3, 2).then({})`, "another modulation"},
	} {
		_, err := eval(t, c.code)
		if err == nil {
			t.Errorf("%s: no error", c.code)
		} else if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %q, want %q in it", c.code, err, c.want)
		}
	}
}
