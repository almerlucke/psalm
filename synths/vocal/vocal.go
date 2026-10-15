// Package vocal sings: a glottal source (the flow of air through the vocal folds, a
// Rosenberg-Klatt pulse with an open quotient, a skew and a return, with jitter, shimmer and breath
// that follows the cycle) through a cascade of five formants (the resonances of the throat and mouth)
// of a bass, tenor, alto, soprano or child, with a nose, a singer's formant, formant tuning to the
// pitch, vibrato and the small unsteadiness of a real voice. It sings twelve vowels (see Vowels) or
// the scale a e i o u, glides from one vowel to another within a note (a diphthong), and speaks words
// of vowels and consonants (stops, fricatives, nasals, liquids and glides) whose formants move from
// the consonant into the vowel. A choir is a voice sung by several singers at once, each a little
// out of tune and time, spread over the stereo field. The source is a Rosenberg-Klatt pulse or the
// Liljencrants-Fant model (one parameter, Rd, for the quality of the voice); the tone can be taken out
// for a whisper of noise shaped by the same tract; p t k are aspirated and b d g voiced; and the
// vibrato can be a singer's, wandering in rate and extent.
//
//	choir := vocal.NewPoly(vocal.Choir(), 8).AddTo(p)
//	choir.Receive(psalm.Message{"vowel": 3.0}, 0) // all sing "o"
package vocal

import (
	"fmt"
	"math"
	"strings"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/modules/poly"
)

// Range is the kind of voice: its formants
type Range int

const (
	Auto Range = iota // by the pitch of the note
	Bass
	Tenor
	Alto
	Soprano
	ChildVoice
	numRanges
)

// Config is a voice, or a section of a choir
type Config struct {
	NumChannels  int
	Range        Range
	Vowel        float64 // 0-4: a e i o u, fractions morph
	VowelGlide   float64 // seconds a change of vowel takes
	Brightness   float64 // 0-1: a soft, breathy voice to a pressed, bright one; the velocity adds
	Breath       float64 // 0-1: the noise of the air
	Attack       float64 // seconds
	Release      float64 // seconds
	VibratoRate  float64 // Hz
	VibratoDepth float64 // semitones
	VibratoDelay float64 // seconds before the vibrato comes in
	Singers      int     // voices singing each note (0 is 1): more is a choir
	Detune       float64 // cents the singers spread over
	Spread       float64 // 0-1: how far they stand apart, left to right
	Steady       bool    // the pitch held perfectly still (a voice wanders a few cents)
	// the vowel moving by itself, from the start of every note: VowelRate Hz, or VowelSync beats a
	// cycle (follows the tempo), VowelDepth steps up and down around Vowel. VowelSteps makes it
	// jump to a new vowel at random every cycle (gliding there in VowelGlide): talking, babbling
	VowelRate  float64
	VowelSync  float64
	VowelDepth float64
	VowelSteps bool
	Level      float64 // output level (0 is treated as 1)

	// Source is the glottal model: 0 the Rosenberg-Klatt pulse (OpenQuotient, SpeedQuotient and
	// ReturnPhase shape it), 1 the Liljencrants-Fant model, which has the one parameter Rd for the
	// quality of the voice: 0.3 pressed and bright to 2.7 breathy and soft (0 is made from Brightness)
	Source int
	Rd     float64
	// Whisper (0-1) takes the tone out of the voice for turbulent noise that the same throat and mouth
	// shape: 0 is the voice (with Breath for a breathy one), 1 a whisper with no pitch
	Whisper float64
	// NaturalVibrato (0-1) makes the vibrato a singer's: the rate 5 to 6.5 Hz, rising a little with the
	// pitch and wandering slowly, the extent wandering too and wider on high notes, late to come in
	NaturalVibrato float64
	// Subglottal (0-1) lets in the resonance of the windpipe below the folds: a faint pole near 600 Hz
	// and zero near 1500 Hz
	Subglottal float64

	// Phoneme sings one of the twelve vowels of Vowels (1 to 12) instead of Vowel (0): the vowel
	// moves along that list, so the steps of the matrix and the wheel go to the neighbours
	Phoneme int
	// PhonemeTo makes a diphthong of the note: the vowel glides from the one sung to this one (1 to
	// 12, 0 is none) in DiphthongTime seconds (0 is 0.3), after holding the first for DiphthongHold
	PhonemeTo     int
	DiphthongTime float64
	DiphthongHold float64

	// the glottal source. OpenQuotient (0.2-0.95: the part of the cycle the folds are open), the
	// SpeedQuotient (0.8-4: how much longer they take to open than to close) and the ReturnPhase
	// (0.02-1: how slowly the closing ends; the longer the darker) are made from Brightness unless set
	// (0 is that). Jitter is the spread of the length of the cycles (0.005 is 0.5%), Shimmer that of
	// their loudness (0.03 is 3%): none unless set
	OpenQuotient  float64
	SpeedQuotient float64
	ReturnPhase   float64
	Jitter        float64
	Shimmer       float64

	// Tuning is how far the first formant follows the pitch up when the pitch is higher than it, as
	// a soprano's does (0 is 1; a negative value is none). Training (0-1) is the singer's formant: the
	// third to fifth formants gather near 3 kHz, as in a trained voice. Nasal (0-1) opens the nose
	Tuning   float64
	Training float64
	Nasal    float64
	// Coarticulation is the seconds the formants take to move from a consonant into the vowel
	// (0 is 0.045)
	Coarticulation float64

	Consonant      Consonant // a consonant as the note starts (none unless set)
	ConsonantLevel float64   // 0-1 (0 is treated as 0.5)

	// FormantShift scales all the formants (and their widths): below 1 a bigger throat, a giant;
	// above 1 a smaller one, a child (0 is 1)
	FormantShift float64

	// Word is a text to sing: each note takes the next syllable (its vowel and consonant), looping at
	// the end (see ParseWord). With WordSync beats a syllable (0 is one syllable a note), a held note
	// goes on to the next syllables too. The notes of a Poly share the place in the word
	Word     []Syllable
	WordSync float64

	Filter voicefilter.Config              // a filter after the formants (off unless its Type is set)
	Mods   [modmatrix.Slots]modmatrix.Slot // the matrix: a source to a destination (see Dests) over an amount
	Ext    [modmatrix.Exts]float64         // the values 0-1 of the matrix's sources Ext 1 to 4 (cabled in on InExt1...)
}

// The destinations of the matrix (see Dests), by number
const (
	dVowel = iota
	dBrightness
	dBreath
	dVibratoDepth
	dDetune
	dSpread
	dPitch
	dLevel
	dCutoff
	dWhisper
)

// Dests are the settings the matrix of a voice moves: by their keys, as described (Describe), with
// how far a full source moves them (vowel steps, cents, semitones, octaves of the cutoff, or the
// setting itself)
var Dests = []modmatrix.Dest{
	dVowel:        {Key: "vowel", Label: "vowel", Span: 2},
	dBrightness:   {Key: "brightness", Label: "brightness", Span: 0.5},
	dBreath:       {Key: "breath", Label: "breath", Span: 0.5},
	dVibratoDepth: {Key: "vibratoDepth", Label: "vibrato depth", Span: 1},
	dDetune:       {Key: "detune", Label: "detune", Span: 40},
	dSpread:       {Key: "spread", Label: "spread", Span: 1},
	dPitch:        {Key: "pitch", Label: "pitch", Span: 12},
	dLevel:        {Key: "level", Label: "level", Span: 1},
	dCutoff:       {Key: "cutoff", Label: "cutoff", Span: 4},
	dWhisper:      {Key: "whisper", Label: "whisper", Span: 1},
}

