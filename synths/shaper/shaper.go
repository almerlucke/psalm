// Package shaper is a synth that builds its wave from a chain of shapers (genny's float/shape, see
// blocks/shapechain): a phase from 0 to 1 goes in, the chain bends it into a wave. Up to four
// chains (A to D) are written as text and the synth morphs from one to the next as its morph goes
// from 0 to 1, so the wave itself moves: an envelope, an LFO, the velocity or the mod wheel can
// pull the morph and four macros (m1 to m4 in the chains) that set the numbers inside them. Two
// lines (octave, detune, level, a morph of their own) go through the filter every synth has
// (blocks/voicefilter) and an amp envelope.
//
//	cfg := shaper.Glass()
//	cfg.Chains[1] = "bipolar | mirror -0.4 0.4 | mult 2.5 | sin"  // chain B
//	v := shaper.NewPoly(cfg, 8).AddTo(p)
//
// The shaping is nonlinear, so it makes harmonics far above the pitch: the synth reads the chain
// several times a sample (Oversample) and filters them away before they fold back as aliasing.
package shaper

import (
	"math"
	"math/rand/v2"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/oversample"
	"github.com/almerlucke/psalm/blocks/shapechain"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/modules/poly"
)

// NumChains is how many chains a sound morphs through
const NumChains = 4

// ChainKeys are the messages that set a chain (a string, or a *shapechain.Chain read already), A to D
var ChainKeys = [NumChains]string{"chainA", "chainB", "chainC", "chainD"}

// ChainKeys2 are the messages that set the chains of line 2 (see Config.Chains2)
var ChainKeys2 = [NumChains]string{"chain2A", "chain2B", "chain2C", "chain2D"}

// lineChains are the chains a line reads: the ones from A on that are there (n)
type lineChains struct {
	c [NumChains]*shapechain.Chain
	n int
}

// Oversamples are the factors the chains are read at, by the number of the message "oversample": 0 is
// automatic, then 1 (the chain read once a sample), 2, 4 and 8 times
var Oversamples = [...]int{0, 1, 2, 4, 8}

// DefaultChain is what a chain A that is empty or can't be read is
const DefaultChain = "sin"

// Line is an oscillator: the same chains read at its pitch and its own place in the morph
type Line struct {
	Octave int
	Detune float64 // cents
	Level  float64 // line 1: 0 is 1; line 2: 0 is silent
	Morph  float64 // -1 to 1: added to the morph of the voice
}

// Config is a sound
type Config struct {
	NumChannels int
	// Chains are the chains written as text, A first. The ones from the first empty one on are not
	// used; a morph goes through the ones that are
	Chains [NumChains]string
	// Chains2 are the chains of line 2, written as text like Chains. A line 2 with none (Chains2[0] empty, the default) reads the chains
	// above: the same wave at its own pitch and place in the morph; with chains of its own it is another oscillator (a different wave to
	// fatten the sound, to modulate or to ring against line 1). The morph and the macros are the same for both
	Chains2 [NumChains]string
	Morph   float64                       // 0-1: 0 is chain A, 1 the last chain, in between a blend of two neighbours
	Macros  [shapechain.NumMacros]float64 // 0-1: m1 to m4 in the chains
	Line1   Line
	Line2   Line

	// Unison stacks copies of each line, spread in pitch by Spread cents (the outer copies that far under and over) and
	// across the stereo field by UniStereo (0-1); 1 is one copy, up to MaxUnison. Drift (0-1) lets each copy, the
	// morph and the cutoff wander slowly and at random, as the oscillators of an analog synth do. Sub is the level (0-1)
	// of a sine an octave under the note, Noise the level (0-1) of white noise, which NoiseColor (-1 to 1) makes dark
	// (under 0) or bright (over 0). The unison is taken when a note starts on a voice that is silent
	Unison     int
	Spread     float64
	UniStereo  float64
	Drift      float64
	Sub        float64
	Noise      float64
	NoiseColor float64

	// XMod lets line 1 act on line 2 (0 is off): XModType 0 bends the phase of line 2 by the wave of line 1 (an FM sound; the
	// amount is the index, up to a whole turn, and the octave and detune of the lines the ratio), 1 multiplies the two (a ring
	// modulator, the amount blends it in). XModSilent hides line 1 itself: it only modulates. The amount is a destination
	// of the matrix ("xmod")
	XMod       float64
	XModType   int
	XModSilent bool

	// Oversample is how many times a sample the chains are read: 1 (off), 2, 4 or 8. 0 is automatic: a chain that fits under half
	// the sample rate at the pitch is read as it is, and one that doesn't is read from tables (see tables.go), which hold only the
	// harmonics that fit
	Oversample int
	// StereoMorph (-1 to 1) reads the left and the right channel at different places in the morph: the left that much under the morph and the
	// right that much over it, so a centred sound is wide (a moving image: the morph envelope and the matrix move both). StereoMacros do the same
	// for the macros m1 to m4 (a fold that is deeper on the right). It works with the automatic oversampling (Oversample 0), where the
	// chains are read from tables: two tables a line instead of one
	StereoMorph  float64
	StereoMacros [shapechain.NumMacros]float64
	Width        float64 // 0-1: line 1 to the left, line 2 to the right
	Level        float64 // output level (0 is treated as 1)
	// Trim is dB on top of Level for the mix: it is not a part of a preset (loading one leaves it), so a script can
	// load a preset and set where it sits
	Trim float64

	AmpEnv adsr.Setting
	// MorphEnv moves the morph over the note by MorphAmount (-1 to 1) at its peak; ShapeEnv moves
	// macro m1 by ShapeAmount. An amount of 0 leaves them off
	MorphEnv    adsr.Setting
	MorphAmount float64
	ShapeEnv    adsr.Setting
	ShapeAmount float64

	// PitchEnv bends the pitch over the note by PitchAmount semitones at its peak (a kick that falls, a
	// laser, a riser); off at 0. Glide is the milliseconds a note takes to slide from the pitch the voice
	// was at (for a synth played mono: Mono, Legato)
	PitchEnv    adsr.Setting
	PitchAmount float64
	Glide       float64
	Mono        bool
	Legato      bool

	VibratoRate  float64 // Hz
	VibratoDepth float64 // semitones
	VibratoDelay float64 // seconds
	// the LFOs of the matrix, a sine from the start of every note (LFO1: Hz, or beats a cycle with
	// LFO1Sync, following the tempo), and a second one a sine (LFO2Shape 0) or a triangle (1)
	LFO1Rate  float64
	LFO1Sync  float64
	LFO2Rate  float64
	LFO2Shape int

	Filter voicefilter.Config // after the lines (off unless its Type is set)

	Mods [modmatrix.Slots]modmatrix.Slot // the matrix: a source to a destination (see Dests) over an amount
	Ext  [modmatrix.Exts]float64         // the values 0-1 of the sources Ext 1 to 4 (cabled in on InExt1...)
}

// MaxUnison is the most copies of a line a unison stacks
const MaxUnison = 7

// MaxAuto is the most an automatic oversample goes to
const MaxAuto = 4

// oversampleFactor is the factor a Config's setting stands for: 0 stays 0 (automatic), others the
// nearest of 1, 2, 4, 8 below
func oversampleFactor(n int) int {
	switch {
	case n <= 0:
		return 0
	case n >= 8:
		return 8
	case n >= 4:
		return 4
	case n >= 2:
		return 2
	}
	return 1
}

