// Package waveguide models strings that are plucked or bowed: a wave travelling up and down a string
// (a waveguide), fed by a pluck or held going by a bow, heard through the body of the instrument.
//
// The pluck is an extended Karplus-Strong string: a burst of noise, shaped by where and how hard the
// string is plucked, circulates and loses its high partials first. The bow is the McIntyre,
// Schumacher and Woodhouse model (after the STK): the bow sticks to the string and slips, which
// keeps it going as long as the bow moves; its pressure and position color the sound.
//
//	guitar := waveguide.NewPoly(waveguide.Guitar(), 8).AddTo(p)
//	violin := waveguide.NewPoly(waveguide.Violin(), 4).AddTo(p)
package waveguide

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/modules/poly"
)

// Model is how the string is played
type Model int

const (
	Pluck Model = iota
	Bow
)

// Body is the instrument the string sounds through
type Body int

const (
	NoBody Body = iota
	GuitarBody
	ViolinBody
	CelloBody
	HarpBody
)

// Config describes a string and how it is played
type Config struct {
	NumChannels int
	Model       Model

	// Pluck
	Decay        float64 // seconds the string rings at 220 Hz (60 dB down); lower strings ring longer
	Damping      float64 // 0-1: how much faster the high partials die (0 bright steel, 1 a dull thud)
	Brightness   float64 // 0-1: the pluck, from a soft finger to a hard pick; the velocity adds to it
	PickPosition float64 // 0-0.5 of the string from the bridge: near the bridge is thin, the middle round
	Release      float64 // seconds the string takes to die once let go (a hand on it), 0 lets it ring

	// Bow
	Pressure     float64 // 0-1: light (airy, flautando) to heavy (gritty)
	BowPosition  float64 // 0.02-0.5 of the string from the bridge: near is glassy (ponticello)
	BowAttack    float64 // seconds the bow takes to speed up
	BowRelease   float64 // seconds it takes to stop
	VibratoRate  float64 // Hz
	VibratoDepth float64 // semitones
	VibratoDelay float64 // seconds before the vibrato comes in

	Body    Body
	BodyMix float64 // 0-1: how much of the sound is the body
	Level   float64 // output level (0 is treated as 1)

	Sympathetic float64 // 0-1: the sound fed into strings tuned to its octave and fifth, added back
	LFO2Rate    float64 // Hz: the rate of the plain LFO of the matrix (0 is 2 Hz)

	Filter voicefilter.Config              // a filter after the string and the body (off unless its Type is set)
	Mods   [modmatrix.Slots]modmatrix.Slot // the matrix: a source to a destination (see Dests) over an amount
	Ext    [modmatrix.Exts]float64         // the values 0-1 of the matrix's sources Ext 1 to 4 (cabled in on InExt1...)
}

// The destinations of the matrix (see Dests), by number
const (
	dDecay = iota
	dDamping
	dBrightness
	dPickPosition
	dPressure
	dBowPosition
	dVibratoDepth
	dBodyMix
	dPitch
	dLevel
	dCutoff
)

// Dests are the settings the matrix of a voice moves: by their keys, as described (Describe), with
// how far a full source moves them (octaves of the decay time, semitones, octaves of the cutoff, or the
// setting itself). The brightness and the pick position are those of the pluck, taken as a note
// starts (so the sources that move them are the velocity, the key, the random value, the wheel and
// the cabled values)
var Dests = []modmatrix.Dest{
	dDecay:        {Key: "decay", Label: "decay", Span: 2},
	dDamping:      {Key: "damping", Label: "damping", Span: 0.5},
	dBrightness:   {Key: "brightness", Label: "brightness", Span: 0.5},
	dPickPosition: {Key: "pickPosition", Label: "pick position", Span: 0.25},
	dPressure:     {Key: "bowPressure", Label: "pressure", Span: 0.5},
	dBowPosition:  {Key: "bowPosition", Label: "bow position", Span: 0.2},
	dVibratoDepth: {Key: "vibratoDepth", Label: "vibrato depth", Span: 1},
	dBodyMix:      {Key: "bodyMix", Label: "body", Span: 1},
	dPitch:        {Key: "pitch", Label: "pitch", Span: 12},
	dLevel:        {Key: "level", Label: "level", Span: 1},
	dCutoff:       {Key: "cutoff", Label: "cutoff", Span: 4},
}

