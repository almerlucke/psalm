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
// dampers are up ring along with what is played (sympathetic resonance), see Piano
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
	SoundboardIR   [][]float64 // a measured soundboard response (per side) instead of the modelled one
	Resonance      float64     // 0-1, sympathetic resonance of the strings whose dampers are up
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
		Resonance:      0.5,
		Level:          1,
	}
}

// highestDamper is the highest key with a damper, the keys above ring on
const highestDamper = 88

// the two polarizations of a string: the prompt sound decays promptFactor times as long as the
// ring time of the key, the aftersound afterFactor times, starting afterLevel as loud
const (
	promptFactor = 0.6
	afterFactor  = 2.2
	afterLevel   = 0.2
	afterCents   = 0.3 // the aftersound is a little apart in pitch
)

// the strings of a key share the bridge: what they do together (the vertical polarizations moving
// in phase) drains into the soundboard, unisonCoupling of it per loop over the strings, what they
// do against each other stays (Weinreich); crossCoupling of the bridge motion passes to the
// horizontal polarizations
const (
	unisonCoupling = 0.003
	crossCoupling  = 0.001
)

// the bark: a string stretched by its own motion (the square of it) rings its longitudinal mode,
// at about longitudinalRatio times the fundamental, with partials at the sums of the partials;
// strongest in the bass, gone above barkTop
const (
	barkGain          = 8.0
	longitudinalRatio = 17.0
	barkTop           = 66.0
)

// the soft pedal (una corda): the hammers strike one string less, with a softer part of the felt;
// the string left out only rings along a little
const (
	softHardness = 0.4
	softLevel    = 1.5
	softCoupling = 0.06
)

// Voice is a piano voice: the strings of one key. Notes take "pitch" (or "frequency") and
// "velocity" (0-1, default 0.8); "sustain" (above 0.5 is down) is the sustain pedal, "soft" the
// soft pedal
type Voice struct {
	*psalm.Module
	cfg Config

	strings  []*str
	exc      []float64    // the strike, played into the strings
	knock    []float64    // the knock of the soundboard, heard directly
	tail     [2][]float64 // what was left of the strike before, when the key is struck again
	excPos   int          // position in the strike, negative waits for the sample of the strike
	pitch    float64
	key      int
	velocity float64
	amp      float64       // output gain of the key: the level and the keyboard balance
	voicing  *[128]float64 // the gains of the keys that even out the soundboard of the instrument
	force    float64       // how hard the hammer strikes
	peak     float64       // the loudest sample of the last buffer
	gainL    float64
	gainR    float64

	keyDown   bool
	sustain   bool
	soft      bool
	releaseIn int // samples until a timed note releases, -1 is none

	// the bark: a highpass and a resonator on the square of the string motion
	barkLevel          float64
	barkDC             float64
	b0, a1, a2         float64
	bx1, bx2, by1, by2 float64
	env                float64 // level of the strings
	wasDamped, wasDown bool
	noise              float64 // level of the action noise, decaying
	noiseLP, noiseHP   float64
	thumpRe, thumpIm   float64 // the key landing, a short low thud
	thumpRot           complex128
	quiet              int // buffers the voice has been silent, after the strike
	active             bool
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
	dispersion(v.Config().SampleRate, math.Max(0, cfg.Stiffness)) // fit the strings now, not at the first note
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
	msg.IfFltVal("resonance", func(f float64) { cfg.Resonance = f })
	msg.IfFltVal("sustain", func(f float64) { v.SetSustain(f > 0.5) })
	msg.IfFltVal("soft", func(f float64) { v.soft = f > 0.5 })
}

func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
	}
}