// The destinations of the matrix (see Dests), by number
const (
	dMorph = iota
	dMacro1
	dMacro2
	dMacro3
	dMacro4
	dPitch
	dVibrato
	dWidth
	dLevel
	dDetune
	dCutoff
	dResonance
	dXmod
	numDests
)

// Dests are the settings the matrix of a voice moves: by their keys, as described (Describe), with
// how far a full source moves them (semitones, octaves, cents, or the setting itself). The cutoff and
// the resonance are those of the filter, and do nothing while it is off
var Dests = []modmatrix.Dest{
	dMorph:     {Key: "morph", Label: "morph", Span: 1},
	dMacro1:    {Key: "macro1", Label: "macro 1", Span: 1},
	dMacro2:    {Key: "macro2", Label: "macro 2", Span: 1},
	dMacro3:    {Key: "macro3", Label: "macro 3", Span: 1},
	dMacro4:    {Key: "macro4", Label: "macro 4", Span: 1},
	dPitch:     {Key: "pitch", Label: "pitch", Span: 12},
	dVibrato:   {Key: "vibratoDepth", Label: "vibrato", Span: 1},
	dWidth:     {Key: "width", Label: "width", Span: 1},
	dLevel:     {Key: "level", Label: "level", Span: 1},
	dDetune:    {Key: "detune2", Label: "detune 2", Span: 100},
	dCutoff:    {Key: "cutoff", Label: "cutoff", Span: 6},
	dResonance: {Key: "resonance", Label: "resonance", Span: 1},
	dXmod:      {Key: "xmod", Label: "cross mod", Span: 1},
}

// The value inputs of a voice: 1 sets the morph, 5 to 8 the values of the matrix (Ext 1 to 4)
const (
	InMorph = 1
	InExt1  = 5 // to InExt1+3
)

// DefaultConfig is a glassy sound
func DefaultConfig() Config { return Glass() }

// Voice plays a note
type Voice struct {
	*psalm.Module
	cfg      Config
	own      [2]lineChains // the chains (A to D) and the chains of line 2 (none: line 2 reads the first, see lc)
	freq     float64
	velocity float64
	amp      float64
	phase    [2 * MaxUnison]float64                                   // of each copy of each line (line*MaxUnison + copy)
	os       [2 * MaxUnison][len(Oversamples)]*oversample.Oversampler // by factor, made when one is first needed
	osReady  [2 * MaxUnison][len(Oversamples)]bool                    // its history holds the wave just before now
	osFactor int                                                      // how many times it reads the chain now
	amped    *adsr.ADSR
	morphEnv *adsr.ADSR
	shapeEnv *adsr.ADSR
	pitchEnv *adsr.ADSR
	pitchMul float64 // 2^(the bend in semitones / 12), worked out every 4 samples
	fBlend   float64 // how much of the filter is in the sound: 1, or less while a filter that was off fades in
	pending  *pendingNote
	dcEst    [2]float64 // the mean of the wave of each line, taken off before the envelope (see Synthesize)
	glideOff float64    // semitones the glide is still away from the pitch of the note
	// what the sound is now: the morph, the macros, the levels and the width glide to what they are set to
	// in a few milliseconds, so changing them under a sounding note (a script does it every step) never
	// makes a click; snap says the next sample starts there (a new note)
	smMorph  float64
	smMacros shapechain.Macros
	smLvl    [2]float64
	smGain   float64
	smWidth  float64
	snap     bool
	// a chain changed under a sounding note is faded to over xfLen samples: the old ones are kept to
	// fade out
	old       [2]lineChains // what each line read before the swap
	xfLeft    int
	xf        float64
	lastFreq  float64 // the pitch the voice last sounded at, 0 for none
	t         int
	waitSamps int
	dc, dcIn  [2]float64
	expr      expression.Expression
	filter    *voicefilter.Filter
	mods      *modmatrix.Matrix
	random    float64
	lfo1Ph    float64
	lfo2Ph    float64
	// the unison of the note: the copies of a line, their pan gains and pitch ratios
	uni        int
	smUniSt    float64
	gL, gR     [2 * MaxUnison]float64
	subPh      float64
	smSub      float64
	smNoise    float64
	noiseRng   uint64
	noiseLP    [2]float64
	drifts     []drifter
	copyDrift  [2 * MaxUnison]float64 // the pitch drift of each copy, a ratio
	voiceDrift [3]float64             // morph, cutoff (octaves), pitch (ratio)
	// the tables of the lines (see tables.go), used when table says so; bwEff is the harmonics the chains hold
	tables  [2]lineTables
	tablesR [2]lineTables // the tables of the right channel when the sound is stereo (StereoMorph, StereoMacros)
	stereo  bool
	tR      struct { // the table the right channel reads now, the one it fades from, and how far it is
		cur, prev *waveTable
		w         float64
	}
	rawR      [MaxUnison]float64 // what the right channel reads, for each copy of a line
	table     bool
	bwEff     [2]int // the harmonics the chains of each line hold
	macroUsed [shapechain.NumMacros]bool
	scratch   tableScratch
	smXmod    float64
	openCfg   voicefilter.Config // the open filter that a filter switched off under a note goes to, openIn samples from now
	openIn    int
	swapGen   int // counts the swaps of a chain under a note: a table fades to the new chains
}

// drifter is a slow random wander in -1 to 1: a new target a few times a second, eased to
type drifter struct{ cur, next, ph, inc float64 }

func (d *drifter) init(r *rand.Rand, lo, hi float64) {
	d.cur, d.next, d.ph = r.Float64()*2-1, r.Float64()*2-1, r.Float64()
	d.inc = lo + (hi-lo)*r.Float64()
}

// step moves it on by n samples' worth (inc is per sample) and gives its value
func (d *drifter) step(r *rand.Rand, n float64) float64 {
	d.ph += d.inc * n
	if d.ph >= 1 {
		d.ph -= math.Floor(d.ph)
		d.cur = d.next
		d.next = r.Float64()*2 - 1
	}
	e := d.ph * d.ph * (3 - 2*d.ph)
	return d.cur + (d.next-d.cur)*e
}

// New returns a voice
func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	if cfg.AmpEnv == (adsr.Setting{}) {
		cfg.AmpEnv = *adsr.NewSetting(1, 5, 0.8, 200, 1, 300)
	}
	if cfg.MorphEnv == (adsr.Setting{}) {
		cfg.MorphEnv = *adsr.NewSetting(1, 5, 0.5, 400, 1, 300)
	}
	if cfg.ShapeEnv == (adsr.Setting{}) {
		cfg.ShapeEnv = *adsr.NewSetting(1, 5, 0.5, 400, 1, 300)
	}
	if cfg.PitchEnv == (adsr.Setting{}) {
		cfg.PitchEnv = *adsr.NewSetting(1, 2, 0, 80, 1, 100)
	}
	cfg.Oversample = oversampleFactor(cfg.Oversample)
	cfg.Unison = max(1, min(cfg.Unison, MaxUnison))
	v := &Voice{Module: psalm.NewModule(nil, []int{cfg.NumChannels}), cfg: cfg, freq: 220, pitchMul: 1}
	sr := v.Config().SampleRate
	v.amped = adsr.New(&v.cfg.AmpEnv, adsr.NoteOff, sr)
	v.morphEnv = adsr.New(&v.cfg.MorphEnv, adsr.NoteOff, sr)
	v.shapeEnv = adsr.New(&v.cfg.ShapeEnv, adsr.NoteOff, sr)
	v.pitchEnv = adsr.New(&v.cfg.PitchEnv, adsr.NoteOff, sr)
	v.filter = voicefilter.New(cfg.Filter, sr, cfg.NumChannels)
	v.mods = modmatrix.New(Dests)
	for i, sl := range cfg.Mods {
		v.mods.Set(i, sl)
	}
	for i, e := range cfg.Ext {
		v.mods.SetExt(i, e)
	}
	for i, text := range cfg.Chains {
		v.setChain(0, i, text)
	}
	for i, text := range cfg.Chains2 {
		v.setChain(1, i, text)
	}
	v.pickFactor()
	v.SetSelf(v)
	return v
}

