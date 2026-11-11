// Package drumsynth makes drums from oscillators and noise, the way the classic drum machines do:
// a kick is a sine falling in pitch, a snare a pair of tuned drums with a burst of noise, a hihat
// six square waves at clashing pitches through a high filter. A kit is a set of sounds by name,
// played like the sample kits of drums.NewDrums: a step sequencer row plays a sound by its name
// (stepseq.Sound(kit, "kick")), a note off chokes it, a keyboard plays them on the General MIDI keys.
//
//	kit := drumsynth.NewPoly(drumsynth.DefaultConfig(), 16).AddTo(p) // the 808
package drumsynth

import (
	"math"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/modules/poly"
)

// Kind is what a sound is made of
type Kind int

const (
	Kick    Kind = iota // a sine falling in pitch, a click
	Snare               // two tuned drum heads and filtered noise (the wires)
	Hat                 // six square waves through a high filter, and noise
	Tom                 // a sine falling in pitch, a little noise
	Clap                // bursts of filtered noise, then a tail
	Rim                 // two short high tones and a click
	Cowbell             // two square waves through a band filter, dying in two steps
	Cymbal              // the hat's metal, lower and long
	Shaker              // a shake of filtered noise (maracas, cabasa), with jingles (a tambourine)
)

// Sound is a drum of a kit. What Tone and Snap do depends on the kind
type Sound struct {
	Kind      Kind
	Pitch     float64 // Hz of the body (a hat or cymbal: a factor on its metal, 0 is 1)
	Decay     float64 // seconds to die away (60 dB)
	Sweep     float64 // semitones the pitch falls from
	SweepTime float64 // seconds the sweep takes (to a third)
	Tone      float64 // 0-1: the filter of the noise or metal, low to high
	Snap      float64 // 0-1: the attack: the click of a kick, the wires of a snare, the noise of a hat
	Drive     float64 // 0-1: saturation
	Level     float64 // linear (0 is 1)
	Pan       float64 // -1 left to 1 right
	Choke     int     // a group: a sound starting stops the others of its group (0 is none)
}

// Kit is a set of sounds by name, and the keys that play them
type Kit struct {
	Sounds map[string]Sound
	// Keys maps MIDI keys to sounds, for notes that come with a pitch instead of a name (nil is
	// General MIDI)
	Keys map[int]string
}

// GeneralMIDI are the keys of the General MIDI drums, for the names of the kits here (every kit
// has all of them)
var GeneralMIDI = map[int]string{
	35: "kick", 36: "kick", 37: "rim", 38: "snare", 39: "clap", 40: "snare",
	41: "tomLow", 43: "tomLow", 45: "tomMid", 47: "tomMid", 48: "tomHigh", 50: "tomHigh",
	42: "hat", 44: "hat", 46: "open",
	49: "crash", 52: "crash", 55: "crash", 57: "crash", // crash, china, splash, crash 2
	51: "ride", 53: "rideBell", 59: "ride",
	54: "tambourine", 56: "cowbell", 58: "shaker", // vibraslap
	60: "bongoHigh", 61: "bongoLow", 62: "congaMute", 63: "congaHigh", 64: "congaLow",
	65: "timbaleHigh", 66: "timbaleLow", 67: "agogoHigh", 68: "agogoLow",
	69: "shaker", 70: "shaker", 73: "shaker", 74: "shaker", // cabasa, maracas, guiro
	75: "clave", 76: "woodblockHigh", 77: "woodblockLow", 80: "triangleMute", 81: "triangle",
}

// Config is a drum synth playing a kit
type Config struct {
	NumChannels int
	Kit         Kit
	Level       float64 // output level (0 is treated as 1)
}

// DefaultConfig plays the 808 kit
func DefaultConfig() Config {
	return Config{NumChannels: psalm.Stereo, Kit: Kit808(), Level: 1}
}

// the metal of the 808 hats and cymbal: six square waves
var metal = [6]float64{205.3, 304.4, 369.6, 522.7, 540, 800}

// chokeTime is how long a choked sound takes to stop (short, without a click)
const chokeTime = 0.008

type shared struct{ voices []*Voice }

// Voice plays a sound
type Voice struct {
	*psalm.Module
	cfg       Config
	shared    *shared
	snd       Sound
	speed     float64
	amp       float64
	t         int // samples since the hit
	env       float64
	envCoef   float64
	clickEnv  float64
	bodyEnv   float64
	bodyCoef  float64
	phase     [6]float64
	choke     float64 // the gain while choked, falling
	chokeStep float64
	filt      [2]*svf.SVF
	rng       uint32
	gainL     float64
	gainR     float64
	waitSamps int
	active    bool
	expr      expression.Expression // the pitch bend (the sounds ringing); the mod wheel makes new hits ring longer
}

