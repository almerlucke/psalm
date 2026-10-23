package patcher

import (
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

var sharpNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

// pitchName writes a pitch as a note name ("C4" is 60), or as a number when it is between the keys
func pitchName(p float64) string {
	if p != math.Round(p) || p < 0 || p > 127 {
		return strconv.FormatFloat(p, 'f', -1, 64)
	}
	k := int(p)
	return fmt.Sprintf("%q", sharpNames[k%12]+strconv.Itoa(k/12-1))
}

// scriptDivisions are the clocks a script can play a loop on, a step of 60, 30 or 15 ticks (a sixteenth, a thirty-second, a sixty-fourth)
var scriptDivisions = []struct {
	name  string
	ticks int64
}{{"1/16", 60}, {"1/32", 30}, {"1/64", 15}}

// Script writes a layer as a script of the patcher: a table of the notes by step, played on a clock of the finest division the notes need (a sixteenth
// for notes on the grid), so on a script module with the notes cable to a synth it plays what was played, on the beat. The steps are those of the
// loop, so a script that begins with the song stays with it
func (l *Looper) Script(layer int) string {
	if layer < 0 || layer >= len(l.layers) {
		return ""
	}
	return loopScript([][]loopNote{l.layers[layer]}, []int{layer + 1}, l.Bars, l.Config().PPQ)
}

// ScriptAll writes all the layers in one script: they play together, through one notes output
func (l *Looper) ScriptAll() string {
	if len(l.layers) == 0 {
		return ""
	}
	numbers := make([]int, len(l.layers))
	for i := range numbers {
		numbers[i] = i + 1
	}
	return loopScript(l.layers, numbers, l.Bars, l.Config().PPQ)
}

func loopScript(layers [][]loopNote, numbers []int, bars, ppq int) string {
	scale := float64(ppq) / 240 // the tick values above are of the sixteenth at 240 to the quarter
	div := scriptDivisions[len(scriptDivisions)-1]
	for _, d := range scriptDivisions {
		step := int64(math.Round(float64(d.ticks) * scale))
		if step > 0 && !slices.ContainsFunc(layers, func(notes []loopNote) bool {
			return slices.ContainsFunc(notes, func(n loopNote) bool { return n.start%step != 0 })
		}) {
			div = d
			break
		}
	}
	step := float64(max(1, int64(math.Round(float64(div.ticks)*scale))))
	loopSteps := int(math.Round(float64(bars*4*ppq) / step))
	stepsBar := loopSteps / max(1, bars)
	type ev struct {
		at      int
		lengthF float64
		note    loopNote
	}
	var b strings.Builder
	what := fmt.Sprintf("Layer %d", numbers[0])
	if len(layers) > 1 {
		what = fmt.Sprintf("All %d layers", len(layers))
	}
	fmt.Fprintf(&b, "// %s of a loop of %d bars, from the looper: the notes as [step, length, pitch, velocity], the steps %s notes long and the lengths in steps.\n", what, bars, div.name)
	b.WriteString("// Put it in a script module and draw its notes cable to a synth: it has its own clock and plays the notes on the beat, over and over.\n")
	fmt.Fprintf(&b, "clock(%q)\n\nconst LOOP = %d // steps in the loop\nconst notes = [\n", div.name, loopSteps)
	for k, notes := range layers {
		evs := make([]ev, len(notes))
		for i, n := range notes {
			evs[i] = ev{at: int(math.Round(float64(n.start) / step)), lengthF: math.Round(float64(n.length)/step*1000) / 1000, note: n}
		}
		slices.SortStableFunc(evs, func(a, b ev) int { return a.at - b.at })
		bar := -1
		for _, e := range evs {
			if nb := e.at / max(1, stepsBar); nb != bar {
				bar = nb
				if len(layers) > 1 {
					fmt.Fprintf(&b, "  // layer %d, bar %d\n", numbers[k], bar+1)
				} else {
					fmt.Fprintf(&b, "  // bar %d\n", bar+1)
				}
			}
			fmt.Fprintf(&b, "  [%d, %s, %s, %s],\n", e.at, strconv.FormatFloat(e.lengthF, 'f', -1, 64), pitchName(e.note.pitch),
				strconv.FormatFloat(math.Round(e.note.velocity*100)/100, 'f', -1, 64))
		}
	}
	b.WriteString(`]

const at = {}
for (const n of notes) {
  if (!at[n[0]]) at[n[0]] = []
  at[n[0]].push(n)
}

function trigger(ev) {
  const here = at[ev.step % LOOP]
  if (!here) return
  for (const n of here) note(pitch(n[2]), n[3], n[1])
}
`)
	return b.String()
}

// exportScripts writes a script for every layer to the folder `recordings`, loop-<time>-layer<n>.js, or one for all of them, loop-<time>.js (from a goroutine of its own)
func (l *Looper) exportScripts() {
	if len(l.layers) == 0 {
		l.say("nothing to export: no layers yet")
		return
	}
	stamp := time.Now().Format("20060102-150405")
	files := map[string]string{}
	if l.ScriptFiles == 1 {
		files[filepath.Join(l.dir, fmt.Sprintf("loop-%s.js", stamp))] = l.ScriptAll()
	} else {
		for i := range l.layers {
			files[filepath.Join(l.dir, fmt.Sprintf("loop-%s-layer%d.js", stamp, i+1))] = l.Script(i)
		}
	}
	say, dir := l.say, l.dir
	go func() {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			say("error: can't make %s: %v", dir, err)
			return
		}
		for _, path := range slices.Sorted(maps.Keys(files)) {
			if err := os.WriteFile(path, []byte(files[path]), 0o644); err != nil {
				say("error: can't write %s: %v", path, err)
				return
			}
			say("saved %s", path)
		}
	}()
}
