package lfo

import (
	"github.com/almerlucke/genny/float"
	"github.com/almerlucke/genny/float/interp"
	"github.com/almerlucke/genny/float/phasor"
	"github.com/almerlucke/genny/float/rand"
	"github.com/almerlucke/genny/float/shape"
	"github.com/almerlucke/genny/float/shape/shapers/linear"
	"github.com/almerlucke/genny/float/shape/shapers/lookup"
	"github.com/almerlucke/genny/float/shape/shapers/mirror"
	"github.com/almerlucke/genny/float/shape/shapers/series"
	"github.com/almerlucke/psalm"
)

type WaveForm int

const (
	Sine WaveForm = iota
	Triangle
	Saw
	Square
	Rand
	Custom
)

const (
	randFcMult = 4.0
)

var sineTable = lookup.NewUniSineTable(512)
var triangleTable = lookup.NewUniTriangleTable(512)
var sawTable = lookup.NewUniSawtoothTable(512)
var squareTable = lookup.NewUniSquareTable(512)

func tableForWaveForm(wf WaveForm, cwt lookup.Table) lookup.Table {
	switch wf {
	case Triangle:
		return triangleTable
	case Saw:
		return sawTable
	case Square:
		return squareTable
	case Custom:
		return cwt
	default:
		return sineTable
	}
}

type LFO struct {
	*psalm.Module
	phasor  *phasor.Phasor
	linear  *linear.Linear
	shaper  *series.Series
	ipol    *interp.Interpolator
	sh      *shape.Shape
	r       *rand.Rand
	wf      WaveForm
	wt      lookup.Table
	kr      bool
	last    int64 // the tick of the last control tick (control rate)
	started bool  // it had a control tick
}

func New(kr bool, fc float64, minVal float64, maxVal float64, wf WaveForm, wts ...lookup.Table) *LFO {
	var (
		wt lookup.Table
		sr float64
	)

	if kr {
		sr = psalm.ControlRate()
	} else {
		sr = psalm.SampleRate()
	}

	if len(wts) > 0 {
		wt = wts[0]
	}

	lin := linear.New(maxVal-minVal, minVal)
	r := rand.New(minVal, maxVal)
	ipol := interp.New(float.ToFrame(r), interp.Cubic, (fc*randFcMult)/sr)

	l := &LFO{
		Module: psalm.NewModule(nil, []int{psalm.Mono}),
		phasor: phasor.New(fc, sr, 0.0),
		linear: lin,
		shaper: series.New(tableForWaveForm(wf, wt), lin),
		r:      r,
		ipol:   ipol,
		sh:     shape.New(ipol, mirror.New(minVal, maxVal)),
		wf:     wf,
		wt:     wt,
		kr:     kr,
	}

	l.SetSelf(l)

	return l
}

func (lfo *LFO) SetCustomWaveTable(wt lookup.Table) {
	lfo.wt = wt
}

func (lfo *LFO) SetWaveForm(wf WaveForm) {
	lfo.shaper.Shapers[0] = tableForWaveForm(wf, lfo.wt)
	lfo.wf = wf
}

func (lfo *LFO) SetFrequency(fc float64) {
	var sr float64

	if lfo.kr {
		sr = lfo.Config().ControlRate()
	} else {
		sr = lfo.Config().SampleRate
	}

	lfo.phasor.SetFrequency(fc, sr)
	lfo.ipol.SetDelta((fc * randFcMult) / sr)
}

func (lfo *LFO) SetMinValue(minVal float64) {
	lfo.linear.Shift = minVal
	lfo.r.SetLow(minVal)
}

func (lfo *LFO) SetMaxValue(maxVal float64) {
	lfo.linear.Scale = maxVal - lfo.linear.Shift
	lfo.r.SetHigh(maxVal)
}

func (lfo *LFO) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		lfo.ReceiveMessage(msg)
		return
	}

	switch index {
	case 0:
		lfo.SetFrequency(psalm.FltVal[float64](value))
	case 1:
		lfo.SetMinValue(psalm.FltVal[float64](value))
	case 2:
		lfo.SetMaxValue(psalm.FltVal[float64](value))
	case 3:
		lfo.SetWaveForm(WaveForm(psalm.IntVal[int](value)))
	}
}

func (lfo *LFO) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("frequency", func(v float64) {
		lfo.SetFrequency(v)
	})
	msg.IfFltVal("minValue", func(v float64) {
		lfo.SetMinValue(v)
	})
	msg.IfFltVal("maxValue", func(v float64) {
		lfo.SetMaxValue(v)
	})
	msg.IfIntVal("waveForm", func(v int) {
		lfo.SetWaveForm(WaveForm(v))
	})
}

// Restart starts the LFO again at its starting phase (psalm.Restarter, from Psalm.Restart)
func (lfo *LFO) Restart() {
	lfo.phasor.Reset()
	lfo.last, lfo.started = 0, false
}

// advance moves the LFO on by ticks it didn't play (at control rate)
func (lfo *LFO) advance(n int64) {
	for range n {
		lfo.phasor.Generate()
		lfo.sh.Generate()
	}
}

func (lfo *LFO) CtrlTick(tick psalm.CtrlTick) {
	if !lfo.kr {
		return
	}
	// it follows the clock of its patch: ticks it missed go by, one added while the patch plays is
	// where it would be from the start, a clock going back (a restart) starts it again
	now := tick.TickTimestamp
	switch {
	case lfo.started && now == lfo.last:
		return
	case lfo.started && now < lfo.last:
		lfo.Restart()
		lfo.advance(now)
	case !lfo.started:
		lfo.advance(now)
	default:
		lfo.advance(now - lfo.last - 1)
	}
	lfo.last, lfo.started = now, true

	v := lfo.shaper.Shape(lfo.phasor.Generate())
	r := lfo.sh.Generate()[0]
	if lfo.wf == Rand {
		lfo.Send(r, 0)
	} else {
		lfo.Send(v, 0)
	}
}

func (lfo *LFO) Synthesize() {
	if lfo.kr {
		return
	}

	config := lfo.Config()
	out := lfo.Output(0).Channels[0]

	for i := 0; i < config.BufferSize; i++ {
		v := lfo.shaper.Shape(lfo.phasor.Generate())
		r := lfo.sh.Generate()[0]
		if lfo.wf == Rand {
			out[i] = float32(r)
		} else {
			out[i] = float32(v)
		}
	}
}
