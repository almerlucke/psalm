package patcher

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/chordmodel"
)

// --- the song of a patch ---

// Song is the song of a patch: a form of sections and its transport (arrange), made by a song
// module, followed by the sequences, the clocks and the scripts of the patch (SongOf)
type Song struct {
	tr    atomic.Pointer[arrange.Transport] // nil: no song
	now   atomic.Pointer[SongNow]           // where it is, for the page
	gates gateList                          // its parts
}

// SongNow is where a song is, as the page shows it
type SongNow struct {
	Section  string  `json:"section"`
	Bar      int     `json:"bar"`  // in the section, from 0
	Bars     int     `json:"bars"` // of the section
	Progress float64 `json:"progress"`
	Seconds  float64 `json:"seconds"` // since the start of the song (of this pass), at the tempo it plays at
	Length   float64 `json:"length"`  // of one pass of the song, the sections that are on, at that tempo
	Queued   string  `json:"queued,omitempty"`
	Ended    bool    `json:"ended,omitempty"`
}

var songs sync.Map // *psalm.Env -> *Song

// SongOf returns the song of the patch of an environment (a module's: m.Env())
func SongOf(env *psalm.Env) *Song {
	v, _ := songs.LoadOrStore(env, &Song{})
	return v.(*Song)
}

// At returns where the song is at a tick (on the audio thread; every module asking at a tick gets
// the same position); false: the patch has no song
func (s *Song) At(tick psalm.CtrlTick, ppq int) (arrange.Position, bool) {
	tr := s.tr.Load()
	if tr == nil {
		return arrange.Position{}, false
	}
	return tr.At(tick.TickTimestamp, ppq), true
}

// Clock is the time of the song as a line for the top of the page: "1:23 / 4:00", the time in the song and the length of a pass; "" for no song
func (s *Song) Clock() string {
	now := s.Now()
	if now == nil {
		return ""
	}
	mmss := func(sec float64) string {
		t := int(math.Max(0, sec))
		return fmt.Sprintf("%d:%02d", t/60, t%60)
	}
	if now.Ended {
		return mmss(now.Length) + " / " + mmss(now.Length)
	}
	return mmss(now.Seconds) + " / " + mmss(now.Length)
}

// Transport is the transport of the song, nil for none
func (s *Song) Transport() *arrange.Transport { return s.tr.Load() }

// Now is where the song was at its last tick, nil for no song (from any goroutine)
func (s *Song) Now() *SongNow {
	if s.tr.Load() == nil {
		return nil
	}
	return s.now.Load()
}

// sectionKey tells one passing of a section from another: a section starts again on a repeat, a
// loop, a jump back
func sectionKey(pos arrange.Position) [2]int { return [2]int{pos.Loop, pos.Section.Index} }

// --- sections typed ---

// ParseSections reads the sections of a song: a name and a number of bars each, by commas or lines
// ("Intro 8, Verse 16, Chorus 8"); a name may have spaces ("Middle eight 8")
func ParseSections(text string) ([]arrange.Section, error) {
	secs, _, err := ParseSong(text)
	return secs, err
}

// meter is the bar of a song typed among its sections: "3/4" (the beats of a bar, quarter notes)
var meter = regexp.MustCompile(`^(\d+)/4$`)

// ParseSong reads the sections of a song and the beats of its bar, 4 unless a part of the text says "3/4" (or 2/4, 5/4, 6/4...): "3/4, Day 1 12, Day 2 12"
func ParseSong(text string) ([]arrange.Section, float64, error) {
	var out []arrange.Section
	beats := 4.0
	for _, part := range strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == '\n' || r == ';' }) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if m := meter.FindStringSubmatch(part); m != nil {
			n, _ := strconv.Atoi(m[1])
			if n < 1 || n > 16 {
				return nil, 0, fmt.Errorf("%q: between 1 and 16 beats to the bar", part)
			}
			beats = float64(n)
			continue
		}
		i := strings.LastIndexAny(part, " \t")
		if i < 0 {
			return nil, 0, fmt.Errorf("%q: a name and a number of bars (Verse 16)", part)
		}
		name := strings.TrimSpace(part[:i])
		bars, err := strconv.Atoi(strings.TrimSpace(part[i+1:]))
		if err != nil || bars <= 0 {
			return nil, 0, fmt.Errorf("%q: a name and a number of bars (Verse 16)", part)
		}
		if slices.ContainsFunc(out, func(s arrange.Section) bool { return s.Name == name }) {
			return nil, 0, fmt.Errorf("%q twice: every section a name of its own", name)
		}
		out = append(out, arrange.Section{Name: name, Bars: bars})
	}
	if len(out) == 0 {
		return nil, 0, fmt.Errorf("no sections: a name and a number of bars each (Intro 8, Verse 16)")
	}
	return out, beats, nil
}

