package script

import "testing"

func TestGennyCompose(t *testing.T) {
	for _, c := range []struct{ name, code, want string }{
		{"messiaen", `return [genny.messiaen(60, 1).degree(3), genny.scale(60, "messiaen2").len(), genny.transpositions("messiaen1"), genny.transpositions("major")]`, `[66,8,2,12]`},
		{"messiaen bad", `return genny.messiaen(60, 9)`, ``},
		{"sclScale", `const s = genny.sclScale(60, "just5"); return [s.degree(0), s.degree(7) > 66.9 && s.degree(7) < 67.1]`, `[60,true]`},
		{"sclScale bad", `return genny.sclScale(60, "nonsense")`, ``},
		{"line transpose", `return genny.line.transpose([{pitch: 60, start: 0}], 7)`, `[{"pitch":67,"velocity":0.8,"start":0,"duration":1}]`},
		{"line retrograde", `return genny.line.retrograde([{pitch: 60, start: 0}, {pitch: 62, start: 1}]).map(e => e.pitch)`, `[62,60]`},
		{"line merge", `return genny.line.merge([{pitch: 60, start: 2}], [{pitch: 62, start: 0}]).map(e => e.pitch)`, `[62,60]`},
		{"line stretch loop", `return [genny.line.stretch([{pitch: 60, start: 1, duration: 1}], 2)[0].start, genny.line.loop([{pitch: 60, start: 0}], 3, 4).map(e => e.start)]`, `[2,[0,4,8]]`},
		{"line invert", `return genny.line.invert([{pitch: 64, start: 0}], 60).map(e => e.pitch)`, `[56]`},
		{"line invert scale", `return genny.line.invert([{pitch: 64, start: 0}], 60, genny.scale(60, "major")).map(e => e.pitch)`, `[57]`},
		{"line canon", `return genny.line.canon([{pitch: 60, start: 0}], 4, 7, 3).map(l => [l[0].start, l[0].pitch])`, `[[0,60],[4,67],[8,74]]`},
		{"line quantize", `return genny.line.quantize([{pitch: 61, start: 0}], genny.scale(60, "major")).map(e => e.pitch)`, `[60]`},
		{"line thin", `return genny.line.thin([{pitch: 1, start: 0}, {pitch: 2, start: 1}], [true, false]).map(e => e.pitch)`, `[1]`},
		{"line slice shift", `return genny.line.shift(genny.line.slice([{pitch: 1, start: 0}, {pitch: 2, start: 4}], 2, 8), 1).map(e => e.start)`, `[3]`},
		{"counterpoint", `const r = genny.counterpoint([62, 65, 64, 62, 67, 65, 69, 67, 65, 64, 62]); return [r.line.length, genny.checkCounterpoint([62, 65, 64, 62, 67, 65, 69, 67, 65, 64, 62], r.line).length]`, `[11,0]`},
		{"counterpoint violations", `return genny.checkCounterpoint([60, 62], [67, 69]).length > 0`, `true`},
		{"toLine", `return genny.toLine([60, 62], 2).map(e => e.start)`, `[0,2]`},
		{"chordTones", `const k = genny.scale(60, "major"); return genny.chordTones([[60, 64, 67], [65, 69, 72]], 2, { scale: k }).length > 0`, `true`},
		{"chordTones no scale", `return genny.chordTones([[60, 64, 67]], 1, {})`, ``},
		{"isorhythm", `return [genny.isorhythm([1, 2, 1], [60, 62], 12).length > 0, genny.isorhythmPeriod([1, 2, 1], [60, 62])]`, `[true,6]`},
		{"isorhythmSteps", `const s = genny.isorhythmSteps([2, 1], [60, 62]); return [s.pitch.take(4), s.length.take(4)]`, `[[60,null,62,60],[2,1,1,2]]`},
		{"phase", `const p = genny.phase([{pitch: 60, start: 0}, {pitch: 62, start: 1}, {pitch: 64, start: 2}], 4, 0.5); return [genny.phaseLength([{pitch: 60, start: 0}, {pitch: 62, start: 1}]), genny.phaseCycles([{pitch: 60, start: 0}, {pitch: 62, start: 1}], 0.5), p.first.length > 0, p.second.length > 0]`, `[2,4,true,true]`},
		{"poisson", `return [genny.poisson(2, 20).every((x, i, a) => i === 0 || x >= a[i - 1]), typeof genny.poissonInt(3)]`, `[true,"number"]`},
		{"density", `return [genny.density([1]).take(3), genny.density([0]).take(3)]`, `[[true,true,true],[false,false,false]]`},
		{"cloud", `const e = genny.cloud({ density: 2, pitch: [60, 72] }, 8); return [e.length > 0, e.every(x => x.pitch >= 60 && x.pitch < 72)]`, `[true,true]`},
		{"cloud scale", `return genny.cloud({ density: step => 1, pitch: genny.rand(60, 72), scale: genny.scale(60, "majorPentatonic") }, 4).length >= 0`, `true`},
		{"cloud no pitch", `return genny.cloud({ density: 1 }, 4)`, ``},
		{"freeStochastic", `return genny.freeStochastic(16, 1, 1, 66, 5).length > 0`, `true`},
		{"life", `const l = genny.life(5, 5, "B3/S23", 0, { pattern: "blinker", x: 1, y: 2 }); const a = l.row(2); l.step(); return [a, l.row(2), l.population(), l.generation()]`, `[[false,true,true,true,false],[false,false,true,false,false],3,1]`},
		{"life gates", `const l = genny.life(4, 4, "B3/S23", 0.5); return [l.rowGate(0).take(4).length, l.density().next() >= 0, l.pitches(0, genny.scale(60, "major"), [0, 2, 4]).take(4).length]`, `[4,true,4]`},
		{"life bad", `return genny.life(4, 4, "x")`, ``},
		{"life pattern bad", `return genny.life(4, 4, "B3/S23", 0, { pattern: "zebra" })`, ``},
		{"evolve", `const e = genny.evolve({ length: 6, min: 60, max: 72, fitness: [["smoothness", 1]], mutation: 0.3 }); const b = e.run(30); return [b.length, e.player.take(6).every(x => x >= 60 && x <= 72), e.generation() >= 30]`, `[6,true,true]`},
		{"evolve function", `const e = genny.evolve({ length: 4, min: 0, max: 9, fitness: p => p.reduce((a, b) => a + b, 0) }); return e.run(40).reduce((a, b) => a + b, 0) > 20`, `true`},
		{"evolve bad", `return genny.evolve({ fitness: [["nonsense"]] })`, ``},
		{"learnStyle", `const ev = [60, 62, 64, 65, 67, 65, 64, 62].map((p, i) => ({ pitch: p, start: i, duration: 1 })); const s = genny.learnStyle(ev, 1); return [s.phrase(4).length, s.fill(4).length > 0, s.events(3, 0, 0.8).length, s.transpose(2).next().pitch >= 0]`, `[4,true,3,true]`},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := eval(t, c.code)
			if c.want == "" {
				if err == nil {
					t.Fatalf("no error, got %s", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}
