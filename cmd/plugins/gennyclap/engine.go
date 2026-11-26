package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/spec"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/script"
)

//go:embed default.json
var defaultSpec string

//go:embed default.js
var defaultScript string

// the parameters of the plugin (their numbers are those of the host)
const (
	pSeed = iota
	pRegen
	pTranspose
	pGain
	pSwing
	pMicro
	pLatch
	pThru
	pKeys
	pSnap
	numParams
)

type paramInfo struct {
	name          string
	stepped       bool
	min, max, def float64
}

var params = [numParams]paramInfo{
	pSeed:      {"seed", true, 0, 999, 7},
	pRegen:     {"new each pass", true, 0, 1, 0},
	pTranspose: {"transpose", true, -24, 24, 0},
	pGain:      {"gain", false, 0, 2, 1},
	pSwing:     {"swing (steps)", false, 0, 0.5, 0},
	pMicro:     {"microtones", true, 0, 1, 1},
	pLatch:     {"latch", true, 0, 1, 0},
	pThru:      {"pass input", true, 0, 1, 0},
	pKeys:      {"key trigger", true, 0, 2, 0},
	pSnap:      {"snap", true, 0, 3, 0},
}

// ev is a note of a rendered line, times in beats
type ev struct {
	start, end float64
	pitch      float64
	vel        float64
	channel    int
}

// lineData is one render of the spec: the notes in order of their start, and their order of ending
type lineData struct {
	events  []ev
	byEnd   []int32
	length  float64 // beats
	overrun float64 // how far a note goes on after the end of the line, beats
	step    float64 // a step of the spec or script, beats
}

func buildLine(res spec.Result) *lineData {
	spb := res.File.StepsPerBeat
	var events []ev
	for _, t := range res.File.Tracks {
		for _, e := range t.Events {
			events = append(events, ev{e.Start / spb, (e.Start + e.Duration) / spb, e.Pitch, e.Velocity, t.Channel})
		}
	}
	return newLine(events, res.Steps/spb, 1/spb)
}

func buildPiece(p script.Piece) *lineData {
	var events []ev
	for _, n := range p.Notes {
		events = append(events, ev{n.Start / p.StepsPerBeat, (n.Start + n.Duration) / p.StepsPerBeat, n.Pitch, n.Velocity, n.Channel})
	}
	return newLine(events, p.Steps/p.StepsPerBeat, 1/p.StepsPerBeat)
}

// newLine makes the line of notes: a loop of length beats
func newLine(events []ev, length, step float64) *lineData {
	l := &lineData{events: events, length: max(length, step), step: step}
	slices.SortStableFunc(l.events, func(a, b ev) int {
		switch {
		case a.start < b.start:
			return -1
		case a.start > b.start:
			return 1
		}
		return 0
	})
	l.byEnd = make([]int32, len(l.events))
	for i := range l.byEnd {
		l.byEnd[i] = int32(i)
		l.overrun = max(l.overrun, l.events[i].end-l.length)
	}
	slices.SortStableFunc(l.byEnd, func(a, b int32) int {
		ea, eb := l.events[a].end, l.events[b].end
		switch {
		case ea < eb:
			return -1
		case ea > eb:
			return 1
		}
		return 0
	})
	return l
}

// store is what the audio thread reads: never changed once published
type store struct {
	lines  map[int64]*lineData // by cycle, the one of cycle 0 for all when the plugin does not renew
	renew  bool
	length float64
	// overrun is how far the notes of a cycle go on into the next one, beats
	overrun float64
	// epoch counts the times the whole piece was replaced (not just made longer): the notes that sound are let go then, since the
	// new piece does not know them
	epoch int64
}

