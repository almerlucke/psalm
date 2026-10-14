package script

import (
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

// Presets are scripts to start from
var Presets = []string{"walk", "arpeggio", "function*", "values", "song"}

var codes = map[string]string{
	"walk": `// every trigger: a note of a random walk, in C minor
const minor = scale("C minor")
const walk = genny.rwalk(60, 48, 72, 0.15)
const lengths = genny.bucket([1, 1, 2, 0.5])

function trigger(ev) {
  note(minor.snap(walk.next()), 0.8, lengths.next())
}
`,
	"arpeggio": `// chords arpeggiated, a chord every 8 steps, a bass note under each
const chords = [["C3", "Eb3", "G3", "Bb3"], ["Ab2", "C3", "Eb3", "G3"],
                ["F2", "Ab2", "C3", "Eb3"], ["G2", "B2", "D3", "F3"]]
const arps = chords.map(c => genny.arpeggio(c.map(pitch), "up", "exclusive"))

function trigger(ev) {
  const i = Math.floor(ev.step / 8) % chords.length
  note(arps[i].next() + 12, 0.7, 0.5)
  if (ev.step % 8 === 0) note(pitch(chords[i][0]) - 12, 0.9, 6)
}
`,
	"function*": `// a phrase of a function*, twice, then an answer: over and over
function* phrase() {
  for (const p of ["C3", "D3", "Eb3", "G3"]) yield pitch(p)
  yield* [pitch("Bb3"), pitch("G3")]
}
function* answer() {
  yield* ["F3", "Eb3", "D3", "C3"].map(pitch)
}
const notes = genny.and(genny.repeat(genny.from(phrase), 2), genny.from(answer))

function trigger(ev) {
  note(notes.next(), ev.step % 4 === 0 ? 0.9 : 0.6, 0.8)
}
`,
	"song": `// follows the song (a song module, its bar output into trigger): a chord a bar, others in the chorus
const chords = {
  Verse: [["C3", "Eb3", "G3"], ["Ab2", "C3", "Eb3"], ["Bb2", "D3", "F3"], ["G2", "B2", "D3"]],
  Chorus: [["F3", "Ab3", "C4"], ["Eb3", "G3", "Bb3"]],
}

function trigger(ev) {
  const list = chords[song.section] || chords.Verse
  note(list[song.bar % list.length], song.section === "Chorus" ? 0.9 : 0.7, 1)
}

function section(name, index) {
  log("now:", name)
}
`,
	"values": `// to other modules: a pitch on a (to a notes' pitch), a trigger every 4th step, a wave on mod
const walk = genny.rwalk(0.5, 0, 1, 0.1)

function trigger(ev) {
  send("a", 48 + Math.round(walk.next() * 24))
  if (ev.step % 4 === 0) fire()
  mod(0.5 + 0.5 * Math.sin(ev.step / 16 * Math.PI))
}

function value(v, port) {
  log(port, v)
}
`,
}

// Help is what to know to write a script, shown above it in its side panel
const Help = "its top level runs when it starts: make its generators there\n" +
	"`function trigger(ev)` on every trigger (`ev.step`, `ev.seconds`); `function value(v, port)` for a value on `a` or `b`\n" +
	"`note(pitch, velocity, steps)` (a pitch: `48`, `\"C3\"`, a chord `[48, 51]`), `send(\"a\", v)`, `fire()`, `mod(0 to 1)`, `log(...)`\n" +
	"`pitch(\"C3\")`, `scale(\"C minor\").snap(p)`, `bpm()`, `input(\"a\")`, `shared` (every script of the patch sees it)\n" +
	"`genny.sequence`, `bucket`, `arpeggio`, `repeat`, `and`, `or`, `rand`, `rwalk`, `from(function* ...)`: `.next()`\n" +
	"with a song: `song.section`, `song.bar`, `song.bars`, `song.progress`, `song.sections`; `function section(name, index)` when one starts\n" +
	"Ctrl+Enter (or Run) runs it; an error (its line) or what it logs is below; a call may take 5 ms"

// Code is the code of a preset (the first, "walk", for one not known)
func Code(preset string) string {
	if c, ok := codes[preset]; ok {
		return c
	}
	return codes[Presets[0]]
}

// Kind is the kind of script modules of a patch: all of them in rt
func Kind(rt *Runtime) patcher.Kind {
	return patcher.Kind{Name: "script", Group: "Generators", Presets: Presets,
		Ins: []patcher.Port{{Name: "trigger", Type: patcher.Trigger}, {Name: "a", Type: patcher.Value, Index: 1}, {Name: "b", Type: patcher.Value, Index: 2}},
		Outs: []patcher.Port{{Name: "notes", Type: patcher.Notes}, {Name: "a", Type: patcher.Value}, {Name: "b", Type: patcher.Value, Index: 1},
			{Name: "trigger", Type: patcher.Trigger, Index: 2}, {Name: "mod", Type: patcher.Mod}},
		Make: func(preset string) (*patcher.Made, error) {
			s := New(rt)
			code := Code(preset)
			if err := s.SetCode(code); err != nil {
				return nil, err
			}
			return &patcher.Made{Module: s,
				Texts:   []patcher.Text{{Key: "code", Label: "script", Value: code, Lines: 12, Help: Help}},
				SetText: func(_, v string) error { return s.SetCode(v) },
				Notes:   s.SetTargets, Late: s.late, Status: s.Status,
				Mod: func(int) func() float64 { return s.Mod }}, nil
		}}
}