// The value inputs of a voice: 1 is the bow pressure and 2 the vibrato depth, 3 to 6 the values of
// the matrix (Ext 1 to 4) that a slot can move any setting by
const InExt1 = 3 // to InExt1+3

const middleC = 261.6255653005986

// symRatios are the strings that sound along: an octave below, a fifth, an octave and an octave and a fifth above
var symRatios = [...]float64{0.5, 1.5, 2, 3}

// DefaultConfig is a nylon string guitar
func DefaultConfig() Config {
	return Guitar()
}

// resonance is a mode of a body: its frequency, sharpness and loudness
type resonance struct{ freq, q, gain float64 }

var bodies = map[Body][]resonance{
	GuitarBody: {{98, 8, 1}, {204, 10, 0.8}, {390, 6, 0.5}, {600, 4, 0.3}, {2500, 1.5, 0.2}},
	ViolinBody: {{275, 10, 0.7}, {460, 12, 1}, {550, 12, 0.9}, {1000, 4, 0.4}, {2500, 2, 0.6}},
	CelloBody:  {{100, 8, 0.8}, {175, 10, 1}, {220, 10, 0.8}, {500, 4, 0.4}, {1500, 2, 0.4}},
	HarpBody:   {{140, 6, 0.7}, {290, 6, 0.6}, {600, 4, 0.4}, {1600, 2, 0.2}},
}

// Voice is one string
type Voice struct {
	*psalm.Module
	cfg       Config
	freq      float64
	velocity  float64
	amp       float64
	waitSamps int
	noteOff   int // samples until a note of Note lets go
	released  bool
	quiet     int // samples it has been silent
	active    bool

	// the pluck: one loop
	line   []float64
	pos    int
	n      int     // the whole samples of the loop
	ap     float64 // the allpass coefficient for the fraction
	apX    float64 // the allpass state
	apY    float64
	s      float64 // the loop filter: y = (1-s) x + s x[-1]
	prev   float64
	gain   float64 // per loop
	excite []float64
	exPos  int
	comb   []float64

	// the bow: two lines, from the bow to the nut (neck) and to the bridge
	neck, bridge delay
	bowFilt      float64
	bowEnv       float64
	bowSpeed     float64
	bowDamp      float64 // per line, the bow resting on the string once it stopped
	vibPhase     float64
	vibWait      int
	baseDelay    float64

	body []*svf.SVF
	rng  uint32

	filter  *voicefilter.Filter
	mods    *modmatrix.Matrix
	random  float64 // a random value for the note, -1 to 1
	penv    float64 // the level of a plucked string, 1 to 0 (the amp envelope for the matrix)
	lfo2    float64 // phase
	sym     []*svf.SVF
	symFreq float64

	expr    expression.Expression // the pitch bend; the mod wheel brings in a vibrato
	period  float64               // samples of the pluck's period, unbent
	tunedTo float64               // the ratio the pluck's loop is tuned for now
}

// delay is a delay line read with a linear interpolation
type delay struct {
	buf []float64
	pos int
	len float64
	out float64
}

func (d *delay) tick(x float64) float64 {
	d.buf[d.pos] = x
	r := float64(d.pos) - d.len
	n := len(d.buf)
	for r < 0 {
		r += float64(n)
	}
	i := int(r)
	f := r - float64(i)
	d.out = d.buf[i]*(1-f) + d.buf[(i+1)%n]*f
	d.pos = (d.pos + 1) % n
	return d.out
}

func (d *delay) clear() {
	clear(d.buf)
	d.out = 0
}

// lowest is the lowest frequency a string can play
const lowest = 20.0

