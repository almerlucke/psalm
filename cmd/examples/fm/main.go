package main

import (
	"fmt"
	"math"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/oscillators/lfo"
	"github.com/almerlucke/psalm/synths/fm"
)

// An FM jam followed by a tour of the FM presets, every section showing settings that change from
// note to note. The whole tour loops, the schedule is printed at start

const (
	introBeats   = 32.0 // 8 bars
	sectionBeats = 16.0 // 4 bars
)

// seq plays the values in order and loops
func seq[T any](values ...T) *sequence.Sequence[T] {
	return sequence.NewLoop(values...)
}

// random picks one of the values at random every time
func random[T any](values ...T) *bucket.Bucket[T] {
	return bucket.NewLoop(values...).SetMode(bucket.Random)
}

// slot confines a timer pattern (tick durations, negative is a rest) to one slot of a looping
// tour: it rests until the slot starts, passes the pattern through during the slot and rests from
// the end of the slot to the end of the tour. Any pattern works, also random ones
type slot struct {
	gen   genny.Generator[float64]
	start float64 // ticks
	end   float64
	total float64
	pos   float64
}

func inSlot(gen genny.Generator[float64], startBeats, lengthBeats, totalBeats float64) *slot {
	ppq := float64(psalm.PPQ())
	return &slot{gen: gen, start: startBeats * ppq, end: (startBeats + lengthBeats) * ppq, total: totalBeats * ppq}
}

func (s *slot) Generate() float64 {
	for {
		if s.pos < s.start {
			d := s.start - s.pos
			s.pos = s.start
			return -d
		}

		if s.pos >= s.end {
			d := s.total - s.pos
			s.pos = 0
			if d > 0 {
				return -d
			}
			continue
		}

		v := s.gen.Generate()
		if remaining := s.end - s.pos; math.Abs(v) > remaining {
			s.pos = s.end
			return -remaining
		}
		s.pos += math.Abs(v)
		return v
	}
}

func (s *slot) Continuous() bool { return true }
func (s *slot) Done() bool       { return false }
func (s *slot) Reset() {
	s.pos = 0
	s.gen.Reset()
}

// note is one or more pitches (MIDI, a chord when more than one) with a length in beats, no
// pitches is a rest
type note struct {
	pitches []float64
	beats   float64
}

const rest = 0.0

func single(pitch float64, beats float64) note {
	if pitch == rest {
		return note{nil, beats}
	}
	return note{[]float64{pitch}, beats}
}

// notes builds single notes of equal length
func notes(beats float64, pitches ...float64) []note {
	ns := make([]note, len(pitches))
	for i, p := range pitches {
		ns[i] = single(p, beats)
	}
	return ns
}

// line builds single notes from pitch, beats pairs
func line(pairs ...float64) []note {
	ns := make([]note, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		ns = append(ns, single(pairs[i], pairs[i+1]))
	}
	return ns
}

// chordNotes builds chords of equal length
func chordNotes(beats float64, chords ...[]float64) []note {
	ns := make([]note, len(chords))
	for i, c := range chords {
		ns[i] = note{c, beats}
	}
	return ns
}

// section shows a preset: notes (or chords) played by an FM synth, params are sent with every note
// and step through their values note by note
type section struct {
	name   string
	cfg    fm.Config
	voices int
	amp    float64
	legato float64
	notes  []note
	params map[string][]float64
	patch  func(p *psalm.Psalm, synth psalm.Modular) // optional extra modulation
	beats  float64                                   // length of the section, 0 is sectionBeats
}

func (s *section) length() float64 {
	if s.beats > 0 {
		return s.beats
	}
	return sectionBeats
}

func (s *section) build(p *psalm.Psalm, startBeats float64, totalBeats float64) psalm.Modular {
	var (
		stps    []any
		pitches [][]float64
		durs    []float64
		total   float64
		beatSec = 60.0 / psalm.BPM()
	)

	for _, n := range s.notes {
		total += n.beats
		if len(n.pitches) == 0 {
			stps = append(stps, -n.beats)
			continue
		}
		stps = append(stps, n.beats)
		pitches = append(pitches, n.pitches)
		durs = append(durs, n.beats*beatSec*s.legato)
	}

	if total != s.length() {
		panic(fmt.Sprintf("section %q lasts %v beats, expected %v", s.name, total, s.length()))
	}

	synth := fm.NewPoly(s.cfg, s.voices).AddTo(p)

	msg := template.Template{"pitch": seq(pitches...)}
	for key, values := range s.params {
		msg[key] = seq(values...)
	}

	pattern := steps.New(seq(steps.QuickSteps(stps...)...))
	synth.ValueIn(templ.NewDurationTrigger(seq(durs...), s.amp, msg).
		ValueIn(timer.New(inSlot(pattern, startBeats, s.length(), totalBeats)).AddTo(p)))

	if s.patch != nil {
		s.patch(p, synth)
	}

	return synth
}

