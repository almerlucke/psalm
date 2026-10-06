package mixer

import (
	"math"
	"slices"
	"strings"

	"github.com/almerlucke/psalm"
)

// StripConfig holds the settings of a channel strip
type StripConfig struct {
	Volume   float64            // dB, 0 is unity
	Pan      float64            // -1 left to 1 right, a balance for stereo sources
	Mute     bool               //
	Solo     bool               // when any channel is soloed only soloed channels sound (groups and returns are solo safe)
	Sends    map[string]float64 // amount (0-1) sent to each effect bus
	PreFader bool               // sends taken before the volume and pan
	Group    string             // group the strip plays into, "" is the master
}

// DB returns the gain of a level in dB
func DB(db float64) float64 {
	if db <= MinDB {
		return 0
	}
	return math.Pow(10, db/20)
}

// ToDB returns the level in dB of a gain
func ToDB(gain float64) float64 {
	if gain <= DB(MinDB) {
		return MinDB
	}
	return 20 * math.Log10(gain)
}

// MinDB is silence
const MinDB = -120.0

const (
	smoothMs = 20.0 // volume, pan and mute changes
	muteMs   = 10.0
)

// Strip is a channel strip module: stereo input 0 (mono sources play on both sides), output 0 the
// signal after volume and pan, then one output per send. Value inputs: 1 volume (dB), 2 pan,
// 3 mute (above 0.5). Messages: "volume", "pan", "mute", "solo", "send.<bus>"
type Strip struct {
	*psalm.Module
	name   string
	mixer  *Mixer
	cfg    StripConfig
	sends  []string // bus names in output order
	safe   bool     // groups and returns ignore solo
	volDB  float64  // current volume, ramps towards cfg.Volume
	volLin float64
	ramp   float64 // dB per sample of the current fade, 0 is the short smoothing
	pan    float64
	panL   float64 // pan gains of the current pan
	panR   float64
	open   float64 // mute gain, 0-1
	sendG  []float64
	meter  levelMeter
}

func newStrip(m *Mixer, name string, cfg StripConfig, safe bool) *Strip {
	sends := make([]string, 0, len(cfg.Sends))
	for bus := range cfg.Sends {
		sends = append(sends, bus)
	}
	slices.Sort(sends)
	s := &Strip{
		Module: psalm.NewModule([]int{psalm.Stereo}, psalm.NumSockets(1+len(sends), psalm.Stereo)),
		name:   name,
		mixer:  m,
		cfg:    cfg,
		sends:  sends,
		safe:   safe,
		volDB:  cfg.Volume,
		volLin: DB(cfg.Volume),
		pan:    cfg.Pan,
		sendG:  make([]float64, len(sends)),
	}
	if !s.silenced() {
		s.open = 1
	}
	s.panGains()
	for i, bus := range sends {
		s.sendG[i] = cfg.Sends[bus]
	}
	s.SetSelf(s)
	return s
}

// Name returns the name of the strip
func (s *Strip) Name() string { return s.name }

// Settings returns the current settings
func (s *Strip) Settings() StripConfig { return s.cfg }

// SetVolume sets the volume in dB, it glides there in a few milliseconds
func (s *Strip) SetVolume(db float64) {
	s.cfg.Volume = db
	s.ramp = 0
}

// FadeTo fades the volume to a level in dB over seconds, in even steps of dB
func (s *Strip) FadeTo(db float64, seconds float64) {
	s.cfg.Volume = db
	from := math.Max(s.volDB, MinDB)
	samples := math.Max(1, seconds*s.Config().SampleRate)
	s.ramp = math.Abs(math.Max(db, MinDB)-from) / samples
}

// SetPan sets the pan (-1 left to 1 right)
func (s *Strip) SetPan(pan float64) { s.cfg.Pan = math.Max(-1, math.Min(pan, 1)) }

// SetMute mutes or unmutes the strip (with a short fade)
func (s *Strip) SetMute(mute bool) { s.cfg.Mute = mute }

// SetSolo solos the strip
func (s *Strip) SetSolo(solo bool) { s.cfg.Solo = solo }

// SetSend sets the amount sent to a bus (one of the sends the strip was created with)
func (s *Strip) SetSend(bus string, amount float64) {
	if i := slices.Index(s.sends, bus); i >= 0 {
		s.cfg.Sends[bus] = amount
		s.sendG[i] = amount
	}
}

