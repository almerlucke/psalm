package shimmer

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/pitchshift"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
)

// Config for the shimmer reverb
type Config struct {
	Reverb    fdnreverb.Config // the reverb tank (its Mix is not used)
	Semitones float64          // pitch shift in the feedback loop, 12 is the classic octave up
	Shimmer   float64          // 0-1, amount of shifted signal fed back into the reverb
	Window    float64          // pitch shifter grain size in milliseconds
	LowCut    float64          // Hz, high pass in the feedback loop to keep the tail from getting muddy
	Mix       float64          // 0 is dry only, 1 is wet only
}

func DefaultConfig() Config {
	rev := fdnreverb.DefaultConfig()
	rev.Size = 0.85
	rev.Decay = 6.0
	rev.Damping = 0.4
	rev.Diffusion = 0.8
	rev.ModDepth = 0.6

	return Config{
		Reverb:    rev,
		Semitones: 12.0,
		Shimmer:   0.5,
		Window:    60.0,
		LowCut:    300.0,
		Mix:       0.4,
	}
}

// highPass is a one pole high pass filter
type highPass struct {
	a, x1, y1 float64
}

func (h *highPass) set(fc float64, sr float64) {
	rc := 1.0 / (2.0 * math.Pi * math.Max(1.0, fc))
	h.a = rc / (rc + 1.0/sr)
}

func (h *highPass) process(x float64) float64 {
	y := h.a * (h.y1 + x - h.x1)
	h.x1, h.y1 = x, y
	return y
}

// Shimmer is a reverb with a pitch shifter in its feedback loop: the reverb output is shifted
// (an octave up by default) and fed back into the reverb, so every pass through the loop climbs
// higher and the tail blooms into an ethereal, rising wash. A low cut keeps the loop clean, and the
// feedback backs off when the tail gets loud (and is soft limited), so long decays with a lot of
// shimmer settle at a sane level instead of building up. Stereo input, stereo output
type Shimmer struct {
	*psalm.Module
	cfg      Config
	reverb   *fdnreverb.FDNReverb
	shifters [2]*pitchshift.PitchShift
	lowCuts  [2]highPass
	fb       [2]float64
	env      float64 // level of the wet signal, used to back off the feedback when it gets loud
	envAtt   float64
	envRel   float64
}

// feedbackLevel is the wet level above which the feedback is backed off
const feedbackLevel = 0.5

func New(cfg Config) *Shimmer {
	s := &Shimmer{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		reverb: fdnreverb.New(cfg.Reverb),
	}

	sr := s.Config().SampleRate
	window := cfg.Window
	if window <= 0.0 {
		window = DefaultConfig().Window
	}

	for c := range s.shifters {
		s.shifters[c] = pitchshift.New(cfg.Semitones, window, sr)
		s.lowCuts[c].set(cfg.LowCut, sr)
	}

	s.envAtt = 1.0 - math.Exp(-1.0/(0.01*sr))
	s.envRel = 1.0 - math.Exp(-1.0/(0.3*sr))

	s.SetSelf(s)

	return s
}

func (s *Shimmer) SetSemitones(semitones float64) {
	s.cfg.Semitones = semitones
	for _, sh := range s.shifters {
		sh.SetSemitones(semitones)
	}
}

func (s *Shimmer) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		s.ReceiveMessage(msg)
		return
	}

	if index == 1 {
		s.cfg.Shimmer = psalm.FltVal[float64](value)
	}
}

// ReceiveMessage handles "shimmer", "semitones", "lowCut" and "mix", other keys (like "decay",
// "damping" or "size") are passed on to the reverb tank
func (s *Shimmer) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("shimmer", func(f float64) { s.cfg.Shimmer = f })
	msg.IfFltVal("semitones", s.SetSemitones)
	msg.IfFltVal("mix", func(f float64) { s.cfg.Mix = f })
	msg.IfFltVal("lowCut", func(f float64) {
		s.cfg.LowCut = f
		for c := range s.lowCuts {
			s.lowCuts[c].set(f, s.Config().SampleRate)
		}
	})

	tank := psalm.Message{}
	for k, v := range msg {
		if k != "mix" {
			tank[k] = v
		}
	}
	s.reverb.ReceiveMessage(tank)
}

func (s *Shimmer) Synthesize() {
	ins := s.Input(0).Channels
	outs := s.Output(0).Channels
	wet, dry := s.cfg.Mix, 1.0-s.cfg.Mix
	amount := math.Max(0.0, math.Min(s.cfg.Shimmer, 1.0))

	for i := range s.Config().BufferSize {
		inL, inR := float64(ins[0][i]), float64(ins[1][i])

		// the shifted feedback of the previous sample joins the input of the reverb
		left, right := s.reverb.ProcessSample(inL+s.fb[0], inR+s.fb[1])

		// follow the wet level and back off the feedback above feedbackLevel
		level := math.Max(math.Abs(left), math.Abs(right))
		coef := s.envRel
		if level > s.env {
			coef = s.envAtt
		}
		s.env += (level - s.env) * coef
		gain := amount
		if s.env > feedbackLevel {
			gain *= feedbackLevel / s.env
		}

		// shift, cut the lows and softly limit what goes back into the loop
		for c, x := range [2]float64{left, right} {
			shifted := s.lowCuts[c].process(s.shifters[c].Process(x))
			s.fb[c] = math.Tanh(shifted * gain)
		}

		outs[0][i] = float32(dry*inL + wet*left)
		outs[1][i] = float32(dry*inR + wet*right)
	}
}

// Tail lets the shimmer rest while silent (see the reverb)
func (s *Shimmer) Tail() float64 {
	return s.cfg.Reverb.PreDelay/1000 + s.cfg.Window/1000 + 0.25
}