// transpose shifts all pitches of the notes
func transpose(ns []note, semitones float64) []note {
	out := make([]note, len(ns))
	for i, n := range ns {
		out[i].beats = n.beats
		for _, p := range n.pitches {
			out[i].pitches = append(out[i].pitches, p+semitones)
		}
	}
	return out
}

// paradeBeats is the length of every preset in the parade (2 bars)
const paradeBeats = 8.0

// phrase is a short musical phrase for the parade, with the voices, level and note length that suit it
type phrase struct {
	notes  []note
	voices int
	amp    float64
	legato float64
}

var paradePhrases = map[string]phrase{
	"chords": {chordNotes(4, []float64{60, 64, 67, 72}, []float64{57, 60, 64, 69}), 12, 0.1, 0.9},
	"pad":    {chordNotes(8, []float64{57, 60, 64, 69}), 12, 0.12, 1.0},
	"melody": {line(60, 1, 62, 1, 64, 1, 67, 1, 69, 2, 67, 2), 2, 0.3, 0.9},
	"arp":    {notes(0.5, 72, 76, 79, 84, 79, 76, 72, 76, 69, 72, 76, 81, 76, 72, 69, 72), 8, 0.25, 0.6},
	"bass":   {notes(1, 36, 36, 43, 36, 33, 33, 40, 43), 2, 0.35, 0.8},
	"drums":  {line(45, 1, rest, 0.5, 45, 0.5, 48, 1, 45, 1, 45, 1, rest, 0.5, 45, 0.5, 50, 1, 45, 1), 4, 0.4, 0.5},
}

// paradeSections presents the General MIDI inspired presets, two bars each with a phrase that suits them
func paradeSections() []*section {
	entries := []struct {
		family, preset, phrase string
		transpose              float64
		mono                   bool
		gain                   float64 // level boost for quieter presets, 0 is 1
	}{
		{"pianos and keys", "grandPiano", "chords", 0, false, 2.16},
		{"pianos and keys", "honkyTonk", "chords", 0, false, 2.45},
		{"pianos and keys", "harpsichord", "arp", -12, false, 2.11},
		{"pianos and keys", "clavinet", "melody", 0, false, 1.55},
		{"chromatic percussion", "celesta", "arp", 0, false, 0.987},
		{"chromatic percussion", "glockenspiel", "arp", 12, false, 1.08},
		{"chromatic percussion", "musicBox", "arp", 12, false, 1.11},
		{"chromatic percussion", "xylophone", "arp", 0, false, 1.19},
		{"chromatic percussion", "tubularBells", "melody", 12, false, 3.22},
		{"organs", "percussiveOrgan", "chords", 0, false, 1.63},
		{"organs", "churchOrgan", "chords", 0, false, 1.97},
		{"organs", "accordion", "chords", 0, false, 1.79},
		{"guitars", "nylonGuitar", "arp", -12, false, 1.99},
		{"guitars", "steelGuitar", "arp", -12, false, 1.92},
		{"guitars", "overdriveGuitar", "melody", 0, false, 3.28},
		{"basses", "uprightBass", "bass", 0, false, 1.87},
		{"basses", "slapBass", "bass", 0, false, 1.62},
		{"basses", "fretlessBass", "bass", 0, true, 2.47},
		{"strings", "strings", "chords", 0, false, 2.25},
		{"strings", "pizzicato", "arp", -12, false, 1.33},
		{"strings", "harp", "arp", -12, false, 1.08},
		{"brass", "trumpet", "melody", 0, false, 2.6},
		{"brass", "frenchHorn", "melody", -12, false, 2.36},
		{"reeds and pipes", "saxophone", "melody", 0, false, 2.2},
		{"reeds and pipes", "clarinet", "melody", 0, false, 3.45},
		{"reeds and pipes", "oboe", "melody", 12, false, 1.69},
		{"reeds and pipes", "flute", "melody", 12, false, 1.5},
		{"reeds and pipes", "panFlute", "melody", 12, false, 1.49},
		{"synth leads and pads", "squareLead", "melody", 12, true, 2.8},
		{"synth leads and pads", "sawLead", "melody", 0, true, 2.45},
		{"synth leads and pads", "warmPad", "pad", 0, false, 2.1},
		{"synth leads and pads", "sweepPad", "pad", 0, false, 2.62},
		{"ethnic", "sitar", "melody", 0, false, 2.36},
		{"ethnic", "kalimba", "arp", 0, false, 1.31},
		{"ethnic", "steelDrum", "arp", 0, false, 1},
		{"ethnic", "koto", "arp", -12, false, 1.91},
		{"percussion", "timpani", "drums", 0, false, 1.01},
		{"percussion", "woodblock", "drums", 24, false, 2.23},
		{"percussion", "taiko", "drums", -12, false, 0.873},
	}

	var sections []*section
	for _, e := range entries {
		ph := paradePhrases[e.phrase]
		gain := e.gain
		if gain == 0 {
			gain = 1
		}
		voices := ph.voices
		if e.mono {
			voices = 1 // glide from note to note
		}
		sections = append(sections, &section{
			name:   fmt.Sprintf("Parade (%s): %s", e.family, e.preset),
			cfg:    fm.Presets[e.preset](),
			voices: voices, amp: ph.amp * gain, legato: ph.legato,
			notes: transpose(ph.notes, e.transpose),
			beats: paradeBeats,
		})
	}
	return sections
}

