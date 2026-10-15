// Package piano is a physically modelled piano. Every key has one to three strings (digital
// waveguides: a delay line closed by a loss filter that makes the highs die faster, a chain of
// allpass filters for the stiffness of the string, which stretches the partials sharp, and an
// allpass that tunes the loop to the pitch). Every string vibrates in two polarizations: one that
// passes its energy to the soundboard fast (the loud prompt sound) and one that keeps it (the long,
// quiet aftersound), so a note decays in two stages like a real piano; the strings of a key are a
// little out of tune with each other, so they beat. A felt hammer strikes them: a force pulse that
// gets shorter and peakier the harder the key is played (the felt stiffens), with notches at the
// harmonics the strike position doesn't excite. Dampers stop the strings when the key is released,
// the sustain pedal lifts them, the highest keys have none; the soft pedal shifts the hammers so
// they strike one string less. The voices sound through a soundboard, and the strings whose
// dampers are up ring along with what is played (sympathetic resonance), see Piano. The strings of
// all keys end on one bridge, a bank of soundboard modes every voice loads and is loaded by (see
// bridge.go), so a chord is more than its notes; the bass strings are wound (less inharmonic, a
// step at the break) and a hard struck bass note has a longitudinal precursor
package piano

import (
	"math"
	"math/cmplx"

	"github.com/almerlucke/psalm"
)

// Config describes a piano
type Config struct {
	NumChannels int

	Hardness       float64     // 0-1, hardness of the hammer felt: soft and round to hard and bright
	StrikePosition float64     // where the hammer hits the string, 0.05-0.3 of its length (1/8 is classic)
	Soundboard     float64     // 0-1, how much of the soundboard response is in the strike (body, knock)
	Stiffness      float64     // inharmonicity, 1 is a grand, more is an upright, 0 is perfectly harmonic
	Decay          float64     // scales the ring times, 1 is a grand
	Damping        float64     // how much faster the highs die, 1 is a grand
	Unison         float64     // cents the strings of a key are apart, about 1 (more is honky tonk)
	Strings        int         // strings per key at most, 1-3 (0 is 3): fewer are thinner, cheaper
	DamperTime     float64     // seconds a released key rings on (the time the dampers take)
	Width          float64     // 0-1, the keys spread from left (low) to right (high)
	VelocitySens   float64     // 0-1, how much the velocity changes the level (it always changes the tone)
	Body           float64     // 0-1, how much of the sound comes through the soundboard
	Bark           float64     // 0-1, the metallic bark of the bass strings played hard (phantom partials)
	Mechanics      float64     // 0-1, the noise of the action: dampers, keys, the pedal
	KeyBed         float64     // 0-1, the thud of the key hitting the keybed when struck hard (scaled by Mechanics)
	Stretch        float64     // how much of the stretch tuning of a piano, 1 is what a tuner does, 0 is equal temperament
	Variation      float64     // 0-1, how different the keys are (hammers, levels, ring times) and every strike
	Treble         float64     // dB, the top octave louder or softer than voiced (0 is as voiced)
	Position       Position    // where the piano is heard from
	SoundboardIR   [][]float64 // a measured soundboard response (per side) instead of the modelled one
	Resonance      float64     // 0-1, sympathetic resonance of the strings whose dampers are up
	Bridge         float64     // 0-1, how much the strings of all keys are coupled through the shared bridge and soundboard modes (0 is each key draining on its own)
	Wound          float64     // 0-1, the wound bass strings: less inharmonic, the copper damps the highs, with a step at the break (0 is plain strings all over)
	Precursor      float64     // 0-1, the burst of longitudinal waves ahead of a hard struck bass note
	StrikeRegister float64     // 0-1, how much the strike point moves with the register (the setting in the bass, towards 2/3 of it in the top)
	Level          float64     // output level (0 is treated as 1)
}

// DefaultConfig is a concert grand
func DefaultConfig() Config {
	return Config{
		NumChannels:    psalm.Stereo,
		Hardness:       0.5,
		StrikePosition: 0.125,
		Soundboard:     0.6,
		Stiffness:      1,
		Decay:          1,
		Damping:        1,
		Unison:         1,
		Strings:        3,
		DamperTime:     0.25,
		Width:          0.6,
		VelocitySens:   0.7,
		Body:           0.6,
		Bark:           0.5,
		Mechanics:      0.5,
		KeyBed:         0.5,
		Stretch:        1,
		Variation:      0.5,
		Position:       Player,
		Resonance:      0.5,
		Bridge:         0.6,
		Wound:          1,
		Precursor:      0.5,
		StrikeRegister: 1,
		Level:          1,
	}
}

// Position is where the piano is heard from
type Position int

const (
	Player Position = iota // on the bench: wide, the keys from left to right, the lid above
	Studio                 // close microphones over the open lid: wide, clear and present
	Hall                   // in the audience: narrower, the lid and the floor reflecting, a little air
)

// highestDamper is the highest key with a damper, the keys above ring on
const highestDamper = 88