// lc is the chains a line reads: line 2 has its own when it has any, else it reads the first set
func (v *Voice) lc(k int) *lineChains {
	if k == 1 && v.own[1].n > 0 {
		return &v.own[1]
	}
	return &v.own[0]
}

// setChain sets a chain of a set (0: A to D, 1: the chains of line 2) from text or a chain read already. Text that can't be read leaves the chain as
// it was (chain A that has none is a sine; an empty first chain of line 2 gives it back the chains of line 1)
func (v *Voice) setChain(set, i int, chain any) {
	var c *shapechain.Chain
	cur := &v.own[set]
	switch x := chain.(type) {
	case *shapechain.Chain:
		c = x
	case string:
		if x == "" {
			c = nil
			if i == 0 && set == 0 {
				c = shapechain.MustParse(DefaultChain)
			}
		} else {
			var err error
			if c, err = shapechain.Parse(x); err != nil {
				if cur.c[i] != nil || i != 0 || set == 1 {
					return
				}
				c = shapechain.MustParse(DefaultChain)
			}
		}
	default:
		return
	}
	if c != cur.c[i] && v.t > 0 && v.own[0].n > 0 && v.IsActive() {
		if v.xfLeft == 0 { // fade from the chains as they are
			v.old = [2]lineChains{*v.lc(0), *v.lc(1)}
		}
		v.xfLeft = int(chainFade * v.Config().SampleRate)
		v.swapGen++
	}
	cur.c[i] = c
	texts := &v.cfg.Chains
	if set == 1 {
		texts = &v.cfg.Chains2
	}
	if c != nil {
		texts[i] = c.Text()
	} else {
		texts[i] = ""
	}
	cur.n = 0
	for _, ch := range cur.c {
		if ch == nil {
			break
		}
		cur.n++
	}
}

// pickFactor works out how many times a sample the chains are read: what the config says, or for
// the automatic one the least that keeps the harmonics of the chains (Bandwidth) that lie above
// the audible band from folding back, at the pitch of the voice (and a little over it, for vibrato
// and bend): 1 while the highest harmonic is under half the sample rate, then 2, then 4
func (v *Voice) pickFactor() {
	f := v.cfg.Oversample
	if f == 0 {
		sr := v.Config().SampleRate
		top := v.freq * 1.12 * math.Exp2(v.cfg.Spread/1200)
		for k := range 2 {
			top = max(top, v.freq*1.12*math.Exp2(float64(v.line(k).Octave)+(v.line(k).Detune+v.cfg.Spread)/1200))
		}
		f = 1
		for _, set := range v.own {
			for i := 0; i < set.n; i++ {
				reach := top * float64(set.c[i].Bandwidth()) // the highest frequency with something in it
				need := 1
				switch {
				case reach <= sr/2:
				case reach <= 2*sr-20000:
					need = 2
				default:
					need = MaxAuto
				}
				f = max(f, need)
			}
		}
	}
	v.bwEff = [2]int{1, 1}
	v.macroUsed = [shapechain.NumMacros]bool{}
	for _, set := range v.own {
		for i := 0; i < set.n; i++ {
			for m := range v.macroUsed {
				v.macroUsed[m] = v.macroUsed[m] || set.c[i].Uses(m+1)
			}
		}
	}
	for k := range v.bwEff {
		lc := v.lc(k)
		for i := 0; i < lc.n; i++ {
			v.bwEff[k] = max(v.bwEff[k], lc.c[i].Bandwidth())
		}
	}
	// from tables: when the chains need more than one time a sample, and for a unison, which reads one table for all its copies
	v.stereo = v.cfg.Oversample == 0 && v.cfg.StereoMorph != 0 || v.cfg.Oversample == 0 && v.cfg.StereoMacros != [shapechain.NumMacros]float64{}
	v.table = v.cfg.Oversample == 0 && (f > 1 || v.cfg.Unison > 1 || v.stereo)
	if f != v.osFactor {
		v.osFactor = f
		for k := range v.os {
			for i, o := range v.os[k] {
				if o != nil {
					o.Reset()
				}
				v.osReady[k][i] = false
			}
		}
	}
}

// oversampler is the up and down sampler of a line at the factor now, and whether its history is the wave
// just before now (it is not for one just made, or after the factor changed under a note: prime it)
func (v *Voice) oversampler(line int) (*oversample.Oversampler, *bool) { // line: the copy, line*MaxUnison + j
	idx := 0
	for i, f := range Oversamples {
		if f == v.osFactor {
			idx = i
		}
	}
	if v.os[line][idx] == nil {
		v.os[line][idx] = oversample.New(v.osFactor, 16)
	}
	return v.os[line][idx], &v.osReady[line][idx]
}

