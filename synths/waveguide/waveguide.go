// Package waveguide models strings that are plucked or bowed: a wave travelling up and down a string
// (a waveguide), fed by a pluck or held going by a bow, heard through the body of the instrument.
//
// The pluck is an extended Karplus-Strong string: a burst of noise, shaped by where and how hard the
// string is plucked, circulates and loses its high partials first. The bow is the McIntyre,
// Schumacher and Woodhouse model (after the STK): the bow sticks to the string and slips, which
// keeps it going as long as the bow moves; its pressure and position color the sound. With BowModel 1
// the bow is a friction curve instead (stick-slip with a memory of the rosin's heat), a force and a
// speed that have to fit each other; the violins and cellos of the presets are bowed so.
//
// A string can be made more real, each part off at 0: its stiffness (Inharmonicity: allpass
// sections that stretch the partials, with the loop retuned for them), two polarizations (a prompt
// sound, an aftersound, beats), a bridge that gives way to the body's modes (BodyCoupling), the
// tension that makes a hard pluck sharp (TensionMod), the shape of the pluck (PluckWidth,
// Plectrum), the finger and the fret (FretNoise, FretBuzz: a spring and a damper where the string
// meets the fret), natural harmonics (Harmonic: a finger on the node that damps the other partials)
// and a legato (Slide, also when a mono voice is sent the next pitch).
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
	Membrane // a drum skin, struck (see membrane.go)
	Pipe     // a tube that is blown: a flute, or with Reed a clarinet (see pipe.go)
)

// Body is the instrument the string sounds through
type Body int

const (
	NoBody Body = iota
	GuitarBody
	ViolinBody
	CelloBody
	HarpBody
	BassBody // an upright bass
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

	// A real string (all off at 0)
	Inharmonicity float64 // B at 220 Hz: the stiffness stretches the partials to f0 n sqrt(1+B n^2); thick low strings more (nylon 3e-5, steel 1e-4, wound bass 3e-4)
	Polarization  float64 // 0-1: the string moves up and down and sideways, a little apart in pitch and in loss: a prompt sound, then an aftersound, and beats
	BodyCoupling  float64 // 0-1: the bridge gives way to the body's modes, so the string loses energy there and the body hands some back
	TensionMod    float64 // 0-1: a hard pluck stretches the string: the pitch starts sharp and glides down (the twang)
	PluckWidth    float64 // 0-0.2 of the string: the width of the finger or the pick that pulls it (0 is the plain noise burst of Brightness)
	Edge          float64 // 0-1: the pluck's top partials against its low ones (0 the string pulled aside, as smooth as it is; 1 a sharp pull, the partials falling slowly): a real pluck starts brighter than the string pulled aside is
	Plectrum      float64 // 0-1: the pluck, from a soft finger to a hard pick (the width of the contact, the release, a click)
	Thump         float64 // 0-1: the pluck knocks the body: a short low pulse at the start of the note, which the body rings with its own modes (air, top plate), whatever the pitch
	FretNoise     float64 // 0-1: the finger or the pick on the string at the start of a note
	FretBuzz      float64 // 0-1: the action, from high (0, off) to so low that the string hits the fret while it swings: a stiff one-sided spring and damper at the fret
	FretPosition  float64 // 0.05-0.98 of the string from the bridge: the fret the string hits (0 is the one beside the finger, 0.92)
	Harmonic      float64 // 0-0.5: a finger on the node at this part of the string (0.5 the octave, 1/3 the twelfth): the pluck leaves the partials with a node there and the finger in the loop damps the others (a natural harmonic, also on a string ringing)
	Slide         float64 // seconds a note takes to glide from the pitch of the string still ringing (legato), 0 plucks anew

	// Bow
	Pressure     float64 // 0-1: light (airy, flautando) to heavy (gritty)
	BowPosition  float64 // 0.02-0.5 of the string from the bridge: near is glassy (ponticello)
	BowAttack    float64 // seconds the bow takes to speed up
	BowRelease   float64 // seconds it takes to stop
	VibratoRate  float64 // Hz
	VibratoDepth float64 // semitones
	VibratoDelay float64 // seconds before the vibrato comes in
	BowModel     int     // 0 the bow table (Pressure), 1 a friction curve with stick-slip and the rosin's heat (BowForce, BowSpeed, BowNoise, BowReverse; Pressure is then not used)
	BowForce     float64 // 0-1: the bow's weight on the string (with BowModel 1)
	BowSpeed     float64 // 0-1: the bow's speed (with BowModel 1; 0 follows the velocity, the force following the speed); the force has to fit the speed: too light is airy, too heavy scratches
	BowNoise     float64 // 0-1: the rosin's grain in the friction
	BowReverse   float64 // seconds of a stroke: the bow turns round after it, the sound dips (0 never)

	// Membrane (struck; Decay is the seconds the lowest mode rings, Damping how much faster the higher
	// modes die, Release a hand laid on the skin)
	Tension  float64 // 0-1: the skin tightens as it is struck, the pitch falls back from above (a tom's thump)
	Strike   float64 // 0-1: where the skin is struck, from the middle (0, a few pure tones) to the rim (1)
	Hardness float64 // 0-1: the stick, from a soft mallet to a hard tip; the velocity adds to it
	Snare    float64 // 0-1: the wires under the skin, buzzing along

	// Pipe (blown; the breath is shaped by BowAttack and BowRelease, Damping darkens the tube, the
	// vibrato is the bow's)
	Reed       bool    // a single reed (a clarinet: odd partials) instead of an air jet over an edge (a flute)
	Breath     float64 // 0-1: the pressure of the breath
	Noise      float64 // 0-1: the air in the sound
	Embouchure float64 // 0-1: the jet's reach over the edge (flute), the reed's stiffness (clarinet)

	Body    Body
	BodyMix float64 // 0-1: how much of the sound is the body

	// A measured body for all the strings together, next to the resonators (nil: the resonators alone). It
	// is a stage of NewPoly, after the sum of the voices (a single voice made with New has the resonators only)
	Measured *Measured
	Level    float64 // output level (0 is treated as 1)

	// How a plucked string dies at each pitch (nil: by the Decay and Damping alone, and a lower string rings longer)
	Track *KeyTrack

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
	dBreath
	dEmbouchure
	dBowForce
	dBowSpeed
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
	dBreath:       {Key: "breath", Label: "breath", Span: 0.5},
	dEmbouchure:   {Key: "embouchure", Label: "embouchure", Span: 0.5},
	dBowForce:     {Key: "bowForce", Label: "bow force", Span: 0.5},
	dBowSpeed:     {Key: "bowSpeed", Label: "bow speed", Span: 0.5},
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
	BassBody:   {{58, 8, 0.8}, {95, 10, 1}, {190, 8, 0.7}, {380, 5, 0.4}, {900, 3, 0.3}},
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

	// the pluck: one loop, or two (the string up and down, and sideways)
	str       [2]pol
	s         float64 // the loop filter: y = (1-s) x + s x[-1]
	gain      float64 // per loop
	excite    []float64
	exPos     int
	comb      []float64
	dispDelay float64 // samples the stiffness adds to the loop at the fundamental
	stiff     bool
	polOn     bool
	polAmt    float64    // the sideways string's share of the pluck
	cbody     []*svf.SVF // the body as the bridge feels it, for the coupling
	buzzThr   float64    // the action: the swing where the string meets the fret (0 none)
	dampK     int        // the finger on the node at 1/dampK of the string (0 none)
	inNote    bool       // the message is that of a note starting
	tmodE     float64    // the energy of the string, smoothed; the glide of the tension follows it
	tmodE0    float64
	tmodAmt   float64
	tmodAlpha float64
	tmodRatio float64 // the pitch ratio the stretched string has now
	slide     float64 // the ratio the pitch is still from the note (a legato), and its step per sample
	slideStep float64
	thumpAt   int // samples since the start of the note, while the knock on the body lasts
	thumpAmp  float64
	fretLeft  int // samples of the finger on the string
	fretAmp   float64
	fretLp    float64
	fretLen   float64

	// the bow: two lines, from the bow to the nut (neck) and to the bridge
	neck, bridge delay
	bowFilt      float64
	bowHeat      float64 // the rosin's heat
	heatRate     float64
	bowSlide     float64 // the ratio the bow's line is still longer than the note's (a legato), and its step per sample
	bowSlideStep float64
	bowPos       float64
	fricScale    float64 // the share of its weight the friction bow leans on the string with
	bowTime      int     // samples since the bow came down
	bowGrain     float64
	fric         bool // the bow is a friction curve
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

	mem  membrane // the struck skin
	pipe pipeState

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
	v.str[0].line = make([]float64, size)
	v.str[1].line = make([]float64, size)
	v.slide, v.slideStep, v.tmodRatio = 1, 1, 1
	v.bowSlide, v.bowSlideStep = 1, 1
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
	v.warmBow()
	return v
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	oldFreq := v.freq
	v.receive(msg)
	// a legato: the pitch of a string that sounds changes (the next note of a mono voice) and it glides
	// there over Slide seconds, without a new attack
	if !v.inNote && v.active && !v.released && v.cfg.Slide > 0 && v.freq != oldFreq && (v.cfg.Model == Pluck || v.cfg.Model == Bow) {
		v.freq = max(lowest, min(v.freq, v.Config().SampleRate/4))
		if v.freq != oldFreq {
			v.glide(oldFreq)
		}
	}
}

