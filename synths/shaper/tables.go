package shaper

import (
	"math"
	"sync"

	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/psalm/blocks/shapechain"
)

// The tables. A chain that has harmonics above what the pitch lets through would fold back, and the
// automatic oversampling reads it two or four times a sample, for every copy of the unison. A table reads it
// once, for all of them: one cycle of the wave of a line (the chains at the morph and the macros it has now) is worked
// out, cut to the harmonics that fit under half the sample rate at the pitch of the voice, and played by
// reading it with a cubic interpolation, which is a few multiplications. The wave moves with the morph and the macros:
// when they have moved a little (tableTol) a new table is worked out, and the sound fades from the old one to it over the
// time they took to move that far, so a slow sweep is a series of small steps the ear does not hear. The tables are used
// when the oversampling is automatic and the chains need more than one time a sample, or there is a unison (one table
// serves all the copies); a chain that fits, played by a single copy, is read directly, which is as cheap.

const (
	// tableTol is how far the morph or a macro may move from what a table was made at before a new one is made
	tableTol = 0.01
	// tablePerHarmonic is the points of a table for each harmonic it holds: the cubic reading is clean up to a quarter of
	// the points, so with eight for a harmonic what the reading adds lies far above it
	tablePerHarmonic   = 8
	tableMin, tableMax = 256, 2048
	// the sound fades to a new table over as long as the last one took to make (the time the wave took to move
	// tableTol, at least half a table), but never shorter or longer than this many samples
	blendMin, blendMax = 24, 2048
)

// waveTable is one cycle of a wave at n points (a power of two). d holds them with the one before and the two after
// (wrapped), so reading needs no checks
type waveTable struct {
	n      int
	d      []float64 // n+3
	harm   int       // the harmonics it holds
	pos    float64   // the morph it was made at
	macros shapechain.Macros
}

// read is the table at a phase (0-1) by cubic (Catmull Rom) interpolation
func (t *waveTable) read(ph float64) float64 {
	x := ph * float64(t.n)
	i := int(x)
	f := x - float64(i)
	i &= t.n - 1 // a phase of exactly 1
	d := t.d[i : i+4 : i+4]
	a, b, c, e := d[0], d[1], d[2], d[3]
	return b + 0.5*f*(c-a+f*(2*a-5*b+4*c-e+f*(3*(b-c)+e-a)))
}

// lineTables are the tables of a line: the one the sound goes to, and the one it comes from while it fades
type lineTables struct {
	buf             [2]waveTable
	cur, prev       int
	valid           bool
	blend, blendInc float64 // blend 1: the sound is the current table
	builtAt         int     // the sample of the voice the current one was made at
	gen             int     // the swap of chains (Voice.swapGen) it has seen
}

// ffts are the real FFTs for the cutting of the tables, by size (they have scratch, so one is taken for the
// time of a build)
var ffts [13]sync.Pool

func realFFT(n int) (*fft.Real, int) {
	k := 0
	for 1<<k < n {
		k++
	}
	if f, ok := ffts[k].Get().(*fft.Real); ok {
		return f, k
	}
	return fft.NewReal(n), k
}

// tableScratch is the room a build works in
type tableScratch struct {
	x, re, im, mix []float64
}

func (s *tableScratch) size(n int) {
	if len(s.x) < n {
		s.x = make([]float64, n)
		s.mix = make([]float64, n)
		s.re = make([]float64, n/2+1)
		s.im = make([]float64, n/2+1)
	}
}

