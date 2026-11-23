package waveguide

import (
	_ "embed"
	"encoding/binary"
	"errors"
	"math"
	"sync"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/modules/effects/convolver"
)

// Measured is the body of a real instrument, an impulse response, for the sound of all the strings together:
// the strings play through it (in NewPoly, once for the sum of the voices) next to the body of five
// resonances the voices have (BodyMix), a part of the sound each. The two are lined up: the convolution is
// late by its blocks and by where the response begins, so the resonators are held back to match, and they
// add up instead of combing
type Measured struct {
	// IR is the response, one slice per channel at SampleRate (a mono one for both sides). It is not
	// changed. GuitarIR is that of the guitar this synth was fitted to
	IR         [][]float64
	SampleRate float64 // of the IR (0 is 44100)
	// Blend is how much of the sound is the measured body: 0 only the resonators, 1 only the measured body,
	// in between a crossfade with the resonators lined up with it
	Blend float64
	// Trim is how much louder (dB) the measured body sits next to the resonators when they are blended: a
	// response with much low end wants less
	Trim float64
}

// the measured body of the nylon guitar the strings were fitted to: taps of a finger (below 600 Hz, averaged)
// and of a hard object (above it, one of them), a 16 bit mono wav at 44100 Hz
//
//go:embed ir/guitar.wav
var guitarWav []byte

var (
	guitarOnce sync.Once
	guitarIR   [][]float64
)

// GuitarIR is the measured body of the nylon guitar the strings of OwnGuitar were fitted to, at 44100 Hz, as
// an impulse response of unit energy (see convolver.NormalizeIR). It is shared: do not change it
func GuitarIR() [][]float64 {
	guitarOnce.Do(func() {
		ir, _, err := decodeWav(guitarWav)
		if err != nil {
			panic("waveguide: the embedded guitar body: " + err.Error())
		}
		guitarIR = convolver.NormalizeIR(ir)
	})
	return guitarIR
}

// OwnGuitarBody is the measured body of GuitarIR in an equal blend with the resonators, as OwnGuitar has it
func OwnGuitarBody() *Measured {
	return &Measured{IR: GuitarIR(), SampleRate: 44100, Blend: 0.5, Trim: -1.5}
}

// decodeWav reads a PCM (8 to 32 bit) or float wav: the channels as floats between -1 and 1, and the sample rate
func decodeWav(b []byte) ([][]float64, float64, error) {
	bad := errors.New("not a wav file of PCM or float samples")
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, 0, bad
	}
	var format, channels, bits, rate int
	var data []byte
	for p := 12; p+8 <= len(b); {
		id, size := string(b[p:p+4]), int(binary.LittleEndian.Uint32(b[p+4:p+8]))
		body := b[p+8 : min(len(b), p+8+size)]
		switch id {
		case "fmt ":
			if len(body) < 16 {
				return nil, 0, bad
			}
			format = int(binary.LittleEndian.Uint16(body[0:]))
			channels = int(binary.LittleEndian.Uint16(body[2:]))
			rate = int(binary.LittleEndian.Uint32(body[4:]))
			bits = int(binary.LittleEndian.Uint16(body[14:]))
		case "data":
			data = body
		}
		p += 8 + size + size&1
	}
	width := bits / 8
	if channels < 1 || data == nil || width < 1 || width > 4 || bits%8 != 0 || (format != 1 && format != 3) || (format == 3 && bits != 32) {
		return nil, 0, bad
	}
	frames := len(data) / (width * channels)
	out := make([][]float64, channels)
	for c := range out {
		out[c] = make([]float64, frames)
	}
	for i := range frames {
		for c := range channels {
			s := data[(i*channels+c)*width:]
			var x float64
			switch {
			case format == 3:
				x = float64(math.Float32frombits(binary.LittleEndian.Uint32(s)))
			case width == 1:
				x = (float64(s[0]) - 128) / 128
			case width == 2:
				x = float64(int16(binary.LittleEndian.Uint16(s))) / 32768
			case width == 3:
				x = float64(int32(uint32(s[0])<<8|uint32(s[1])<<16|uint32(s[2])<<24)>>8) / 8388608
			default:
				x = float64(int32(binary.LittleEndian.Uint32(s))) / 2147483648
			}
			out[c][i] = x
		}
	}
	return out, float64(rate), nil
}

// resample is the response at another sample rate, by a windowed sinc
func resample(ir [][]float64, from, to float64) [][]float64 {
	if from == to || from <= 0 || to <= 0 {
		return ir
	}
	ratio := from / to
	cut := min(1, to/from) // below the lower of the two rates' Nyquist
	const half = 16
	out := make([][]float64, len(ir))
	for c, x := range ir {
		y := make([]float64, int(float64(len(x))/ratio))
		for i := range y {
			t := float64(i) * ratio
			k0 := int(math.Floor(t))
			acc := 0.0
			for k := k0 - half + 1; k <= k0+half; k++ {
				if k < 0 || k >= len(x) {
					continue
				}
				d := t - float64(k)
				w := 0.5 + 0.5*math.Cos(math.Pi*d/half)
				if d == 0 {
					acc += x[k] * w * cut
				} else {
					acc += x[k] * w * math.Sin(math.Pi*d*cut) / (math.Pi * d)
				}
			}
			y[i] = acc
		}
		out[c] = y
	}
	return out
}