// the two polarizations of a string: both ring with the losses of the string (ringTime), the
// vertical one drains fast through the bridge (see couplingTable), the horizontal one, starting
// afterLevel as loud and a little apart in pitch, doesn't: the aftersound
const (
	promptFactor = 1.0
	afterCents   = 0.3
)

// afterLevel is how loud the horizontal polarization starts, against the vertical one
const afterLevel = 0.3

// the strings of a key share the bridge: what they do together (the vertical polarizations moving
// in phase) drains into the soundboard fast, couplingTable per second (spread over the loops of
// the string: a treble string goes around thousands of times a second, a bass string a few
// dozen); what they do against each other only loses what the strings themselves lose, and lasts
// (Weinreich). That is the two stages of a piano note. crossCoupling per second of the bridge
// motion passes to the horizontal polarizations
const crossCoupling = 0.26

// the soundboard drains the bridge motion below drainHz drainLow times, above it drainHigh times
const (
	drainHz   = 300.0
	drainLow  = 1.3
	drainHigh = 0.5
)

// couplingTable is how fast (nepers per second) what the strings of a key do together drains into
// the soundboard, fitted to the first second of the decay of a recorded concert grand
var couplingTable = [][2]float64{{21, 1.05}, {36, 2.12}, {48, 1.9}, {60, 3.26}, {72, 2.52}, {84, 1.1}, {96, 0.31}, {108, 0.05}}

// bridgeDrain returns the coupling of a key per second
func bridgeDrain(pitch float64) float64 { return interpolate(couplingTable, pitch) }

// a string played hard starts sharp and settles: cents at full velocity, and how fast
// it settles (seconds to fall to a third), by key
var (
	settleCents   = [][2]float64{{21, 6}, {60, 3}, {96, 1}}
	settleSeconds = [][2]float64{{21, 0.3}, {60, 0.2}, {108, 0.1}}
)

// the bark: a string stretched by its own motion (the square of it) rings its longitudinal mode,
// at about longitudinalRatio times the fundamental, with partials at the sums of the partials;
// strongest in the bass, gone above barkTop
const (
	barkGain          = 20.0
	longitudinalRatio = 17.0
	barkTop           = 66.0
)

// the precursor: a burst of longitudinal waves ahead of the note, in the bass struck hard
const (
	precursorGain = 0.0006
	precursorFrom = 45.0 // keys up to here have all of it
	precursorTop  = 72.0 // and none above this
)

// velocityPower is how the level goes with the speed of the hammer at VelocitySens 1, fitted to
// the dynamics of a recorded grand (at the default 0.7, about 1.5: pianissimo to fortissimo about 25 dB)
const velocityPower = 2.1

// the soft pedal (una corda): the hammers strike one string less, with a softer part of the felt;
// the string left out only rings along a little
const (
	softHardness = 0.4
	softLevel    = 1.25
	softCoupling = 0.06
)

// Voice is a piano voice: the strings of one key. Notes take "pitch" (or "frequency") and
// "velocity" (0-1, default 0.8); "sustain" is the sustain pedal (0 up, 1 down, half pedaling in
// between), "soft" the soft pedal
type Voice struct {
	*psalm.Module
	cfg Config

	strings  []*str
	exc      []float64 // the strike, played into the strings
	knock    []float64 // the knock of the soundboard, heard directly
	knockOut []float64 // the knock of this buffer, when an instrument hears it apart
	scratch  hammerScratch
	tail     [2][]float64 // what was left of the strike before, when the key is struck again
	excPos   int          // position in the strike, negative waits for the sample of the strike
	pitch    float64
	key      int
	velocity float64
	amp      float64       // output gain of the key: the level and the keyboard balance
	knockAmp float64       // output gain of the knock
	hardness float64       // the felt of this key and strike
	strikeAt float64       // where this hammer hits
	targets  []float64     // the loop gains the dampers leave the strings
	voicing  *[128]float64 // the gains of the keys that even out the soundboard of the instrument
	force    float64       // how hard the hammer strikes
	peak     float64       // the loudest sample of the last buffer
	loudest  float64       // the loudest buffer since the strike
	gainL    float64
	gainR    float64

	// the buffer being rendered, and the bridge the voice is loaded by (shared by the instrument)
	run       runState
	bridge    *bridge
	own       *bridge // of a voice on its own
	bridgeAmt float64
	bridgeOff bool // the voice is rendered alone for a buffer (faded out), outside the bridge

	// the precursor: a burst of longitudinal waves
	pre      [3]complex128
	preRot   [3]complex128
	preAmp   [3]float64
	preLevel float64
	preOn    bool
	preN     int

	keyDown   bool
	pedal     float64    // the sustain pedal, 0 up to 1 down
	sostenuto *[128]bool // the keys the sostenuto pedal keeps undamped (of the instrument)
	soft      bool
	releaseIn int // samples until a timed note releases, -1 is none

	// the bark: a highpass and a resonator on the square of the string motion
	barkLevel          float64
	settle             float64 // cents the strings are sharp for now, settling
	settleDecay        float64
	drainLP            float64 // the low part of the bridge motion
	barkDC             float64
	b0, a1, a2         float64
	bx1, bx2, by1, by2 float64
	env                float64 // level of the strings
	wasDamped, wasDown bool
	noise              float64 // level of the action noise, decaying
	keyNoise           float64 // the noise of the key coming up, in the units of the strings
	noiseLP, noiseHP   float64
	thumpRe, thumpIm   float64 // the key landing, a short low thud
	thumpRot           complex128
	bedIn              int     // samples until the key reaches the keybed, -1 is none
	bedLevel           float64 // how hard it lands, in the units of the strings
	quiet              int     // buffers the voice has been silent, after the strike
	active             bool
}