// sectionLabel is a line of a text per section: "Verse: ..."
var sectionLabel = regexp.MustCompile(`^\s*([A-Za-z][\w ]*?)\s*:(.*)$`)

// PerSection splits a text into what it says for every section ("Verse: C3 Eb3" a line) and what
// it says for the others (the lines without a section)
func PerSection(text string) (each map[string]string, rest string) {
	var others []string
	for _, line := range strings.Split(text, "\n") {
		if m := sectionLabel.FindStringSubmatch(line); m != nil {
			if each == nil {
				each = map[string]string{}
			}
			each[m[1]] += m[2] + "\n"
			continue
		}
		others = append(others, line)
	}
	return each, strings.Join(others, "\n")
}

// --- the song module ---

// SongForm is the song of a patch: its sections (typed: "Intro 8, Verse 16, Chorus 8") played by
// a transport, which follows the clock of the patch (a restart starts the song again). Outputs: a
// trigger on every bar and at the start of every section (a note of a step lasts a bar), the
// section's number (a value) and how far it is through it (mod). Its settings: the sections to play
// next and those that are off (tiles), repeat, when a section asked for starts, loop
type SongForm struct {
	*psalm.Module
	song  *Song
	mu    sync.Mutex // building a form
	text  string
	loop  bool
	harm  *Harmony           // the harmony inside a song made by the patcher (nil: none)
	model *chordmodel.Module // the chord model inside it
	tr    *arrange.Transport // the one it made
	last  [2]int             // the section passing, the bar: sent once
	lastB int
	begun bool
}

// NewSongForm returns a song of an intro, a verse, a chorus and an outro, looping
func NewSongForm() *SongForm {
	s := &SongForm{Module: psalm.NewControl(), loop: true}
	s.SetSelf(s)
	s.song = SongOf(s.Env())
	_ = s.SetSections(DefaultSections)
	return s
}

// DefaultSections are the sections of a new song
const DefaultSections = "Intro 4, Verse 8, Chorus 8, Outro 4"

// SetSections makes the song of sections typed (from any goroutine); the transport keeps what it
// can of the one before: the repeat, the jump, the sections switched off
func (s *SongForm) SetSections(text string) error {
	secs, beats, err := ParseSong(text)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.text = text
	s.build(secs, beats)
	return nil
}

// SetLoop makes the song start again after its last section, or end
func (s *SongForm) SetLoop(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if on == s.loop {
		return
	}
	s.loop = on
	if secs, beats, err := ParseSong(s.text); err == nil {
		s.build(secs, beats)
	}
}

// build makes a form and its transport (with the lock held) and makes it the patch's song
func (s *SongForm) build(secs []arrange.Section, beats float64) {
	f := arrange.NewForm(secs...)
	f.BeatsPerBar = beats
	f.Loop = s.loop
	tr := f.Transport()
	if old := s.tr; old != nil {
		tr.SetRepeat(old.Repeat()).SetJump(old.Jump())
		for _, name := range old.Sections() {
			if !old.Enabled(name) {
				if _, ok := f.Section(name); ok {
					tr.SetEnabled(name, false)
				}
			}
		}
		s.song.tr.CompareAndSwap(old, tr)
	}
	if s.song.tr.Load() == nil {
		s.song.tr.Store(tr)
	}
	s.tr = tr
}

// close lets the patch go without its song (the module taken away)
func (s *SongForm) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.song.tr.CompareAndSwap(s.tr, nil)
}