// instance is one plugin in the host
type instance struct {
	param [numParams]atomic.Uint64 // the values (float bits)
	store atomic.Pointer[store]
	buf   [maxEvents]goEvent // the events of the block being made (the audio thread's)
	want  atomic.Int64       // the cycle the transport is in, as last asked for
	stop  chan struct{}
	done  chan struct{} // closed when the goroutine has ended

	// the notes played into the plugin (written by the audio thread, read by the goroutine that renders)
	held      [128]atomic.Int32 // the velocity (1 to 127) of each key that counts, 0 for none
	down      [128]atomic.Bool  // the keys that are physically down (a latched key can be held without being down)
	downs     atomic.Int32      // how many keys are down
	inVersion atomic.Uint64     // counts the changes of held

	// the key trigger: the piece made for each key (by the goroutine that renders), and the voices of the keys that are down (the
	// audio thread's)
	keys     atomic.Pointer[keySet]
	voices   [128]voice
	salt     int // the last voice number
	pending  [64]goEvent
	npending int

	mu        sync.Mutex // the spec text and its file (never taken by the audio thread)
	text      string
	isScript  bool // the text is a script (a file that ends in .js), else a spec
	path      string
	modTime   time.Time
	fromSave  bool // the spec is that of the saved project, the file counts when it changes
	specGen   uint64
	lastErr   string
	savedPath string   // the file the project was saved with
	handle    int      // of the instance, for the host log
	logLines  []string // what it said, the last ones (the file next to the spec has them)
}

// Severities of a message, as CLAP numbers them
const (
	sevInfo  = 1
	sevWarn  = 2
	sevError = 3
)

// logHook sends a message to the log of the host (set by the shim, nil in tests that do not want it)
var logHook func(handle, severity int, text string)

// say tells the user something: the log of the host, and a file next to the spec (spec.json.log) that has the last lines, since a
// plugin has no window yet where an error of a spec or a script could be shown. It is called from the goroutine of the instance
func (in *instance) say(severity int, format string, args ...any) {
	text := fmt.Sprintf(format, args...)
	if logHook != nil {
		logHook(in.handle, severity, "genny: "+text)
	}
	label := [...]string{sevInfo: "info", sevWarn: "warning", sevError: "error"}[severity]
	line := time.Now().Format("15:04:05") + " " + label + ": " + text
	in.mu.Lock()
	in.logLines = append(in.logLines, line)
	if len(in.logLines) > 200 {
		in.logLines = in.logLines[len(in.logLines)-200:]
	}
	path, body := in.path, strings.Join(in.logLines, "\n")+"\n"
	in.mu.Unlock()
	if path != "" {
		_ = os.WriteFile(path+".log", []byte(body), 0o644)
	}
}

var (
	instances [64]atomic.Pointer[instance]
	renderMu  sync.Mutex // genny has one stream of random numbers: one render at a time
)

func (in *instance) get(id int) float64 { return math.Float64frombits(in.param[id].Load()) }
func (in *instance) set(id int, v float64) {
	p := params[id]
	v = min(max(v, p.min), p.max)
	if p.stepped {
		v = math.Round(v)
	}
	in.param[id].Store(math.Float64bits(v))
}

func newInstance() *instance {
	in := &instance{}
	for i, p := range params {
		in.set(i, p.def)
	}
	return in
}