// Syllable is a vowel position (0-4: a e i o u, fractions morph) and a consonant before it. A
// Diphthong (1 to 5, see DiphthongNames) sings that pair of vowels instead of Vowel
type Syllable struct {
	Vowel     float64
	Consonant Consonant
	Diphthong int
}

// DiphthongNames are the diphthongs of a syllable, by number
var DiphthongNames = []string{"none", "ai", "au", "ei", "ou", "oi"}

// the vowels each diphthong glides from and to, by their place in Vowels (from 1)
var diphthongs = [...][2]int{{}, {6, 2}, {6, 9}, {3, 2}, {8, 9}, {7, 2}}

// wordPlace is the place in the word, shared by the voices of a Poly: the syllable the next note sings
type wordPlace struct{ next int }

// The value inputs of a voice: 1 is the vowel and 2 the vibrato depth, 3 to 6 the values of the
// matrix (Ext 1 to 4) that a slot can move any setting by
const InExt1 = 3 // to InExt1+3

const middleC = 261.6255653005986

// Consonant is the sound a note starts with
type Consonant int

const (
	NoConsonant Consonant = iota
	T
	K
	S
	Sh
	F
	M
	N
	Ng
	L
	R
	W
	Y
	P // the voiceless p t k are aspirated; the voiced b d g start their voice soon after the burst
	B
	D
	G
	Z // the voiced fricatives hum through the noise (the voice bar)
	Zh
	V
	numConsonants
)

// ConsonantNames are the names of the consonants, by number
var ConsonantNames = []string{"none", "t", "k", "s", "sh", "f", "m", "n", "ng", "l", "r", "w", "y", "p", "b", "d", "g", "z", "zh", "v"}

// locus is where the formants of a consonant stand: the first three (and their widths) at the start,
// held for hold seconds, then moving into the vowel in trans seconds. A nasal opens the nose (with a
// zero at zero Hz); the voice is as loud as amp while it is at the locus
type locus struct {
	f, bw       [3]float64
	hold, trans float64
	nasal, zero float64
	amp         float64
}

// consonantShape is a consonant: a burst of noise through a band (freq, q), as long as dur seconds,
// either dying away at once (a plosive, with the time constant tau) or swelling and fading (a
// fricative); and the locus its formants start from. A nasal, a liquid or a glide has no noise
type consonantShape struct {
	freq, q, dur, tau, gain float64
	plosive                 bool
	loc                     locus
	// vot is the seconds after the start until the voice does (the voice onset time: a voiceless
	// consonant keeps it out, with aspiration of the level asp through the throat, a voiced stop starts it
	// soon); bar is how far a voiced fricative hums, through the noise, in a murmur of low partials
	vot, asp, bar float64
}

var consonants = [...]consonantShape{
	T: {freq: 5200, q: 1.2, dur: 0.05, tau: 0.01, gain: 3, plosive: true, vot: 0.055, asp: 0.5,
		loc: locus{f: [3]float64{300, 1800, 2900}, bw: [3]float64{100, 150, 200}, trans: 0.045, amp: 1}},
	K: {freq: 2000, q: 2, dur: 0.07, tau: 0.018, gain: 3, plosive: true, vot: 0.065, asp: 0.5,
		loc: locus{f: [3]float64{300, 1500, 2500}, bw: [3]float64{100, 150, 200}, trans: 0.045, amp: 1}},
	S: {freq: 7500, q: 1.5, dur: 0.12, gain: 2.5, vot: 0.1,
		loc: locus{f: [3]float64{350, 1700, 2700}, bw: [3]float64{100, 150, 200}, trans: 0.05, amp: 1}},
	Sh: {freq: 3500, q: 1, dur: 0.14, gain: 2.5, vot: 0.12,
		loc: locus{f: [3]float64{350, 1900, 2400}, bw: [3]float64{100, 150, 200}, trans: 0.05, amp: 1}},
	F: {freq: 4500, q: 0.4, dur: 0.1, gain: 1.8, vot: 0.08,
		loc: locus{f: [3]float64{350, 1100, 2300}, bw: [3]float64{100, 150, 200}, trans: 0.05, amp: 1}},
	M: {loc: locus{f: [3]float64{250, 1000, 2200}, bw: [3]float64{90, 200, 250}, hold: 0.04, trans: 0.055,
		nasal: 1, zero: 1000, amp: 0.55}},
	N: {loc: locus{f: [3]float64{250, 1500, 2500}, bw: [3]float64{90, 180, 250}, hold: 0.04, trans: 0.055,
		nasal: 1, zero: 1600, amp: 0.55}},
	Ng: {loc: locus{f: [3]float64{250, 1900, 2500}, bw: [3]float64{90, 180, 250}, hold: 0.04, trans: 0.055,
		nasal: 1, zero: 2100, amp: 0.55}},
	L: {loc: locus{f: [3]float64{360, 1100, 2800}, bw: [3]float64{80, 110, 140}, hold: 0.025, trans: 0.055, amp: 0.8}},
	R: {loc: locus{f: [3]float64{400, 1300, 1650}, bw: [3]float64{90, 110, 100}, hold: 0.03, trans: 0.06, amp: 0.85}},
	W: {loc: locus{f: [3]float64{300, 700, 2300}, bw: [3]float64{70, 90, 130}, hold: 0.015, trans: 0.07, amp: 0.8}},
	Y: {loc: locus{f: [3]float64{290, 2250, 3000}, bw: [3]float64{60, 90, 130}, hold: 0.015, trans: 0.065, amp: 0.85}},
	P: {freq: 1200, q: 0.7, dur: 0.04, tau: 0.008, gain: 2.4, plosive: true, vot: 0.05, asp: 0.5,
		loc: locus{f: [3]float64{300, 900, 2200}, bw: [3]float64{100, 150, 200}, trans: 0.045, amp: 1}},
	B: {freq: 800, q: 0.7, dur: 0.03, tau: 0.006, gain: 1.2, plosive: true, vot: 0.012,
		loc: locus{f: [3]float64{300, 900, 2200}, bw: [3]float64{100, 150, 200}, trans: 0.045, amp: 1}},
	D: {freq: 4500, q: 1.2, dur: 0.04, tau: 0.008, gain: 1.5, plosive: true, vot: 0.012,
		loc: locus{f: [3]float64{300, 1800, 2900}, bw: [3]float64{100, 150, 200}, trans: 0.045, amp: 1}},
	G: {freq: 2000, q: 2, dur: 0.05, tau: 0.012, gain: 1.5, plosive: true, vot: 0.015,
		loc: locus{f: [3]float64{300, 1500, 2500}, bw: [3]float64{100, 150, 200}, trans: 0.045, amp: 1}},
	Z: {freq: 7500, q: 1.5, dur: 0.11, gain: 2, bar: 1,
		loc: locus{f: [3]float64{350, 1700, 2700}, bw: [3]float64{100, 150, 200}, trans: 0.05, amp: 0.8}},
	Zh: {freq: 3500, q: 1, dur: 0.12, gain: 2, bar: 1,
		loc: locus{f: [3]float64{350, 1900, 2400}, bw: [3]float64{100, 150, 200}, trans: 0.05, amp: 0.8}},
	V: {freq: 4500, q: 0.4, dur: 0.09, gain: 1.3, bar: 1,
		loc: locus{f: [3]float64{350, 1100, 2300}, bw: [3]float64{100, 150, 200}, trans: 0.05, amp: 0.8}},
}