func (v *Voice) receive(msg psalm.Message) {
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
	msg.IfFltVal("sympathetic", func(f float64) { v.cfg.Sympathetic = max(0, min(f, 1)) })
	msg.IfFltVal("lfo2Rate", func(f float64) { v.cfg.LFO2Rate = max(0, f) })
	// "ampAttack" the bow speeding up, "ampDecay" the ring of a plucked string, "ampRelease" a
	// string let go (plucked or bowed)
	envelope.Seconds(msg, "amp", &v.cfg.BowAttack, &v.cfg.Decay, &v.cfg.Release)
	envelope.Seconds(msg, "amp", nil, nil, &v.cfg.BowRelease)
	msg.IfFltVal("frequency", func(f float64) { v.freq = f })
	msg.IfFltVal("pitch", func(p float64) { v.freq = 440 * math.Exp2((p-69)/12) })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = max(0, min(f, 1)) })
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
	msg.IfFltVal("inharmonicity", func(f float64) { v.cfg.Inharmonicity = f })
	msg.IfFltVal("stiffness", func(f float64) { v.cfg.Inharmonicity = f })
	msg.IfFltVal("polarization", func(f float64) { v.cfg.Polarization = f })
	msg.IfFltVal("bodyCoupling", func(f float64) { v.cfg.BodyCoupling = f })
	msg.IfFltVal("tensionMod", func(f float64) { v.cfg.TensionMod = f })
	msg.IfFltVal("pluckWidth", func(f float64) { v.cfg.PluckWidth = f })
	msg.IfFltVal("edge", func(f float64) { v.cfg.Edge = f })
	msg.IfFltVal("plectrum", func(f float64) { v.cfg.Plectrum = f })
	msg.IfFltVal("thump", func(f float64) { v.cfg.Thump = f })
	msg.IfFltVal("fretNoise", func(f float64) { v.cfg.FretNoise = f })
	msg.IfFltVal("fretBuzz", func(f float64) { v.cfg.FretBuzz = f })
	msg.IfFltVal("harmonic", func(f float64) { v.cfg.Harmonic = f })
	msg.IfFltVal("slide", func(f float64) { v.cfg.Slide = f })
	msg.IfFltVal("fretPosition", func(f float64) { v.cfg.FretPosition = f })
	msg.IfFltVal("bowModel", func(f float64) { v.cfg.BowModel = int(math.Round(max(0, min(f, 1)))) })
	msg.IfFltVal("bowForce", func(f float64) { v.cfg.BowForce = f })
	msg.IfFltVal("bowSpeed", func(f float64) { v.cfg.BowSpeed = f })
	msg.IfFltVal("bowNoise", func(f float64) { v.cfg.BowNoise = f })
	msg.IfFltVal("bowReverse", func(f float64) { v.cfg.BowReverse = f })
	msg.IfFltVal("tension", func(f float64) { v.cfg.Tension = f })
	msg.IfFltVal("strike", func(f float64) { v.cfg.Strike = f })
	msg.IfFltVal("hardness", func(f float64) { v.cfg.Hardness = f })
	msg.IfFltVal("snare", func(f float64) { v.cfg.Snare = f })
	msg.IfBoolVal("reed", func(b bool) { v.cfg.Reed = b })
	msg.IfFltVal("breath", func(f float64) { v.cfg.Breath = f })
	msg.IfFltVal("noise", func(f float64) { v.cfg.Noise = f })
	msg.IfFltVal("embouchure", func(f float64) { v.cfg.Embouchure = f })
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