// CtrlTick moves the song on and sends its bars and sections
func (s *SongForm) CtrlTick(tick psalm.CtrlTick) {
	s.mu.Lock()
	tr := s.tr
	s.mu.Unlock()
	if tr == nil || s.song.tr.Load() != tr { // another song module has the patch's song
		return
	}
	ppq := s.Config().PPQ
	pos := tr.At(tick.TickTimestamp, ppq)
	sec := pos.Section
	form := tr.Form()
	bars := 0
	for _, o := range tr.Order() {
		bars += o.Bars
	}
	secPerBeat := 60 / s.BPM()
	now := &SongNow{Section: sec.Name, Bar: pos.SectionBar(), Bars: sec.Bars, Progress: pos.Progress, Queued: tr.Queued(), Ended: pos.Ended,
		Seconds: pos.Beat * secPerBeat, Length: float64(bars) * form.BeatsPerBar * secPerBeat}
	if pos.Ended {
		now.Section = ""
	}
	s.song.now.Store(now)
	if pos.Ended {
		return
	}
	barSeconds := tr.Form().BeatsPerBar * 60 / s.BPM()
	key := sectionKey(pos)
	if !s.begun || key != s.last {
		s.begun, s.last, s.lastB = true, key, pos.Bar
		s.Send(float64(sec.Index), 2)
		s.Send(Event{Tick: tick, Step: int64(sec.Index), Seconds: barSeconds}, 1)
		s.Send(Event{Tick: tick, Step: int64(pos.SectionBar()), Seconds: barSeconds}, 0)
		return
	}
	if pos.Bar != s.lastB {
		s.lastB = pos.Bar
		s.Send(Event{Tick: tick, Step: int64(pos.SectionBar()), Seconds: barSeconds}, 0)
	}
}

// Restart starts the song again with the patch (Psalm.Restart): its first bar is sent again
func (s *SongForm) Restart() { s.begun = false }

// progress is how far the song is through its section (mod), 0-1
func (s *SongForm) progress() float64 {
	if now := s.song.Now(); now != nil && !now.Ended {
		return math.Max(0, math.Min(now.Progress, 1))
	}
	return 0
}

// songCues shows the transport of the song a module made as a cue list (whatever transport it
// has now: typed again, the sections change)
type songCues struct{ s *SongForm }

func (c songCues) transport() *arrange.Transport {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()
	return c.s.tr
}

func (c songCues) CueNames() []string {
	if tr := c.transport(); tr != nil {
		return tr.Sections()
	}
	return nil
}

func (c songCues) CueState() panel.CueState {
	st := panel.CueState{Playing: -1, Queued: -1}
	tr := c.transport()
	if tr == nil {
		return st
	}
	names := tr.Sections()
	st.Names = names
	playing, progress := tr.Playing()
	st.Playing, st.Progress = slices.Index(names, playing), progress
	st.Queued = slices.Index(names, tr.Queued())
	for _, n := range names {
		st.Enabled = append(st.Enabled, tr.Enabled(n))
	}
	return st
}

func (c songCues) QueueCue(i int) {
	tr := c.transport()
	if tr == nil {
		return
	}
	if names := tr.Sections(); i >= 0 && i < len(names) {
		tr.Queue(names[i])
	} else {
		tr.Queue("")
	}
}

func (c songCues) EnableCue(i int, on bool) {
	if tr := c.transport(); tr != nil {
		if names := tr.Sections(); i >= 0 && i < len(names) {
			tr.SetEnabled(names[i], on)
		}
	}
}

// songSettings are the settings of a song module kept with a patch: repeat, jump, loop
type songSettings struct {
	repeat, jump, loop, key, scale, low *panel.Param
}

func (st songSettings) params() map[string]*panel.Param {
	all := map[string]*panel.Param{"repeat": st.repeat, "jump": st.jump, "loop": st.loop, "key": st.key, "scale": st.scale, "low": st.low}
	for k, p := range all {
		if p == nil {
			delete(all, k)
		}
	}
	return all
}

func (st songSettings) Values() map[string]float64 {
	out := map[string]float64{}
	for k, p := range st.params() {
		out[k] = p.Value()
	}
	return out
}

func (st songSettings) Apply(values map[string]float64) {
	for key, p := range st.params() {
		if v, ok := values[key]; ok {
			p.Set(v)
		}
	}
}

// SongHelp is what to know to type the sections of a song
const SongHelp = "a name and a number of bars a section, by commas or lines: `Intro 4, Verse 8, Chorus 8, Outro 4`; `3/4` among them makes the bar three beats (four unless it says)\n" +
	"its tiles below: a tap plays a section next, its dot switches it off (skipped); repeat plays the one playing again\n" +
	"outputs: a trigger every bar and at every section (a note of a step lasts a bar), the section's number, how far through it (mod)\n" +
	"a sequence, a clock's pattern can be per section: a line `Verse: C3 Eb3` each; a script asks `song.section`\n" +
	"its parts below the sections: the gates of the patch (sound, notes, triggers through in some sections), a tap switches one; a lane is a curve per section\n" +
	"Ctrl+Enter (or Run) takes them; a restart starts the song again"