// New returns a voice; voices of New don't choke each other, NewPoly's do
func New(cfg Config) *Voice {
	return newVoice(cfg, nil)
}

func newVoice(cfg Config, sh *shared) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	v := &Voice{Module: psalm.NewModule(nil, []int{cfg.NumChannels}), cfg: cfg}
	sr := v.Config().SampleRate
	v.filt = [2]*svf.SVF{svf.New(1000, 0, sr), svf.New(1000, 0, sr)}
	v.rng = uint32(v.Rand().Uint64()) | 1
	if sh == nil {
		sh = &shared{}
	}
	sh.voices = append(sh.voices, v)
	v.shared = sh
	v.SetSelf(v)
	return v
}

// ReceiveMessage takes the level of the whole ("level"), the bend and the wheel, and the settings
// of each sound of the kit by its name: "<sound><Setting>" ("kickPitch", "snareDecay", "hatTone"),
// the settings Pitch, Decay, Sweep, SweepTime, Tone, Snap, Drive, Level and Pan (see Sound). A
// sound takes them from its next hit
func (v *Voice) ReceiveMessage(msg psalm.Message) {
	v.expr.Receive(msg)
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
	for key, val := range msg {
		for _, st := range soundSettings {
			name, ok := strings.CutSuffix(key, st.name)
			if !ok || name == "" {
				continue
			}
			snd, ok := v.cfg.Kit.Sounds[name]
			if !ok {
				break
			}
			*st.field(&snd) = psalm.FltVal[float64](val)
			v.cfg.Kit.Sounds[name] = snd // the kit is shared by the voices of a poly: all of them hear it
			break
		}
	}
}

// soundSettings are the settings of a sound by message, the longer names first (SweepTime before
// Sweep)
var soundSettings = []struct {
	name  string
	field func(*Sound) *float64
}{
	{"SweepTime", func(s *Sound) *float64 { return &s.SweepTime }},
	{"Sweep", func(s *Sound) *float64 { return &s.Sweep }},
	{"Pitch", func(s *Sound) *float64 { return &s.Pitch }},
	{"Decay", func(s *Sound) *float64 { return &s.Decay }},
	{"Tone", func(s *Sound) *float64 { return &s.Tone }},
	{"Snap", func(s *Sound) *float64 { return &s.Snap }},
	{"Drive", func(s *Sound) *float64 { return &s.Drive }},
	{"Level", func(s *Sound) *float64 { return &s.Level }},
	{"Pan", func(s *Sound) *float64 { return &s.Pan }},
}

func (v *Voice) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
	}
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.hit(amplitude, message, tick)
}

// Note plays a sound to its end, whatever the duration: a drum is a hit
func (v *Voice) Note(_ float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.hit(amplitude, message, tick)
}

// soundOf returns the sound a note asks for: by "sound" (its name), or by "pitch" (a key)
func (v *Voice) soundOf(msg psalm.Message) (Sound, bool) {
	kit := &v.cfg.Kit
	name, ok := msg.StrVal("sound")
	if !ok {
		key := -1
		msg.IfFltVal("pitch", func(p float64) { key = int(math.Round(p)) })
		keys := kit.Keys
		if keys == nil {
			keys = GeneralMIDI
		}
		if name, ok = keys[key]; !ok {
			return Sound{}, false
		}
	}
	s, ok := kit.Sounds[name]
	return s, ok
}