// New returns a string
func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	v := &Voice{
		Module: psalm.NewModule(nil, []int{cfg.NumChannels}),
		cfg:    cfg,
		freq:   220,
	}
	sr := v.Config().SampleRate
	size := int(sr/lowest) + 4
	v.line = make([]float64, size)
	v.neck.buf = make([]float64, size)
	v.bridge.buf = make([]float64, size)
	v.rng = uint32(v.Rand().Uint64())
	v.filter = voicefilter.New(cfg.Filter, sr, cfg.NumChannels)
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
	msg.IfFltVal("sympathetic", func(f float64) { v.cfg.Sympathetic = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("lfo2Rate", func(f float64) { v.cfg.LFO2Rate = math.Max(0, f) })
	// "ampAttack" the bow speeding up, "ampDecay" the ring of a plucked string, "ampRelease" a
	// string let go (plucked or bowed)
	envelope.Seconds(msg, "amp", &v.cfg.BowAttack, &v.cfg.Decay, &v.cfg.Release)
	envelope.Seconds(msg, "amp", nil, nil, &v.cfg.BowRelease)
	msg.IfFltVal("frequency", func(f float64) { v.freq = f })
	msg.IfFltVal("pitch", func(p float64) { v.freq = 440 * math.Exp2((p-69)/12) })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("decay", func(f float64) { v.cfg.Decay = f })
	msg.IfFltVal("damping", func(f float64) { v.cfg.Damping = f })
	msg.IfFltVal("brightness", func(f float64) { v.cfg.Brightness = f })
	msg.IfFltVal("pickPosition", func(f float64) { v.cfg.PickPosition = f })
	msg.IfFltVal("release", func(f float64) { v.cfg.Release = f })
	msg.IfFltVal("bowPressure", func(f float64) { v.cfg.Pressure = f })
	msg.IfFltVal("bowPosition", func(f float64) { v.cfg.BowPosition = f })
	msg.IfFltVal("vibratoRate", func(f float64) { v.cfg.VibratoRate = f })
	msg.IfFltVal("vibratoDepth", func(f float64) { v.cfg.VibratoDepth = f })
	msg.IfFltVal("bodyMix", func(f float64) { v.cfg.BodyMix = f })
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
}

// Receive takes messages, or the bow pressure on input 1 and the vibrato depth on input 2, and on 3
// to 6 (InExt1...) values 0-1 that the matrix takes as sources Ext 1 to 4
func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}
	switch index {
	case 1:
		v.cfg.Pressure = psalm.FltVal[float64](value)
	case 2:
		v.cfg.VibratoDepth = psalm.FltVal[float64](value)
	case InExt1, InExt1 + 1, InExt1 + 2, InExt1 + 3:
		v.mods.SetExt(index-InExt1, psalm.FltVal[float64](value))
		v.cfg.Ext[index-InExt1] = v.mods.Ext(index - InExt1)
	}
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(0, amplitude, message, tick)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(duration, amplitude, message, tick)
}

func (v *Voice) note(duration, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	v.velocity = 0.8
	v.ReceiveMessage(message)
	v.freq = math.Max(lowest, math.Min(v.freq, v.Config().SampleRate/4))
	v.waitSamps = tick.SampleOffset
	v.amp = amplitude
	v.released = false
	v.quiet = 0
	v.noteOff = 0
	if duration > 0 {
		v.noteOff = max(1, int(duration*v.Config().SampleRate))
	}
	if !v.active {
		v.vibPhase = 0
		for _, b := range v.sym {
			b.Clear()
		}
	}
	// drawn only when a slot uses it: a voice that doesn't keeps the random stream it had
	if v.mods.Uses(modmatrix.Random) {
		v.random = v.Rand().Float64()*2 - 1
	}
	v.penv = 1
	if v.mods.Active() {
		// the pluck is shaped as it starts: the sources that are still are what moves it
		mv := &v.mods.Values
		*mv = [modmatrix.NumSources]float64{}
		mv[modmatrix.AmpEnv] = v.penv
		if v.cfg.Model == Bow {
			mv[modmatrix.AmpEnv] = 0
		}
		v.fillStatic()
		v.mods.Eval(v.Config().SampleRate)
	}
	mode := adsr.NoteOff
	if duration > 0 {
		mode = adsr.Duration
	}
	v.filter.Trigger(duration*1000, mode)
	v.prepareBody()
	if v.cfg.Model == Bow {
		v.bowNote()
	} else {
		v.pluck()
	}
	v.active = true
}