// DefaultConfig is a solo voice
func DefaultConfig() Config {
	return Solo()
}

// rangeOf returns the range that sings a pitch best
func rangeOf(pitch float64) Range {
	switch {
	case pitch < 50:
		return Bass
	case pitch < 58:
		return Tenor
	case pitch < 66:
		return Alto
	}
	return Soprano
}

// maxSingers is the most singers of a voice
const maxSingers = 16

// the noise: the level of a whisper and of the aspiration of p, t, k at the throat, how much of a
// consonant's noise excites the tract too, and how loud the voice bar hums
const (
	whisperGain = 3.2
	aspGain     = 0.5
	fricFeed    = 0.25
	barGain     = 2.5
)

// softClip is the last resort against a level that runs away: the sound passes untouched up to 0.9 and
// bends towards 1.5 above it, which it never reaches
func softClip(x float64) float64 {
	const knee, limit = 0.9, 1.5
	a := math.Abs(x)
	if a <= knee {
		return x
	}
	return math.Copysign(knee+(limit-knee)*math.Tanh((a-knee)/(limit-knee)), x)
}

// outScale brings the cascade of formants to the level the voice had before
const outScale = 0.12

// singer is one of the voices of a note: its own source, a little out of tune and time
type singer struct {
	g       glottis
	detune  float64 // as a ratio
	vibRate float64
	vibPh   float64
	drift   float64 // a slow random wander of the pitch, in semitones...
	driftTo float64 // ...towards a new aim every so often
	delay   int     // samples it comes in late
	gainL   float64
	gainR   float64
	pos     float64 // -1 to 1, where it stands in the choir
	jitter  float64 // its own detune, as a fraction of the detune
	// the natural vibrato: the rate and extent now, and their slow wander
	natRate, natExt, rateDev, extDev float64
}

// Voice sings one note
type Voice struct {
	*psalm.Module
	cfg       Config
	singers   [maxSingers]singer
	n         int
	pitch     float64
	velocity  float64
	amp       float64
	rng       *randSource
	tract     [2]tractFilter
	vowel     float64 // the vowel sung, gliding to its target: a position on the scale or along Vowels
	mode      int     // 0 when vowel is on the scale a e i o u, else the phoneme it is a position along Vowels
	phTo      int     // the vowel a diphthong glides to (0 none)
	oq, sq    float64 // the shape of the glottal pulse now
	tilt      [2]float64
	tiltCoef  float64
	breathLP  [2]float64
	env       float64
	released  bool
	t         int
	noteOff   int
	waitSamps int
	active    bool
	rangeNow  Range
	update    int
	stepTo    float64               // the vowel a step of VowelSteps aims for
	expr      expression.Expression // the pitch bend; the mod wheel opens the vowel (up to two steps)
	filter    *voicefilter.Filter
	mods      *modmatrix.Matrix
	random    float64 // a random value for the note, -1 to 1
	lfo2      float64 // phase
	cons      *svf.SVF
	consT     int // samples into the burst of the consonant (-1 none)
	consNow   Consonant
	artT      int // samples since the consonant began moving the formants (-1 over)
	artGain   float64
	tuneGain  float64 // the gain that keeps a tuned tract at the level of the untuned one (<= 1)
	tuneNow   float64 // the same, smoothed
	compSnap  bool    // the next gain found is taken at once (a note begins)
	compN     int     // how often the formants were set (the gain is found every eighth time)
	artTarget float64
	syllT0    int // the sample the syllable (or the note) began
	place     *wordPlace
	syl       int     // the syllable this note started on
	sylNow    int     // the syllable sung now (-1 when there is no word)
	sylVowel  float64 // its vowel
	sylPh     int     // its vowel as a phoneme (0 none), and the one it glides to
	sylTo     int
	fnow      [5]float64 // the formants now (Hz)
	lf        lfPulse    // the Liljencrants-Fant pulse now
	lfRd      float64    // the Rd and the closing time it was solved for
	lfTa      float64
	harm      [harmN]float64 // the power of the harmonics of the pulse now
	harmKey   [4]float64     // what they were found for
	nrng      randSource     // the noise of the whisper and the aspiration, and the natural vibrato: a stream of its own
	nPrev     [2]float64     // the last noise of each channel (the lips differentiate it)
	bar       [2][2]float64  // the low-pass of the voice bar
	gateEnd   int            // samples into the syllable the voicing of the consonant is over
}

// gauss is a random value of mean 0 and spread 1
func (r *randSource) gauss() float64 {
	return (r.float() + r.float() + r.float() + r.float() - 2) * math.Sqrt(3)
}

// wander moves the vibrato of a singer a step on (every 64 samples): a rate of 5 to 6.5 Hz, higher on
// high notes, and an extent that is wider on high notes, each wandering slowly around that. key is the
// pitch, -1 to 1 over two octaves around middle C, and amount how far the voice is natural (0-1)
func (s *singer) wander(r *randSource, amount, key, sr float64, step bool) {
	if step {
		dt := 64 / sr
		k := dt / 2.0
		s.rateDev += -s.rateDev*k + 0.3*math.Sqrt(2*k)*r.gauss()
		k = dt / 3.0
		s.extDev += -s.extDev*k + 0.15*math.Sqrt(2*k)*r.gauss()
	}
	rate := math.Max(5, math.Min(5.75+0.35*key+s.rateDev, 6.5))
	s.natRate = s.vibRate*(1-amount) + rate*amount
	if s.vibRate == 0 {
		s.natRate = rate
	}
	s.natExt = math.Max(0.4, math.Min(1+amount*(0.3*key+s.extDev), 1.8))
}

// randSource is a small fast random generator
type randSource struct{ s uint64 }

func (r *randSource) float() float64 {
	if r.s == 0 {
		r.s = 0x9E3779B97F4A7C15
	}
	r.s ^= r.s << 13
	r.s ^= r.s >> 7
	r.s ^= r.s << 17
	return float64(r.s>>11) / (1 << 53)
}