// apply makes the settings of a config the voice's: the chains (faded to if it is sounding), the envelopes, the
// lines, the filter, the matrix, the LFOs. What sounds goes on: the settings glide, the envelopes are of the
// next note
func (v *Voice) apply(c Config) {
	for i, text := range c.Chains {
		v.setChain(0, i, text)
	}
	for i, text := range c.Chains2 {
		v.setChain(1, i, text)
	}
	cfg := &v.cfg
	cfg.Morph, cfg.Macros, cfg.Line1, cfg.Line2, cfg.Width, cfg.Level = c.Morph, c.Macros, c.Line1, c.Line2, c.Width, c.Level
	cfg.StereoMorph, cfg.StereoMacros = max(-1, min(c.StereoMorph, 1)), c.StereoMacros
	cfg.Oversample = oversampleFactor(c.Oversample)
	cfg.Unison, cfg.Spread, cfg.UniStereo = max(1, min(c.Unison, MaxUnison)), c.Spread, c.UniStereo
	cfg.Drift, cfg.Sub, cfg.Noise, cfg.NoiseColor = c.Drift, c.Sub, c.Noise, c.NoiseColor
	cfg.XMod, cfg.XModType, cfg.XModSilent = c.XMod, c.XModType, c.XModSilent
	if c.AmpEnv != (adsr.Setting{}) {
		cfg.AmpEnv = c.AmpEnv
	}
	if c.MorphEnv != (adsr.Setting{}) {
		cfg.MorphEnv = c.MorphEnv
	}
	if c.ShapeEnv != (adsr.Setting{}) {
		cfg.ShapeEnv = c.ShapeEnv
	}
	if c.PitchEnv != (adsr.Setting{}) {
		cfg.PitchEnv = c.PitchEnv
	}
	cfg.MorphAmount, cfg.ShapeAmount, cfg.PitchAmount, cfg.Glide = c.MorphAmount, c.ShapeAmount, c.PitchAmount, c.Glide
	cfg.VibratoRate, cfg.VibratoDepth, cfg.VibratoDelay = c.VibratoRate, c.VibratoDepth, c.VibratoDelay
	cfg.LFO1Rate, cfg.LFO1Sync, cfg.LFO2Rate, cfg.LFO2Shape = c.LFO1Rate, c.LFO1Sync, c.LFO2Rate, c.LFO2Shape
	sounding := v.t > 0 && v.IsActive()
	wasOn := v.filter.On()
	v.openIn = 0
	if c.Filter.Type == voicefilter.Off && wasOn && sounding {
		// a filter switched off under a note steps: leave it on, as it is while the chains of the old sound fade out (the
		// edges of a saw would come through an open filter), then open it (it glides)
		c.Filter = v.cfg.Filter
		v.openCfg = c.Filter
		v.openCfg.Cutoff, v.openCfg.Resonance, v.openCfg.EnvAmount, v.openCfg.KeyTrack = 20000, 0, 0, 0
		v.openIn = int(chainFade*v.Config().SampleRate) + 1
	}
	v.filter.SetConfig(c.Filter)
	if !wasOn && v.filter.On() && sounding {
		v.filter.Snap() // a filter that comes in under a note starts empty and where it is going, and is crossfaded in quickly
	}
	cfg.Filter = v.filter.Config()
	for i, sl := range c.Mods {
		v.mods.Set(i, sl)
		cfg.Mods[i] = v.mods.Slot(i)
	}
	for i, e := range c.Ext {
		v.mods.SetExt(i, e)
		cfg.Ext[i] = v.mods.Ext(i)
	}
	v.pickFactor()
}

// setFreq changes the pitch of the voice; with a glide the sound slides there from where it was
func (v *Voice) setFreq(f float64) {
	if f <= 0 {
		return
	}
	if v.cfg.Glide > 0 && v.lastFreq > 0 {
		v.glideOff += 12 * math.Log2(v.freq/f)
	}
	v.freq = f
}

// lineTable is the table a line reads for a morph and macros: made when it has none, again when the wave or the pitch has moved from what the
// current one was made for (the sound fades from the old to the new), and the table it fades from while it does (nil when it is done)
func (v *Voice) lineTable(lt *lineTables, k int, pos float64, macros *shapechain.Macros, fTop float64) (cur, prev *waveTable, w float64) {
	switch {
	case !lt.valid:
		v.buildTable(lt, k, pos, macros, fTop)
	case lt.gen != v.swapGen: // a chain was swapped: fade to the table of the new chains as the voice does
		v.rebuildTable(lt, k, pos, macros, fTop, true)
	case v.t%4 == 0 && v.tableDue(lt, k, pos, macros, fTop):
		v.rebuildTable(lt, k, pos, macros, fTop, false)
	}
	w = 1
	if lt.blend < 1 {
		lt.blend = min(1, lt.blend+lt.blendInc)
		w = lt.blend
		prev = &lt.buf[lt.prev]
	}
	return &lt.buf[lt.cur], prev, w
}

// side is the morph and the macros a channel reads (-1 the left, 1 the right) when the sound is stereo: that much under or over
// the morph, a half of StereoMorph and of each of StereoMacros each way
func (v *Voice) side(pos float64, ch float64) (float64, shapechain.Macros) {
	m := v.smMacros
	for i := range m {
		m[i] = clamp01(m[i] + ch*0.5*v.cfg.StereoMacros[i])
	}
	return clamp01(pos + ch*0.5*v.cfg.StereoMorph), m
}

func (v *Voice) line(i int) *Line {
	if i == 0 {
		return &v.cfg.Line1
	}
	return &v.cfg.Line2
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	// a preset by its name: every setting of it (but the mono play of a synth, which is the Poly's), before the
	// other settings of the message, which then change it
	if name, ok := msg.StrVal("preset"); ok {
		if f, ok := Presets[name]; ok {
			v.apply(f())
		}
	}
	v.expr.Receive(msg)
	v.filter.Receive(msg)
	v.cfg.Filter = v.filter.Config()
	envelope.Receive(msg, "amp", &v.cfg.AmpEnv)
	envelope.Receive(msg, "morphEnv", &v.cfg.MorphEnv)
	envelope.Receive(msg, "shapeEnv", &v.cfg.ShapeEnv)
	envelope.Receive(msg, "pitchEnv", &v.cfg.PitchEnv)
	for i, key := range ChainKeys {
		if c, ok := msg[key]; ok {
			v.setChain(0, i, c)
		}
	}
	for i, key := range ChainKeys2 {
		if c, ok := msg[key]; ok {
			v.setChain(1, i, c)
		}
	}
	msg.IfFltVal("frequency", func(f float64) { v.setFreq(f) })
	msg.IfFltVal("pitch", func(p float64) { v.setFreq(440 * math.Exp2((p-69)/12)) })
	msg.IfFltVal("pitchAmount", func(f float64) { v.cfg.PitchAmount = max(-60, min(f, 60)) })
	msg.IfFltVal("glide", func(f float64) { v.cfg.Glide = max(0, min(f, 3000)) })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = max(0, min(f, 1)) })
	msg.IfFltVal("morph", func(f float64) { v.cfg.Morph = max(0, min(f, 1)) })
	msg.IfFltVal("stereoMorph", func(f float64) { v.cfg.StereoMorph = max(-1, min(f, 1)) })
	for i := range v.cfg.StereoMacros {
		msg.IfFltVal([]string{"stereoMacro1", "stereoMacro2", "stereoMacro3", "stereoMacro4"}[i], func(f float64) { v.cfg.StereoMacros[i] = max(-1, min(f, 1)) })
	}
	for i := range v.cfg.Macros {
		msg.IfFltVal([]string{"macro1", "macro2", "macro3", "macro4"}[i], func(f float64) { v.cfg.Macros[i] = max(0, min(f, 1)) })
	}
	for i, n := range []string{"1", "2"} {
		l := v.line(i)
		msg.IfIntVal("octave"+n, func(o int) { l.Octave = max(-4, min(o, 4)) })
		msg.IfFltVal("detune"+n, func(f float64) { l.Detune = f })
		msg.IfFltVal("level"+n, func(f float64) { l.Level = max(0, f) })
		msg.IfFltVal("morph"+n, func(f float64) { l.Morph = max(-1, min(f, 1)) })
	}
	msg.IfFltVal("detune", func(f float64) { v.cfg.Line2.Detune = f }) // line 2, as on the other synths with two
	msg.IfIntVal("oversample", func(o int) { v.cfg.Oversample = Oversamples[max(0, min(o, len(Oversamples)-1))] })
	msg.IfIntVal("unison", func(n int) { v.cfg.Unison = max(1, min(n, MaxUnison)) })
	msg.IfFltVal("spread", func(f float64) { v.cfg.Spread = max(0, min(f, 100)) })
	msg.IfFltVal("uniStereo", func(f float64) { v.cfg.UniStereo = clamp01(f) })
	msg.IfFltVal("drift", func(f float64) { v.cfg.Drift = clamp01(f) })
	msg.IfFltVal("sub", func(f float64) { v.cfg.Sub = clamp01(f) })
	msg.IfFltVal("noise", func(f float64) { v.cfg.Noise = clamp01(f) })
	msg.IfFltVal("noiseColor", func(f float64) { v.cfg.NoiseColor = max(-1, min(f, 1)) })
	msg.IfFltVal("xmod", func(f float64) { v.cfg.XMod = clamp01(f) })
	msg.IfIntVal("xmodType", func(n int) { v.cfg.XModType = max(0, min(n, 1)) })
	msg.IfBoolVal("xmodSilent", func(b bool) { v.cfg.XModSilent = b })
	msg.IfFltVal("width", func(f float64) { v.cfg.Width = f })
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
	msg.IfFltVal("trim", func(f float64) { v.cfg.Trim = max(-48, min(f, 24)) })
	msg.IfFltVal("morphAmount", func(f float64) { v.cfg.MorphAmount = max(-1, min(f, 1)) })
	msg.IfFltVal("shapeAmount", func(f float64) { v.cfg.ShapeAmount = max(-1, min(f, 1)) })
	msg.IfFltVal("vibratoRate", func(f float64) { v.cfg.VibratoRate = f })
	msg.IfFltVal("vibratoDepth", func(f float64) { v.cfg.VibratoDepth = f })
	msg.IfFltVal("lfo1Rate", func(f float64) { v.cfg.LFO1Rate = f })
	msg.IfFltVal("lfo2Rate", func(f float64) { v.cfg.LFO2Rate = f })
	msg.IfIntVal("lfo2Shape", func(n int) { v.cfg.LFO2Shape = max(0, min(n, 1)) })
	v.mods.Receive(msg)
	for i := range v.cfg.Mods {
		v.cfg.Mods[i] = v.mods.Slot(i)
	}
	for i := range v.cfg.Ext {
		v.cfg.Ext[i] = v.mods.Ext(i)
	}
	v.pickFactor() // the chains, or the factor, may have changed
}