// fillStatic sets the sources that are the same for the whole note
func (v *Voice) fillStatic() {
	mv := &v.mods.Values
	mv[modmatrix.Velocity] = v.velocity
	mv[modmatrix.Random] = v.random
	mv[modmatrix.Key] = math.Max(-1, math.Min(math.Log2(v.freq/middleC)/5, 1))
	mv[modmatrix.Wheel] = v.expr.Wheel()
}

// setLoss sets the loop filter and the gain per loop for a damping and the seconds of decay at 220 Hz
func (v *Voice) setLoss(damping, decay float64) {
	sr := v.Config().SampleRate
	f := v.freq
	v.s = 0.05 + 0.45*math.Max(0, math.Min(damping, 1))
	// the gain per loop that rings Decay seconds at 220 Hz (longer below, shorter above), over what
	// the loop filter takes at the fundamental
	t60 := math.Max(0.01, decay) * math.Sqrt(220/f)
	w := math.Pi * f / sr
	h := math.Sqrt(1 - 4*v.s*(1-v.s)*math.Sin(w)*math.Sin(w))
	v.gain = math.Min(0.99999, math.Pow(10, -3/(f*t60))/h)
}

// pluck tunes the loop and fills the excitation: noise, as bright as the pluck, with the notch of
// the pick position
func (v *Voice) pluck() {
	cfg := &v.cfg
	sr := v.Config().SampleRate
	f := v.freq
	v.setLoss(cfg.Damping+v.mods.Mod(dDamping), cfg.Decay*math.Exp2(v.mods.Mod(dDecay)))
	// the loop: whole samples, the allpass for the fraction (0.5-1.5) and the filter's half sample
	v.period = sr / f
	n := int(math.Floor(v.period - v.s - 0.5))
	if v.n != n {
		clear(v.line)
		v.prev, v.apX, v.apY = 0, 0, 0
	}
	v.tune(1)

	// the excitation, one loop long
	if cap(v.excite) < n+1 {
		v.excite = make([]float64, n+1)
	}
	ex := v.excite[:n+1]
	bright := math.Max(0, math.Min(cfg.Brightness+v.mods.Mod(dBrightness)+0.4*(v.velocity-0.5), 1))
	coef := math.Exp(-2 * math.Pi * math.Min(f*(1+60*bright*bright), sr/3) / sr) // the pluck's lowpass, 12 dB/oct
	lp, lp2 := 0.0, 0.0
	mean := 0.0
	for i := range ex {
		v.rng = v.rng*1664525 + 1013904223
		lp = (1-coef)*float64(int32(v.rng))/2147483648 + coef*lp
		lp2 = (1-coef)*lp + coef*lp2
		ex[i] = lp2
		mean += lp2
	}
	mean /= float64(len(ex))
	peak := 0.0
	for i := range ex {
		ex[i] -= mean
		peak = math.Max(peak, math.Abs(ex[i]))
	}
	// the pick position: the reflection off the near end cancels the partials with a node there
	// (around the loop: the burst circulates)
	if p := int(math.Round(math.Max(0, math.Min(cfg.PickPosition+v.mods.Mod(dPickPosition), 0.5)) * float64(n))); p > 0 {
		v.comb = append(v.comb[:0], ex[:n]...)
		for i := range n {
			ex[i] -= v.comb[(i-p+n)%n]
		}
		ex[n] = ex[0]
		peak = 0
		for _, x := range ex {
			peak = math.Max(peak, math.Abs(x))
		}
	}
	if peak > 0 {
		g := 1 / peak
		for i := range ex {
			ex[i] *= g
		}
	}
	v.excite = ex
	v.exPos = 0
}

// tune sets the pluck's loop for its pitch moved by a ratio (the bend, the vibrato): the whole
// samples of the line, and the allpass for the fraction
func (v *Voice) tune(ratio float64) {
	d := v.period/ratio - v.s
	n := int(math.Floor(d - 0.5))
	n = max(1, min(n, len(v.line)-2))
	frac := d - float64(n)
	v.n = n
	v.ap = (1 - frac) / (1 + frac)
	v.tunedTo = ratio
}

