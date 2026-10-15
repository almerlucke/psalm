// Package vocal sings vowels: a buzzing source like the vocal folds, shaped by five formants (the
// resonances of the throat and mouth) of a bass, tenor, alto or soprano, with breath, vibrato and
// the small unsteadiness of a real voice. A choir is a voice sung by several singers at once, each a
// little out of tune and time, spread over the stereo field.
//
//	choir := vocal.NewPoly(vocal.Choir(), 8).AddTo(p)
//	choir.Receive(psalm.Message{"vowel": 3.0}, 0) // all sing "o"
package vocal

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
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

	Consonant      Consonant // a burst of noise as the note starts (none unless set)
	ConsonantLevel float64   // 0-1 (0 is treated as 0.5)

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
}

// The value inputs of a voice: 1 is the vowel and 2 the vibrato depth, 3 to 6 the values of the
// matrix (Ext 1 to 4) that a slot can move any setting by
const InExt1 = 3 // to InExt1+3

const middleC = 261.6255653005986

// Consonant is the burst of noise a note starts with
type Consonant int

const (
	NoConsonant Consonant = iota
	T
	K
	S
	Sh
	F
	numConsonants
)

// ConsonantNames are the names of the consonants, by number
var ConsonantNames = []string{"none", "t", "k", "s", "sh", "f"}

// consonantShape is a burst: noise through a band (freq, q), as long as dur seconds, either
// dying away at once (a plosive, with the time constant tau) or swelling and fading (a fricative)
type consonantShape struct {
	freq, q, dur, tau, gain float64
	plosive                 bool
}

var consonants = [...]consonantShape{
	T:  {freq: 5200, q: 1.2, dur: 0.05, tau: 0.01, gain: 3, plosive: true},
	K:  {freq: 2000, q: 2, dur: 0.07, tau: 0.018, gain: 3, plosive: true},
	S:  {freq: 7500, q: 1.5, dur: 0.12, gain: 2.5},
	Sh: {freq: 3500, q: 1, dur: 0.14, gain: 2.5},
	F:  {freq: 4500, q: 0.4, dur: 0.1, gain: 1.8},
}

// DefaultConfig is a solo voice
func DefaultConfig() Config {
	return Solo()
}

// formant is a resonance: its frequency, its level (dB) and its width (Hz)
type formant struct{ freq, db, bw float64 }