// Receive takes messages, or the morph (0-1) on input 1, and on 5 to 8 (InExt1...) values 0-1 that
// the matrix takes as sources Ext 1 to 4
func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}
	switch index {
	case InMorph:
		v.cfg.Morph = max(0, min(psalm.FltVal[float64](value), 1))
	case InExt1, InExt1 + 1, InExt1 + 2, InExt1 + 3:
		v.mods.SetExt(index-InExt1, psalm.FltVal[float64](value))
		v.cfg.Ext[index-InExt1] = v.mods.Ext(index - InExt1)
	}
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(adsr.NoteOff, 0, amplitude, message, tick)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(adsr.Duration, duration, amplitude, message, tick)
}

func (v *Voice) note(mode adsr.ReleaseMode, duration, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	if v.t > 0 && v.IsActive() && tick.SampleOffset > 0 {
		v.pending = &pendingNote{at: tick.SampleOffset, mode: mode, duration: duration, amplitude: amplitude, message: message}
		return
	}
	v.waitSamps = tick.SampleOffset
	v.startNote(mode, duration, amplitude, message)
}

// startNote starts a note: the settings of it, the envelopes, the wave from its start. A note that starts on a voice
// that sounds goes on from the sound it was: the phase, the envelope level (the envelope starts from where it is) and
// the settings glide, so a mono synth's next note never clicks
func (v *Voice) startNote(mode adsr.ReleaseMode, duration, amplitude float64, message psalm.Message) {
	fresh := !(v.t > 0 && v.IsActive())
	v.velocity = 0.8
	v.ReceiveMessage(message)
	v.osReady = [2 * MaxUnison][len(Oversamples)]bool{} // the history of the oversamplers is the last note's: fill it again
	v.pickFactor()                                      // the pitch of the note
	v.amp = amplitude
	v.t = 0
	v.lfo1Ph, v.lfo2Ph = 0, 0
	if fresh {
		v.snap = true
		v.xfLeft, v.xf = 0, 0
		v.fBlend = 0
		if v.filter.On() { // a note starts with its filter in
			v.fBlend = 1
		}
		// the unison of the note; the first copy of a line starts the wave at the same place every time, the others
		// anywhere (a unison that starts in phase is a single, loud oscillator for a moment)
		v.uni = max(1, min(v.cfg.Unison, MaxUnison))
		v.phase = [2 * MaxUnison]float64{}
		v.subPh = 0
		v.tables[0].valid, v.tables[1].valid, v.tablesR[0].valid, v.tablesR[1].valid = false, false, false, false
		v.openIn = 0
		if v.uni > 1 {
			r := v.Rand()
			for k := range 2 {
				for j := 1; j < v.uni; j++ {
					v.phase[k*MaxUnison+j] = r.Float64()
				}
			}
		}
	}
	if v.uni == 0 {
		v.uni = 1
	}
	// drawn only when a slot uses it: a voice that doesn't keeps the random stream it had
	if v.mods.Uses(modmatrix.Random) {
		v.random = v.Rand().Float64()*2 - 1
	}
	v.filter.Trigger(duration*1000, mode)
	if v.cfg.PitchAmount != 0 {
		v.pitchEnv.TriggerFull(duration*1000, 1, &v.cfg.PitchEnv, mode)
	}
	v.amped.TriggerFull(duration*1000, 1, &v.cfg.AmpEnv, mode)
	if v.cfg.MorphAmount != 0 {
		v.morphEnv.TriggerFull(duration*1000, 1, &v.cfg.MorphEnv, mode)
	}
	if v.cfg.ShapeAmount != 0 {
		v.shapeEnv.TriggerFull(duration*1000, 1, &v.cfg.ShapeEnv, mode)
	}
}

func (v *Voice) NoteOff() {
	v.filter.Release()
	v.amped.Release()
	v.morphEnv.Release()
	v.shapeEnv.Release()
	v.pitchEnv.Release()
}