func (v *Voice) hit(amplitude float64, msg psalm.Message, tick psalm.CtrlTick) {
	snd, ok := v.soundOf(msg)
	if !ok {
		v.active = false
		return
	}
	if snd.Choke != 0 {
		for _, o := range v.shared.voices {
			if o != v && o.active && o.snd.Choke == snd.Choke {
				o.NoteOff()
			}
		}
	}
	sr := v.Config().SampleRate
	v.snd = snd
	v.speed = 1
	msg.IfFltVal("speed", func(f float64) {
		if f > 0 {
			v.speed = f
		}
	})
	v.amp = amplitude
	msg.IfFltVal("velocity", func(f float64) { v.amp *= max(0, min(f, 1)) })
	v.waitSamps = tick.SampleOffset
	v.t = 0
	v.env = 1
	v.clickEnv = 1
	v.expr.Settle()
	v.envCoef = fall(snd.Decay*(1+2*v.expr.Wheel()), sr) // the wheel: up to three times as long
	v.bodyEnv = 1
	v.bodyCoef = fall(snd.Decay*0.5, sr) // the drum heads of a snare die before its wires
	v.choke, v.chokeStep = 1, 0
	for i := range v.phase {
		v.phase[i] = 0
	}
	v.filt[0].Clear()
	v.filt[1].Clear()
	pitch := v.pitch()
	tone := max(0, min(snd.Tone, 1))
	switch snd.Kind {
	case Snare:
		v.filt[0].SetFrequencyQ(min(1500+tone*6000, sr*0.45), 0.7)
	case Hat, Cymbal:
		base := 7000.0
		if snd.Kind == Cymbal {
			base = 4000
		}
		v.filt[0].SetFrequencyQ(min((base+3000*tone)*math.Sqrt(pitch), sr*0.45), 1.2)
		v.filt[1].SetFrequencyQ(min(base*0.8, sr*0.45), 0.7)
	case Clap:
		v.filt[0].SetFrequencyQ(min(900+tone*1500, sr*0.45), 1.5)
	case Rim:
		v.filt[0].SetFrequencyQ(min(400, sr*0.45), 0.7)
	case Cowbell:
		v.filt[0].SetFrequencyQ(min(pitch*4.8, sr*0.45), 1.5)
	case Shaker:
		v.filt[0].SetFrequencyQ(min(3000+6000*tone, sr*0.45), 0.9)
		v.filt[1].SetFrequencyQ(min(6000*math.Sqrt(pitch), sr*0.45), 2)
	case Tom, Kick:
		v.filt[0].SetFrequencyQ(min(2000+tone*6000, sr*0.45), 0.7)
	}
	a := (max(-1, min(snd.Pan, 1)) + 1) * math.Pi / 4
	v.gainL, v.gainR = math.Cos(a)*math.Sqrt2, math.Sin(a)*math.Sqrt2
	v.active = true
}

// pitch returns the body frequency of the sound, at the speed of the note
func (v *Voice) pitch() float64 {
	p := v.snd.Pitch
	if p <= 0 {
		switch v.snd.Kind {
		case Hat, Cymbal, Shaker:
			p = 1
		default:
			p = 100
		}
	}
	return p * v.speed
}

// fall returns the gain per sample that dies 60 dB in seconds
func fall(seconds, sr float64) float64 {
	return math.Pow(10, -3/(max(0.002, seconds)*sr))
}

// NoteOff chokes the sound
func (v *Voice) NoteOff() {
	if v.active && v.chokeStep == 0 {
		v.chokeStep = 1 / (chokeTime * v.Config().SampleRate)
	}
}