// Levels returns the peak and RMS level in dB since the meters were reset
func (s *Strip) Levels() (peak float64, rms float64) {
	return s.meter.levels()
}

func (s *Strip) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		s.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		s.SetVolume(v)
	case 2:
		s.SetPan(v)
	case 3:
		s.SetMute(v >= 0.5)
	}
}

func (s *Strip) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("volume", s.SetVolume)
	msg.IfFltVal("pan", s.SetPan)
	msg.IfFltVal("mute", func(v float64) { s.SetMute(v >= 0.5) })
	msg.IfFltVal("solo", func(v float64) { s.SetSolo(v >= 0.5) })
	for k := range msg {
		if bus, ok := strings.CutPrefix(k, "send."); ok {
			msg.IfFltVal(k, func(v float64) { s.SetSend(bus, v) })
		}
	}
}

// silenced returns true when the strip is muted, or another channel is soloed
func (s *Strip) silenced() bool {
	if s.cfg.Mute {
		return true
	}
	return !s.safe && !s.cfg.Solo && s.mixer.anySolo()
}

func (s *Strip) Synthesize() {
	var (
		sr       = s.Config().SampleRate
		in       = s.Input(0).Channels
		main     = s.Output(0).Channels
		smooth   = 1 - math.Exp(-1/(smoothMs*0.001*sr))
		muteCoef = 1 - math.Exp(-1/(muteMs*0.001*sr))
		target   = math.Max(s.cfg.Volume, MinDB)
		openTo   = 1.0
	)
	if s.silenced() {
		openTo = 0
	}

	for i := range main[0] {
		// volume: a fade moves in even dB steps, other changes glide quickly
		if s.volDB != target {
			if s.ramp > 0 {
				if s.volDB < target {
					s.volDB = math.Min(target, s.volDB+s.ramp)
				} else {
					s.volDB = math.Max(target, s.volDB-s.ramp)
				}
			} else {
				s.volDB += (target - s.volDB) * smooth
				if math.Abs(target-s.volDB) < 1e-4 {
					s.volDB = target
				}
			}
			s.volLin = DB(s.volDB)
		}
		if s.pan != s.cfg.Pan {
			s.pan += (s.cfg.Pan - s.pan) * smooth
			if math.Abs(s.cfg.Pan-s.pan) < 1e-5 {
				s.pan = s.cfg.Pan
			}
			s.panGains()
		}
		s.open += (openTo - s.open) * muteCoef

		l, r := float64(in[0][i]), float64(in[1][i])

		// pre fader sends
		if s.cfg.PreFader {
			for k, g := range s.sendG {
				out := s.Output(k + 1).Channels
				out[0][i] = float32(l * g * s.open)
				out[1][i] = float32(r * g * s.open)
			}
		}

		g := s.volLin * s.open
		l *= g * s.panL
		r *= g * s.panR

		main[0][i] = float32(l)
		main[1][i] = float32(r)
		s.meter.add(l, r)

		if !s.cfg.PreFader {
			for k, sg := range s.sendG {
				out := s.Output(k + 1).Channels
				out[0][i] = float32(l * sg)
				out[1][i] = float32(r * sg)
			}
		}
	}
	if s.ramp > 0 && s.volDB == target {
		s.ramp = 0
	}
}

// panGains sets the gains of an equal power pan (a balance for stereo), unity in the centre
func (s *Strip) panGains() {
	theta := (s.pan + 1) * math.Pi / 4
	s.panL = math.Cos(theta) * math.Sqrt2
	s.panR = math.Sin(theta) * math.Sqrt2
}

// levelMeter keeps the peak and the energy of a signal
type levelMeter struct {
	peak  float64
	sum   float64
	count int
}

func (m *levelMeter) add(l, r float64) {
	m.peak = math.Max(m.peak, math.Max(math.Abs(l), math.Abs(r)))
	m.sum += (l*l + r*r) / 2
	m.count++
}

func (m *levelMeter) levels() (float64, float64) {
	if m.count == 0 {
		return MinDB, MinDB
	}
	return ToDB(m.peak), ToDB(math.Sqrt(m.sum / float64(m.count)))
}

func (m *levelMeter) reset() {
	*m = levelMeter{}
}