func (v *Voice) Clear() {
	v.filter.Clear()
	v.amped.Clear()
	v.morphEnv.Clear()
	v.shapeEnv.Clear()
	v.pitchEnv.Clear()
	v.glideOff, v.lastFreq = 0, 0
	v.xfLeft, v.xf = 0, 0
	v.pending = nil
	v.phase = [2 * MaxUnison]float64{}
	v.subPh = 0
	v.tables[0].valid, v.tables[1].valid, v.tablesR[0].valid, v.tablesR[1].valid = false, false, false, false
	v.noiseLP = [2]float64{}
	for _, line := range v.os {
		for _, o := range line {
			if o != nil {
				o.Reset()
			}
		}
	}
	v.dc, v.dcIn = [2]float64{}, [2]float64{}
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool { return !v.amped.Done() }

// wave is the chains at a phase, blended at a place in the morph (0-1 over the chains in use); while a
// chain is being swapped, the chains it replaces fade out
func (v *Voice) wave(k int, ph, pos float64, m *shapechain.Macros) float64 {
	lc := v.lc(k)
	y := waveOf(&lc.c, lc.n, ph, pos, m)
	if v.xf > 0 {
		o := &v.old[k]
		y += (waveOf(&o.c, o.n, ph, pos, m) - y) * v.xf
	}
	return y
}

func waveOf(chains *[NumChains]*shapechain.Chain, n int, ph, pos float64, m *shapechain.Macros) float64 {
	if n <= 1 {
		return chains[0].Eval(ph, m)
	}
	f := pos * float64(n-1)
	i := int(f)
	if i >= n-1 {
		return chains[n-1].Eval(ph, m)
	}
	a := chains[i].Eval(ph, m)
	frac := f - float64(i)
	if frac <= 0 {
		return a
	}
	return a + (chains[i+1].Eval(ph, m)-a)*frac
}

// chainFade is the seconds a chain that is swapped under a sounding note takes to fade in
const chainFade = 0.012

// filterFade is the seconds a filter that is switched on under a note takes to come in
const filterFade = 0.002

// dcPoints is how many points of a cycle the mean of a wave is worked out from at the start of a note
const dcPoints = 128

// dcTrackHz is how fast the mean of the wave of a line follows it
const dcTrackHz = 3

// glideTime is the seconds the morph, the macros, the levels and the width take to reach a new setting
const glideTime = 0.005

// silent is the level of the amp envelope (-80 dB) below which a line makes no sound
const silent = 1e-4

func orOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// pendingNote is a note that starts in the middle of a buffer on a voice that is still sounding
type pendingNote struct {
	at                  int
	mode                adsr.ReleaseMode
	duration, amplitude float64
	message             psalm.Message
}

func (v *Voice) Synthesize() {
	out := v.Output(0)
	out.Clear()
	n := v.Config().BufferSize
	if p := v.pending; p != nil {
		// a note starting on a voice that sounds (a mono synth's next note): the old one sounds until the
		// sample the new one starts at, and does not stop at the start of the buffer
		v.pending = nil
		at := max(0, min(p.at, n))
		v.render(out, 0, at)
		v.startNote(p.mode, p.duration, p.amplitude, p.message)
		v.render(out, at, n)
		v.waitSamps = 0
		return
	}
	v.render(out, v.waitSamps, n)
	v.waitSamps = 0
}

// render makes the samples from one to another of the buffer
func (v *Voice) render(out *psalm.Socket, from, to int) {
	if !v.IsActive() {
		return
	}
	cfg := &v.cfg
	sr := v.Config().SampleRate
	level := orOne(cfg.Level) * math.Pow(10, cfg.Trim/20)
	mods := v.mods
	modded := mods.Active()
	morphModded := modded && mods.Targets(dMorph)
	var macroModded [shapechain.NumMacros]bool
	for i := range macroModded {
		macroModded[i] = modded && mods.Targets(dMacro1+i)
	}
	pitchModded := modded && mods.Targets(dPitch)
	vibModded := modded && mods.Targets(dVibrato)
	levelModded := modded && mods.Targets(dLevel)
	detuneModded := modded && mods.Targets(dDetune)
	xmodModded := modded && mods.Targets(dXmod)
	filtered := v.filter.On()
	filterTick := filtered || (modded && mods.Uses(modmatrix.FilterEnv))
	morphEnvOn := cfg.MorphAmount != 0
	shapeEnvOn := cfg.ShapeAmount != 0
	pitchEnvOn := cfg.PitchAmount != 0
	glideCoef := 0.0
	if cfg.Glide > 0 {
		glideCoef = 1 - math.Exp(-1/(cfg.Glide*0.001*sr))
	}
	gain := level * v.amp
	smooth := 1 - math.Exp(-1/(glideTime*sr))
	dcTrack := 2 * math.Pi * dcTrackHz / sr
	pan := func(p float64) (float64, float64) {
		a := (p + 1) * math.Pi / 4
		return math.Cos(a) * math.Sqrt2, math.Sin(a) * math.Sqrt2
	}
	// the copies of the unison: where each lies between -1 and 1, its pitch ratio from the spread
	uni := max(1, v.uni)
	var offs, spreadRatio [MaxUnison]float64
	for j := range uni {
		if uni > 1 {
			offs[j] = 2*float64(j)/float64(uni-1) - 1
		}
		spreadRatio[j] = 1
		if cfg.Spread > 0 {
			spreadRatio[j] = math.Exp2(offs[j] * cfg.Spread / 1200)
		}
	}
	uniNorm := 1 / math.Sqrt(float64(uni))
	// setPans gives every copy its gains: its line to the left or the right by the width, and spread by the unison's stereo
	setPans := func(width, spread float64) {
		for k := range 2 {
			base := width
			if k == 0 {
				base = -width
			}
			for j := range uni {
				v.gL[k*MaxUnison+j], v.gR[k*MaxUnison+j] = pan(max(-1, min(base+offs[j]*spread, 1)))
			}
		}
	}
	panned, pannedU := v.smWidth, v.smUniSt // where the width is now (a new note starts it again)
	setPans(panned, pannedU)
	driftOn := cfg.Drift > 0
	if driftOn && v.drifts == nil {
		r := v.Rand()
		v.drifts = make([]drifter, 3+2*MaxUnison)
		for i := range v.drifts {
			lo, hi := 0.2, 0.7
			if i < 3 {
				lo, hi = 0.12, 0.4
			}
			v.drifts[i].init(r, lo/sr, hi/sr)
		}
	}
	if !driftOn {
		v.voiceDrift = [3]float64{0, 0, 1}
		for i := range v.copyDrift {
			v.copyDrift[i] = 1
		}
	}
	// the noise: white, made dark (a low pass) or bright (a high pass) by its colour
	noiseOn := cfg.Noise > 0 || v.smNoise > 1e-4
	var noiseA, noiseDark float64 // the one pole's coefficient, and the make-up gain of dark noise
	if noiseOn {
		if v.noiseRng == 0 {
			v.noiseRng = v.Rand().Uint64() | 1
		}
		fc := 20000.0
		if cfg.NoiseColor < 0 {
			fc = 250 * math.Pow(80, 1+cfg.NoiseColor)
			noiseDark = min(6, math.Pow(20000/fc, 0.35))
		} else if cfg.NoiseColor > 0 {
			fc = 250 * math.Pow(32, cfg.NoiseColor)
		}
		noiseA = 1 - math.Exp(-2*math.Pi*fc/sr)
	}
	vibDelay := int(cfg.VibratoDelay * sr)
	lfo1Inc := cfg.LFO1Rate / sr
	if cfg.LFO1Sync > 0 {
		lfo1Inc = v.BPM() / 60 / cfg.LFO1Sync / sr
	}
	lfo1On := lfo1Inc > 0 && modded && mods.Uses(modmatrix.LFO1)
	lfo2On := modded && mods.Uses(modmatrix.LFO2) && cfg.LFO2Rate > 0
	lfo2Inc := cfg.LFO2Rate / sr
	dcCoef := 1 - 2*math.Pi*10/sr
	var ratio [2]float64
	for i := range ratio {
		l := v.line(i)
		ratio[i] = math.Exp2(float64(l.Octave) + l.Detune/1200)
	}
	if modded {
		mods.Values[modmatrix.Velocity] = v.amp
		mods.Values[modmatrix.Random] = v.random
		mods.Values[modmatrix.Key] = max(-1, min(math.Log2(v.freq/130.81/2)/5, 1))
	}
	factor := v.osFactor
	invFactor := 1 / float64(factor)
	var lfo1, lfo2, morphEnv, shapeEnv float64
	channels := out.Channels
	stereo := len(channels) > 1
	for i := from; i < to; i++ {
		v.expr.Tick(sr)
		dca := v.amped.Generate()
		if morphEnvOn {
			morphEnv = v.morphEnv.Generate() * cfg.MorphAmount
		}
		if shapeEnvOn {
			shapeEnv = v.shapeEnv.Generate() * cfg.ShapeAmount
		}
		if lfo1On {
			lfo1 = shapechain.SinTurns(v.lfo1Ph)
			v.lfo1Ph += lfo1Inc
			v.lfo1Ph -= math.Floor(v.lfo1Ph)
		}
		if lfo2On {
			if cfg.LFO2Shape == 1 {
				t := v.lfo2Ph + 0.25
				t -= math.Floor(t)
				lfo2 = 1 - 4*math.Abs(t-0.5)
			} else {
				lfo2 = shapechain.SinTurns(v.lfo2Ph)
			}
			v.lfo2Ph += lfo2Inc
			v.lfo2Ph -= math.Floor(v.lfo2Ph)
		}
		if modded {
			mv := &mods.Values
			mv[modmatrix.FilterEnv], mv[modmatrix.AmpEnv] = v.filter.Env(), dca
			mv[modmatrix.LFO1], mv[modmatrix.LFO2] = lfo1, lfo2
			mv[modmatrix.Wheel] = v.expr.Wheel()
			mods.Eval(sr)
		}
		if driftOn && v.t%4 == 0 {
			// the pitch of each copy, the morph and the cutoff wander (a ratio from cents: 1 cent is 5.776e-4)
			r := v.Rand()
			v.voiceDrift[0] = v.drifts[0].step(r, 4) * 0.04 * cfg.Drift
			v.voiceDrift[1] = v.drifts[1].step(r, 4) * 0.25 * cfg.Drift
			v.voiceDrift[2] = 1 + v.drifts[2].step(r, 4)*5*cfg.Drift*5.776e-4
			for k := range 2 {
				for j := range uni {
					c := k*MaxUnison + j
					v.copyDrift[c] = 1 + v.drifts[3+c].step(r, 4)*8*cfg.Drift*5.776e-4
				}
			}
		}
		morph := cfg.Morph + morphEnv + v.voiceDrift[0]
		if morphModded {
			morph += mods.Mod(dMorph)
		}
		var macros shapechain.Macros
		for k := range macros {
			macros[k] = cfg.Macros[k]
			if k == 0 {
				macros[k] += shapeEnv
			}
			if macroModded[k] {
				macros[k] += mods.Mod(dMacro1 + k)
			}
			macros[k] = clamp01(macros[k])
		}
		// the targets for this sample, and the sound gliding to them
		gainT := gain
		if levelModded {
			gainT = max(0, level+mods.Mod(dLevel)) * v.amp
		}
		widthT := max(0, min(cfg.Width+mods.Mod(dWidth), 1))
		xmodT := cfg.XMod
		if xmodModded {
			xmodT += mods.Mod(dXmod)
		}
		xmodT = clamp01(xmodT)
		lvlT := [2]float64{orOne(cfg.Line1.Level), max(0, cfg.Line2.Level)}
		if cfg.XModSilent && xmodT > 1e-4 {
			lvlT[0] = 0 // line 1 is heard only in line 2
		}
		if v.snap {
			v.snap = false
			v.smMorph, v.smMacros, v.smGain, v.smWidth, v.smLvl = morph, macros, gainT, widthT, lvlT
			v.smUniSt, v.smSub, v.smNoise, v.smXmod = cfg.UniStereo, cfg.Sub, cfg.Noise, xmodT
			panned, pannedU = widthT, v.smUniSt
			setPans(panned, pannedU)
			// the mean of the wave of each line over one cycle: a lopsided wave (a narrow pulse) is not
			// centred, and an envelope on it would switch the offset on and off with every note, a thump
			for k := 0; k < 2; k++ {
				pos := clamp01(v.smMorph + v.line(k).Morph)
				sum := 0.0
				for j := 0; j < dcPoints; j++ {
					sum += max(-2, min(v.wave(k, float64(j)/dcPoints, pos, &v.smMacros), 2))
				}
				v.dcEst[k] = sum / dcPoints
			}
		} else {
			v.smMorph += (morph - v.smMorph) * smooth
			for k := range macros {
				v.smMacros[k] += (macros[k] - v.smMacros[k]) * smooth
			}
			v.smGain += (gainT - v.smGain) * smooth
			v.smWidth += (widthT - v.smWidth) * smooth
			v.smLvl[0] += (lvlT[0] - v.smLvl[0]) * smooth
			v.smLvl[1] += (lvlT[1] - v.smLvl[1]) * smooth
			v.smUniSt += (cfg.UniStereo - v.smUniSt) * smooth
			v.smXmod += (xmodT - v.smXmod) * smooth
			v.smSub += (cfg.Sub - v.smSub) * smooth
			v.smNoise += (cfg.Noise - v.smNoise) * smooth
			if v.t%4 == 0 && (math.Abs(v.smWidth-panned) > 1e-5 || math.Abs(v.smUniSt-pannedU) > 1e-5) {
				panned, pannedU = v.smWidth, v.smUniSt
				setPans(panned, pannedU)
			}
		}
		if v.xfLeft > 0 {
			v.xf = float64(v.xfLeft) / (chainFade * sr)
			v.xfLeft--
		} else {
			v.xf = 0
		}
		semis := v.expr.Semitones()
		if pitchEnvOn {
			semis += v.pitchEnv.Generate() * cfg.PitchAmount
		}
		if v.glideOff != 0 {
			if v.glideOff *= 1 - glideCoef; math.Abs(v.glideOff) < 1e-4 || glideCoef == 0 {
				v.glideOff = 0
			}
			semis += v.glideOff
		}
		if pitchModded {
			semis += mods.Mod(dPitch)
		}
		if vd := cfg.VibratoDepth + mods.Mod(dVibrato); vd != 0 && v.t > vibDelay && (cfg.VibratoDepth != 0 || vibModded) {
			semis += vd * shapechain.SinTurns(cfg.VibratoRate*float64(v.t-vibDelay)/sr)
		}
		f := v.freq
		if semis != 0 {
			// the pitch moves slowly, a fourth of a millisecond at a time is plenty (Exp2 is not cheap)
			if v.t%4 == 0 {
				v.pitchMul = math.Exp2(semis / 12)
			}
			f *= v.pitchMul
		}
		f *= v.voiceDrift[2]
		if detuneModded {
			ratio[1] = math.Exp2(float64(cfg.Line2.Octave) + (cfg.Line2.Detune+mods.Mod(dDetune))/1200)
		}
		var o [2]float64
		// the cross modulation: what line 1 is this sample (the mean of its copies, no level, no envelope) acts on line 2
		xmodOn := v.smXmod > 1e-4 && v.smLvl[1] > 1e-4
		modSig := 0.0
		for k := 0; k < 2; k++ {
			l := v.line(k)
			lvl := v.smLvl[k]
			if lvl < 1e-4 && !(k == 0 && xmodOn) {
				continue
			}
			base := f * ratio[k] / sr
			if dca < silent { // the end of a long release: nothing to hear, nothing to work out
				for j := range uni {
					ph := v.phase[k*MaxUnison+j] + base*spreadRatio[j]*v.copyDrift[k*MaxUnison+j]
					v.phase[k*MaxUnison+j] = ph - math.Floor(ph)
				}
				continue
			}
			pos := clamp01(v.smMorph + l.Morph)
			// from tables: made when the line has none, and again when the wave or the pitch has moved from what the
			// current one was made for; the sound fades from the old table to the new one
			w := 1.0
			var tcur, tprev *waveTable
			stereoLine := false // the right channel reads tables of its own (v.tR, v.rawR)
			if v.table {
				lt := &v.tables[k]
				if lt.valid && lt.gen == v.swapGen && lt.blend >= 1 && v.t&3 != 0 && !v.stereo {
					tcur = &lt.buf[lt.cur] // nothing to make or fade: the usual sample
				} else {
					fTop := f * ratio[k] * spreadRatio[uni-1] * 1.01
					if v.stereo {
						posL, macL := v.side(pos, -1)
						tcur, tprev, w = v.lineTable(lt, k, posL, &macL, fTop)
						posR, macR := v.side(pos, 1)
						v.tR.cur, v.tR.prev, v.tR.w = v.lineTable(&v.tablesR[k], k, posR, &macR, fTop)
						stereoLine = true
					} else {
						tcur, tprev, w = v.lineTable(lt, k, pos, &v.smMacros, fTop)
					}
				}
			}
			pm := 0.0 // the phase of line 2 bent by line 1, in turns
			if k == 1 && xmodOn && cfg.XModType == 0 {
				pm = v.smXmod * modSig
			}
			var raw [MaxUnison]float64
			mean := 0.0
			for j := range uni {
				c := k*MaxUnison + j
				inc := base * spreadRatio[j] * v.copyDrift[c]
				ph := v.phase[c]
				rd := ph // where the wave is read
				if pm != 0 {
					rd = ph + pm
					rd -= math.Floor(rd)
				}
				var s float64
				switch {
				case tcur != nil:
					s = tcur.read(rd)
					if tprev != nil {
						s = tprev.read(rd)*(1-w) + s*w
					}
					if stereoLine {
						sr := v.tR.cur.read(rd)
						if v.tR.prev != nil {
							sr = v.tR.prev.read(rd)*(1-v.tR.w) + sr*v.tR.w
						}
						v.rawR[j] = sr
					}
				case factor == 1:
					s = v.wave(k, rd, pos, &v.smMacros)
				default:
					step := inc * invFactor
					os, ready := v.oversampler(c)
					// the sampler delays by its filter: read the wave that far ahead and what comes out is the
					// wave of now, as the direct reading (factor 1) gives it, so the factor can change under a note
					ahead := os.DownDelay() * inc
					if !*ready {
						for b := 20; b > 0; b-- { // fill its history with the wave just before now
							pb := rd + ahead - float64(b)*inc
							os.Down(func(j int) float64 {
								p := pb + float64(j)*step
								return v.wave(k, p-math.Floor(p), pos, &v.smMacros)
							})
						}
						*ready = true
					}
					s = os.Down(func(j int) float64 {
						p := rd + ahead + float64(j)*step
						return v.wave(k, p-math.Floor(p), pos, &v.smMacros)
					})
				}
				if s != s { // NaN: a chain that divided by nothing
					s = 0
				}
				raw[j] = s
				mean += s
				ph += inc
				v.phase[c] = ph - math.Floor(ph)
			}
			if tcur != nil {
				v.dcEst[k] = 0 // a table has no offset
			} else {
				v.dcEst[k] += (mean/float64(uni) - v.dcEst[k]) * dcTrack // the mean moves with the macros, slowly
			}
			if k == 0 && xmodOn {
				modSig = max(-1.5, min((mean/float64(uni))-v.dcEst[0], 1.5))
			}
			amp := dca * lvl * uniNorm
			ring := k == 1 && xmodOn && cfg.XModType == 1
			for j := range uni {
				s := raw[j] - v.dcEst[k]
				if s > 2 {
					s = 2
				} else if s < -2 {
					s = -2
				}
				if ring {
					s *= 1 - v.smXmod + v.smXmod*1.6*modSig
				}
				c := k*MaxUnison + j
				if !stereoLine {
					o[0] += s * amp * v.gL[c]
					o[1] += s * amp * v.gR[c]
					continue
				}
				sr := max(-2, min(v.rawR[j]-v.dcEst[k], 2)) // the right channel reads another wave (stereo)
				if ring {
					sr *= 1 - v.smXmod + v.smXmod*1.6*modSig
				}
				o[0] += s * amp * v.gL[c]
				o[1] += sr * amp * v.gR[c]
			}
		}
		if !stereo {
			o[0] += o[1] // a mono output has the sum of both sides
			o[1] = 0
		}
		if v.smSub > 1e-4 {
			sub := 0.8 * v.smSub * dca * shapechain.SinTurns(v.subPh)
			v.subPh += f * 0.5 / sr
			v.subPh -= math.Floor(v.subPh)
			o[0] += sub
			o[1] += sub
		}
		if noiseOn {
			for c := range 2 {
				v.noiseRng ^= v.noiseRng << 13
				v.noiseRng ^= v.noiseRng >> 7
				v.noiseRng ^= v.noiseRng << 17
				x := float64(int64(v.noiseRng)) / (1 << 63)
				v.noiseLP[c] += (x - v.noiseLP[c]) * noiseA
				switch {
				case cfg.NoiseColor < 0:
					x = v.noiseLP[c] * noiseDark
				case cfg.NoiseColor > 0:
					x -= v.noiseLP[c]
				}
				o[c] += 0.5 * v.smNoise * dca * x
			}
		}
		g := v.smGain
		if v.openIn > 0 {
			if v.openIn--; v.openIn == 0 {
				v.filter.SetConfig(v.openCfg)
				cfg.Filter = v.filter.Config()
			}
		}
		if filterTick {
			v.filter.Tick(f, mods.Mod(dCutoff)+1.5*v.expr.Wheel()+v.voiceDrift[1], mods.Mod(dResonance))
		}
		for c := range o {
			if c > 0 && !stereo {
				break
			}
			// the chains are not always centred: a DC blocker
			x := o[c]
			v.dc[c] = x - v.dcIn[c] + dcCoef*v.dc[c]
			v.dcIn[c] = x
			val := v.dc[c] * g * 0.5
			if filtered {
				fv := v.filter.Process(c, val)
				val += (fv - val) * v.fBlend // a filter switched on under a note fades in
			}
			channels[c][i] = float32(val)
		}
		if filtered && v.fBlend < 1 {
			v.fBlend = min(1, v.fBlend+1/(filterFade*sr))
		} else if !filtered {
			v.fBlend = 0
		}
		v.t++
	}
	v.lastFreq = v.freq
}

// NewPoly creates a set of voices (mono, and legato, if the config says: one note at a time, a new one
// sliding there when the config has a glide)
func NewPoly(cfg Config, numVoices int) *poly.Poly[*Voice] {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	voices := make([]*Voice, numVoices)
	for i := range numVoices {
		voices[i] = New(cfg)
	}
	p := poly.New[*Voice](cfg.NumChannels, voices)
	if cfg.Mono {
		p.SetMono(true, cfg.Legato)
	}
	return p
}