func (v *Voice) Clear() {
	v.active = false
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool {
	return v.active
}

func (v *Voice) noise() float64 {
	v.rng ^= v.rng << 13
	v.rng ^= v.rng >> 17
	v.rng ^= v.rng << 5
	return float64(int32(v.rng)) / 2147483648
}

func (v *Voice) Synthesize() {
	out := v.Output(0)
	out.Clear()
	if !v.active {
		return
	}
	snd := &v.snd
	n := v.Config().BufferSize
	sr := v.Config().SampleRate
	level := v.cfg.Level
	if level <= 0 {
		level = 1
	}
	if snd.Level > 0 {
		level *= snd.Level
	}
	gain := level * v.amp
	v.expr.Settle()
	pitch := v.pitch() * v.expr.Ratio()
	snap := max(0, min(snd.Snap, 1))
	drive := 1 + max(0, snd.Drive)*6
	sweepTime := max(0.001, snd.SweepTime) * sr
	click := math.Exp(-1 / (0.0015 * sr))
	channels := out.Channels
	stereo := len(channels) > 1
	for i := v.waitSamps; i < n; i++ {
		t := float64(v.t)
		s := 0.0
		switch snd.Kind {
		case Kick, Tom:
			f := pitch * math.Exp2(snd.Sweep/12*math.Exp(-t/sweepTime))
			v.phase[0] += f / sr
			s = math.Sin(2*math.Pi*v.phase[0]) * v.env
			// the click of the beater: a short burst of filtered noise
			if v.clickEnv > 1e-4 {
				s += v.filt[0].Process(v.noise(), svf.LowPass) * v.clickEnv * snap * 1.5
			}
		case Snare:
			f := pitch * math.Exp2(snd.Sweep/12*math.Exp(-t/sweepTime))
			v.phase[0] += f / sr
			v.phase[1] += f * 1.83 / sr
			body := (math.Sin(2*math.Pi*v.phase[0]) + 0.6*math.Sin(2*math.Pi*v.phase[1])) * v.bodyEnv
			wires := v.filt[0].Process(v.noise(), svf.HighPass) * v.env
			s = body*0.7 + wires*(0.2+snap)
			v.bodyEnv *= v.bodyCoef
		case Hat, Cymbal:
			m := 0.0
			for k, f := range metal {
				v.phase[k] += f * pitch / sr
				v.phase[k] -= math.Floor(v.phase[k])
				if v.phase[k] < 0.5 {
					m++
				} else {
					m--
				}
			}
			m = m/6*(1-0.5*snap) + v.noise()*snap*0.5
			m = v.filt[0].Process(m, svf.BandPass)
			m = v.filt[1].Process(m, svf.HighPass)
			s = m * v.env * 2.5
		case Clap:
			// four quick claps 10 ms apart, then the room
			burst := 0.0
			if t < 0.04*sr {
				phase := math.Mod(t, 0.01*sr)
				burst = math.Exp(-phase / (0.002 * sr))
			}
			tail := 0.0
			if t > 0.03*sr {
				tail = v.env
			}
			s = v.filt[0].Process(v.noise(), svf.BandPass) * (burst*(0.5+snap) + tail) * 2
		case Rim:
			v.phase[0] += pitch / sr
			v.phase[1] += pitch * 0.273 / sr
			tone := math.Sin(2*math.Pi*v.phase[0]) + 0.5*math.Sin(2*math.Pi*v.phase[1])
			s = v.filt[0].Process(tone, svf.HighPass) * v.env
			if v.clickEnv > 1e-4 {
				s += v.noise() * v.clickEnv * snap
			}
		case Cowbell:
			v.phase[0] += pitch / sr
			v.phase[1] += pitch * 1.48 / sr
			v.phase[0] -= math.Floor(v.phase[0])
			v.phase[1] -= math.Floor(v.phase[1])
			sq := 0.0
			for _, p := range v.phase[:2] {
				if p < 0.5 {
					sq += 0.5
				} else {
					sq -= 0.5
				}
			}
			// it dies in two steps: a fast drop to 40%, then the ring
			e := v.env * (0.4 + 0.6*math.Exp(-t/(0.012*sr)))
			s = v.filt[0].Process(sq, svf.BandPass) * e * 1.5
		case Shaker:
			// the beads: noise rising in a few milliseconds; the jingles of a tambourine: the
			// metal, high (Snap is how much of them)
			m := v.filt[0].Process(v.noise(), svf.BandPass)
			if snap > 0 {
				j := 0.0
				for k, f := range metal {
					v.phase[k] += f * 2.2 * pitch / sr
					v.phase[k] -= math.Floor(v.phase[k])
					if v.phase[k] < 0.5 {
						j++
					} else {
						j--
					}
				}
				m = m*(1-0.5*snap) + v.filt[1].Process(j/6, svf.BandPass)*snap*2
			}
			s = m * v.env * (1 - math.Exp(-t/(0.004*sr))) * 2.5
		}
		if snd.Kind != Clap || t > 0.03*sr {
			v.env *= v.envCoef
		}
		v.clickEnv *= click
		if drive > 1 {
			s = math.Tanh(s*drive) / math.Tanh(drive)
		}
		if v.chokeStep > 0 {
			v.choke -= v.chokeStep
			if v.choke <= 0 {
				v.active = false
				break
			}
			s *= v.choke
		}
		s *= gain
		if stereo {
			channels[0][i] = float32(s * v.gainL)
			channels[1][i] = float32(s * v.gainR)
		} else {
			channels[0][i] = float32(s)
		}
		v.t++
		if v.env < 1e-3 && (snd.Kind != Clap || t > 0.05*sr) {
			v.active = false
			break
		}
	}
	v.waitSamps = 0
}

// NewPoly returns a drum synth: voices that play the sounds of a kit and choke each other
func NewPoly(cfg Config, numVoices int) *poly.Poly[*Voice] {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	voices := make([]*Voice, numVoices)
	var sh *shared
	for i := range numVoices {
		voices[i] = newVoice(cfg, sh)
		sh = voices[i].shared
	}
	return poly.New[*Voice](cfg.NumChannels, voices)
}