// SongChordsHelp is what to know to type the chords of a song: the harmony inside it
const SongChordsHelp = ChordsHelp + "\nthe chord model below draws chords for a section (Draw chords in the grid of the sections, or the buttons of its window) and writes them here"

// SongKind is the kind of song modules: the song of the patch (one at a time: the last made), with its harmony (the chords per section, which the parts, melodists and rolls follow)
// and the chord model that draws them inside it. Without chords it has no harmony; the chords the others follow are the first ones typed
func SongKind() Kind {
	trig := func(name string, i int) Port { return Port{Name: name, Type: Trigger, Index: i} }
	return Kind{Name: "song", Group: "Song",
		Outs: []Port{trig("bar", 0), trig("section", 1), {Name: "number", Type: Value, Index: 2}, {Name: "progress", Type: Mod}, {Name: "hear chords", Type: Notes, Index: 4}},
		Make: func(string) (*Made, error) {
			s := NewSongForm()
			h, m := newHarmony(), newChordModel()
			s.harm, s.model = h, m
			var st songSettings
			return &Made{Module: s,
				Texts: []Text{{Key: "sections", Label: "sections", Value: DefaultSections, Help: SongHelp}, {Key: "chords", Label: "chords", Value: "", Help: SongChordsHelp}},
				SetText: func(k, v string) error {
					if k != "chords" {
						return s.SetSections(v)
					}
					if strings.TrimSpace(v) == "" {
						h.release()
						h.clear()
						return nil
					}
					if err := h.SetChords(v); err != nil {
						return err
					}
					h.claim()
					return nil
				},
				Added: func() []psalm.Modular { return []psalm.Modular{h, m} },
				Notes: func(targets []psalm.Receiver) { m.SetTargets(targets...) },
				Mod:   func(int) func() float64 { return s.progress },
				Close: func() { s.close(); h.release() },
				// the chord model is the state of a song (a patch keeps it, and the settings of the editor with it: a module with a state does not have them set again)
				State: func() (string, error) {
					b, err := json.Marshal(map[string]any{"model": m.StateText(), "settings": st.Values()})
					return string(b), err
				},
				SetState: func(text string) error {
					var v struct {
						Model    string             `json:"model"`
						Settings map[string]float64 `json:"settings"`
					}
					if err := json.Unmarshal([]byte(text), &v); err != nil {
						return err
					}
					st.Apply(v.Settings)
					return m.SetStateText(v.Model)
				},
				Edit: func(page *panel.Panel, name, _ string) Settings {
					page.Section("Song")
					page.Cues(name+" sections", songCues{s}, panel.CueSlots(32))
					page.Cues(name+" parts", partCues{s.song}, panel.CueSlots(32), panel.CueToggles()) // its gates: a tap switches one
					st = songSettings{
						repeat: page.Toggle(name+" repeat", panel.Func(func(v float64) {
							if tr := (songCues{s}).transport(); tr != nil {
								tr.SetRepeat(v >= 0.5)
							}
						}), panel.Label("Repeat")),
						jump: page.Select(name+" jump", []string{"section end", "next bar"}, func(i int) {
							if tr := (songCues{s}).transport(); tr != nil {
								tr.SetJump(arrange.JumpAt(i))
							}
						}, panel.Label("Jump at")),
						loop: page.Toggle(name+" loop", panel.Func(func(v float64) { s.SetLoop(v >= 0.5) }), panel.Label("Loop"), panel.Default(1)),
					}
					page.Section("Harmony")
					st.key = page.Select(name+" key", Keys, func(i int) { h.ReceiveMessage(psalm.Message{"key": i}) }, panel.Label("Key (of the numerals)"))
					st.scale = page.Select(name+" scale", Scales, func(i int) { h.ReceiveMessage(psalm.Message{"scale": i}) }, panel.Label("Scale (of the numerals)"), panel.Default(float64(indexOf(Scales, "major"))))
					st.low = page.Knob(name+" low", 24, 84, panel.Linear, panel.Func(func(v float64) { h.ReceiveMessage(psalm.Message{"low": v}) }), panel.Label("Lowest note"), panel.Step(1), panel.Default(48))
					page.Section("Chords")
					chordmodel.Add(page, name+" chords", m)
					return st
				},
			}, nil
		}}
}

// Harmony and Model are the harmony and the chord model inside a song made by the patcher (nil: the song has none)
func (s *SongForm) Harmony() *Harmony         { return s.harm }
func (s *SongForm) Model() *chordmodel.Module { return s.model }