// runState is what a voice keeps between the halves of the samples of a buffer
type runState struct {
	contact, glide, period, coupling, g, cross, mech float64
	drainCoef, dcCoef, envCoef                       float64
	noiseLPCoef, noiseHPCoef, noiseDecay             float64
	damped                                           bool
	targets                                          []float64
	peak                                             float64
	t, bridge, kn                                    float64
	pos                                              int
}

// New creates a piano voice
func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	v := &Voice{
		Module:    psalm.NewModule(nil, []int{cfg.NumChannels}),
		cfg:       cfg,
		pitch:     60,
		velocity:  0.8,
		releaseIn: -1,
	}
	dispersion(v.Config().SampleRate, math.Max(0, cfg.Stiffness), math.Max(0, math.Min(cfg.Wound, 1))) // fit the strings now, not at the first note
	v.SetSelf(v)
	return v
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	cfg := &v.cfg
	msg.IfFltVal("hardness", func(f float64) { cfg.Hardness = f })
	msg.IfFltVal("strikePosition", func(f float64) { cfg.StrikePosition = f })
	msg.IfFltVal("soundboard", func(f float64) { cfg.Soundboard = f })
	msg.IfFltVal("stiffness", func(f float64) { cfg.Stiffness = f })
	msg.IfFltVal("decay", func(f float64) { cfg.Decay = f })
	msg.IfFltVal("damping", func(f float64) { cfg.Damping = f })
	msg.IfFltVal("unison", func(f float64) { cfg.Unison = f })
	msg.IfIntVal("strings", func(i int) { cfg.Strings = i })
	msg.IfFltVal("damperTime", func(f float64) { cfg.DamperTime = f })
	msg.IfFltVal("width", func(f float64) { cfg.Width = f })
	msg.IfFltVal("velocitySens", func(f float64) { cfg.VelocitySens = f })
	msg.IfFltVal("level", func(f float64) { cfg.Level = f })
	msg.IfFltVal("pitch", func(f float64) { v.pitch = f })
	msg.IfFltVal("frequency", func(f float64) { v.pitch = 69 + 12*math.Log2(f/440) })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("body", func(f float64) { cfg.Body = f })
	msg.IfFltVal("stretch", func(f float64) { cfg.Stretch = f })
	msg.IfFltVal("treble", func(f float64) { cfg.Treble = f })
	msg.IfFltVal("variation", func(f float64) { cfg.Variation = f })
	msg.IfIntVal("position", func(i int) { cfg.Position = Position(i) })
	msg.IfFltVal("bark", func(f float64) { cfg.Bark = f })
	msg.IfFltVal("mechanics", func(f float64) { cfg.Mechanics = f })
	msg.IfFltVal("keyBed", func(f float64) { cfg.KeyBed = f })
	msg.IfFltVal("resonance", func(f float64) { cfg.Resonance = f })
	msg.IfFltVal("bridge", func(f float64) { cfg.Bridge = f })
	msg.IfFltVal("wound", func(f float64) { cfg.Wound = f })
	msg.IfFltVal("precursor", func(f float64) { cfg.Precursor = f })
	msg.IfFltVal("strikeRegister", func(f float64) { cfg.StrikeRegister = f })
	msg.IfFltVal("sustain", v.SetSustain)
	msg.IfFltVal("soft", func(f float64) { v.soft = f > 0.5 })
}

func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
	}
}

// SetSustain sets the sustain pedal: 0 up, 1 down, in between the dampers only touch the strings
// (half pedaling)
func (v *Voice) SetSustain(amount float64) { v.pedal = math.Max(0, math.Min(amount, 1)) }

// contact returns how firmly the dampers of the voice are on its strings, 0 lifted (the key is
// down, the sostenuto holds it, the keys at the top have none) to 1
func (v *Voice) contact() float64 {
	if v.keyDown || v.pitch > highestDamper || (v.sostenuto != nil && v.sostenuto[v.key]) {
		return 0
	}
	return damperContact(v.pedal, v.pitch)
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.strike(amplitude, message, tick, -1, false)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.strike(amplitude, message, tick, int(duration*v.Config().SampleRate)+tick.SampleOffset, false)
}

func (v *Voice) NoteOff() { v.keyDown = false }