// specPath is the file the piece is read from: $GENNY_SPEC, else ~/.config/psalm/genny.js when there is one (a
// script), else ~/.config/psalm/genny.json (a spec)
func specPath() string {
	if p := os.Getenv("GENNY_SPEC"); p != "" {
		return p
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	if js := filepath.Join(dir, "psalm", "genny.js"); fileExists(js) {
		return js
	}
	return filepath.Join(dir, "psalm", "genny.json")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isScriptFile is whether a file is a script, by its name
func isScriptFile(path string) bool { return strings.EqualFold(filepath.Ext(path), ".js") }

// defaultFor is the text a new file starts with
func defaultFor(path string) string {
	if isScriptFile(path) {
		return defaultScript
	}
	return defaultSpec
}

// loadFile reads the spec file (writing the default one when there is none). It reports whether the text changed
func (in *instance) loadFile() bool {
	in.mu.Lock()
	defer in.mu.Unlock()
	// the file is looked for each time: a genny.js that appears (or goes) changes which file is played
	if p := specPath(); p != in.path {
		// a project keeps its own copy of the file it was saved with; another file is not a copy of that one
		if in.path != "" || (in.savedPath != "" && in.savedPath != p) {
			in.fromSave = false
		}
		in.path, in.modTime = p, time.Time{}
	}
	if in.path == "" {
		if in.text == "" {
			in.text, in.specGen = defaultSpec, in.specGen+1
			return true
		}
		return false
	}
	fi, err := os.Stat(in.path)
	if err != nil {
		if in.text != "" {
			return false
		}
		def := defaultFor(in.path)
		_ = os.MkdirAll(filepath.Dir(in.path), 0o755)
		_ = os.WriteFile(in.path, []byte(def), 0o644)
		fi, err = os.Stat(in.path)
		if err != nil {
			in.text, in.isScript, in.specGen = def, isScriptFile(in.path), in.specGen+1
			return true
		}
	}
	if !fi.ModTime().After(in.modTime) {
		return false
	}
	in.modTime = fi.ModTime()
	if in.fromSave {
		// the project has its own copy of the spec: the file only counts once it is edited
		in.fromSave = false
		return false
	}
	data, err := os.ReadFile(in.path)
	if err != nil || (string(data) == in.text && isScriptFile(in.path) == in.isScript) {
		return false
	}
	in.text, in.isScript, in.specGen = string(data), isScriptFile(in.path), in.specGen+1
	return true
}

// renderSpec renders the spec for a seed
func renderSpec(text string, seed uint64) (*lineData, error) {
	s, err := spec.ParseBytes([]byte(text))
	if err != nil {
		return nil, err
	}
	renderMu.Lock()
	defer renderMu.Unlock()
	genny.Seed(seed)
	res, err := s.Render(&seed)
	if err != nil {
		return nil, err
	}
	return buildLine(res), nil
}

// renderScript renders the script for a seed, in the runtime of the instance (which only its goroutine uses)
func (in *instance) renderScript(rt *script.Runtime, text string, seed uint64, held []script.HeldNote) (*lineData, error) {
	renderMu.Lock()
	piece, err := rt.RenderHeld(text, seed, held)
	renderMu.Unlock()
	for _, line := range piece.Log {
		in.say(sevInfo, "script: %s", line)
	}
	if err != nil {
		return nil, err
	}
	return buildPiece(piece), nil
}

// failure is what a render that failed depended on
type failure struct {
	gen   uint64
	seed  float64
	renew bool
	cycle int64
	input uint64 // the notes held, when the script uses them
}

// input takes a note played into the plugin (on the audio thread, without locks or allocation). With latch on, the notes stay held
// when they are let go, until the next note is played with nothing down: a new chord replaces the old one
func (in *instance) input(key int, velocity float64, on bool) {
	if key < 0 || key > 127 {
		return
	}
	if on {
		if in.get(pLatch) > 0.5 && in.downs.Load() == 0 {
			for k := range in.held {
				in.held[k].Store(0)
			}
		}
		if !in.down[key].Swap(true) {
			in.downs.Add(1)
		}
		in.held[key].Store(int32(min(max(math.Round(velocity*127), 1), 127)))
	} else {
		if in.down[key].Swap(false) {
			in.downs.Add(-1)
		}
		if in.get(pLatch) < 0.5 {
			in.held[key].Store(0)
		}
	}
	in.inVersion.Add(1)
}

// unlatch lets go of the notes that were only held by the latch (it was turned off)
func (in *instance) unlatch() {
	for k := range in.held {
		if !in.down[k].Load() {
			in.held[k].Store(0)
		}
	}
	in.inVersion.Add(1)
}

// heldNotes are the notes held, lowest first
func (in *instance) heldNotes() []script.HeldNote {
	var out []script.HeldNote
	for k := range in.held {
		if v := in.held[k].Load(); v > 0 {
			out = append(out, script.HeldNote{Pitch: k, Velocity: float64(v) / 127})
		}
	}
	return out
}

// usesInput is whether a script looks at the notes played (it is not rendered again for them when it does not)
func usesInput(text string) bool {
	return strings.Contains(text, "held") || strings.Contains(text, "pitches") || chordArg.MatchString(text)
}

// chordArg is p.chord (not genny.chord, which makes a chord)
var chordArg = regexp.MustCompile(`\bp\.chord\b`)

// run keeps the store up to date: the spec file, the seed, and the cycles ahead of the transport
func (in *instance) run() {
	if in.done != nil {
		defer close(in.done)
	}
	var (
		have    *store
		haveGen uint64 = math.MaxUint64
		haveSd         = -1.0
		tick    int
		rt      *script.Runtime // made when the first script is rendered (here, the only goroutine that uses it)
		failed  failure         // the last render that failed (the zero value is none: no spec has generation 0)
		haveVer uint64          // the notes held that the piece was made for
		lastVer uint64          // the notes held at the last look
		settle  int             // ticks to wait for more notes of a chord
		latched bool
		keysGen uint64 = math.MaxUint64 // what the pieces of the keys were made for
		keysSd         = -1.0
	)
	for {
		select {
		case <-in.stop:
			return
		case <-time.After(15 * time.Millisecond):
		}
		if tick++; tick%30 == 1 {
			in.loadFile()
			if tick == 1 {
				if line := editorLine(); line != "" {
					in.say(sevInfo, "%s", line)
				}
			}
		}
		in.mu.Lock()
		text, gen, isScript := in.text, in.specGen, in.isScript
		in.mu.Unlock()
		if text == "" {
			continue
		}
		seed, renew := in.get(pSeed), in.get(pRegen) > 0.5
		if in.get(pKeys) > 0.5 {
			// the key trigger: the piece of each key instead of the one that loops
			if kf := (failure{gen, seed, false, -2, 0}); (gen != keysGen || seed != keysSd) && kf != failed {
				switch err := in.renderKeys(&rt, text, isScript, uint64(seed)); {
				case err == nil:
					keysGen, keysSd = gen, seed
					in.mu.Lock()
					in.lastErr = ""
					in.mu.Unlock()
				case errors.Is(err, errStopped):
					return
				default:
					failed = kf
					in.mu.Lock()
					fresh := in.lastErr != err.Error()
					in.lastErr = err.Error()
					in.mu.Unlock()
					if fresh {
						in.say(sevError, "%v (nothing plays until it is fixed)", err)
					}
				}
			}
			continue
		}
		keysGen = math.MaxUint64 // made again when the trigger is turned on again
		cycle := in.want.Load()
		// the notes played: a script that looks at them is made again when they change, once a chord has settled
		now := in.get(pLatch) > 0.5
		if latched && !now {
			in.unlatch()
		}
		latched = now
		withInput := isScript && usesInput(text)
		ver := in.inVersion.Load()
		if ver != lastVer {
			lastVer, settle = ver, 3
			if withInput {
				in.say(sevInfo, "input changed: %d notes held", len(in.heldNotes()))
			}
		}
		if settle > 0 {
			settle--
			if withInput {
				continue
			}
		}
		if !withInput {
			ver = haveVer // not looked at
		}
		// what failed is not tried again until something it depends on changes (the cycle only when each pass has its own seed)
		key := failure{gen, seed, renew, 0, 0}
		if renew {
			key.cycle = cycle
		}
		if withInput {
			key.input = ver
		}
		if key == failed {
			continue
		}
		rebuilt := gen != haveGen || seed != haveSd || have == nil || have.renew != renew
		remade := rebuilt || ver != haveVer // for the notes as well
		var lines map[int64]*lineData
		if remade {
			lines = map[int64]*lineData{}
		} else {
			lines = maps(have.lines)
		}
		first, last := cycle, cycle+2
		if !renew {
			first, last = 0, 0
		}
		changed := remade
		var held []script.HeldNote
		if withInput {
			held = in.heldNotes()
		}
		for c := first; c <= last; c++ {
			if lines[c] != nil {
				continue
			}
			var l *lineData
			var err error
			if isScript {
				if rt == nil {
					rt = script.NewRuntime()
				}
				l, err = in.renderScript(rt, text, uint64(seed)+uint64(c), held)
			} else {
				l, err = renderSpec(text, uint64(seed)+uint64(c))
			}
			if err != nil {
				in.mu.Lock()
				fresh := in.lastErr != err.Error()
				in.lastErr = err.Error()
				in.mu.Unlock()
				if fresh {
					if have != nil {
						in.say(sevError, "%v (the last good version goes on)", err)
					} else {
						in.say(sevError, "%v (nothing plays until it is fixed)", err)
					}
				}
				failed = key
				break
			}
			in.mu.Lock()
			wasErr := in.lastErr != ""
			in.lastErr = ""
			in.mu.Unlock()
			if rebuilt || wasErr || remade {
				in.say(sevInfo, "%d notes in %.4g beats, seed %d, %d held", len(l.events), l.length, int(seed)+int(c), len(held))
			}
			lines[c], changed = l, true
		}
		for c := range lines {
			if renew && c < cycle-1 { // without renew the one piece is cycle 0 for ever
				delete(lines, c)
				changed = true
			}
		}
		if !changed || len(lines) == 0 {
			continue
		}
		var length, overrun float64
		for _, l := range lines {
			length = l.length
			overrun = max(overrun, l.overrun)
		}
		epoch := int64(1)
		if have != nil {
			epoch = have.epoch
			if remade {
				epoch++
			}
		}
		have = &store{lines: lines, renew: renew, length: length, overrun: overrun, epoch: epoch}
		haveGen, haveSd = gen, seed
		if withInput {
			haveVer = ver
		}
		in.store.Store(have)
	}
}

func maps(m map[int64]*lineData) map[int64]*lineData {
	out := make(map[int64]*lineData, len(m)+2)
	for k, v := range m {
		out[k] = v
	}
	return out
}

// collect writes the note on and off events in the beats [from, to), sorted by time. It does not allocate. The piece that plays with
// the transport (the store), or the voices of the keys that were played (with the key trigger on)
func (in *instance) collect(from, to float64) []goEvent {
	out := in.buf[:]
	n := 0
	put := func(e goEvent) {
		if n < len(out) {
			out[n] = e
			n++
		}
	}
	if in.get(pKeys) < 0.5 {
		in.endVoices(from, put)
		if st := in.store.Load(); st != nil {
			in.want.Store(int64(math.Floor(from / st.length)))
			in.emit(st, 0, from, to, 0, false, 0, 1, put)
		}
	} else {
		in.collectVoices(from, to, put)
	}
	slices.SortFunc(out[:n], func(a, b goEvent) int {
		switch {
		case a.beat < b.beat:
			return -1
		case a.beat > b.beat:
			return 1
		case !a.on && b.on: // an off (or the end of a voice) before an on at the same time
			return -1
		case a.on && !b.on:
			return 1
		}
		return 0
	})
	return out[:n]
}

// emit puts the events of a store in the beats [from, to) (counted from offset, the events go out at offset + their time) in put. A
// voice is a store of one line that plays once or for ever, with its own id for the notes, shifted by a number of semitones and
// with the velocity of its key
func (in *instance) emit(st *store, offset, from, to float64, voice int, once bool, shift, velocity float64, put func(goEvent)) {
	loop := st.length
	transpose, gain, swing := in.get(pTranspose)+shift, in.get(pGain)*velocity, in.get(pSwing)
	micro := in.get(pMicro) > 0.5
	for c := int64(math.Floor((from - st.overrun - 1.0) / loop)); c <= int64(math.Floor(to/loop)); c++ {
		key := c
		if !st.renew {
			key = 0
		}
		l := st.lines[key]
		if l == nil || c < 0 || (once && c > 0) {
			continue
		}
		base := float64(c) * loop
		emit := func(i int, on bool) {
			e := l.events[i]
			start, end := e.start, e.end
			if swing > 0 {
				// the steps are 1/4 beat in a spec of 4 steps a beat: a note that starts on an odd step is late
				// by the swing, the step is read from the start of the note
				start, end = swungTimes(e.start, e.end, swing, l.step)
			}
			exact := e.pitch + transpose
			p := math.Round(exact)
			if p < 0 || p > 127 {
				return
			}
			id := int(((c*2654435+int64(i))*2 + 1 + int64(voice)*7919003) & 0x7fffffff)
			ce := goEvent{key: int(p), channel: e.channel, id: id, voice: voice}
			if on {
				ce.beat, ce.on, ce.velocity = base+start, true, min(max(e.vel*gain, 0.01), 1)
				if micro {
					ce.tune = exact - p // a pitch between two keys: the key and how far from it, in semitones
				}
			} else {
				ce.beat = base + end
			}
			if ce.beat >= from && ce.beat < to {
				ce.beat += offset
				put(ce)
			}
		}
		// the notes that start in the range (the swing moves a start by at most 0.5 step)
		lo := sortSearch(len(l.events), func(i int) bool { return base+l.events[i].start >= from-1.0 })
		for i := lo; i < len(l.events) && base+l.events[i].start < to; i++ {
			emit(i, true)
		}
		lo = sortSearch(len(l.byEnd), func(i int) bool { return base+l.events[l.byEnd[i]].end >= from-1.0 })
		for i := lo; i < len(l.byEnd) && base+l.events[l.byEnd[i]].end < to+1.0; i++ {
			emit(int(l.byEnd[i]), false)
		}
	}
}

// goEvent is a note on or off for the shim, at a position in beats
type goEvent struct {
	beat, velocity float64
	tune           float64 // semitones from the key, a note on only
	key, channel   int
	id             int
	on             bool
	kill           bool // not a note: let go of the notes of a voice (its voice number) that still sound
	voice          int  // the voice that made the note, 0 for the piece of the transport
}

// maxEvents is the most events one block hands the shim
const maxEvents = 512

// swungTimes delays a note that starts on an odd step (times in beats, a step is step beats long) by swing steps
func swungTimes(start, end, swing, step float64) (float64, float64) {
	n := start / step
	if math.Abs(n-math.Round(n)) < 1e-6 && int64(math.Round(n))%2 == 1 {
		d := swing * step
		return start + d, end + d
	}
	return start, end
}

func sortSearch(n int, f func(int) bool) int {
	lo, hi := 0, n
	for lo < hi {
		m := int(uint(lo+hi) >> 1)
		if !f(m) {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo
}

// state is what is saved in the project
type stateDoc struct {
	Params map[string]float64 `json:"params"`
	Spec   string             `json:"spec"` // the text of the spec or script
	Script bool               `json:"script,omitempty"`
	Path   string             `json:"path,omitempty"` // the file it was read from
}

func (in *instance) saveState() []byte {
	doc := stateDoc{Params: map[string]float64{}}
	for i, p := range params {
		doc.Params[p.name] = in.get(i)
	}
	in.mu.Lock()
	doc.Spec, doc.Script, doc.Path = in.text, in.isScript, in.path
	in.mu.Unlock()
	b, _ := json.Marshal(doc)
	return b
}

func (in *instance) loadState(data []byte) {
	var doc stateDoc
	if json.Unmarshal(data, &doc) != nil {
		return
	}
	for i, p := range params {
		if v, ok := doc.Params[p.name]; ok {
			in.set(i, v)
		}
	}
	if doc.Spec != "" {
		in.mu.Lock()
		in.text, in.isScript, in.specGen, in.fromSave, in.savedPath = doc.Spec, doc.Script, in.specGen+1, true, doc.Path
		in.mu.Unlock()
	}
}