// CutsOnRestart: a note on a voice that sounds retunes and excites it again from what it was (poly fades the sound that was there and clears the voice, as it does for a
// voice it steals)
func (v *Voice) CutsOnRestart() bool { return true }

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(0, amplitude, message, tick)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(duration, amplitude, message, tick)
}

func (v *Voice) note(duration, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	v.velocity = 0.8
	oldFreq, wasReleased := v.freq, v.released
	v.inNote = true
	v.ReceiveMessage(message)
	v.inNote = false
	v.freq = max(lowest, min(v.freq, v.Config().SampleRate/4))
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
		if v.cfg.Model == Bow || v.cfg.Model == Pipe {
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
	switch v.cfg.Model {
	case Bow:
		// a legato: the bow goes on and the string glides to the new pitch
		if v.cfg.Slide > 0 && v.active && !wasReleased && oldFreq != v.freq {
			v.bowLegato(oldFreq)
		} else {
			v.bowNote()
		}
	case Membrane:
		v.strike()
	case Pipe:
		v.pipeNote()
	default:
		// a legato: a string still ringing glides to the new pitch instead of being plucked again
		if v.cfg.Slide > 0 && v.active && !wasReleased && oldFreq != v.freq {
			v.legato(oldFreq)
		} else {
			v.pluck()
		}
	}
	v.active = true
}

// fillStatic sets the sources that are the same for the whole note
func (v *Voice) fillStatic() {
	mv := &v.mods.Values
	mv[modmatrix.Velocity] = v.velocity
	mv[modmatrix.Random] = v.random
	mv[modmatrix.Key] = max(-1, min(math.Log2(v.freq/middleC)/5, 1))
	mv[modmatrix.Wheel] = v.expr.Wheel()
}

// minDamping is how far below 0 the damping goes: the loop filter (1-s) x + s x[-1] has s = 0.05 at 0, a little
// loss of the high partials that real strings do not all have; at -1/9 it takes nothing
const minDamping = 1.0 / 9

// edgeSpan is the part of the loop (1/edgeSpan) the Edge of the pluck takes its difference over
const edgeSpan = 40

// the knock of the pluck on the body (Config.Thump): the time constant of its pulse and its level at the full Thump
const (
	thumpTau  = 0.0008
	thumpGain = 8.0
)

// edgeReach is how many times higher the lowpass of the pluck goes at the full Edge
const edgeReach = 3

// setLoss sets the loop filter and the gain per loop for a damping and the seconds of decay at 220 Hz
func (v *Voice) setLoss(damping, decay float64) {
	sr := v.Config().SampleRate
	f := v.freq
	v.s = 0.05 + 0.45*max(-minDamping, min(damping, 1))
	// the gain per loop that rings Decay seconds at 220 Hz (longer below, shorter above), over what
	// the loop filter takes at the fundamental
	t60 := max(0.01, decay) * math.Sqrt(220/f)
	w := math.Pi * f / sr
	h := math.Sqrt(1 - 4*v.s*(1-v.s)*math.Sin(w)*math.Sin(w))
	v.gain = min(0.99999, math.Pow(10, -3/(f*t60))/h)
}

// stiffen fits the stiffness of the string to its pitch: the allpass sections, and the delay they
// add to the loop. Under a string that rings (a legato) the old sections fade out over 40 ms, the
// states of the new would not fit the wave
func (v *Voice) stiffen(fade bool) {
	sr := v.Config().SampleRate
	a, m, delay, _ := fitDispersion(stiffnessAt(v.cfg.Inharmonicity, v.freq), v.freq, sr)
	for i := range v.str {
		p := &v.str[i]
		if fade && m > 0 && (p.disp.a != a || p.disp.m != m) {
			p.oldDisp = p.disp
			p.fadeLen = max(1, int(0.04*sr))
			p.fade = p.fadeLen
		}
	}
	v.dispDelay, v.stiff = delay, m > 0
	for i := range v.str {
		v.str[i].disp.a, v.str[i].disp.m = a, m
		if v.str[i].fade > 0 {
			v.str[i].disp.clear()
		}
	}
}

// pluck tunes the loop and fills the excitation: noise, as bright as the pluck, with the notch of
// the pick position, or a string pulled aside by a finger or a pick
func (v *Voice) pluck() {
	cfg := &v.cfg
	sr := v.Config().SampleRate
	f := v.freq
	kd, kt := v.keyTrack()
	v.setLoss(cfg.Damping+kd+v.mods.Mod(dDamping), cfg.Decay*kt*math.Exp2(v.mods.Mod(dDecay)))
	// the loop: whole samples, the allpass for the fraction (0.5-1.5), the filter's half sample and
	// the stiffness's delay
	v.period = sr / f
	v.stiffen(false)
	v.polOn = cfg.Polarization > 0
	v.polAmt = polSideways * min(cfg.Polarization, 1)
	v.slide, v.slideStep = 1, 1
	old := v.str[0].n
	v.tune(1)
	n := v.str[0].n
	if old != n {
		v.str[0].clear()
	}
	if old != n || !v.active {
		v.str[1].clear()
	}

	// the excitation, one loop long
	if cap(v.excite) < n+1 {
		v.excite = make([]float64, n+1)
	}
	ex := v.excite[:n+1]
	bright := max(0, min(cfg.Brightness+v.mods.Mod(dBrightness)+0.4*(v.velocity-0.5)+0.4*cfg.Plectrum, 1))
	edge := max(0, min(cfg.Edge+v.keyEdge(), 1))
	// the pluck's lowpass, 12 dB/oct: a sharp pull reaches higher in the spectrum too
	coef := math.Exp(-2 * math.Pi * min(f*(1+60*bright*bright)*(1+edgeReach*edge), sr/3) / sr)
	pick := max(0, min(cfg.PickPosition+v.mods.Mod(dPickPosition), 0.5))
	peak, edgePeak := 0.0, 0.0
	if cfg.PluckWidth > 0 {
		// a string pulled aside: the triangle held at the pick, rounded by the contact, then the
		// lowpass of the pluck; a pick clicks as it lets go
		shapeExcite(ex, n, max(pick, 0.02), min(cfg.PluckWidth, 0.2), max(0, min(cfg.Plectrum, 1)))
		lp, lp2 := 0.0, 0.0
		for pass := 0; pass < 2; pass++ {
			for i := range n {
				lp = (1-coef)*ex[i] + coef*lp
				lp2 = (1-coef)*lp + coef*lp2
				if pass == 1 {
					ex[i] = lp2
				}
			}
		}
		ex[n] = ex[0]
		if edge > 0 {
			// the sharp pull: what the string pulled aside lacks in its top partials. A difference over a
			// fortieth of the loop, twice (a rise in the number of the partial, whatever the pitch, as far as
			// its twentieth or so), as far as the Edge goes. The pluck keeps the energy it had: the string is
			// as loud as it was
			before := energy(ex[:n])
			for _, x := range ex[:n] {
				edgePeak = max(edgePeak, math.Abs(x))
			}
			d := max(1, int(math.Round(float64(n)/edgeSpan)))
			for range 2 {
				v.comb = append(v.comb[:0], ex[:n]...)
				for i := range n {
					ex[i] = v.comb[i] - edge*v.comb[(i-d+n)%n]
				}
			}
			if after := energy(ex[:n]); after > 0 {
				g := math.Sqrt(before / after)
				for i := range n {
					ex[i] *= g
				}
			}
			ex[n] = ex[0]
		}
		if click := 0.12 * cfg.Plectrum * cfg.Plectrum * bright; click > 0 {
			lp, lp2 = 0, 0
			for i := range n {
				v.rng = v.rng*1664525 + 1013904223
				lp = float64(int32(v.rng))/2147483648 - lp*0.5
				ex[i] += click * lp * math.Exp(-float64(i)/(0.0005*sr))
			}
			ex[n] = ex[0]
		}
	} else {
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
		for i := range ex {
			ex[i] -= mean
			peak = max(peak, math.Abs(ex[i]))
		}
		// the pick position: the reflection off the near end cancels the partials with a node there
		// (around the loop: the burst circulates)
		if p := int(math.Round(pick * float64(n))); p > 0 {
			v.comb = append(v.comb[:0], ex[:n]...)
			for i := range n {
				ex[i] -= v.comb[(i-p+n)%n]
			}
			ex[n] = ex[0]
			peak = 0
			for _, x := range ex {
				peak = max(peak, math.Abs(x))
			}
		}
	}
	// a finger on a node of the string: only the partials that have a node there are left (the
	// loop, summed with itself shifted by the node's share)
	if k := harmonicK(cfg.Harmonic, n); k > 0 {
		v.comb = append(v.comb[:0], ex[:n]...)
		for i := range n {
			sum := 0.0
			for j := range k {
				sum += v.comb[(i+j*n/k)%n]
			}
			ex[i] = sum
		}
		ex[n] = ex[0]
	}
	if cfg.PluckWidth > 0 || (cfg.Harmonic > 0 && cfg.Harmonic <= 0.5) {
		peak = 0
		for _, x := range ex {
			peak = max(peak, math.Abs(x))
		}
	}
	if edgePeak > 0 {
		// the peak of the pluck without the Edge sets the level: the sharp one is peakier, and as loud
		peak = edgePeak
	}
	if peak > 0 {
		g := 1 / peak
		for i := range ex {
			ex[i] *= g
		}
	}
	v.excite = ex
	v.exPos = 0

	// the energy of the loop, to follow for the tension: a hard pluck starts sharp
	v.tmodAmt, v.tmodRatio = 0, 1
	if cfg.TensionMod > 0 {
		e := 0.0
		for _, x := range ex[:n] {
			e += x * x
		}
		v.tmodE0 = max(e/float64(n), 1e-9)
		v.tmodE = v.tmodE0
		v.tmodAmt = min(cfg.TensionMod, 1) * v.velocity * v.velocity
		v.tmodAlpha = 1 / (1.5*v.period + 1)
		v.tmodRatio = math.Exp2(tensionCents / 1200 * v.tmodAmt)
	}
	// the finger or the pick on the string, and the fret
	v.thumpAt, v.thumpAmp = 0, 0
	if cfg.Thump > 0 {
		v.thumpAmp = thumpGain * min(cfg.Thump, 1) * (0.3 + 0.7*v.velocity)
	}
	v.fretLeft = 0
	if cfg.FretNoise > 0 {
		v.fretLen = 0.012 * sr
		v.fretLeft = int(3 * v.fretLen)
		v.fretAmp = 0.7 * cfg.FretNoise * (0.4 + 0.6*v.velocity)
		v.fretLp = 0
	}
	v.buzzThr = 0
	if cfg.FretBuzz > 0 {
		v.buzzThr = (1.1 - min(cfg.FretBuzz, 1)) / (0.3 + 0.7*v.velocity)
	}
}

// fretPosition is the part of the string from the bridge where the fret that is hit is
func fretPosition(cfg Config) float64 {
	if cfg.FretPosition <= 0 {
		return 0.92
	}
	return max(0.05, min(cfg.FretPosition, 0.98))
}

// harmonicK is the k of a finger on the node at 1/k of the string (the nearest), 0 for none
func harmonicK(h float64, n int) int {
	if h <= 0 {
		return 0
	}
	k := int(math.Round(1 / max(h, 0.01)))
	if k < 2 || k >= n/4 {
		return 0
	}
	return k
}

// legato glides a ringing string to the pitch of a new note, without plucking it again
func (v *Voice) legato(from float64) {
	cfg := &v.cfg
	sr := v.Config().SampleRate
	kd, kt := v.keyTrack()
	v.setLoss(cfg.Damping+kd+v.mods.Mod(dDamping), cfg.Decay*kt*math.Exp2(v.mods.Mod(dDecay)))
	v.period = sr / v.freq
	v.stiffen(true)
	v.polOn = cfg.Polarization > 0
	// the string is still tuned for the old note: the ratio from the new one to it, falling to 1
	v.slide = from * v.slide / v.freq
	v.slideStep = math.Pow(1/v.slide, 1/max(1, cfg.Slide*sr))
	v.tune(v.slide * v.tmodRatio)
}

// tune sets the pluck's loop for its pitch moved by a ratio (the bend, the vibrato): the whole
// samples of the line, and the allpass for the fraction
func (v *Voice) tune(ratio float64) {
	v.tuneString(&v.str[0], v.period/ratio)
	if v.polOn {
		v.tuneString(&v.str[1], v.period/ratio*(1+polDetune))
	}
	v.tunedTo = ratio
}

func (v *Voice) tuneString(p *pol, period float64) {
	d := period - v.s - v.dispDelay
	n := int(math.Floor(d - 0.5))
	n = max(1, min(n, len(p.line)-2))
	frac := d - float64(n)
	p.n = n
	p.ap = (1 - frac) / (1 + frac)
}

// tensionCents is how sharp a full pluck of a string with TensionMod 1 starts
const tensionCents = 35.0

// bowNote sets the two lines of the bow
func (v *Voice) bowNote() {
	sr := v.Config().SampleRate
	pos := max(0.02, min(v.cfg.BowPosition+v.mods.Mod(dBowPosition), 0.5))
	v.fric = v.cfg.BowModel == 1
	v.bowSpeed = v.tableSpeed()
	if v.fric && v.cfg.BowSpeed > 0 {
		v.bowSpeed = bowSpeedOf(v.cfg.BowSpeed)
	}
	v.baseDelay = v.tunedBowDelay(pos, 0)
	v.bowPos = pos
	v.bowSlide, v.bowSlideStep = 1, 1
	v.bridge.len = max(v.baseDelay*pos, minBridge)
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
	// the friction curve: the speed and the force are the bow's own
	v.bowTime = 0
	v.heatRate = 1 / (frictionCool * sr)
	if v.fric {
		if !v.active {
			v.bowHeat = 0
		}
	}
}

// tunedBowDelay is the length of the lines for the pitch: for a friction bow with what it adds to the
// loop taken off (see fricTuning)
func (v *Voice) tunedBowDelay(pos float64, only float64) float64 {
	d := v.bowDelay()
	v.fricScale = 1
	if !v.fric {
		return d
	}
	cfg := &v.cfg
	sr := v.Config().SampleRate
	follow := cfg.BowSpeed <= 0 && !v.mods.Targets(dBowSpeed)
	// the tuning is looked up; a pitch that has none yet is estimated and asked for (never made here)
	key := fricKey{freq: v.freq, pos: pos, force: v.bowForce(cfg.BowForce), speed: v.bowSpeed, attack: cfg.BowAttack, only: only, sr: sr, follow: follow}
	fit, ok := fricLookup(key)
	if !ok {
		fit = fricEstimate(key)
		fricRequest(key)
	}
	if !fit.holds && (follow || v.samplesPerPeriod() < 40) {
		// no weight of the bow keeps this pitch (the very highest notes): the table bow plays it
		v.fric = false
		v.bowSpeed = v.tableSpeed()
		tk := fricKey{freq: v.freq, pos: pos, force: bowSlope(cfg.Pressure), speed: v.bowSpeed, attack: cfg.BowAttack, sr: sr, table: true}
		tfit, ok := fricLookup(tk)
		if !ok {
			tfit = fricEstimate(tk)
			fricRequest(tk)
		}
		return d + tfit.d
	}
	v.fricScale = fit.scale
	return d + fit.d
}

// samplesPerPeriod is the samples of the period of the pitch
func (v *Voice) samplesPerPeriod() float64 { return v.Config().SampleRate / v.freq }

// tableSpeed is the speed of the bow of the table, after the amplitude and the velocity
func (v *Voice) tableSpeed() float64 { return tableSpeedOf(v.amp, v.velocity) }

func tableSpeedOf(amp, velocity float64) float64 {
	return 0.03 + 0.1*min(amp, 1)*(0.5+0.5*velocity)
}

// bowDelay is the length of the two lines of the bow for the pitch: the loop is the lines, a sample
// each for reading them, and the filter
func (v *Voice) bowDelay() float64 { return bowDelayFor(v.freq, v.Config().SampleRate) }

func bowDelayFor(freq, sr float64) float64 {
	w := 2 * math.Pi * freq / sr
	p := bowPole(sr)
	d := sr/freq - 2 - math.Atan(p*math.Sin(w)/(1-p*math.Cos(w)))/w - bowTune
	// high up, the short lines' interpolation and the bow make the loop play sharp, measured: about
	// a sixth of a sample once the period is under 45 samples, easing in from 70
	if period := sr / freq; period < 70 {
		d += 0.17 * min(1, (70-period)/25)
	}
	return d
}

// bowLegato glides a bowed string to a new pitch: the bow goes on, the lines are longer by the ratio
// of the old pitch to the new and shorten to it over Slide seconds
func (v *Voice) bowLegato(from float64) {
	sr := v.Config().SampleRate
	old := v.baseDelay * v.bowSlide
	v.baseDelay = v.tunedBowDelay(v.bowPos, v.fricScale)
	v.bowSlide = old / v.baseDelay
	v.bowSlideStep = math.Pow(1/v.bowSlide, 1/max(1, v.cfg.Slide*sr))
	v.bowDamp = math.Pow(10, -3/(v.freq*bowStop)/2)
}

// glide moves the sounding string to the new pitch
func (v *Voice) glide(from float64) {
	if v.cfg.Model == Bow {
		v.bowLegato(from)
	} else {
		v.legato(from)
	}
}

// bowSpeedOf is the speed of the bow for a BowSpeed 0-1
func bowSpeedOf(x float64) float64 { return 0.02 + 0.2*max(0, min(x, 1)) }

// bowForce is the force of the bow for a BowForce: with a speed of the bow's own it is the force;
// a bow that follows the velocity has a force that follows the speed (the same weight holds at a
// slow bow and a fast one only when it goes with it)
func (v *Voice) bowForce(x float64) float64 {
	f := bowForceOf(x)
	if v.fricScale > 0 {
		f *= v.fricScale
	}
	if v.cfg.BowSpeed <= 0 && !v.mods.Targets(dBowSpeed) {
		f *= v.bowSpeed / bowSpeedRef
	}
	return f
}

// bowSpeedRef is the speed a bow follows the velocity to at a full note (amplitude 1, velocity 0.8)
const bowSpeedRef = 0.12

// bowForceOf is the force of the bow for a BowForce 0-1
func bowForceOf(x float64) float64 { return 0.05 + 2*max(0, min(x, 1)) }

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
		v.body[i].SetFrequencyQ(min(m.freq, sr*0.45), m.q)
	}
	// the bridge as the string feels it: the same modes
	if v.cfg.BodyCoupling > 0 && v.cfg.Model == Pluck {
		for len(v.cbody) < len(modes) {
			v.cbody = append(v.cbody, svf.New(1000, 0, sr))
		}
		v.cbody = v.cbody[:len(modes)]
		for i, m := range modes {
			if !v.active {
				v.cbody[i].Clear()
			}
			v.cbody[i].SetFrequencyQ(min(m.freq, sr*0.45), m.q)
		}
	} else {
		v.cbody = v.cbody[:0]
	}
}