func nextPow2(n int) int {
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

// tableDue says whether the table of a line no longer fits what the voice is doing: the pitch rose until its harmonics
// would fold back, or fell far enough to make it dull, or the morph or a macro moved
func (v *Voice) tableDue(lt *lineTables, pos float64, macros *shapechain.Macros, fTop float64) bool {
	t := &lt.buf[lt.cur]
	fit := int(v.Config().SampleRate / 2 / fTop) // the harmonics that fit now
	if fit < t.harm {
		return true // the pitch rose until the table would fold back: at once
	}
	// a table is not made more often than it costs less than reading the chain: a build is a chain evaluated n times, and
	// reading it directly four times a sample is eight evaluations, so every n/2 samples at most (a fast sweep of a macro by an
	// envelope is a table every n/2 samples, and the fade between them is smooth)
	if v.t-lt.builtAt < max(blendMin, t.n/2) {
		return false
	}
	if min(fit, v.bwEff, tableMax/tablePerHarmonic) > 2*t.harm {
		return true
	}
	if v.numChains > 1 && math.Abs(pos-t.pos) > tableTol {
		return true
	}
	for i := range macros {
		if v.macroUsed[i] && math.Abs(macros[i]-t.macros[i]) > tableTol {
			return true
		}
	}
	return false
}

// retire gets a line ready for a new current table. The one it was going to becomes the one it comes from; if the sound was
// still on its way from one table to the other, the mix it has reached is made a table of its own first, so a new table
// can follow at any time without a step
func (v *Voice) retire(lt *lineTables) {
	if lt.blend >= 1 {
		lt.prev, lt.cur = lt.cur, 1-lt.cur
		return
	}
	p, c := &lt.buf[lt.prev], &lt.buf[lt.cur]
	n := c.n
	sc := &v.scratch
	sc.size(n)
	w := lt.blend
	for i := range n {
		ph := float64(i) / float64(n)
		sc.mix[i] = p.read(ph)*(1-w) + c.read(ph)*w
	}
	p.setPoints(sc.mix[:n])
	p.harm = c.harm
}

// setPoints makes x (n points, a power of two) the table
func (t *waveTable) setPoints(x []float64) {
	n := len(x)
	if cap(t.d) < n+3 {
		t.d = make([]float64, n+3)
	}
	t.d, t.n = t.d[:n+3], n
	copy(t.d[1:], x)
	t.d[0], t.d[n+1], t.d[n+2] = x[n-1], x[0], x[1]
}

// fill makes the table from the chains given (the ones the voice has, or the ones that are fading out) at the morph and
// macros given, cut to the harmonics that fit under half the sample rate at fTop (the highest frequency a copy plays,
// with room to rise)
func (v *Voice) fill(t *waveTable, chains *[NumChains]*shapechain.Chain, num, bw int, pos float64, macros *shapechain.Macros, fTop float64) {
	sr := v.Config().SampleRate
	nyq := int(sr / 2 / (fTop * 1.12))
	harm := max(4, min(nyq, bw))
	n := min(tableMax, max(tableMin, nextPow2(tablePerHarmonic*harm)))
	harm = min(harm, n/tablePerHarmonic)
	cut := bw > harm // something lies above: it is cut away
	sc := &v.scratch
	sc.size(n)
	x := sc.x[:n]
	mean := 0.0
	for i := range x {
		y := waveOf(chains, num, float64(i)/float64(n), pos, macros)
		if y != y {
			y = 0
		}
		y = max(-2, min(y, 2))
		x[i] = y
		mean += y
	}
	mean /= float64(n)
	for i := range x {
		x[i] -= mean
	}
	if cut {
		f, k := realFFT(n)
		re, im := sc.re[:n/2+1], sc.im[:n/2+1]
		f.Forward(x, re, im)
		// the harmonics above `harm` go, the top tenth of what stays fades out (a cut with no slope rings)
		from := float64(harm) * 0.9
		for b := range re {
			switch {
			case b > harm:
				re[b], im[b] = 0, 0
			case float64(b) > from:
				g := 0.5 + 0.5*math.Cos(math.Pi*(float64(b)-from)/(float64(harm)-from))
				re[b] *= g
				im[b] *= g
			}
		}
		f.Inverse(re, im, x)
		ffts[k].Put(f)
	}
	t.setPoints(x)
	t.harm, t.pos, t.macros = harm, pos, *macros
}

// oldBandwidth is the harmonics of the chains that are fading out
func (v *Voice) oldBandwidth() int {
	bw := 1
	for i := 0; i < v.oldNum; i++ {
		bw = max(bw, v.old[i].Bandwidth())
	}
	return bw
}

// buildTable makes the first table of a line (a note starts). A line that is picked up while a swapped chain is fading
// out (the voice read directly before) gets the table of the chains that go, and the sound goes on from where the fade is
func (v *Voice) buildTable(lt *lineTables, pos float64, macros *shapechain.Macros, fTop float64) {
	lt.cur, lt.prev = 0, 0
	lt.blend, lt.blendInc = 1, 1
	if v.xf > 0 && v.oldNum > 0 && v.xfLeft > 0 {
		lt.prev = 1
		v.fill(&lt.buf[1], &v.old, v.oldNum, v.oldBandwidth(), pos, macros, fTop)
		lt.blend = 1 - v.xf
		lt.blendInc = v.xf / float64(v.xfLeft)
	}
	v.fill(&lt.buf[0], &v.chains, v.numChains, v.bwEff, pos, macros, fTop)
	lt.valid, lt.builtAt, lt.gen = true, v.t, v.swapGen
}

// rebuildTable makes a new current table for the morph and macros the line has now, and the sound fades to it over as
// long as the last one took to make (the time the wave took to move tableTol); after a swap of chains the new table is that of
// the chains, and the fade the one of the swap
func (v *Voice) rebuildTable(lt *lineTables, pos float64, macros *shapechain.Macros, fTop float64, swap bool) {
	v.retire(lt)
	v.fill(&lt.buf[lt.cur], &v.chains, v.numChains, v.bwEff, pos, macros, fTop)
	span := min(blendMax, max(blendMin, v.t-lt.builtAt))
	if swap {
		span = max(blendMin, v.xfLeft)
		lt.gen = v.swapGen
	}
	lt.blend, lt.blendInc = 0, 1/float64(span)
	lt.builtAt = v.t
}