// the formants of the vowels a e i o u of each range (after the formant table of Csound)
var ranges = map[Range][5][5]formant{
	Bass: {
		{{600, 0, 60}, {1040, -7, 70}, {2250, -9, 110}, {2450, -9, 120}, {2750, -20, 130}},
		{{400, 0, 40}, {1620, -12, 80}, {2400, -9, 100}, {2800, -12, 120}, {3100, -18, 120}},
		{{250, 0, 60}, {1750, -30, 90}, {2600, -16, 100}, {3050, -22, 120}, {3340, -28, 120}},
		{{400, 0, 40}, {750, -11, 80}, {2400, -21, 100}, {2600, -20, 120}, {2900, -40, 120}},
		{{350, 0, 40}, {600, -20, 80}, {2400, -32, 100}, {2675, -28, 120}, {2950, -36, 120}},
	},
	Tenor: {
		{{650, 0, 80}, {1080, -6, 90}, {2650, -7, 120}, {2900, -8, 130}, {3250, -22, 140}},
		{{400, 0, 70}, {1700, -14, 80}, {2600, -12, 100}, {3200, -14, 120}, {3580, -20, 120}},
		{{290, 0, 40}, {1870, -15, 90}, {2800, -18, 100}, {3250, -20, 120}, {3540, -30, 120}},
		{{400, 0, 40}, {800, -10, 80}, {2600, -12, 100}, {2800, -12, 120}, {3000, -26, 120}},
		{{350, 0, 40}, {600, -20, 60}, {2700, -17, 100}, {2900, -14, 120}, {3300, -26, 120}},
	},
	Alto: {
		{{800, 0, 80}, {1150, -4, 90}, {2800, -20, 120}, {3500, -36, 130}, {4950, -60, 140}},
		{{400, 0, 60}, {1600, -24, 80}, {2700, -30, 120}, {3300, -35, 150}, {4950, -60, 200}},
		{{350, 0, 50}, {1700, -20, 100}, {2700, -30, 120}, {3700, -36, 150}, {4950, -60, 200}},
		{{450, 0, 70}, {800, -9, 80}, {2830, -16, 100}, {3500, -28, 130}, {4950, -55, 135}},
		{{325, 0, 50}, {700, -12, 60}, {2530, -30, 170}, {3500, -40, 180}, {4950, -64, 200}},
	},
	Soprano: {
		{{800, 0, 80}, {1150, -6, 90}, {2900, -32, 120}, {3900, -20, 130}, {4950, -50, 140}},
		{{350, 0, 60}, {2000, -20, 100}, {2800, -15, 120}, {3600, -40, 150}, {4950, -56, 200}},
		{{270, 0, 60}, {2140, -12, 90}, {2950, -26, 100}, {3900, -26, 120}, {4950, -44, 120}},
		{{450, 0, 40}, {800, -11, 80}, {2830, -22, 100}, {3800, -22, 120}, {4950, -50, 120}},
		{{325, 0, 50}, {700, -16, 60}, {2700, -35, 170}, {3800, -40, 180}, {4950, -60, 200}},
	},
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

// singer is one of the voices of a note: its own source, a little out of tune and time
type singer struct {
	osc     *blosc.Blosc
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
	formants  [2][5]*svf.SVF
	gains     [5]float64
	vowel     float64 // the vowel sung, gliding to cfg.Vowel
	tilt      [2]float64
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
	consT     int // samples into the consonant (-1 none)
}

// randSource is a small fast random generator
type randSource struct{ s uint64 }

func (r *randSource) float() float64 {
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
	v := &Voice{Module: psalm.NewModule(nil, []int{cfg.NumChannels}), cfg: cfg, pitch: 60, vowel: cfg.Vowel}
	sr := v.Config().SampleRate
	for i := range v.singers {
		v.singers[i].osc = blosc.New(220, sr, 0, 0.5, 1, blosc.Sawtooth)
	}
	for c := range v.formants {
		for k := range v.formants[c] {
			v.formants[c][k] = svf.New(1000, 0, sr)
		}
	}
	v.rng = &randSource{s: v.Rand().Uint64() | 1}
	v.filter = voicefilter.New(cfg.Filter, sr, cfg.NumChannels)
	v.cons = svf.New(1000, 0, sr)
	v.consT = -1
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
	msg.IfIntVal("consonant", func(c int) { v.cfg.Consonant = Consonant(max(0, min(c, int(numConsonants)-1))) })
	msg.IfFltVal("consonantLevel", func(f float64) { v.cfg.ConsonantLevel = math.Max(0, math.Min(f, 1)) })
	envelope.Seconds(msg, "amp", &v.cfg.Attack, nil, &v.cfg.Release)
	msg.IfFltVal("frequency", func(f float64) {
		if f > 0 {
			v.pitch = 69 + 12*math.Log2(f/440)
		}
	})
	msg.IfFltVal("pitch", func(p float64) { v.pitch = p })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("vowel", func(f float64) { v.cfg.Vowel = f })
	msg.IfFltVal("vowelGlide", func(f float64) { v.cfg.VowelGlide = f })
	msg.IfFltVal("brightness", func(f float64) { v.cfg.Brightness = f })
	msg.IfFltVal("breath", func(f float64) { v.cfg.Breath = f })
	msg.IfFltVal("vibratoRate", func(f float64) { v.cfg.VibratoRate = f })
	msg.IfFltVal("vibratoDepth", func(f float64) { v.cfg.VibratoDepth = f })
	msg.IfIntVal("range", func(r int) { v.cfg.Range = Range(r) })
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
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
	v.startConsonant()
	if !v.active {
		v.env = 0
		v.vowel = cfg.Vowel
		for c := range v.formants {
			for _, f := range v.formants[c] {
				f.Clear()
			}
			v.tilt[c], v.breathLP[c] = 0, 0
		}
	}
	v.rangeNow = cfg.Range
	if v.rangeNow == Auto {
		v.rangeNow = rangeOf(v.pitch)
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
		s.delay = 0
		if v.n > 1 {
			s.delay = int(v.rng.float() * 0.03 * sr) // they don't all start at once
		}
		if !v.active {
			s.osc.SetPhase(v.rng.float())
		}
		a := (pos*spread + 1) * math.Pi / 4
		g := 1 / math.Sqrt(float64(v.n))
		s.gainL, s.gainR = g*math.Cos(a)*math.Sqrt2, g*math.Sin(a)*math.Sqrt2
	}
	v.update = 0
	v.active = true
}

// setFormants tunes the formants to the vowel (the first follows the pitch up when it is higher,
// as singers do)
func (v *Voice) setFormants(f0 float64) {
	table := ranges[v.rangeNow]
	pos := math.Max(0, math.Min(v.vowel, 4))
	i := min(int(pos), 3)
	frac := pos - float64(i)
	sr := v.Config().SampleRate
	for k := range 5 {
		a, b := table[i][k], table[i+1][k]
		freq := a.freq * math.Pow(b.freq/a.freq, frac)
		if k == 0 {
			freq = math.Max(freq, f0*1.1)
		}
		bw := a.bw + (b.bw-a.bw)*frac
		v.gains[k] = math.Pow(10, (a.db+(b.db-a.db)*frac)/20)
		freq = math.Min(freq, sr*0.45)
		for c := range v.formants {
			v.formants[c][k].SetFrequencyQ(freq, freq/bw)
		}
	}
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

// startConsonant starts the burst of the consonant
func (v *Voice) startConsonant() {
	v.consT = -1
	c := v.cfg.Consonant
	if c <= NoConsonant || c >= numConsonants {
		return
	}
	sh := consonants[c]
	v.cons.Clear()
	v.cons.SetFrequencyQ(math.Min(sh.freq, v.Config().SampleRate*0.45), sh.q)
	v.consT = 0
}

// consonant returns the next sample of the burst (0 once it is over)
func (v *Voice) consonant(sr float64) float64 {
	if v.consT < 0 {
		return 0
	}
	sh := consonants[v.cfg.Consonant]
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
	v.consT = -1
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
	// the source's tilt: a soft voice loses its high partials sooner
	tiltCoef := math.Exp(-2 * math.Pi * (300 + 4000*bright*bright) / sr)
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
	for i := v.waitSamps; i < n; i++ {
		// the vowel glides; the formants follow every 32 samples
		v.expr.Tick(sr)
		bend := v.expr.Ratio()
		target := cfg.Vowel + 2*v.expr.Wheel()
		vib0 := cfg.VibratoDepth
		breathNow := breath
		pitchMul := 1.0
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
			target += mods.Mod(dVowel)
			vib0 += mods.Mod(dVibratoDepth)
			breathNow = math.Max(0, breath+mods.Mod(dBreath))
			pitchMul = math.Exp2(mods.Mod(dPitch) / 12)
			if brightModded {
				b := math.Max(0, math.Min(cfg.Brightness+mods.Mod(dBrightness)+0.4*(v.velocity-0.5), 1))
				tiltCoef = math.Exp(-2 * math.Pi * (300 + 4000*b*b) / sr)
			}
		}
		if moving {
			ph := vowelInc * float64(v.t)
			if cfg.VowelSteps {
				if v.t == 0 || math.Floor(ph) != math.Floor(ph-vowelInc) {
					v.stepTo = (v.rng.float()*2 - 1) * cfg.VowelDepth // a new vowel every cycle
				}
				target += v.stepTo
			} else {
				target += cfg.VowelDepth * math.Sin(2*math.Pi*ph)
			}
		}
		v.vowel += (target - v.vowel) * glide
		if v.update == 0 {
			v.setFormants(f0)
			if reshape {
				v.reshape()
			}
		}
		v.update = (v.update + 1) % 32

		// the vibrato comes in gently after its delay
		vib := 0.0
		if v.t > vibDelay {
			vib = vib0 * math.Min(1, float64(v.t-vibDelay)/(0.5*sr))
		}
		var src [2]float64
		for k := range v.n {
			s := &v.singers[k]
			if v.t < s.delay {
				continue
			}
			s.vibPh += s.vibRate / sr
			if s.vibPh >= 1 {
				s.vibPh--
			}
			// the drift: a voice never holds a pitch quite still (5 cents or so)
			if (v.t+k*977)%driftEvery == 0 && !cfg.Steady {
				s.driftTo = (v.rng.float() - 0.5) * 0.1
			}
			s.drift += (s.driftTo - s.drift) * driftCoef
			semis := vib*math.Sin(2*math.Pi*s.vibPh) + s.drift
			s.osc.SetFrequency(f0*bend*pitchMul*s.detune*(1+semis*0.05776), sr) // 2^(x/12) for small x
			x := s.osc.Tick(0, false, 0)
			if stereo {
				src[0] += x * s.gainL
				src[1] += x * s.gainR
			} else {
				src[0] += x * (s.gainL + s.gainR) / 2
			}
		}
		// the envelope
		if !v.released {
			v.env = math.Min(1, v.env+attack)
		} else {
			v.env *= release
		}
		var y [2]float64
		chans := 1
		if stereo {
			chans = 2
		}
		air := 0.0
		if breathNow > 0 {
			air = v.rng.float() - 0.5 // the breath is one, in the middle
		}
		for c := range chans {
			v.tilt[c] = (1-tiltCoef)*src[c] + tiltCoef*v.tilt[c]
			x := v.tilt[c]
			if breathNow > 0 {
				v.breathLP[c] += (air - v.breathLP[c]) * 0.5
				x += v.breathLP[c] * breathNow * 0.6
			}
			for k, f := range v.formants[c] {
				_, band, _ := f.Tick(x)
				y[c] += band * f.K() * v.gains[k]
			}
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
		cons := v.consonant(sr) * gain
		if stereo {
			if filtered {
				y[0], y[1] = v.filter.Process(0, y[0]*v.env*g), v.filter.Process(1, y[1]*v.env*g)
			} else {
				y[0], y[1] = y[0]*v.env*g, y[1]*v.env*g
			}
			channels[0][i] = float32(y[0] + cons)
			channels[1][i] = float32(y[1] + cons)
		} else {
			x := y[0] * v.env * g
			if filtered {
				x = v.filter.Process(0, x)
			}
			channels[0][i] = float32(x + cons)
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

// NewPoly creates a set of voices
func NewPoly(cfg Config, numVoices int) *poly.Poly[*Voice] {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	voices := make([]*Voice, numVoices)
	for i := range numVoices {
		voices[i] = New(cfg)
	}
	return poly.New[*Voice](cfg.NumChannels, voices)
}