func (v *Voice) NoteOff() {
	v.filter.Release()
	v.released = true
	v.noteOff = 0
}

func (v *Voice) Clear() {
	v.active = false
	v.str[0].clear()
	v.str[1].clear()
	v.neck.clear()
	v.bridge.clear()
	v.bowFilt, v.bowEnv, v.bowHeat = 0, 0, 0
	v.fretLeft = 0
	v.slide, v.slideStep, v.tmodRatio = 1, 1, 1
	v.bowSlide, v.bowSlideStep = 1, 1
	for _, b := range v.cbody {
		b.Clear()
	}
	v.mem.clear()
	v.pipe = pipeState{}
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
	if cfg.Model == Membrane {
		gain *= v.amp // the velocity is in the strike
	}
	mods := v.mods
	modded := mods.Active()
	lossModded := modded && (cfg.Model == Pluck || cfg.Model == Membrane) && (mods.Targets(dDecay) || mods.Targets(dDamping))
	pressModded := modded && mods.Targets(dPressure)
	posModded := modded && cfg.Model == Bow && mods.Targets(dBowPosition)
	breathP, noiseP, embP := cfg.Breath, cfg.Noise, cfg.Embouchure
	windModded := modded && cfg.Model == Pipe && (mods.Targets(dBreath) || mods.Targets(dEmbouchure))
	filtered := v.filter.On()
	tickFilter := filtered || (modded && mods.Uses(modmatrix.FilterEnv))
	lfo1Used := modded && mods.Uses(modmatrix.LFO1)
	lfo2Used := modded && mods.Uses(modmatrix.LFO2)
	lfo2Inc := cfg.LFO2Rate / sr
	if lfo2Inc <= 0 {
		lfo2Inc = 2 / sr
	}
	penvStep := 1.0
	if (cfg.Model == Pluck || cfg.Model == Membrane) && modded {
		t60 := max(0.01, cfg.Decay) * math.Sqrt(220/v.freq)
		if cfg.Model == Membrane {
			t60 = max(0.01, cfg.Decay)
		}
		penvStep = math.Pow(10, -3/(t60*sr))
	}
	if modded {
		v.fillStatic()
	}
	mixBase := max(0, min(cfg.BodyMix, 1))
	if len(v.body) == 0 {
		mixBase = 0
	}
	bodyModded := modded && len(v.body) > 0 && mods.Targets(dBodyMix)
	mix := mixBase
	sym := max(0, min(cfg.Sympathetic, 1))
	if sym > 0 {
		v.prepareSym()
	}
	channels := out.Channels
	quiet := true
	modes := bodies[cfg.Body]

	// letting go: the string dies in Release seconds (a pluck), or the bow stops
	loopGain := v.gain
	if v.released && cfg.Model == Pluck && cfg.Release > 0 {
		loopGain = min(loopGain, math.Pow(10, -3/(v.freq*cfg.Release)))
	}
	// a real string: the stiffness, the two polarizations, the bridge's give, the tension, the fret
	pluck := cfg.Model == Pluck
	stiff := pluck && v.stiff
	polOn := pluck && v.polOn
	polC := polCoupling(min(cfg.Polarization, 1))
	lgV, lgH := loopGain, loopGain
	setPol := func() {
		if polOn {
			gv, gh := polGains(v.gain, min(cfg.Polarization, 1))
			if v.released && cfg.Release > 0 {
				r := math.Pow(10, -3/(v.freq*cfg.Release))
				gv, gh = min(gv, r), min(gh, r)
			}
			lgV, lgH = gv, gh
		}
	}
	setPol()
	coupled := pluck && len(v.cbody) > 0
	kappa := 0.03 * min(cfg.BodyCoupling, 1)
	tmodOn := pluck && v.tmodAmt > 0
	buzz := pluck && v.buzzThr > 0 && v.period/v.tunedTo >= 8
	fx := fretPosition(*cfg)
	// the displacement where the string is at its widest is not what it is at the fret
	fretGap := v.buzzThr * max(0.2, math.Sin(math.Pi*fx))
	dampK := 0
	if pluck {
		dampK = harmonicK(cfg.Harmonic, v.str[0].n)
	}
	// the friction curve of the bow
	fric := cfg.Model == Bow && v.fric
	forceModded := modded && fric && (mods.Targets(dBowForce) || mods.Targets(dBowSpeed))
	force := v.bowForce(cfg.BowForce)
	grainOn := fric && cfg.BowNoise > 0
	follow := fric && cfg.BowSpeed <= 0 && !mods.Targets(dBowSpeed)
	reverse := fric && cfg.BowReverse > 0
	reversePeriod := cfg.BowReverse * sr
	bowStep := 1 / max(1, max(0.005, cfg.BowAttack)*sr)
	bowFall := 1 / max(1, max(0.005, cfg.BowRelease)*sr)
	pressure := cfg.Pressure
	slope := 5 - 4*max(0, min(pressure, 1))
	vibInc := cfg.VibratoRate / sr
	filterPole := bowPole(sr)

	for i := v.waitSamps; i < n; i++ {
		var s float64
		v.expr.Tick(sr)
		bend := v.expr.VibratoRatio(sr, 5.5, 0.3)
		depth := cfg.VibratoDepth
		if modded {
			mv := &mods.Values
			if cfg.Model == Bow || cfg.Model == Pipe {
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
				mix = max(0, min(mixBase+mods.Mod(dBodyMix), 1))
			}
			if pressModded {
				pressure = cfg.Pressure + mods.Mod(dPressure)
				slope = 5 - 4*max(0, min(pressure, 1))
			}
			if lossModded && i&15 == 0 {
				if cfg.Model == Membrane {
					v.mem.setLoss(v, cfg.Damping+mods.Mod(dDamping), cfg.Decay*math.Exp2(mods.Mod(dDecay)))
				} else {
					kd, kt := v.keyTrack()
					v.setLoss(cfg.Damping+kd+mods.Mod(dDamping), cfg.Decay*kt*math.Exp2(mods.Mod(dDecay)))
					loopGain = v.gain
					if v.released && cfg.Release > 0 {
						loopGain = min(loopGain, math.Pow(10, -3/(v.freq*cfg.Release)))
					}
					setPol()
				}
			}
			if windModded {
				breathP = cfg.Breath + mods.Mod(dBreath)
				embP = cfg.Embouchure + mods.Mod(dEmbouchure)
			}
			if forceModded {
				if cfg.BowSpeed > 0 || mods.Targets(dBowSpeed) {
					v.bowSpeed = bowSpeedOf(cfg.BowSpeed + mods.Mod(dBowSpeed))
				}
				force = v.bowForce(cfg.BowForce + mods.Mod(dBowForce))
			}
			if posModded {
				v.bowPos = max(0.02, min(cfg.BowPosition+mods.Mod(dBowPosition), 0.5))
				v.bridge.len = max(v.baseDelay*v.bowSlide*v.bowPos, minBridge)
			}
		}
		if cfg.Model == Pluck {
			ratio := bend * v.slide * v.tmodRatio
			if v.slide != 1 {
				v.slide *= v.slideStep
				if (v.slideStep < 1) == (v.slide < 1) || v.slideStep == 1 {
					v.slide, v.slideStep = 1, 1
				}
			}
			if ratio != v.tunedTo {
				v.tune(ratio)
			}
			// the loop: the line (read the loop's length back), the allpass for the fraction, the
			// loss filter
			f := v.str[0].filter(v.s, stiff)
			if coupled {
				// the bridge gives way to the body: the modes take some of the wave and ring on
				b := 0.0
				for j, m := range modes {
					_, band, _ := v.cbody[j].Tick(f)
					b += band * v.cbody[j].K() * m.gain
				}
				f -= kappa * b
			}
			period := v.period / v.tunedTo
			if buzz {
				v.str[0].contact(period, max(2, fx*period/2), fretGap)
			}
			var in float64
			if !polOn {
				in = f * loopGain
				if dampK > 0 {
					in = v.str[0].damp(in, period/float64(dampK), dampK)
				}
				if v.exPos < len(v.excite) {
					in += v.excite[v.exPos]
					v.exPos++
				}
				v.str[0].push(in)
				s = in
			} else {
				fh := v.str[1].filter(v.s, stiff)
				if buzz {
					v.str[1].contact(period*(1+polDetune), max(2, fx*period*(1+polDetune)/2), fretGap)
				}
				inV := lgV * ((1-polC)*f + polC*fh)
				inH := lgH * ((1-polC)*fh + polC*f)
				if dampK > 0 {
					inV = v.str[0].damp(inV, period/float64(dampK), dampK)
					inH = v.str[1].damp(inH, period*(1+polDetune)/float64(dampK), dampK)
				}
				if v.exPos < len(v.excite) {
					e := v.excite[v.exPos]
					inV += e
					inH += v.polAmt * e
					v.exPos++
				}
				v.str[0].push(inV)
				v.str[1].push(inH)
				in = inV + polOutput*inH
				s = in
			}
			if tmodOn {
				// a stretched string is sharp: the pitch follows the energy it has left
				v.tmodE += (in*in - v.tmodE) * v.tmodAlpha
				if i&7 == 0 {
					v.tmodRatio = math.Exp2(tensionCents / 1200 * v.tmodAmt * min(1, v.tmodE/v.tmodE0))
				}
			}
			if v.thumpAmp != 0 {
				// the knock of the pluck on the body: the derivative of an alpha function (no DC), a low pulse
				// that has its energy where the body has its modes
				t := float64(v.thumpAt) / (thumpTau * sr)
				s += v.thumpAmp * (1 - t) * math.Exp(-t)
				if v.thumpAt++; t > 12 {
					v.thumpAmp = 0
				}
			}
			if v.fretLeft > 0 {
				// the finger or the pick on the string: a short rustle
				v.rng = v.rng*1664525 + 1013904223
				nz := float64(int32(v.rng)) / 2147483648
				v.fretLp += 0.4 * (nz - v.fretLp)
				s += v.fretAmp * (nz - v.fretLp) * math.Exp(-float64(3*int(v.fretLen)-v.fretLeft)/v.fretLen)
				v.fretLeft--
			}
			if lfo1Used {
				v.vibPhase += vibInc
				if v.vibPhase >= 1 {
					v.vibPhase--
				}
			}
		} else if cfg.Model == Membrane {
			s = v.mem.tick(v, bend)
		} else {
			// the bow speeds up while held, and slows down when let go
			if !v.released {
				v.bowEnv = min(1, v.bowEnv+bowStep)
			} else {
				v.bowEnv = max(0, v.bowEnv-bowFall)
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
			if cfg.Model == Pipe {
				s = v.pipeTick(ratio/bend, breathP, noiseP, embP, sr)
			} else {
				if v.bowSlide != 1 {
					v.bowSlide *= v.bowSlideStep
					if (v.bowSlideStep < 1) == (v.bowSlide < 1) || v.bowSlideStep == 1 {
						v.bowSlide, v.bowSlideStep = 1, 1
					}
					v.bridge.len = max(v.baseDelay*v.bowSlide*v.bowPos, minBridge)
				}
				v.neck.len = v.baseDelay*v.bowSlide*ratio/bend - v.bridge.len
				nut := -v.neck.out
				v.bowFilt = (1-filterPole)*v.bridge.out*0.95 + filterPole*v.bowFilt
				bridge := -v.bowFilt
				spd := v.bowSpeed * v.bowEnv
				forceEnv := 1.0
				if follow {
					// the speed is the velocity's and swells and fades with the bow: the weight goes with it
					forceEnv = v.bowEnv
				}
				if reverse && !v.released {
					spd *= bowReversal(float64(v.bowTime), reversePeriod, 0.03*sr)
				}
				v.bowTime++
				dv := spd - (nut + bridge)
				push := 0.0
				if v.bowEnv > 0 {
					if fric {
						grain := 1.0
						if grainOn {
							// the rosin's grain: the force flickers
							v.rng = v.rng*1664525 + 1013904223
							v.bowGrain += 0.05 * (float64(int32(v.rng))/2147483648 - v.bowGrain)
							grain = 1 + 3*cfg.BowNoise*v.bowGrain
						}
						push = v.friction(dv, force*forceEnv, grain)
					} else {
						push = dv * bowTable(dv, slope)
					}
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
			g = gain / level * max(0, level+mods.Mod(dLevel))
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
	held := (cfg.Model == Bow || cfg.Model == Pipe) && (!v.released || v.bowEnv > 0)
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
		v.sym[i].SetFrequencyQ(min(v.freq*r, sr*0.45), 40)
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
	return max(0.01, min(x, 0.98))
}

// NewPoly creates a set of strings
func NewPoly(cfg Config, numVoices int) *poly.Poly[*Voice] {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	// a measured body is one stage for the sum of the strings: the voices have no resonators of their own then
	var stage *bodyStage
	if m := cfg.Measured; m != nil && len(m.IR) > 0 && m.Blend > 0 {
		stage = newBodyStage(cfg.Body, cfg.BodyMix, m, cfg.NumChannels)
		cfg.BodyMix = 0
	}
	voices := make([]*Voice, numVoices)
	for i := range numVoices {
		voices[i] = New(cfg)
	}
	p := poly.New[*Voice](cfg.NumChannels, voices)
	if stage != nil {
		p.SetPost(stage.process)
	}
	return p
}

// bowReversal is the bow's speed as a factor of its stroke: after each stroke of period samples it
// slows over ramp samples, stops, and goes back the other way
func bowReversal(t, period, ramp float64) float64 {
	stroke := math.Floor(t / period)
	x := t - stroke*period
	dir := 1.0
	if int(stroke)%2 == 1 {
		dir = -1
	}
	f := min(1, (period-x)/ramp)
	if stroke > 0 {
		f = min(f, x/ramp)
	}
	return dir * f
}

// energy is the sum of the squares of x
func energy(x []float64) float64 {
	e := 0.0
	for _, s := range x {
		e += s * s
	}
	return e
}
