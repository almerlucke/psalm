package waveguide

import (
	"math"

	"github.com/almerlucke/psalm"
)

// The open strings of a guitar ring along with what is played: six strings, each with its own pitch, that nothing
// plucks and the sound of the other strings drives. They are strings of the model, a loop of a delay, an allpass for
// the fraction of a sample and a gentle lowpass for the loss of the highs, fed through the bridge by the sum of the
// voices (before the body, which colours them as it does the rest). A string rings along with the partials of
// the note that are its own, or its octave and fifth, and dies as the plucked one does.

// openString is a string of the instrument's tuning: its MIDI pitch and the seconds it takes to fall 60 dB when
// plucked, those of the real guitar of the reference (the fall of the level at the pitch)
type openString struct {
	pitch, t60 float64
}

// openTuning is the standard tuning of a guitar: E2 A2 D3 G3 B3 E4
var openTuning = [...]openString{{40, 18}, {45, 5.3}, {50, 11}, {55, 5.8}, {59, 5.1}, {64, 4.8}}

// openLoss is the lowpass of the loop: how much of the highs a round trip takes (the same filter as the plucked string's)
const openLoss = 0.08

// openCoupling is how much of the sound drives a string at the full OpenStrings
const openCoupling = 0.02

type openLoop struct {
	buf      []float64
	pos      int
	n        int     // the whole samples of the delay
	apA      float64 // the allpass for the fraction
	apX, apY float64
	lpPrev   float64
	g        float64 // the gain of a round trip
}

type openStrings struct {
	loops []openLoop
	level float64
}

func newOpenStrings(amount float64) *openStrings {
	sr := psalm.CurrentConfig().SampleRate
	st := &openStrings{level: max(0, min(amount, 1))}
	for _, s := range openTuning {
		f := 440 * math.Exp2((s.pitch-69)/12)
		total := sr / f
		n := int(math.Floor(total - openLoss - 0.5))
		d := total - openLoss - float64(n) // the fraction the allpass has to give: 0.5-1.5 samples
		st.loops = append(st.loops, openLoop{
			buf: make([]float64, n+2),
			n:   n,
			apA: (1 - d) / (1 + d),
			g:   math.Pow(10, -3*total/(sr*s.t60)),
		})
	}
	return st
}

// process drives the strings with the sum of the channels and adds what they sound to each of them
func (s *openStrings) process(ch psalm.Channels) {
	if len(ch) == 0 {
		return
	}
	drive := s.level * openCoupling
	for i := range ch[0] {
		in := 0.0
		for _, buf := range ch {
			in += float64(buf[i])
		}
		in *= drive / float64(len(ch))
		sum := 0.0
		for k := range s.loops {
			l := &s.loops[k]
			read := l.buf[(l.pos-l.n+len(l.buf))%len(l.buf)]
			// the allpass for the fraction of a sample
			y := l.apA*read + l.apX - l.apA*l.apY
			l.apX, l.apY = read, y
			// the loss of the highs
			lp := (1-openLoss)*y + openLoss*l.lpPrev
			l.lpPrev = y
			out := l.g * lp
			l.buf[l.pos] = out + in
			l.pos = (l.pos + 1) % len(l.buf)
			sum += out
		}
		for _, buf := range ch {
			buf[i] += float32(sum)
		}
	}
}