// bowNote sets the two lines of the bow
func (v *Voice) bowNote() {
	sr := v.Config().SampleRate
	pos := math.Max(0.02, math.Min(v.cfg.BowPosition+v.mods.Mod(dBowPosition), 0.5))
	// the loop is the two lines there and back, and the filter
	// the loop is the lines, a sample each for reading them, and the filter
	w := 2 * math.Pi * v.freq / sr
	p := bowPole(sr)
	v.baseDelay = sr/v.freq - 2 - math.Atan(p*math.Sin(w)/(1-p*math.Cos(w)))/w - bowTune
	// high up, the short lines' interpolation and the bow make the loop play sharp, measured: about
	// a sixth of a sample once the period is under 45 samples, easing in from 70
	if period := sr / v.freq; period < 70 {
		v.baseDelay += 0.17 * math.Min(1, (70-period)/25)
	}
	v.bridge.len = math.Max(v.baseDelay*pos, minBridge)
	v.neck.len = v.baseDelay * (1 - pos)
	if !v.active {
		v.neck.clear()
		v.bridge.clear()
		v.bowFilt = 0
		v.bowEnv = 0
		v.vibPhase = 0
	}
	v.vibWait = int(v.cfg.VibratoDelay * sr)
	// the bow speed is the loudness: faster than this the string jumps to its octave
	// slowing down, the bow takes the string's energy with it: as it stops, the string dies in
	// 60 ms (per period, over the two lines), it doesn't ring on as if plucked
	v.bowDamp = math.Pow(10, -3/(v.freq*bowStop)/2)
	v.bowSpeed = 0.03 + 0.1*math.Min(v.amp, 1)*(0.5+0.5*v.velocity)
}

// bowTune is what the friction of the bow adds to the loop, in samples
const bowTune = -0.15

// minBridge is the shortest the bow side of the string can be: shorter, the string jumps to a
// higher mode
const minBridge = 8.0

// bowPole is the pole of the string filter of the bow
func bowPole(sr float64) float64 { return 0.75 - 0.2*22050/sr }

// prepareBody sets the resonances of the body
func (v *Voice) prepareBody() {
	modes := bodies[v.cfg.Body]
	sr := v.Config().SampleRate
	for len(v.body) < len(modes) {
		v.body = append(v.body, svf.New(1000, 0, sr))
	}
	v.body = v.body[:len(modes)]
	for i, m := range modes {
		if !v.active {
			v.body[i].Clear()
		}
		v.body[i].SetFrequencyQ(math.Min(m.freq, sr*0.45), m.q)
	}
}

func (v *Voice) NoteOff() {
	v.filter.Release()
	v.released = true
	v.noteOff = 0
}