func (v *Voice) Clear() {
	v.active = false // the strings are reset by the next strike
	v.excPos = len(v.exc)
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool { return v.active }

// Held returns whether the key of the voice is down (its dampers are up)
func (v *Voice) Held() bool { return v.active && v.keyDown }

// Pitch returns the pitch of the key the voice plays
func (v *Voice) Pitch() float64 { return v.pitch }

// Key returns the key the voice plays (the nearest MIDI note)
func (v *Voice) Key() int { return v.key }

// keyOf returns the key of a pitch
func keyOf(pitch float64) int { return max(0, min(int(math.Round(pitch)), 127)) }

// strike sets up the strings of the key and the hammer. Striking a key that still sounds again
// (restrike) strikes its strings as they vibrate, like a piano does
func (v *Voice) strike(amplitude float64, message psalm.Message, tick psalm.CtrlTick, releaseIn int, restrike bool) {
	v.velocity = 0.8
	v.ReceiveMessage(message)
	cfg := &v.cfg
	sr := v.Config().SampleRate

	restrike = restrike && v.active && v.key == keyOf(v.pitch)
	v.key = keyOf(v.pitch)
	vary := math.Max(0, math.Min(cfg.Variation, 1))

	// every key's hammer is a little different, and every strike: the felt, where it hits, and
	// the moment it reaches each string
	v.hardness = cfg.Hardness + vary*(0.06*keyRandom(v.key, 1)+0.03*(2*v.Rand().Float64()-1))
	v.strikeAt = keyStrikeAt(cfg, v.key)
	spread := vary * maxStrikeSpread
	v.releaseIn = releaseIn
	v.keyDown = true
	if !restrike {
		v.loudest = 0
	}
	v.quiet = 0
	v.active = true

	f0 := 440 * math.Exp2((stretched(v.pitch, cfg)-69)/12) // tuned like a piano: the octaves stretched
	n := numStrings(v.pitch, cfg.Strings)
	disp := dispersion(sr, math.Max(0, cfg.Stiffness), math.Max(0, math.Min(cfg.Wound, 1)))[max(0, min(int(math.Round(v.pitch)), 127))]
	woundRingF, woundDampF, _ := woundTraits(v.pitch, cfg)
	design := stringDesign{
		sr:      sr,
		stages:  disp.stages,
		ad:      disp.ad,
		t60:     ringTime(v.pitch) * woundRingF * math.Max(0.05, cfg.Decay),
		damping: highDamping * woundDampF * math.Max(0, cfg.Damping),
		// the dampers on the long bass strings take longer to stop them
		damperTime: math.Max(0.02, cfg.DamperTime) * (1 + math.Max(0, (48-v.pitch)/24)),
	}

	// the strings of the key: a little apart (the same for every strike of the key, like a tuned
	// piano), every one in two polarizations. With the soft pedal down the hammer misses one
	if len(v.strings) != 2*n {
		v.strings = make([]*str, 2*n)
		restrike = false
	}
	struck := n
	if v.soft && n > 1 {
		struck = n - 1
	}
	offsets := unisonOffsets(v.pitch, n)
	for i := range n {
		d := design
		d.f0 = f0 * math.Exp2(offsets[i]*cfg.Unison/1200)
		d.t60 *= 1 + 0.08*float64(i)               // the strings decay a little differently
		d.t60 *= 1 + 0.08*vary*keyRandom(v.key, 3) // and every key a little differently
		in := 1.0
		if i >= struck {
			in = softCoupling
		}
		d.m, d.eps = irregular(sr/d.f0, d.f0, v.key, i, vary)
		prompt, after := d, d
		prompt.t60 *= promptFactor
		after.f0 *= math.Exp2(afterCents / 1200)
		for k, pd := range []stringDesign{prompt, after} {
			s := v.strings[2*i+k]
			switch {
			case s == nil:
				s = newString(designString(pd))
				v.strings[2*i+k] = s
			case !restrike:
				s.reset(designString(pd))
			}
			s.in = in
			if k == 1 {
				s.in *= afterLevel
			}
		}
	}

	for _, s := range v.strings {
		s.delay = int(spread * v.Rand().Float64())
	}

	// key position in the stereo field, low keys on the left
	pan := keyPan(v.pitch, cfg)
	v.gainL = math.Cos(pan * math.Pi / 2)
	v.gainR = math.Sin(pan * math.Pi / 2)

	// the level of the key and the keyboard balance, so the keys sound about equally loud (the
	// output gain stays the same when the key is struck again, the strike sets the loudness)
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	v.amp = level * keyBalance(v.pitch) / math.Sqrt(float64(n))
	v.amp *= math.Pow(10, vary*keyRandom(v.key, 4)/20)                             // every key ±1 dB
	v.amp *= math.Pow(10, cfg.Treble*math.Max(0, math.Min((v.pitch-84)/12, 1))/20) // the voicing of the top
	// the knock is the soundboard, not the strings: the balance of the strings leaves it alone
	v.knockAmp = v.amp / keyBalance(v.pitch)
	if v.voicing != nil {
		v.amp *= v.voicing[v.key]
	}
	v.amp *= math.Pow(10, -interpolateLinear(hardnessBalance, v.pitch)*(cfg.Hardness-balanceHardness)/20)
	// the key coming up is the action, not the strings: the voicing leaves it alone, the treble
	// (whose short notes are quiet when the key comes up) half follows the balance. It is as gentle
	// as the key was played (a key played softly is let go softly), but a key struck hard doesn't
	// come up harder: it falls back by its own weight
	touch := math.Min(hammerVelocity(v.velocity)/hammerVelocity(0.8), 1)
	v.keyNoise = keyNoise * amplitude * touch * math.Sqrt(math.Min(keyBalance(v.pitch), 1)) * v.knockAmp / v.amp

	// a string played hard is stretched by its own motion: it starts sharp and settles as the
	// motion dies down (the more the harder, most in the bass)
	speed := hammerVelocity(v.velocity) / hammerVelocity(1)
	v.settle += interpolate(settleCents, v.pitch) * speed * speed
	v.settleDecay = math.Exp(-1 / (interpolate(settleSeconds, v.pitch) * sr))

	// the bark: the longitudinal resonance, only the bass and tenor
	w := math.Max(0, math.Min(1, (barkTop-v.pitch)/(barkTop-45)))
	v.barkLevel = barkGain * math.Max(0, math.Min(cfg.Bark, 1)) * w * speed
	fl := math.Min(math.Max(longitudinalRatio*f0, 300), 3000)
	wl := 2 * math.Pi * fl / sr
	alpha := math.Sin(wl) / (2 * 6) // Q 6
	v.b0 = alpha / (1 + alpha)
	v.a1 = -2 * math.Cos(wl) / (1 + alpha)
	v.a2 = (1 - alpha) / (1 + alpha)
	if !restrike {
		v.settle = interpolate(settleCents, v.pitch) * speed * speed
		v.barkDC, v.bx1, v.bx2, v.by1, v.by2, v.env = 0, 0, 0, 0, 0, 0
		v.noise, v.thumpRe, v.thumpIm = 0, 0, 0
	}
	v.wasDown, v.wasDamped = true, false

	// the precursor: the hammer pulls the string along its length, a short burst of longitudinal
	// waves (their own high resonances, no transverse partials) in the bass struck hard
	v.preOn, v.preN = false, 0
	v.preLevel = 0
	if cfg.Precursor > 0 {
		low := math.Max(0, math.Min(1, (precursorTop-v.pitch)/(precursorTop-precursorFrom)))
		// in the units of the output (the balance of the keys keeps it as loud against the note)
		v.preLevel = precursorGain * level / v.amp * math.Min(cfg.Precursor, 1) * low * speed * speed * speed
		fl := math.Min(math.Max(longitudinalRatio*f0, 400), 3000)
		for j := range v.pre {
			// the longitudinal modes and the sums of transverse partials they carry (2x-ish)
			fj := math.Min(fl*[...]float64{1, 1.62, 2.4}[j], 0.4*sr)
			tau := 0.012 / (1 + fj/2500)
			v.preRot[j] = cmplx.Rect(math.Exp(-1/(tau*sr)), 2*math.Pi*fj/sr)
			v.preAmp[j] = 1 / float64(j+1)
		}
	}

	// the key bottoms out on the keybed (a loud strike lands sooner and harder, a soft one is let
	// down slowly and hardly heard); the action, not the strings, so the voicing leaves it alone
	v.bedIn = -1
	if bed := math.Max(0, math.Min(cfg.KeyBed, 1)) * math.Max(0, math.Min(cfg.Mechanics, 1)); bed > 0 {
		v.bedLevel = bed * keyBedNoise * amplitude * math.Pow(speed, keyBedPower) * v.knockAmp / v.amp
		v.bedIn = int((0.004+0.016*(1-speed))*sr) + tick.SampleOffset
	}

	// the strike: the velocity sets the speed of the hammer, the level goes with a power of it (a
	// recorded grand is some 24 dB louder at fortissimo than at pianissimo), VelocitySens of it;
	// velocity 0.8 is the level the presets are calibrated at
	sens := math.Max(0, math.Min(cfg.VelocitySens, 1))
	v.force = amplitude * math.Pow(hammerVelocity(v.velocity)/hammerVelocity(0.8), velocityPower*sens-interpolateLinear(velocityBalance, v.pitch))
	v.force *= math.Pow(10, vary*0.6*(2*v.Rand().Float64()-1)/20) // no strike is the same
	if v.soft {
		v.force *= softLevel
	}
	v.preLevel *= v.force

	// what is left of a strike still playing joins the new one
	for k, b := range [2][]float64{v.exc, v.knock} {
		v.tail[k] = v.tail[k][:0]
		if restrike && v.excPos >= 0 && v.excPos < len(b) {
			v.tail[k] = append(v.tail[k], b[v.excPos:]...)
		}
	}
	v.excitation(f0, sr)
	for i := range v.exc {
		v.exc[i] *= v.force
		v.knock[i] *= v.force
	}
	for k, b := range [2][]float64{v.exc, v.knock} {
		for i, x := range v.tail[k] {
			if i >= len(b) {
				break
			}
			b[i] += x
		}
	}
	v.excPos = -tick.SampleOffset // the strike starts at its sample, the strings keep sounding
}

// Synthesize renders a buffer of the voice alone (in an instrument the voices step together, see
// Piano.Synthesize, so they share the bridge)
func (v *Voice) Synthesize() {
	if !v.begin() {
		return
	}
	solo := v.bridge == nil && v.bridgeAmt > 0
	if solo {
		// the voice alone is loaded by a bridge of its own
		if v.own == nil {
			v.own = newBridge(v.Config().SampleRate)
		}
		v.bridge = v.own
	}
	for i := range v.Config().BufferSize {
		v.stringsStep(i)
		if solo {
			v.bridge.tick()
		}
		v.outStep(i)
	}
	if solo {
		v.bridge = nil
	}
	v.end()
}

// begin sets up a buffer, returns whether the voice sounds
func (v *Voice) begin() bool {
	out := v.Output(0)
	out.Clear()
	if !v.active {
		return false
	}

	var (
		sr = v.Config().SampleRate
		r  = &v.run
	)
	// the dampers: lifted while the key is down or a pedal holds them, on the strings they lose
	// enough per loop to fall 60 dB in DamperTime, touching them lightly (half pedaling) less
	r.contact = v.contact()
	r.damped = r.contact > 0.5
	r.targets = v.damperTargets(r.contact)
	r.glide = 1 - math.Exp(-1/(0.01*sr))
	r.peak = 0
	r.period = sr / (440 * math.Exp2((stretched(v.pitch, &v.cfg)-69)/12))
	_, _, mass := woundTraits(v.pitch, &v.cfg)
	r.coupling = bridgeDrain(v.pitch) * mass * r.period / sr / float64(len(v.strings)/2) // per loop
	r.g = math.Sqrt(r.coupling)
	r.drainCoef = 1 - math.Exp(-2*math.Pi*drainHz/sr)
	r.cross = crossCoupling * r.period / sr
	r.mech = math.Max(0, math.Min(v.cfg.Mechanics, 1))
	r.dcCoef = 1 - math.Exp(-2*math.Pi*20/sr)
	r.envCoef = 1 - math.Exp(-1/(0.005*sr))
	r.noiseLPCoef = 1 - math.Exp(-2*math.Pi*2500/sr)
	r.noiseHPCoef = 1 - math.Exp(-2*math.Pi*300/sr)
	r.noiseDecay = math.Exp(-1 / (0.035 * sr))
	v.thumpRot = cmplx.Rect(math.Exp(-1/(0.02*sr)), 2*math.Pi*90/sr)
	v.bridgeAmt = 0
	if !v.bridgeOff {
		v.bridgeAmt = math.Max(0, math.Min(v.cfg.Bridge, 1))
	}
	return true
}

// stringsStep is the first half of a sample: the strike, what comes around the loop of every string,
// and what the strings put on the bridge. The bridge moves, then outStep follows
func (v *Voice) stringsStep(i int) {
	r := &v.run
	if v.releaseIn >= 0 {
		if v.releaseIn == 0 {
			v.keyDown = false
			r.contact = v.contact()
			r.damped = r.contact > 0.5
			r.targets = v.damperTargets(r.contact)
		}
		v.releaseIn--
	}

	// the strike: every string gets it a little later or earlier (the hammer doesn't reach
	// the strings of a key at exactly the same moment)
	pos := v.excPos
	r.kn = 0
	if pos >= 0 && pos < len(v.knock) {
		r.kn = v.knock[pos] * v.knockAmp
	}
	if pos < len(v.exc)+maxStrikeSpread {
		v.excPos++
	}
	r.pos = pos

	// settling: the loops a little shorter while the strings are still stretched
	if v.settle > 1e-3 {
		v.settle *= v.settleDecay
		shift := r.period * v.settle * (math.Ln2 / 1200)
		for _, s := range v.strings {
			s.shift = shift
		}
	} else if v.settle > 0 {
		v.settle = 0
		for _, s := range v.strings {
			s.shift = 0
		}
	}

	// the strings: what comes around every loop reaches the bridge
	t, bridge := 0.0, 0.0
	for k, s := range v.strings {
		s.damper += (r.targets[k] - s.damper) * r.glide
		s.y = s.next()
		t += s.y
		if k%2 == 0 {
			bridge += s.y
		}
	}
	r.t, r.bridge = t, bridge
	if v.bridgeAmt > 0 {
		v.bridge.f += r.g * bridge
	}
}

// outStep is the second half of a sample: the strings take what the bridge gives back, the bark,
// the precursor and the noise of the action are added, and the voice is written out
func (v *Voice) outStep(i int) {
	var (
		r      = &v.run
		sr     = v.Config().SampleRate
		outs   = v.Output(0).Channels
		t      = r.t
		bridge = r.bridge
		pos    = r.pos
	)

	// the soundboard takes the low frequencies of the bridge more than the highs (its low modes
	// move most), so the fundamental and the low partials fade first and a bass note gets
	// brighter as it rings
	v.drainLP += r.drainCoef * (bridge - v.drainLP)
	drain := drainLow*v.drainLP + drainHigh*(bridge-v.drainLP)
	// the loss of the vertical polarizations: the drain of the key on its own, or the bridge the
	// strings of all keys share (the admittance of the board and what the other strings put on it)
	loss := r.coupling * drain
	if v.bridgeAmt > 0 {
		loss = (1-v.bridgeAmt)*loss + v.bridgeAmt*r.g*v.bridge.v
	}

	// the bridge: the vertical polarizations moving together lose to the soundboard, a
	// little passes to the horizontal ones
	for k, s := range v.strings {
		x := 0.0
		if i := pos - s.delay; i >= 0 && i < len(v.exc) {
			x = v.exc[i]
		}
		if k%2 == 0 {
			s.push(x*s.in + s.y - loss)
		} else {
			s.push(x*s.in + s.y + r.cross*bridge)
		}
	}

	// the bark: phantom partials from the square of the motion, in the longitudinal mode
	if v.barkLevel > 0 && v.force > 0 {
		// the stretch goes with the square of the motion: relative to the strike (the speed
		// of the hammer is in barkLevel), so it doesn't depend on the level of the note
		tn := t / v.force
		sq := tn * tn * v.force
		v.barkDC += r.dcCoef * (sq - v.barkDC)
		hp := sq - v.barkDC
		bp := v.b0*(hp-v.bx2) - v.a1*v.by1 - v.a2*v.by2
		v.bx2, v.bx1 = v.bx1, hp
		v.by2, v.by1 = v.by1, bp
		t += v.barkLevel * (0.7*bp + 0.3*hp)
	}

	// the precursor: the burst of longitudinal waves that outruns the transverse note
	if v.preLevel > 0 {
		if pos == 0 {
			v.preOn = true
			for j := range v.pre {
				v.pre[j] = cmplx.Rect(1, math.Pi*keyRandom(v.key, 10+float64(j)))
			}
		}
		if v.preOn {
			x := 0.0
			for j := range v.pre {
				x += v.preAmp[j] * real(v.pre[j])
				v.pre[j] *= v.preRot[j]
			}
			t += v.preLevel * x
			if v.preN++; v.preN > int(0.08*sr) {
				v.preOn = false
			}
		}
	}

	// the key landing on the keybed: a low thud and a dull knock of wood
	if v.bedIn >= 0 {
		if v.bedIn == 0 {
			v.noise += v.bedLevel * 0.5
			v.thumpRe += v.bedLevel
		}
		v.bedIn--
	}

	// the action: the dampers landing on the vibrating strings, the key coming up
	v.env += r.envCoef * (math.Abs(t) - v.env)
	if v.keyDown != v.wasDown || r.damped != v.wasDamped {
		if !v.keyDown && v.wasDown {
			// as loud as what is left of the note: a key held while the note died away is let go
			// in a quiet room, and comes up gently
			left := 1.0
			if v.loudest > 0 {
				left = math.Min(v.peak/v.loudest, 1)
			}
			v.noise += r.mech * v.keyNoise * left
			v.thumpRe += r.mech * v.keyNoise * left * 2
		}
		if r.damped && !v.wasDamped {
			v.noise += r.mech * damperNoise * v.env
		}
		v.wasDown, v.wasDamped = v.keyDown, r.damped
	}
	if v.noise > 1e-7 || math.Abs(v.thumpRe) > 1e-7 {
		n := (v.Rand().Float64()*2 - 1) * v.noise
		v.noiseLP += r.noiseLPCoef * (n - v.noiseLP)
		v.noiseHP += r.noiseHPCoef * (v.noiseLP - v.noiseHP)
		t += v.noiseLP - v.noiseHP
		v.noise *= r.noiseDecay
		c := complex(v.thumpRe, v.thumpIm) * v.thumpRot
		v.thumpRe, v.thumpIm = real(c), imag(c)
		t += v.thumpRe
	}

	y := t * v.amp
	kn := r.kn
	r.peak = math.Max(r.peak, math.Max(math.Abs(y), math.Abs(kn)))
	// the knock is the soundboard already: in an instrument it is heard beside the soundboard,
	// not through it again
	if v.knockOut != nil {
		v.knockOut[i] = kn
	} else {
		y += kn
	}

	if len(outs) == 1 {
		outs[0][i] = float32(y)
	} else {
		outs[0][i] = float32(y * v.gainL)
		outs[1][i] = float32(y * v.gainR)
	}
}

// end closes a buffer: the voice rests when it has died away
func (v *Voice) end() {
	peak := v.run.peak
	v.peak = peak
	v.loudest = math.Max(v.loudest, peak)

	// rest when the strings have died away and the key is up (a key held down keeps its dampers
	// up, its strings can ring along with others)
	// (silent: below -90 dB, or 60 dB below the loudest it was, inaudible beside anything)
	if v.excPos >= len(v.exc)+maxStrikeSpread && peak < math.Max(3e-5, 1e-3*v.loudest) && !v.keyDown {
		v.quiet++
		if v.quiet > 2 {
			v.active = false
		}
	} else {
		v.quiet = 0
	}
}

// the noise of the action: the key coming up (small, at velocity 0.8) and the dampers landing on
// the strings (as loud as the strings still are)
const (
	keyNoise    = 0.00013
	damperNoise = 0.3
	// the key landing on the keybed at full velocity, and how fast that falls off with the speed of
	// the hammer
	keyBedNoise = 0.02
	keyBedPower = 3
)

// damperTargets returns the loop gain the dampers leave every string at a contact
func (v *Voice) damperTargets(contact float64) []float64 {
	v.targets = v.targets[:0]
	for _, s := range v.strings {
		v.targets = append(v.targets, dampedGain(s.damped, contact))
	}
	return v.targets
}

// the half pedal: between halfPedalLow and halfPedalHigh of the pedal travel the dampers touch the
// strings more and more lightly
const (
	halfPedalLow  = 0.3
	halfPedalHigh = 0.7
)

// damperContact returns how firmly a damper is on the string of a key at a sustain pedal position:
// 1 fully on (the pedal up), 0 lifted. In between the heavy bass strings push the light touch away
// more easily than the treble, so half pedaling clears the treble and lets the bass ring
func damperContact(pedal, pitch float64) float64 {
	c := math.Max(0, math.Min((halfPedalHigh-pedal)/(halfPedalHigh-halfPedalLow), 1))
	if c == 0 || c == 1 {
		return c
	}
	treble := math.Max(0, math.Min((pitch-36)/48, 1))
	return math.Pow(c, 1.6-0.8*treble)
}

// dampedGain returns the loop gain of a string with its damper at a contact, from the gain fully
// damped
func dampedGain(damped, contact float64) float64 {
	return math.Pow(damped, contact*contact)
}

// maxStrikeSpread is the most samples the hammer reaches one string of a key later than another
const maxStrikeSpread = 6

// keyRandom returns a number from -1 to 1 for a key, the same every time (a property of the
// instrument), another for every salt
func keyRandom(key int, salt float64) float64 {
	x := math.Sin(float64(key)*12.9898+salt*78.233) * 43758.5453
	return 2*(x-math.Floor(x)) - 1
}

// keyBalanceTable is the gain of the keys that evens out their loudness (the higher strings keep
// more of the strike, the ear hears the bass less, the soundboard is not even): fitted key by key
// to the loudness (K-weighted) of the presets at velocity 0.8, at balanceHardness
var keyBalanceTable = func() [][2]float64 {
	db := []float64{ // from key 21
		13.7, 13.9, 14.2, 13.5, 14.4, 13.6, 13.4, 12.7, 9.4, 10.3, 9.0, 8.9,
		8.4, 8.7, 7.9, 7.0, 7.3, 5.9, 8.2, 6.0, 4.2, 4.5, 5.4, 5.4,
		2.7, 2.8, 1.4, 1.7, 1.7, 1.1, 2.5, 1.0, -0.7, 0.0, 2.0, 1.7,
		0.2, 1.3, 0.5, 0.6, 0.6, -0.5, 0.2, 0.1, -1.6, -1.5, 1.8, 0.5,
		-1.0, -0.1, -1.5, -2.4, -1.1, -1.2, -0.7, -0.3, -1.2, 0.3, 1.3, -0.6,
		-1.8, -1.0, -1.7, -1.8, -2.0, -3.7, -3.6, -0.2, -3.4, -5.6, -4.5, -9.3,
		-8.8, -7.8, -13.1, -9.7, -9.7, -10.8, -9.4, -10.8, -6.9, -12.6, -7.4, -9.2,
		-8.5, -10.4, -9.2, -8.5,
	}
	t := make([][2]float64, len(db))
	for i, x := range db {
		t[i] = [2]float64{float64(21 + i), math.Pow(10, x/20)}
	}
	return t
}()

// balanceHardness is the hardness keyBalanceTable is fitted at
const balanceHardness = 0.42

// hardnessBalance is how much louder (dB) a key sounds per unit of hardness above balanceHardness,
// against the middle of the keyboard: harder felt brightens the middle (louder to the ear), the
// treble is nearly pure and gains nothing. It is taken away again
var hardnessBalance = [][2]float64{{21, 1.0}, {27, 0.1}, {33, -2.3}, {39, -1.0}, {45, -1.3}, {51, -1.9}, {57, -1.2}, {63, 0.9}, {69, 1.2}, {75, 2.1}, {81, 1.8}, {87, -0.4}, {93, -2.2}, {99, -4.8}, {105, -6.4}, {108, -5.6}}

// velocityBalance is how much more than middle C (dB per dB of hammer speed) a key grows louder
// with the velocity: a hard strike brightens the keys above it most where the ear is most
// sensitive, the bass and the nearly pure treble less. It is taken away again, so every key has
// the same dynamic range
var velocityBalance = [][2]float64{{21, -0.06}, {27, -0.10}, {33, -0.17}, {39, -0.21}, {45, -0.19}, {51, -0.12}, {57, -0.07}, {63, 0.01}, {69, 0.04}, {75, 0.13}, {81, 0.08}, {87, -0.07}, {93, -0.21}, {99, -0.32}, {105, -0.38}, {108, -0.39}}

// keyBalance evens out the loudness of the keys
func keyBalance(pitch float64) float64 { return interpolate(keyBalanceTable, pitch) }