// the gain of the convolver: the response has the energy of one, but a plucked string drives its few bass modes hard
const bodyConvolverGain = 0.17

// blendTrim is how much louder (dB) the resonators and the measured body together are made: the sounds do not add up
// the same at all frequencies, and together they come out about 3 dB quieter than either alone
const blendTrim = 2.8

// bodyBlock is the block of the convolver: its latency in samples
const bodyBlock = 128

// bodyStage is the body of an instrument that all its strings share: the resonators, the measured body, or both
type bodyStage struct {
	banks [][]*svf.SVF // the resonators, per channel
	modes []resonance
	mix   float64 // how much of the sound the resonators are

	conv  *convolver.Convolver
	late  [][]float32 // the resonators' delay, per channel
	pos   int
	dry   float32 // the share of the resonators when blended
	wet   float32
	solo  bool // only the measured body
	plain bool // only the resonators
}

// newBodyStage makes the stage of the resonators of body (mix of the sound) and the measured body (m, or nil)
func newBodyStage(body Body, mix float64, m *Measured, channels int) *bodyStage {
	sr := psalm.CurrentConfig().SampleRate
	st := &bodyStage{modes: bodies[body], mix: max(0, min(mix, 1))}
	if len(st.modes) == 0 {
		st.mix = 0
	}
	for range channels {
		var bank []*svf.SVF
		for _, md := range st.modes {
			f := svf.New(1000, 0, sr)
			f.SetFrequencyQ(min(md.freq, sr*0.45), md.q)
			bank = append(bank, f)
		}
		st.banks = append(st.banks, bank)
	}
	if m == nil || len(m.IR) == 0 || m.Blend <= 0 {
		st.plain = true
		return st
	}
	ir := m.IR
	if m.SampleRate > 0 && m.SampleRate != sr {
		// the convolver does not resample
		ir = resample(ir, m.SampleRate, sr)
	} else if m.SampleRate == 0 && sr != 44100 {
		ir = resample(ir, 44100, sr)
	}
	st.conv = convolver.New(ir, convolver.Config{PartitionSize: bodyBlock, Mix: 1, Gain: bodyConvolverGain})
	blend := min(m.Blend, 1)
	if blend >= 1 {
		st.solo = true
		return st
	}
	peak := 0
	for i, x := range ir[0] {
		if math.Abs(x) > math.Abs(ir[0][peak]) {
			peak = i
		}
	}
	st.late = make([][]float32, channels)
	for c := range st.late {
		st.late[c] = make([]float32, st.conv.SignalLatency()+peak)
	}
	db := func(x float64) float32 { return float32(math.Pow(10, x/20)) }
	st.dry = db(blendTrim) * float32(1-blend)
	st.wet = db(blendTrim+m.Trim) * float32(blend)
	return st
}

// resonate puts a sample through the resonators of a channel
func (s *bodyStage) resonate(c int, x float64) float64 {
	if s.mix <= 0 {
		return x
	}
	b := 0.0
	for j, md := range s.modes {
		_, band, _ := s.banks[c][j].Tick(x)
		b += band * s.banks[c][j].K() * md.gain
	}
	return x*(1-s.mix) + b*s.mix*1.5
}

// process puts the buffers through the stage, in place
func (s *bodyStage) process(ch psalm.Channels) {
	if s.conv != nil {
		in, out := s.conv.Input(0).Channels, s.conv.Output(0).Channels
		in.Copy(ch)
		s.conv.Synthesize()
		if s.solo {
			ch.Copy(out)
			return
		}
		for c, buf := range ch {
			if c >= len(s.banks) {
				break
			}
			d := s.late[c]
			for i, x := range buf {
				y := float32(s.resonate(c, float64(x)))
				if n := len(d); n > 0 {
					j := (s.pos + i) % n
					y, d[j] = d[j], y
				}
				buf[i] = s.dry*y + s.wet*out[c][i]
			}
		}
		if n := len(s.late[0]); n > 0 {
			s.pos = (s.pos + len(ch[0])) % n
		}
		return
	}
	for c, buf := range ch {
		if c >= len(s.banks) {
			break
		}
		for i, x := range buf {
			buf[i] = float32(s.resonate(c, float64(x)))
		}
	}
}

// BodyEffect is the body as a module of its own, for a signal from anywhere: the resonators of body (mix of the
// sound) and the measured body, as the strings of NewPoly have them
type BodyEffect struct {
	*psalm.Module
	stage *bodyStage
	tail  float64
}

// NewBodyEffect makes the body for a stereo signal; m is nil for the resonators alone
func NewBodyEffect(body Body, mix float64, m *Measured) *BodyEffect {
	e := &BodyEffect{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		stage:  newBodyStage(body, mix, m, psalm.Stereo),
	}
	if e.stage.conv != nil {
		e.tail = e.stage.conv.Tail()
	}
	e.SetSelf(e)
	return e
}

func (e *BodyEffect) Synthesize() {
	out := e.Output(0).Channels
	out.Copy(e.Input(0).Channels)
	e.stage.process(out)
}

// Tail is how long it sounds after its input stopped
func (e *BodyEffect) Tail() float64 { return e.tail + 0.5 }