func main() {
	psalm.PushConfig(psalm.Config{
		BPM:        92,
		PPQ:        240,
		SampleRate: 44100,
		BufferSize: 1024,
	})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	ppq := float64(psalm.PPQ())
	beatSec := 60.0 / psalm.BPM()

	sections := []*section{
		{
			name: "Electric piano: velocity rising from soft to hard, harder is brighter",
			cfg:  fm.EPianoPreset(), voices: 16, amp: 0.106, legato: 0.9,
			notes: chordNotes(2,
				[]float64{62, 65, 69, 72}, []float64{62, 65, 69, 72},
				[]float64{59, 62, 65, 67}, []float64{59, 62, 65, 67},
				[]float64{60, 64, 67, 71}, []float64{60, 64, 67, 71},
				[]float64{57, 60, 64, 67}, []float64{57, 60, 64, 67},
			),
			params: map[string][]float64{"velocity": {0.1, 0.2, 0.35, 0.5, 0.6, 0.75, 0.9, 1.0}},
		},
		{
			name: "Bell: the ratio of operator 2 changes the color (3.5, 2, 1.41, 5)",
			cfg:  fm.BellPreset(), voices: 6, amp: 0.359, legato: 0.5,
			notes:  notes(4, 72, 79, 76, 72),
			params: map[string][]float64{"op2Ratio": {3.5, 2.0, 1.41, 5.0}},
		},
		{
			name: "Bass: mod depth (brightness) rising note by note",
			cfg:  fm.BassPreset(), voices: 2, amp: 0.874, legato: 0.7,
			notes: notes(1, 36, 36, 48, 36, 43, 43, 55, 43, 41, 41, 53, 41, 43, 43, 55, 47),
			params: map[string][]float64{"modDepth": {
				0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7,
			}},
		},
		{
			name: "Brass: swelling chords with delayed vibrato",
			cfg:  fm.BrassPreset(), voices: 12, amp: 0.367, legato: 0.9,
			notes: chordNotes(4,
				[]float64{60, 64, 67}, []float64{58, 62, 65}, []float64{57, 60, 65}, []float64{55, 60, 64},
			),
		},
		{
			name: "Organ: pulling out drawbars (the levels of operators 3 and 4) chord by chord",
			cfg:  fm.OrganPreset(), voices: 16, amp: 0.182, legato: 0.95,
			notes: chordNotes(4,
				[]float64{48, 60, 64, 67}, []float64{48, 60, 64, 67}, []float64{53, 60, 65, 69}, []float64{55, 59, 62, 67},
			),
			params: map[string][]float64{
				"op3Level": {0.0, 0.6, 0.6, 0.9},
				"op4Level": {0.0, 0.0, 0.4, 0.8},
			},
		},
		{
			name: "Mallet: an arpeggio with accents (velocity)",
			cfg:  fm.MalletPreset(), voices: 8, amp: 0.237, legato: 0.5,
			notes: notes(1, 60, 64, 67, 72, 76, 72, 67, 64, 62, 65, 69, 74, 77, 74, 69, 65),
			params: map[string][]float64{"velocity": {
				1.0, 0.4, 0.6, 0.4, 1.0, 0.4, 0.6, 0.4, 1.0, 0.4, 0.6, 0.4, 1.0, 0.5, 0.7, 0.5,
			}},
		},
		{
			name: "Pad: an LFO on value input 1 sweeps the mod depth",
			cfg:  fm.PadPreset(), voices: 12, amp: 0.178, legato: 1.0,
			notes: chordNotes(8, []float64{57, 60, 64, 69}, []float64{53, 57, 60, 65}),
			patch: func(p *psalm.Psalm, synth psalm.Modular) {
				synth.ValueIn(lfo.New(true, 0.3, 0.1, 2.0, lfo.Sine).AddTo(p), 0, 1)
			},
		},
		{
			name: "Pluck: feedback rising note by note, from round to buzzing",
			cfg:  fm.PluckPreset(), voices: 8, amp: 0.537, legato: 0.6,
			notes: notes(1, 64, 67, 71, 72, 64, 67, 71, 74, 64, 67, 71, 76, 64, 67, 71, 72),
			params: map[string][]float64{"feedback": {
				0.0, 0.05, 0.1, 0.15, 0.2, 0.25, 0.3, 0.35, 0.4, 0.45, 0.5, 0.55, 0.6, 0.7, 0.8, 0.9,
			}},
		},
		{
			name: "Algorithms: the metallic preset's operators through all 8 routings",
			cfg:  fm.MetallicPreset(), voices: 4, amp: 0.81, legato: 0.6,
			notes:  notes(2, 48, 48, 48, 48, 48, 48, 48, 48),
			params: map[string][]float64{"algorithm": {0, 1, 2, 3, 4, 5, 6, 7}},
		},
		{
			name: "Lead (mono): glide and vibrato, feedback rising per phrase",
			cfg:  fm.LeadPreset(), voices: 1, amp: 0.734, legato: 1.0,
			notes:  line(72, 2, 74, 1, 76, 1, 79, 3, 76, 1, 74, 2, 72, 2, 69, 3, rest, 1),
			params: map[string][]float64{"feedback": {0.2, 0.2, 0.4, 0.4, 0.6, 0.6, 0.85, 0.85}},
		},
	}

	sections = append(sections, paradeSections()...)

	totalBeats := introBeats
	for _, s := range sections {
		totalBeats += s.length()
	}
	beatsToSec := func(beats float64) float64 { return beats * beatSec }

	// Intro jam (the first 8 bars of the tour)
	fmt.Printf("%6.1fs  Jam: electric piano chords, FM bass and bells\n", 0.0)

	// Electric piano: a ii-V-I-vi progression in C (Dm7 G7 Cmaj7 Am7), each chord played twice per
	// bar with a little syncopation. Random velocities make some chords brighter than others
	epiano := fm.NewPoly(fm.EPianoPreset(), 16).AddTo(p)

	chords := seq(
		[]float64{62, 65, 69, 72}, []float64{62, 65, 69, 72}, // Dm7
		[]float64{59, 62, 65, 67}, []float64{59, 62, 65, 67}, // G7
		[]float64{60, 64, 67, 71}, []float64{60, 64, 67, 71}, // Cmaj7
		[]float64{57, 60, 64, 67}, []float64{57, 60, 64, 67}, // Am7
	)

	jamPattern := func() *steps.Steps { return steps.New(seq(steps.QuickSteps(1.5, 2.5)...)) }

	epiano.ValueIn(templ.NewDurationTrigger(seq(1.2*beatSec, 2.3*beatSec), 0.106, template.Template{
		"pitch":    chords,
		"velocity": random(0.5, 0.7, 0.9, 1.0),
	}).ValueIn(timer.New(inSlot(jamPattern(), 0, introBeats, totalBeats)).AddTo(p)))

	// Bass: roots and fifths on the FM bass
	bass := fm.NewPoly(fm.BassPreset(), 2).AddTo(p)

	bass.ValueIn(templ.NewDurationTrigger(0.8*beatSec, 0.753, template.Template{
		"pitch":    seq(38.0, 45.0, 43.0, 50.0, 36.0, 43.0, 45.0, 40.0),
		"velocity": random(0.7, 0.85, 1.0),
	}).ValueIn(timer.New(inSlot(jamPattern(), 0, introBeats, totalBeats)).AddTo(p)))

	// Bells: sparse, random notes from the C major pentatonic, high up
	bells := fm.NewPoly(fm.BellPreset(), 8).AddTo(p)

	bells.ValueIn(templ.NewDurationTrigger(0.5, random(0.0714, 0.1, 0.129), template.Template{
		"pitch":    random(84.0, 86.0, 88.0, 91.0, 93.0, 96.0),
		"velocity": random(0.4, 0.6, 0.8),
	}).ValueIn(timer.New(inSlot(random(2*ppq, 3*ppq, -4*ppq, -6*ppq), 0, introBeats, totalBeats)).AddTo(p)))

	mix := []any{epiano, 0, 0, bass, 0, 0, bells, 0, 0}

	// The preset tour
	start := introBeats
	for _, s := range sections {
		fmt.Printf("%6.1fs  %s\n", beatsToSec(start), s.name)
		mix = append(mix, s.build(p, start, totalBeats), 0, 0)
		start += s.length()
	}

	fmt.Printf("%6.1fs  (tour loops)\n", beatsToSec(totalBeats))

	// A warm room
	rcfg := fdnreverb.DefaultConfig()
	rcfg.Decay = 2.2
	rcfg.Mix = 0.25

	p.Out(fdnreverb.New(rcfg).AddTo(p).In(mix...))

	//err := p.ToSoundFile("fm.wav", writer.WAV, beatsToSec(totalBeats), 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