// SetSustain puts the sustain pedal down or lifts it
func (v *Voice) SetSustain(down bool) { v.sustain = down }

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
	v.releaseIn = releaseIn
	v.keyDown = true
	v.quiet = 0
	v.active = true

	f0 := 440 * math.Exp2((v.pitch-69)/12)
	n := numStrings(v.pitch, cfg.Strings)
	disp := dispersion(sr, math.Max(0, cfg.Stiffness))[max(0, min(int(math.Round(v.pitch)), 127))]
	design := stringDesign{
		sr:      sr,
		stages:  disp.stages,
		ad:      disp.ad,
		t60:     ringTime(v.pitch) * math.Max(0.05, cfg.Decay),
		damping: highDamping * math.Max(0, cfg.Damping),
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
		d.t60 *= 1 + 0.08*float64(i) // the strings decay a little differently
		in := 1.0
		if i >= struck {
			in = softCoupling
		}
		prompt, after := d, d
		prompt.t60 *= promptFactor
		after.t60 *= afterFactor
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

	// key position in the stereo field, low keys on the left
	pan := 0.5 + 0.5*math.Max(0, math.Min(cfg.Width, 1))*math.Max(-1, math.Min((v.pitch-64)/44, 1))
	v.gainL = math.Cos(pan * math.Pi / 2)
	v.gainR = math.Sin(pan * math.Pi / 2)

	// the level of the key and the keyboard balance, so the keys sound about equally loud (the
	// output gain stays the same when the key is struck again, the strike sets the loudness)
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	v.amp = level * keyBalance(v.pitch) / math.Sqrt(float64(n))
	if v.voicing != nil {
		v.amp *= v.voicing[v.key]
	}

	// the bark: the longitudinal resonance, only the bass and tenor
	w := math.Max(0, math.Min(1, (barkTop-v.pitch)/(barkTop-45)))
	v.barkLevel = barkGain * math.Max(0, math.Min(cfg.Bark, 1)) * w
	fl := math.Min(math.Max(longitudinalRatio*f0, 300), 3000)
	wl := 2 * math.Pi * fl / sr
	alpha := math.Sin(wl) / (2 * 6) // Q 6
	v.b0 = alpha / (1 + alpha)
	v.a1 = -2 * math.Cos(wl) / (1 + alpha)
	v.a2 = (1 - alpha) / (1 + alpha)
	if !restrike {
		v.barkDC, v.bx1, v.bx2, v.by1, v.by2, v.env = 0, 0, 0, 0, 0, 0
		v.noise, v.thumpRe, v.thumpIm = 0, 0, 0
	}
	v.wasDown, v.wasDamped = true, false

	// the strike: the velocity (always the tone, VelocitySens of it the level)
	vel := v.velocity
	sens := math.Max(0, math.Min(cfg.VelocitySens, 1))
	v.force = amplitude * (1 - sens + sens*vel*vel/0.64)
	if v.soft {
		v.force *= softLevel
	}

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

func (v *Voice) Synthesize() {
	out := v.Output(0)
	out.Clear()
	if !v.active {
		return
	}

	var (
		sr   = v.Config().SampleRate
		outs = out.Channels
		bs   = v.Config().BufferSize
	)

	// the damper: lifted while the key is down or the pedal holds it, otherwise the strings lose
	// enough per loop to fall 60 dB in DamperTime (the keys at the top have no dampers)
	damped := !v.keyDown && !v.sustain && v.pitch <= highestDamper
	glide := 1 - math.Exp(-1/(0.01*sr))
	peak := 0.0
	coupling := unisonCoupling / float64(len(v.strings)/2)
	mech := math.Max(0, math.Min(v.cfg.Mechanics, 1))
	dcCoef := 1 - math.Exp(-2*math.Pi*20/sr)
	envCoef := 1 - math.Exp(-1/(0.005*sr))
	noiseLPCoef := 1 - math.Exp(-2*math.Pi*2500/sr)
	noiseHPCoef := 1 - math.Exp(-2*math.Pi*300/sr)
	noiseDecay := math.Exp(-1 / (0.035 * sr))
	v.thumpRot = cmplx.Rect(math.Exp(-1/(0.02*sr)), 2*math.Pi*90/sr)

	for i := range bs {
		if v.releaseIn >= 0 {
			if v.releaseIn == 0 {
				v.keyDown = false
				damped = !v.sustain && v.pitch <= highestDamper
			}
			v.releaseIn--
		}

		x, y := 0.0, 0.0
		if v.excPos < 0 {
			v.excPos++
		} else if v.excPos < len(v.exc) {
			x, y = v.exc[v.excPos], v.knock[v.excPos]
			v.excPos++
		}

		// the strings: what comes around every loop reaches the bridge
		t, bridge := 0.0, 0.0
		for k, s := range v.strings {
			target := 1.0
			if damped {
				target = s.damped
			}
			s.damper += (target - s.damper) * glide
			s.y = s.next()
			t += s.y
			if k%2 == 0 {
				bridge += s.y
			}
		}
		// the bridge: the vertical polarizations moving together lose to the soundboard, a
		// little passes to the horizontal ones
		for k, s := range v.strings {
			if k%2 == 0 {
				s.push(x*s.in + s.y - coupling*bridge)
			} else {
				s.push(x*s.in + s.y + crossCoupling*bridge)
			}
		}

		// the bark: phantom partials from the square of the motion, in the longitudinal mode
		if v.barkLevel > 0 {
			sq := t * t
			v.barkDC += dcCoef * (sq - v.barkDC)
			hp := sq - v.barkDC
			bp := v.b0*(hp-v.bx2) - v.a1*v.by1 - v.a2*v.by2
			v.bx2, v.bx1 = v.bx1, hp
			v.by2, v.by1 = v.by1, bp
			t += v.barkLevel * (0.7*bp + 0.3*hp)
		}

		// the action: the dampers landing on the vibrating strings, the key coming up
		v.env += envCoef * (math.Abs(t) - v.env)
		if v.keyDown != v.wasDown || damped != v.wasDamped {
			if !v.keyDown && v.wasDown {
				v.noise += mech * keyNoise
				v.thumpRe += mech * keyNoise * 2
			}
			if damped && !v.wasDamped {
				v.noise += mech * damperNoise * v.env
			}
			v.wasDown, v.wasDamped = v.keyDown, damped
		}
		if v.noise > 1e-7 || math.Abs(v.thumpRe) > 1e-7 {
			n := (v.Rand().Float64()*2 - 1) * v.noise
			v.noiseLP += noiseLPCoef * (n - v.noiseLP)
			v.noiseHP += noiseHPCoef * (v.noiseLP - v.noiseHP)
			t += v.noiseLP - v.noiseHP
			v.noise *= noiseDecay
			c := complex(v.thumpRe, v.thumpIm) * v.thumpRot
			v.thumpRe, v.thumpIm = real(c), imag(c)
			t += v.thumpRe
		}

		y += t
		y *= v.amp
		peak = math.Max(peak, math.Abs(y))

		if len(outs) == 1 {
			outs[0][i] = float32(y)
		} else {
			outs[0][i] = float32(y * v.gainL)
			outs[1][i] = float32(y * v.gainR)
		}
	}
	v.peak = peak

	// rest when the strings have died away and the key is up (a key held down keeps its dampers
	// up, its strings can ring along with others)
	if v.excPos >= len(v.exc) && peak < 3e-5 && !v.keyDown {
		v.quiet++
		if v.quiet > 2 {
			v.active = false
		}
	} else {
		v.quiet = 0
	}
}

// the noise of the action: the key coming up (a constant, small) and the dampers landing on the
// strings (as loud as the strings still are)
const (
	keyNoise    = 0.0004
	damperNoise = 0.05
)

// keyBalanceTable is the gain of the keys that evens out their loudness (the higher strings keep
// more of the strike, the ear hears the bass less), the top a little quieter like a real piano
var keyBalanceTable = func() [][2]float64 {
	db := [][2]float64{{21, 15}, {33, 10}, {45, 4.5}, {57, 0}, {69, -1}, {81, -2}, {93, -3}, {99, -1.5}, {105, 1.8}, {108, 0}}
	for i := range db {
		db[i][1] = math.Pow(10, db[i][1]/20)
	}
	return db
}()

// keyBalance evens out the loudness of the keys
func keyBalance(pitch float64) float64 { return interpolate(keyBalanceTable, pitch) }