// New returns a voice
func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	v := &Voice{Module: psalm.NewModule(nil, []int{cfg.NumChannels}), cfg: cfg, pitch: 60, vowel: cfg.Vowel, place: &wordPlace{}, sylNow: -1}
	sr := v.Config().SampleRate
	v.rng = &randSource{s: v.Rand().Uint64() | 1}
	v.nrng.s = (v.rng.s ^ 0xD1B54A32D192ED03) | 1
	for i := range v.singers {
		// the irregularities of each source have a stream of their own, so they never move the others
		v.singers[i].g.rng.s = (v.rng.s ^ uint64(i+1)*0xBF58476D1CE4E5B9) | 1
	}
	v.filter = voicefilter.New(cfg.Filter, sr, cfg.NumChannels)
	v.cons = svf.New(1000, 0, sr)
	v.consT, v.artT, v.artGain, v.artTarget = -1, -1, 1, 1
	v.tuneGain, v.tuneNow, v.compSnap = 1, 1, true
	v.mods = modmatrix.New(Dests)
	for i, sl := range cfg.Mods {
		v.mods.Set(i, sl)
	}
	for i, e := range cfg.Ext {
		v.mods.SetExt(i, e)
	}
	v.SetSelf(v)
	return v
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	v.expr.Receive(msg)
	v.filter.Receive(msg)
	v.cfg.Filter = v.filter.Config()
	v.mods.Receive(msg)
	for i := range v.cfg.Mods {
		v.cfg.Mods[i] = v.mods.Slot(i)
	}
	for i := range v.cfg.Ext {
		v.cfg.Ext[i] = v.mods.Ext(i)
	}
	cfg := &v.cfg
	msg.IfFltVal("formantShift", func(f float64) { cfg.FormantShift = math.Max(0.25, math.Min(f, 4)) })
	msg.IfFltVal("wordSync", func(f float64) { cfg.WordSync = math.Max(0, f) })
	msg.IfStrVal("word", func(s string) {
		if w, err := ParseWord(s); err == nil {
			cfg.Word = w
		}
	})
	// the next syllable to sing (the place is shared in a Poly)
	for _, k := range []string{"syllable", "wordStep"} {
		msg.IfIntVal(k, func(i int) { v.place.next = max(0, i) })
	}
	msg.IfIntVal("consonant", func(c int) { cfg.Consonant = Consonant(max(0, min(c, int(numConsonants)-1))) })
	msg.IfFltVal("consonantLevel", func(f float64) { cfg.ConsonantLevel = math.Max(0, math.Min(f, 1)) })
	msg.IfIntVal("phoneme", func(i int) { cfg.Phoneme = max(0, min(i, len(Vowels))) })
	msg.IfIntVal("phonemeTo", func(i int) { cfg.PhonemeTo = max(0, min(i, len(Vowels))) })
	// a named diphthong sets both vowels
	msg.IfIntVal("diphthong", func(i int) {
		i = max(0, min(i, len(diphthongs)-1))
		if i == 0 {
			cfg.PhonemeTo = 0
			return
		}
		cfg.Phoneme, cfg.PhonemeTo = diphthongs[i][0], diphthongs[i][1]
	})
	msg.IfFltVal("diphthongTime", func(f float64) { cfg.DiphthongTime = math.Max(0, math.Min(f, 4)) })
	msg.IfFltVal("diphthongHold", func(f float64) { cfg.DiphthongHold = math.Max(0, math.Min(f, 4)) })
	msg.IfFltVal("openQuotient", func(f float64) { cfg.OpenQuotient = zeroOr(f, 0.2, 0.95) })
	msg.IfFltVal("speedQuotient", func(f float64) { cfg.SpeedQuotient = zeroOr(f, 0.8, 4) })
	msg.IfFltVal("returnPhase", func(f float64) { cfg.ReturnPhase = zeroOr(f, 0.02, 1) })
	msg.IfIntVal("source", func(i int) { cfg.Source = max(0, min(i, 1)) })
	msg.IfFltVal("rd", func(f float64) { cfg.Rd = zeroOr(f, 0.3, 2.7) })
	msg.IfFltVal("whisper", func(f float64) { cfg.Whisper = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("naturalVibrato", func(f float64) { cfg.NaturalVibrato = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("subglottal", func(f float64) { cfg.Subglottal = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("jitter", func(f float64) { cfg.Jitter = math.Max(0, math.Min(f, 0.05)) })
	msg.IfFltVal("shimmer", func(f float64) { cfg.Shimmer = math.Max(0, math.Min(f, 0.3)) })
	msg.IfFltVal("formantTuning", func(f float64) {
		cfg.Tuning = math.Max(0, math.Min(f, 1))
		if cfg.Tuning == 0 {
			cfg.Tuning = -1
		}
	})
	msg.IfFltVal("training", func(f float64) { cfg.Training = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("nasal", func(f float64) { cfg.Nasal = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("coarticulation", func(f float64) { cfg.Coarticulation = math.Max(0.005, math.Min(f, 0.2)) })
	envelope.Seconds(msg, "amp", &cfg.Attack, nil, &cfg.Release)
	msg.IfFltVal("frequency", func(f float64) {
		if f > 0 {
			v.pitch = 69 + 12*math.Log2(f/440)
		}
	})
	msg.IfFltVal("pitch", func(p float64) { v.pitch = p })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("vowel", func(f float64) { cfg.Vowel = f })
	msg.IfFltVal("vowelGlide", func(f float64) { cfg.VowelGlide = f })
	msg.IfFltVal("brightness", func(f float64) { cfg.Brightness = f })
	msg.IfFltVal("breath", func(f float64) { cfg.Breath = f })
	msg.IfFltVal("vibratoRate", func(f float64) { cfg.VibratoRate = f })
	msg.IfFltVal("vibratoDepth", func(f float64) { cfg.VibratoDepth = f })
	msg.IfIntVal("range", func(r int) { cfg.Range = Range(max(0, min(r, int(numRanges)-1))) })
	msg.IfFltVal("level", func(f float64) { cfg.Level = f })
}

// zeroOr keeps 0 (the setting is made from others) and limits anything else to lo-hi
func zeroOr(f, lo, hi float64) float64 {
	if f <= 0 {
		return 0
	}
	return math.Max(lo, math.Min(f, hi))
}

// Receive takes messages, or the vowel on input 1 and the vibrato depth on input 2, and on 3 to 6
// (InExt1...) values 0-1 that the matrix takes as sources Ext 1 to 4
func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}
	switch index {
	case 1:
		v.cfg.Vowel = psalm.FltVal[float64](value)
	case 2:
		v.cfg.VibratoDepth = psalm.FltVal[float64](value)
	case InExt1, InExt1 + 1, InExt1 + 2, InExt1 + 3:
		v.mods.SetExt(index-InExt1, psalm.FltVal[float64](value))
		v.cfg.Ext[index-InExt1] = v.mods.Ext(index - InExt1)
	}
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.sing(0, amplitude, message, tick)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.sing(duration, amplitude, message, tick)
}

func (v *Voice) sing(duration, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	v.velocity = 0.8
	v.ReceiveMessage(message)
	cfg := &v.cfg
	sr := v.Config().SampleRate
	v.waitSamps = tick.SampleOffset
	v.amp = amplitude
	v.released = false
	v.noteOff = 0
	if duration > 0 {
		v.noteOff = max(1, int(duration*sr))
	}
	v.t = 0
	// drawn only when a slot uses it: a voice that doesn't keeps the random stream it had
	if v.mods.Uses(modmatrix.Random) {
		v.random = v.Rand().Float64()*2 - 1
	}
	mode := adsr.NoteOff
	if duration > 0 {
		mode = adsr.Duration
	}
	v.filter.Trigger(duration*1000, mode)
	v.sylNow = -1
	v.consNow = cfg.Consonant
	if n := len(cfg.Word); n > 0 {
		v.syl = v.place.next % n
		v.place.next = (v.syl + 1) % n
		v.setSyllable(v.syl)
	}
	v.startConsonant()
	v.rangeNow = cfg.Range
	if v.rangeNow == Auto {
		v.rangeNow = rangeOf(v.pitch)
	}
	if !v.active {
		v.env = 0
		ph, _, pos := v.vowelBase()
		v.mode, v.vowel = ph, pos
		v.compN, v.compSnap = 0, true
		for c := range v.tract {
			v.tract[c].clear()
			v.tilt[c], v.breathLP[c] = 0, 0
		}
	}
	v.n = max(1, min(cfg.Singers, maxSingers))
	spread := math.Max(0, math.Min(cfg.Spread, 1))
	for i := range v.n {
		s := &v.singers[i]
		// the singers stand apart, detuned over the spread of cents, each with a vibrato of its own
		pos := 0.0
		if v.n > 1 {
			pos = float64(i)/float64(v.n-1)*2 - 1
		}
		s.pos, s.jitter = pos, 0
		if v.n > 1 {
			s.jitter = (v.rng.float() - 0.5) * 0.3
		}
		s.detune = math.Exp2(s.cents(cfg.Detune) / 1200)
		s.vibRate = math.Max(0, cfg.VibratoRate) * (1 + (v.rng.float()-0.5)*0.15)
		s.vibPh = v.rng.float()
		s.drift, s.driftTo = 0, 0
		if nat := cfg.NaturalVibrato; nat > 0 {
			s.rateDev, s.extDev = 0.3*v.nrng.gauss(), 0.15*v.nrng.gauss()
			s.wander(&v.nrng, nat, math.Max(-1, math.Min(math.Log2(pitchHz(v.pitch)/middleC)/2, 1)), sr, false)
		}
		s.delay = 0
		if v.n > 1 {
			s.delay = int(v.rng.float() * 0.03 * sr) // they don't all start at once
		}
		if !v.active {
			s.g.phase = v.rng.float()
			s.g.mul, s.g.amp = 1, 1
		}
		a := (pos*spread + 1) * math.Pi / 4
		g := 1 / math.Sqrt(float64(v.n))
		s.gainL, s.gainR = g*math.Cos(a)*math.Sqrt2, g*math.Sin(a)*math.Sqrt2
	}
	v.update = 0
	v.active = true
}

func pitchHz(pitch float64) float64 { return 440 * math.Exp2((pitch-69)/12) }

// setSyllable makes the syllable i of the word the one sung now
func (v *Voice) setSyllable(i int) {
	s := v.cfg.Word[i]
	v.sylNow, v.sylVowel, v.consNow = i, s.Vowel, s.Consonant
	v.sylPh, v.sylTo = 0, 0
	if s.Diphthong > 0 && s.Diphthong < len(diphthongs) {
		v.sylPh, v.sylTo = diphthongs[s.Diphthong][0], diphthongs[s.Diphthong][1]
	}
}

// vowelBase returns the vowel to sing without the movement of the wheel and the matrix: the phoneme
// (0 for the scale a e i o u), the one it glides to, and the position (on the scale, or along Vowels)
func (v *Voice) vowelBase() (ph, to int, pos float64) {
	cfg := &v.cfg
	if v.sylNow >= 0 {
		ph, to, pos = v.sylPh, v.sylTo, v.sylVowel
	} else {
		ph, to, pos = cfg.Phoneme, cfg.PhonemeTo, cfg.Vowel
	}
	if ph > 0 {
		pos = float64(ph - 1)
	}
	return
}

// diphFrac is how far the glide of a diphthong has come (0-1)
func (v *Voice) diphFrac(sr float64) float64 {
	d := v.cfg.DiphthongTime
	if d <= 0 {
		d = 0.3
	}
	u := (float64(v.t-v.syllT0)/sr - v.cfg.DiphthongHold) / d
	if u <= 0 {
		return 0
	}
	if u >= 1 {
		return 1
	}
	return u * u * (3 - 2*u)
}

// setFormants tunes the tract to the vowel, the diphthong, the consonant the syllable began with, the
// pitch (the first formant follows it up when it is higher, as sopranos tune it), the singer's
// formant and the nose
func (v *Voice) setFormants(f0, wh float64) {
	cfg := &v.cfg
	sr := v.Config().SampleRate
	shift := cfg.FormantShift
	if shift <= 0 {
		shift = 1
	}
	f, bw := tractAt(v.rangeNow, v.mode, v.vowel)
	if v.phTo > 0 {
		if d := v.diphFrac(sr); d > 0 {
			fb, bb := tractAt(v.rangeNow, 1, float64(v.phTo-1))
			for k := range f {
				f[k] *= math.Pow(fb[k]/f[k], d)
				bw[k] += (bb[k] - bw[k]) * d
			}
		}
	}
	for k := range f {
		f[k] *= shift
		bw[k] *= shift
	}
	// the singer's formant: the third to fifth formants gather
	if tr := math.Max(0, math.Min(cfg.Training, 1)); tr > 0 {
		fs := voiceTypes[v.rangeNow].singerForm * shift
		f[2] += tr * 0.6 * (fs*0.97 - f[2])
		f[3] += tr * (fs*1.05 - f[3])
		f[4] += tr * (fs*1.2 - f[4])
		for k := 2; k < 5; k++ {
			bw[k] *= 1 - 0.3*tr
		}
	}
	ref, bwRef := f, bw // the tract as it is before it is tuned to the pitch
	// formant tuning: the first formant rises towards the pitch (a soft maximum, so it bends)
	if tn := tuningOf(cfg.Tuning); tn > 0 {
		const p = 6.0
		f[0] = math.Pow(math.Pow(f[0], p)+math.Pow(tn*f0*1.08, p), 1/p)
		bw[0] = math.Max(bw[0], 0.07*f[0]) // a high first formant is not a narrow one
		// it never climbs past the second formant (it would pile its gain on it): the upper ones make room
		for k := 1; k < 5; k++ {
			if f[k] < f[k-1]*1.12 {
				f[k] = f[k-1] * 1.12
				bw[k] = math.Max(bw[k], 0.07*f[k])
			}
		}
	}
	nasal := math.Max(0, math.Min(cfg.Nasal, 1))
	zero := 1000 * shift
	v.artTarget = 1
	if v.artT >= 0 {
		l := &consonants[v.consNow].loc
		coart := cfg.Coarticulation
		if coart <= 0 {
			coart = 0.045
		}
		trans := l.trans * coart / 0.045
		el := float64(v.artT) / sr
		u := math.Max(0, math.Min((el-l.hold)/trans, 1))
		if u >= 1 {
			v.artT = -1
		}
		u = u * u * (3 - 2*u)
		for k := range 3 {
			f[k] = l.f[k]*shift + (f[k]-l.f[k]*shift)*u
			bw[k] = l.bw[k]*shift + (bw[k]-l.bw[k]*shift)*u
			ref[k] = l.f[k]*shift + (ref[k]-l.f[k]*shift)*u
			bwRef[k] = l.bw[k]*shift + (bwRef[k]-l.bw[k]*shift)*u
		}
		if l.nasal > 0 {
			nasal = l.nasal*(1-u) + nasal*u
			zero = (l.zero*(1-u) + 1000*u) * shift
		}
		v.artTarget = l.amp*(1-u) + u
	}
	v.fnow = f
	chans := 1
	if v.cfg.NumChannels > 1 {
		chans = 2
	}
	t := &v.tract[0]
	for k := range f {
		t.f[k].set(f[k], bw[k], sr)
	}
	t.np.set(270*shift, 100*shift, sr)
	t.nz.set(zero, 120*shift, sr)
	t.nasal = nasal
	t.sub = math.Max(0, math.Min(cfg.Subglottal, 1))
	if t.sub > 0 {
		t.sp.set(600, 160, sr)
		t.sz.set(1500, 250, sr)
	}
	key := [4]float64{float64(cfg.Source), v.oq, v.sq, v.lfTa + v.lfRd}
	if key != v.harmKey {
		v.harmKey = key
		if cfg.Source == 1 {
			v.harm = sourceHarmonics(0, 0, &v.lf)
		} else {
			v.harm = sourceHarmonics(v.oq, v.sq, nil)
		}
	}
	// the level of the tract as tuned at this pitch against the untuned one at the middle pitch
	v.compN++
	if f == ref && bw == bwRef && t.nasal <= 0 {
		v.tuneGain = 1
		if v.compSnap {
			v.tuneNow, v.compSnap = 1, false
		}
	} else if v.compN%8 == 1 {
		tilt := 0.0
		if v.tiltCoef > 0 && v.tiltCoef < 1 {
			tilt = -math.Log(v.tiltCoef) * sr / (2 * math.Pi)
		}
		tu := newCascade(&f, &bw, t, tilt, &v.harm, sr)
		un := newCascade(&ref, &bwRef, t, tilt, &v.harm, sr)
		v.tuneGain = levelCompensation(un, tu, f0, sr, wh, cfg.Breath)
		if v.compSnap {
			v.tuneNow, v.compSnap = v.tuneGain, false
		}
	}
	for c := 1; c < chans; c++ {
		u := &v.tract[c]
		for k := range f {
			u.f[k].a, u.f[k].b, u.f[k].c = t.f[k].a, t.f[k].b, t.f[k].c
		}
		u.np.a, u.np.b, u.np.c = t.np.a, t.np.b, t.np.c
		u.nz.a, u.nz.b, u.nz.c = t.nz.a, t.nz.b, t.nz.c
		u.nasal = nasal
		u.sub = t.sub
		u.sp.a, u.sp.b, u.sp.c = t.sp.a, t.sp.b, t.sp.c
		u.sz.a, u.sz.b, u.sz.c = t.sz.a, t.sz.b, t.sz.c
	}
}

// tuningOf is how much a voice tunes its first formant: 0 is full, a negative value none
func tuningOf(t float64) float64 {
	switch {
	case t < 0:
		return 0
	case t == 0:
		return 1
	}
	return math.Min(t, 1)
}

// shape sets the glottal pulse and the tilt from the brightness of the voice: a soft voice is breathy
// (the folds open long, close slowly and the high partials fall away); a pressed one is bright
func (v *Voice) shape(bright, f0 float64) {
	cfg := &v.cfg
	sr := v.Config().SampleRate
	if cfg.Source == 1 {
		// Liljencrants-Fant: Rd from the brightness (a soft voice is breathy, 2.7; a pressed one 0.3), the
		// return time at least two samples so the closing never aliases; solved again only when it moves
		rd := cfg.Rd
		if rd <= 0 {
			rd = 0.3 + 2.4*math.Pow(1-bright, 1.6)
		}
		rd = math.Max(0.3, math.Min(rd, 2.7))
		tp, te, ta := lfRd(rd)
		ta = math.Max(ta, 2*f0/sr)
		if !v.lf.ok || math.Abs(rd-v.lfRd) > 1e-3 || math.Abs(ta-v.lfTa) > 0.02*ta {
			v.lf = lfSolve(tp, te, ta)
			v.lfRd, v.lfTa = rd, ta
		}
		v.tiltCoef = 0
		return
	}
	oq := cfg.OpenQuotient
	if oq <= 0 {
		oq = 0.85 - 0.5*bright
	}
	sq := cfg.SpeedQuotient
	if sq <= 0 {
		sq = 1 + 0.8*bright
	}
	v.oq, v.sq = math.Max(0.2, math.Min(oq, 0.95)), math.Max(0.8, math.Min(sq, 4))
	b := bright
	if cfg.ReturnPhase > 0 {
		b = 1 - cfg.ReturnPhase
	}
	v.tiltCoef = math.Exp(-2 * math.Pi * (300 + 4000*b*b) / sr)
}

// reshape sets the detune and the spread of the singers as the matrix moves them
func (v *Voice) reshape() {
	cfg := &v.cfg
	detune := cfg.Detune + v.mods.Mod(dDetune)
	spread := math.Max(0, math.Min(cfg.Spread+v.mods.Mod(dSpread), 1))
	for i := range v.n {
		s := &v.singers[i]
		s.detune = math.Exp2(s.cents(detune) / 1200)
		a := (s.pos*spread + 1) * math.Pi / 4
		g := 1 / math.Sqrt(float64(v.n))
		s.gainL, s.gainR = g*math.Cos(a)*math.Sqrt2, g*math.Sin(a)*math.Sqrt2
	}
}

// cents is how far the singer is out of tune when the choir is detuned over a spread of cents
func (s *singer) cents(detune float64) float64 { return s.pos*detune/2 + s.jitter*detune }

// startConsonant starts the consonant: its burst of noise and the formants moving from its locus
func (v *Voice) startConsonant() {
	v.consT, v.artT = -1, -1
	v.syllT0 = v.t
	c := v.consNow
	if c <= NoConsonant || c >= numConsonants {
		return
	}
	v.artT = 0
	sh := consonants[c]
	v.gateEnd = 0
	if sh.vot > 0 || sh.bar > 0 {
		v.gateEnd = int((math.Max(sh.vot+0.08, sh.dur+0.01)) * v.Config().SampleRate)
	}
	if sh.gain == 0 {
		return
	}
	v.cons.Clear()
	v.cons.SetFrequencyQ(math.Min(sh.freq, v.Config().SampleRate*0.45), sh.q)
	v.consT = 0
}

func smooth01(u float64) float64 {
	switch {
	case u <= 0:
		return 0
	case u >= 1:
		return 1
	}
	return u * u * (3 - 2*u)
}

// gate returns how the consonant shapes the voice at this instant: how much of the voicing sounds (a
// voiceless consonant keeps it out until its voice onset time), how far the voice bar of a voiced
// fricative hums in its place, and the level of the aspiration through the throat
func (v *Voice) gate(sr float64) (voicing, bar, asp float64) {
	voicing = 1
	el := v.t - v.syllT0
	if v.consNow <= NoConsonant || el < 0 || el >= v.gateEnd {
		return
	}
	sh := &consonants[v.consNow]
	t := float64(el) / sr
	if sh.vot > 0 {
		voicing = smooth01((t - sh.vot) / 0.008)
		if sh.asp > 0 {
			asp = sh.asp * math.Min(1, t/0.006)
			if t > sh.vot {
				asp *= math.Exp(-(t - sh.vot) / 0.02)
			}
		}
	}
	if sh.bar > 0 && t < sh.dur {
		bar = sh.bar * math.Min(1, t/0.01) * math.Min(1, (sh.dur-t)/0.03)
	}
	return
}

// consonant returns the next sample of the burst (0 once it is over)
func (v *Voice) consonant(sr float64) float64 {
	if v.consT < 0 {
		return 0
	}
	sh := consonants[v.consNow]
	t := float64(v.consT) / sr
	v.consT++
	if t >= sh.dur {
		v.consT = -1
		return 0
	}
	var env float64
	if sh.plosive {
		env = math.Exp(-t/sh.tau) * math.Min(1, t/0.0008)
	} else {
		env = math.Sin(math.Pi * t / sh.dur)
	}
	_, band, _ := v.cons.Tick(v.rng.float()*2 - 1)
	level := v.cfg.ConsonantLevel
	if level <= 0 {
		level = 0.5
	}
	return band * v.cons.K() * env * sh.gain * level
}

func (v *Voice) NoteOff() {
	v.filter.Release()
	v.released = true
	v.noteOff = 0
}

func (v *Voice) Clear() {
	v.filter.Clear()
	v.consT, v.artT = -1, -1
	v.active = false
	v.env = 0
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool {
	return v.active
}

func (v *Voice) Synthesize() {
	out := v.Output(0)
	out.Clear()
	if !v.active {
		return
	}
	cfg := &v.cfg
	n := v.Config().BufferSize
	sr := v.Config().SampleRate
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	gain := level * v.amp
	bright := math.Max(0, math.Min(cfg.Brightness+0.4*(v.velocity-0.5), 1))
	breath := math.Max(0, cfg.Breath)
	attack := 1 / math.Max(1, cfg.Attack*sr)
	release := math.Pow(10, -3/(math.Max(0.01, cfg.Release)*sr))
	glide := 1.0
	if cfg.VowelGlide > 0 {
		glide = 1 - math.Exp(-1/(cfg.VowelGlide*sr))
	}
	vibDelay := int(cfg.VibratoDelay * sr)
	driftCoef := 1 - math.Exp(-1/(0.15*sr))
	driftEvery := int(0.3 * sr)
	channels := out.Channels
	stereo := len(channels) > 1
	f0 := 440 * math.Exp2((v.pitch-69)/12)
	vowelInc := cfg.VowelRate / sr
	if cfg.VowelSync > 0 {
		vowelInc = v.BPM() / 60 / cfg.VowelSync / sr
	}
	moving := vowelInc > 0 && cfg.VowelDepth != 0
	sylInc := 0.0
	if len(cfg.Word) > 0 && cfg.WordSync > 0 {
		sylInc = v.BPM() / 60 / cfg.WordSync / sr
	}
	mods := v.mods
	modded := mods.Active()
	filtered := v.filter.On()
	tickFilter := filtered || (modded && mods.Uses(modmatrix.FilterEnv))
	lfo2Used := modded && mods.Uses(modmatrix.LFO2)
	brightModded := modded && mods.Targets(dBrightness)
	reshape := modded && (mods.Targets(dDetune) || mods.Targets(dSpread))
	if modded {
		mv := &mods.Values
		mv[modmatrix.Velocity] = v.velocity
		mv[modmatrix.Random] = v.random
		mv[modmatrix.Key] = math.Max(-1, math.Min(math.Log2(f0/middleC)/5, 1))
	}
	chans := 1
	if stereo {
		chans = 2
	}
	jitter, shimmer := cfg.Jitter, cfg.Shimmer
	// the natural vibrato, the whisper (voice and noise cross-fade with equal power), the voice bar
	nat := cfg.NaturalVibrato
	natKey := math.Max(-1, math.Min(math.Log2(f0/middleC)/2, 1))
	depth0 := cfg.VibratoDepth
	vibDelay2, vibRamp := vibDelay, 0.5*sr
	if nat > 0 {
		if depth0 <= 0 {
			depth0 = 0.35 * nat
		}
		if cfg.VibratoDelay <= 0 {
			vibDelay2 = int(0.25 * nat * sr)
		}
		vibRamp = (0.5 + 0.5*nat) * sr
	}
	whisperModded := modded && mods.Targets(dWhisper)
	wh := math.Max(0, math.Min(cfg.Whisper, 1))
	whPrev := -1.0
	gv, gn := 1.0, 0.0
	barK := 1 - math.Exp(-2*math.Pi*350/sr)
	for i := v.waitSamps; i < n; i++ {
		// the vowel glides; the formants follow every 32 samples
		v.expr.Tick(sr)
		bend := v.expr.Ratio()
		if v.sylNow >= 0 {
			// the word: a held note goes on to the next syllable on the clock
			if sylInc > 0 {
				if k := (v.syl + int(sylInc*float64(v.t))) % len(cfg.Word); k != v.sylNow {
					v.setSyllable(k)
					v.startConsonant()
				}
			}
		}
		ph, phTo, base := v.vowelBase()
		off := 2 * v.expr.Wheel() // the vowel's movement, in steps
		vib0 := depth0
		breathNow := breath
		pitchMul := 1.0
		bnow := bright
		if modded {
			mv := &mods.Values
			mv[modmatrix.AmpEnv] = v.env
			mv[modmatrix.FilterEnv] = v.env
			if filtered {
				mv[modmatrix.FilterEnv] = v.filter.Env()
			}
			mv[modmatrix.Wheel] = v.expr.Wheel()
			mv[modmatrix.LFO1] = math.Sin(2 * math.Pi * v.singers[0].vibPh)
			if lfo2Used {
				mv[modmatrix.LFO2] = math.Sin(2 * math.Pi * v.lfo2)
				v.lfo2 += 2 / sr
				if v.lfo2 >= 1 {
					v.lfo2--
				}
			}
			mods.Eval(sr)
			off += mods.Mod(dVowel)
			if whisperModded {
				wh = math.Max(0, math.Min(cfg.Whisper+mods.Mod(dWhisper), 1))
			}
			vib0 += mods.Mod(dVibratoDepth)
			breathNow = math.Max(0, breath+mods.Mod(dBreath))
			pitchMul = math.Exp2(mods.Mod(dPitch) / 12)
			if brightModded {
				bnow = math.Max(0, math.Min(cfg.Brightness+mods.Mod(dBrightness)+0.4*(v.velocity-0.5), 1))
			}
		}
		if moving {
			p := vowelInc * float64(v.t)
			if cfg.VowelSteps {
				if v.t == 0 || math.Floor(p) != math.Floor(p-vowelInc) {
					v.stepTo = (v.rng.float()*2 - 1) * cfg.VowelDepth // a new vowel every cycle
				}
				off += v.stepTo
			} else {
				off += cfg.VowelDepth * math.Sin(2*math.Pi*p)
			}
		}
		target := base + off
		if ph > 0 {
			target = math.Max(0, math.Min(target, float64(len(Vowels)-1)))
		}
		if ph != v.mode {
			v.mode, v.vowel = ph, target
		}
		v.phTo = phTo
		v.vowel += (target - v.vowel) * glide
		if v.update == 0 {
			v.shape(bnow, f0)
			v.setFormants(f0, wh)
			if reshape {
				v.reshape()
			}
		}
		v.update = (v.update + 1) % 32
		if v.artT >= 0 {
			v.artT++
		}
		v.artGain += (v.artTarget - v.artGain) * 0.003
		if v.tuneGain < v.tuneNow {
			v.tuneNow = v.tuneGain // a tract that swells is held down at once
		} else {
			v.tuneNow += (v.tuneGain - v.tuneNow) * 0.001
		}

		if wh != whPrev {
			whPrev = wh
			gv, gn = math.Cos(wh*math.Pi/2), math.Sin(wh*math.Pi/2)
		}
		// the vibrato comes in gently after its delay
		vib := 0.0
		if v.t > vibDelay2 {
			vib = vib0 * math.Min(1, float64(v.t-vibDelay2)/vibRamp)
		}
		var src [2]float64
		flow, sung := 0.0, 0
		voices := v.n
		if gv < 1e-6 {
			voices = 0 // a whisper has no voice to sing
		}
		for k := range voices {
			s := &v.singers[k]
			if v.t < s.delay {
				continue
			}
			rate, ext := s.vibRate, 1.0
			if nat > 0 {
				if (v.t+k*37)&63 == 0 {
					s.wander(&v.nrng, nat, natKey, sr, true)
				}
				rate, ext = s.natRate, s.natExt
			}
			s.vibPh += rate / sr
			if s.vibPh >= 1 {
				s.vibPh--
			}
			// the drift: a voice never holds a pitch quite still (5 cents or so)
			if (v.t+k*977)%driftEvery == 0 && !cfg.Steady {
				s.driftTo = (v.rng.float() - 0.5) * 0.1
			}
			s.drift += (s.driftTo - s.drift) * driftCoef
			semis := vib*ext*math.Sin(2*math.Pi*s.vibPh) + s.drift
			inc := f0 * bend * pitchMul * s.detune * (1 + semis*0.05776) / sr // 2^(x/12) for small x
			var x, fl float64
			if cfg.Source == 1 {
				x, fl = s.g.tickLF(inc, &v.lf, jitter, shimmer)
			} else {
				x, fl = s.g.tick(inc, v.oq, v.sq, jitter, shimmer)
			}
			flow += fl
			sung++
			if stereo {
				src[0] += x * s.gainL
				src[1] += x * s.gainR
			} else {
				src[0] += x * (s.gainL + s.gainR) / 2
			}
		}
		if sung > 0 {
			flow /= float64(sung)
		}
		// the envelope
		if !v.released {
			v.env = math.Min(1, v.env+attack)
		} else {
			v.env *= release
		}
		var y [2]float64
		air := 0.0
		if breathNow > 0 {
			// the breath is one, in the middle, and swells with the open folds
			air = (v.rng.float() - 0.5) * (0.42 + 0.98*flow)
		}
		// the consonant gates the voice (a voiceless one keeps it out until its voice onset time), may
		// hum it low (the voice bar of z, zh, v), and breathes aspiration through the throat; its
		// noise excites the throat too, and a whisper is turbulence at the glottis
		cn := v.consonant(sr)
		vg, barAmt, aspC := v.gate(sr)
		flowGate := (1-wh)*flow + wh*0.5
		for c := range chans {
			v.tilt[c] = (1-v.tiltCoef)*src[c]*(gv*vg) + v.tiltCoef*v.tilt[c]
			x := v.tilt[c]
			if barAmt > 0 {
				b := &v.bar[c]
				b[0] += barK * (x - b[0])
				b[1] += barK * (b[0] - b[1])
				x += barAmt * (b[1]*barGain - x)
			}
			if breathNow > 0 {
				v.breathLP[c] += (air - v.breathLP[c]) * 0.5
				x += v.breathLP[c] * breathNow * 0.6
			}
			if gn > 0 || aspC > 0 {
				nz := v.nrng.float()*2 - 1
				d := nz - 0.6*v.nPrev[c] // the lips radiate: the noise is differentiated too
				v.nPrev[c] = nz
				x += d * (gn*whisperGain*(0.35+0.65*flowGate) + aspC*aspGain)
			}
			x += cn * fricFeed
			y[c] = v.tract[c].tick(x) * outScale * v.artGain * v.tuneNow
		}
		g := gain
		if modded && mods.Targets(dLevel) {
			g = gain / level * math.Max(0, level+mods.Mod(dLevel))
		}
		if tickFilter {
			extra := 0.0
			if modded {
				extra = mods.Mod(dCutoff)
			}
			v.filter.Tick(f0*bend*pitchMul, extra, 0)
		}
		cons := cn * gain
		if stereo {
			if filtered {
				y[0], y[1] = v.filter.Process(0, y[0]*v.env*g), v.filter.Process(1, y[1]*v.env*g)
			} else {
				y[0], y[1] = y[0]*v.env*g, y[1]*v.env*g
			}
			channels[0][i] = float32(softClip(y[0] + cons))
			channels[1][i] = float32(softClip(y[1] + cons))
		} else {
			x := y[0] * v.env * g
			if filtered {
				x = v.filter.Process(0, x)
			}
			channels[0][i] = float32(softClip(x + cons))
		}
		v.t++
	}
	if v.noteOff > 0 {
		if left := n - v.waitSamps; v.noteOff > left {
			v.noteOff -= left
		} else {
			v.NoteOff()
		}
	}
	v.waitSamps = 0
	if v.released && v.env < 1e-4 && v.consT < 0 {
		v.active = false
	}
}

// NewPoly creates a set of voices. They share their place in the word (Config.Word), so a sequence
// of notes sings the text whichever voice each note lands on
func NewPoly(cfg Config, numVoices int) *poly.Poly[*Voice] {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	voices := make([]*Voice, numVoices)
	place := &wordPlace{}
	for i := range numVoices {
		voices[i] = New(cfg)
		voices[i].place = place
	}
	return poly.New[*Voice](cfg.NumChannels, voices)
}

// ParseWord turns a text into syllables: words split by spaces, dashes or commas, each a consonant
// (t k s sh f m n ng l r w y p b d g z zh v, or none) and a vowel (a e i o u, or a diphthong: ai au ei
// ou oi), so "ta ke so" is three syllables, "sha-fu" two and "mai" one that glides. The voiceless
// p t k are aspirated and s sh f keep the voice out; b d g start the voice at once and z zh v hum
// through their noise. A lone consonant or an unknown letter is an error
func ParseWord(text string) ([]Syllable, error) {
	vowels := map[string]float64{"a": 0, "e": 1, "i": 2, "o": 3, "u": 4}
	var word []Syllable
	for _, tok := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return r == ' ' || r == '-' || r == ',' || r == '\t' || r == '\n'
	}) {
		var c Consonant
		rest := tok
		// the longest name first (sh and ng before s and n)
		for l := 2; l >= 1 && c == NoConsonant; l-- {
			if len(rest) > l {
				for i, n := range ConsonantNames[1:] {
					if len(n) == l && rest[:l] == n {
						c, rest = Consonant(i+1), rest[l:]
						break
					}
				}
			}
		}
		syl := Syllable{Consonant: c}
		if vw, ok := vowels[rest]; ok {
			syl.Vowel = vw
		} else if d := diphthongOf(rest); d > 0 {
			syl.Vowel, syl.Diphthong = vowels[rest[:1]], d
		} else {
			return nil, fmt.Errorf("vocal: cannot sing %q", tok)
		}
		word = append(word, syl)
	}
	return word, nil
}

func diphthongOf(s string) int {
	for i := 1; i < len(DiphthongNames); i++ {
		if DiphthongNames[i] == s {
			return i
		}
	}
	return 0
}