func (v *Voice) Clear() {
	v.active = false
	clear(v.line)
	v.neck.clear()
	v.bridge.clear()
	v.prev, v.apX, v.apY, v.bowFilt, v.bowEnv = 0, 0, 0, 0, 0
	for _, b := range v.body {
		b.Clear()
	}
	for _, b := range v.sym {
		b.Clear()
	}
	v.filter.Clear()
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
	n := v.Config().BufferSize
	sr := v.Config().SampleRate
	cfg := &v.cfg
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	gain := level // the bow's speed is its loudness
	if cfg.Model == Pluck {
		gain *= v.amp * (0.3 + 0.7*v.velocity)
	}
	mods := v.mods
	modded := mods.Active()
	lossModded := modded && cfg.Model == Pluck && (mods.Targets(dDecay) || mods.Targets(dDamping))
	pressModded := modded && mods.Targets(dPressure)
	posModded := modded && cfg.Model == Bow && mods.Targets(dBowPosition)
	filtered := v.filter.On()
	tickFilter := filtered || (modded && mods.Uses(modmatrix.FilterEnv))
	lfo1Used := modded && mods.Uses(modmatrix.LFO1)
	lfo2Used := modded && mods.Uses(modmatrix.LFO2)
	lfo2Inc := cfg.LFO2Rate / sr
	if lfo2Inc <= 0 {
		lfo2Inc = 2 / sr
	}
	penvStep := 1.0
	if cfg.Model == Pluck && modded {
		t60 := math.Max(0.01, cfg.Decay) * math.Sqrt(220/v.freq)
		penvStep = math.Pow(10, -3/(t60*sr))
	}
	if modded {
		v.fillStatic()
	}
	mixBase := math.Max(0, math.Min(cfg.BodyMix, 1))
	if len(v.body) == 0 {
		mixBase = 0
	}
	bodyModded := modded && len(v.body) > 0 && mods.Targets(dBodyMix)
	mix := mixBase
	sym := math.Max(0, math.Min(cfg.Sympathetic, 1))
	if sym > 0 {
		v.prepareSym()
	}
	channels := out.Channels
	quiet := true
	modes := bodies[cfg.Body]

	// letting go: the string dies in Release seconds (a pluck), or the bow stops
	loopGain := v.gain
	if v.released && cfg.Model == Pluck && cfg.Release > 0 {
		loopGain = math.Min(loopGain, math.Pow(10, -3/(v.freq*cfg.Release)))
	}
	bowStep := 1 / math.Max(1, math.Max(0.005, cfg.BowAttack)*sr)
	bowFall := 1 / math.Max(1, math.Max(0.005, cfg.BowRelease)*sr)
	pressure := cfg.Pressure
	slope := 5 - 4*math.Max(0, math.Min(pressure, 1))
	vibInc := cfg.VibratoRate / sr
	filterPole := bowPole(sr)

	size := len(v.line)
	for i := v.waitSamps; i < n; i++ {
		var s float64
		v.expr.Tick(sr)
		bend := v.expr.VibratoRatio(sr, 5.5, 0.3)
		depth := cfg.VibratoDepth
		if modded {
			mv := &mods.Values
			if cfg.Model == Bow {
				mv[modmatrix.AmpEnv] = v.bowEnv
			} else {
				v.penv *= penvStep
				if v.released {
					v.penv *= penvStep * penvStep * penvStep
				}
				mv[modmatrix.AmpEnv] = v.penv
			}
			mv[modmatrix.FilterEnv] = mv[modmatrix.AmpEnv]
			if filtered {
				mv[modmatrix.FilterEnv] = v.filter.Env()
			}
			mv[modmatrix.Wheel] = v.expr.Wheel()
			if lfo1Used {
				mv[modmatrix.LFO1] = math.Sin(2 * math.Pi * v.vibPhase)
			}
			if lfo2Used {
				mv[modmatrix.LFO2] = math.Sin(2 * math.Pi * v.lfo2)
				v.lfo2 += lfo2Inc
				if v.lfo2 >= 1 {
					v.lfo2--
				}
			}
			mods.Eval(sr)
			depth += mods.Mod(dVibratoDepth)
			if m := mods.Mod(dPitch); m != 0 {
				bend *= math.Exp2(m / 12)
			}
			if bodyModded {
				mix = math.Max(0, math.Min(mixBase+mods.Mod(dBodyMix), 1))
			}
			if pressModded {
				pressure = cfg.Pressure + mods.Mod(dPressure)
				slope = 5 - 4*math.Max(0, math.Min(pressure, 1))
			}
			if lossModded && i&15 == 0 {
				v.setLoss(cfg.Damping+mods.Mod(dDamping), cfg.Decay*math.Exp2(mods.Mod(dDecay)))
				loopGain = v.gain
				if v.released && cfg.Release > 0 {
					loopGain = math.Min(loopGain, math.Pow(10, -3/(v.freq*cfg.Release)))
				}
			}
			if posModded {
				v.bridge.len = math.Max(v.baseDelay*math.Max(0.02, math.Min(cfg.BowPosition+mods.Mod(dBowPosition), 0.5)), minBridge)
			}
		}
		if cfg.Model == Pluck {
			if bend != v.tunedTo {
				v.tune(bend)
			}
			// the loop: the line (read the loop's length back), the allpass for the fraction, the
			// loss filter
			x := v.line[(v.pos-v.n+size)%size]
			y := v.ap*x + v.apX - v.ap*v.apY
			v.apX, v.apY = x, y
			f := (1-v.s)*y + v.s*v.prev
			v.prev = y
			in := f * loopGain
			if v.exPos < len(v.excite) {
				in += v.excite[v.exPos]
				v.exPos++
			}
			v.line[v.pos] = in
			v.pos++
			if v.pos >= size {
				v.pos = 0
			}
			s = in
			if lfo1Used {
				v.vibPhase += vibInc
				if v.vibPhase >= 1 {
					v.vibPhase--
				}
			}
		} else {
			// the bow speeds up while held, and slows down when let go
			if !v.released {
				v.bowEnv = math.Min(1, v.bowEnv+bowStep)
			} else {
				v.bowEnv = math.Max(0, v.bowEnv-bowFall)
			}
			// the vibrato stretches the neck side
			if v.vibWait > 0 {
				v.vibWait--
			} else if depth != 0 || lfo1Used {
				v.vibPhase += vibInc
				if v.vibPhase >= 1 {
					v.vibPhase--
				}
			}
			ratio := 1.0
			if depth != 0 {
				ratio = math.Exp2(-depth * math.Sin(2*math.Pi*v.vibPhase) / 12)
			}
			v.neck.len = v.baseDelay*ratio/bend - v.bridge.len
			nut := -v.neck.out
			v.bowFilt = (1-filterPole)*v.bridge.out*0.95 + filterPole*v.bowFilt
			bridge := -v.bowFilt
			dv := v.bowSpeed*v.bowEnv - (nut + bridge)
			push := 0.0
			if v.bowEnv > 0 {
				push = dv * bowTable(dv, slope)
				if v.released {
					push *= v.bowEnv * v.bowEnv // lifting, the bow lets go of the string as it slows
				}
			}
			if v.released {
				// the damping grows as the bow comes to a stop
				still := 1 - v.bowEnv
				still *= still
				d := 1 - (1-v.bowDamp)*still*still
				bridge *= d
				nut *= d
			}
			v.neck.tick(bridge + push)
			v.bridge.tick(nut + push)
			s = v.bridge.out * 1.2
		}
		if mix > 0 {
			b := 0.0
			for j, m := range modes {
				_, band, _ := v.body[j].Tick(s)
				b += band * v.body[j].K() * m.gain
			}
			s = s*(1-mix) + b*mix*1.5
		}
		if sym > 0 {
			in := 0.0
			for _, r := range v.sym {
				_, band, _ := r.Tick(s)
				in += band * r.K()
			}
			s += in * sym * 0.5
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
			v.filter.Tick(v.freq*bend, extra, 0)
		}
		if filtered {
			for c, ch := range channels {
				ch[i] = float32(v.filter.Process(c, s*g))
			}
		} else {
			val := float32(s * g)
			for _, ch := range channels {
				ch[i] = val
			}
		}
		if math.Abs(s) > 1e-4 {
			quiet = false
		}
	}
	// a note of Note lets go
	if v.noteOff > 0 {
		if left := n - v.waitSamps; v.noteOff > left {
			v.noteOff -= left
		} else {
			v.NoteOff()
		}
	}
	v.waitSamps = 0
	// the string rests once it has been silent a while, bowed or plucked
	held := cfg.Model == Bow && (!v.released || v.bowEnv > 0)
	if quiet && !held && v.exPos >= len(v.excite) {
		v.quiet += n
		if v.quiet > int(0.05*sr) {
			v.active = false
		}
	} else {
		v.quiet = 0
	}
}

// prepareSym tunes the strings that sound along to the note
func (v *Voice) prepareSym() {
	if len(v.sym) == len(symRatios) && v.symFreq == v.freq {
		return
	}
	sr := v.Config().SampleRate
	for len(v.sym) < len(symRatios) {
		v.sym = append(v.sym, svf.New(1000, 0, sr))
	}
	for i, r := range symRatios {
		v.sym[i].SetFrequencyQ(math.Min(v.freq*r, sr*0.45), 40)
	}
	v.symFreq = v.freq
}

// bowStop is the seconds a string dies in once the bow has stopped
const bowStop = 0.06

// bowTable is the friction of the bow: it holds the string (1) while their speeds are close and lets
// it slip as they differ, the sooner the lighter the bow
func bowTable(dv, slope float64) float64 {
	x := math.Abs(dv*slope) + 0.75
	x *= x
	x = 1 / (x * x)
	return math.Max(0.01, math.Min(x, 0.98))
}

// NewPoly creates a set of strings
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
