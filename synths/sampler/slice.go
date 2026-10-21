package sampler

import (
	"math"

	"github.com/almerlucke/psalm/blocks/fft"
)

// Slicing: a recording cut into pieces, each piece on a key of its own, so a drum loop or a phrase becomes a kit played from a keyboard.
// The cuts are the frames where one piece ends and the next begins; EvenCuts and OnsetCuts find them, Slices makes the regions.
//
//	snd, _ := sampler.LoadFile("loop.wav")
//	inst := sampler.MustInstrument(sampler.Slices(snd, 36, sampler.OnsetCuts(snd, 9, 0.08))...)

// EvenCuts returns the cuts that divide a sound into n pieces of the same length (n+1 frames, the first 0 and the last the end), moved to the
// nearest zero crossing so a piece does not begin or end with a click
func EvenCuts(snd Sound, n int) []int64 {
	total := snd.NumFrames()
	n = max(1, n)
	cuts := make([]int64, n+1)
	for i := range cuts {
		cuts[i] = total * int64(i) / int64(n)
	}
	return snapCuts(snd, cuts)
}

// OnsetCuts returns the cuts at the starts of the sounds in a recording (the hits of a loop, the syllables of a phrase): the places the
// level jumps. rise is how many dB the level of a short stretch (about six milliseconds) must exceed the average of the 50 milliseconds before
// it (6 to 12 is usual: the lower, the more it finds), minGap the least seconds between two. The first cut is 0 and the last the end of the
// sound, so n+1 cuts are n pieces; a cut is moved to the nearest zero crossing.
func OnsetCuts(snd Sound, rise, minGap float64) []int64 {
	total := snd.NumFrames()
	sr := snd.SampleRate()
	hop := max(16, int(sr*0.006))
	nh := int(total) / hop
	// the level of every hop, the channels together
	level := make([]float64, nh)
	for h := range level {
		sum, count := 0.0, 0
		for c := range snd.NumChannels() {
			buf := snd.Buffer(c, 0)
			for _, v := range buf[h*hop : (h+1)*hop] {
				sum += float64(v) * float64(v)
				count++
			}
		}
		level[h] = sum / float64(count)
	}
	// the quietest level counts for something: a pause is not the floor of the hits to come
	peak := 0.0
	for _, l := range level {
		peak = max(peak, l)
	}
	floor := peak * 1e-6
	db := func(x float64) float64 { return 10 * math.Log10(x+floor) }
	back := max(2, int(0.05*sr)/hop)
	gap := max(1, int(minGap*sr)/hop)
	cuts := []int64{0}
	last := -gap
	for h := 1; h < nh; h++ {
		// the average of the stretch before, in power
		from := max(0, h-back)
		avg := 0.0
		for _, l := range level[from:h] {
			avg += l
		}
		avg /= float64(h - from)
		if db(level[h])-db(avg) < rise || h-last < gap {
			continue
		}
		// the rise goes on for a hop or two: the onset is where it began
		at := h
		for at > from && level[at-1] < level[at] {
			at--
		}
		if at-last < gap {
			continue
		}
		last = h
		if frame := int64(at * hop); frame > cuts[len(cuts)-1] {
			cuts = append(cuts, frame)
		}
	}
	cuts = append(cuts, total)
	return snapCuts(snd, cuts)
}

// ChangeCuts returns the cuts where a recording changes, for what has no hits: a sung line, a phrase of speech or a pad whose notes flow into
// each other at the same level (OnsetCuts finds nothing there). It follows the spectrum, not the level: where the sound of one syllable,
// vowel or note passes into another, the spectrum moves more than it does anywhere nearby. sensitivity is how many times more than
// the average of the second around it the movement must be (1.1 for a sung line, where it changes gently, 1.5 and more for
// speech or notes that jump: the lower, the more it finds), minGap the least seconds
// between two cuts. Each cut is put at the quietest place near the change, and on a zero crossing.
func ChangeCuts(snd Sound, sensitivity, minGap float64) []int64 {
	total := int(snd.NumFrames())
	sr := snd.SampleRate()
	mono := make([]float64, total)
	for c := range snd.NumChannels() {
		for i, v := range snd.Buffer(c, 0)[:total] {
			mono[i] += float64(v)
		}
	}
	const size, hop = 2048, 512
	nf := (total-size)/hop + 1
	if nf < 3 {
		return []int64{0, int64(total)}
	}
	win := make([]float64, size)
	for i := range win {
		win[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/size)
	}
	f := fft.NewReal(size)
	x, re, im := make([]float64, size), make([]float64, size/2+1), make([]float64, size/2+1)
	// the log spectrum of every frame between 100 Hz and 6 kHz, and the level of the frame
	lo, hi := int(100*size/sr), min(size/2, int(6000*size/sr))
	spec := make([][]float64, nf)
	level := make([]float64, nf)
	peak := 0.0
	for t := range spec {
		for i := range x {
			x[i] = mono[t*hop+i] * win[i]
			level[t] += x[i] * x[i]
		}
		f.Forward(x, re, im)
		spec[t] = make([]float64, hi-lo)
		for b := lo; b < hi; b++ {
			m := math.Hypot(re[b], im[b])
			spec[t][b-lo] = m
			peak = max(peak, m)
		}
	}
	maxLevel := 0.0
	for _, l := range level {
		maxLevel = max(maxLevel, l)
	}
	for t := range spec {
		for b, m := range spec[t] {
			spec[t][b] = math.Log(m + peak*1e-3)
		}
	}
	// the movement of the spectrum from a frame to the next, a little smoothed
	flux := make([]float64, nf)
	for t := 1; t < nf; t++ {
		for b := range spec[t] {
			flux[t] += math.Abs(spec[t][b] - spec[t-1][b])
		}
		flux[t] /= float64(hi - lo)
	}
	smooth := make([]float64, nf)
	for t := range smooth {
		sum, n := 0.0, 0
		for k := max(0, t-1); k <= min(nf-1, t+1); k++ {
			sum += flux[k]
			n++
		}
		smooth[t] = sum / float64(n)
	}
	around := max(2, int(sr/hop)/2) // half a second each way
	gap := max(1, int(minGap*sr)/hop)
	cuts := []int64{0}
	last := -gap
	for t := 2; t < nf-1; t++ {
		if level[t] < maxLevel*1e-5 { // silence
			continue
		}
		sum, n := 0.0, 0
		for k := max(0, t-around); k <= min(nf-1, t+around); k++ {
			sum += smooth[k]
			n++
		}
		if smooth[t] < sensitivity*sum/float64(n) || smooth[t] < smooth[t-1] || smooth[t] < smooth[t+1] {
			continue
		}
		if t-last < gap {
			if smooth[t] <= smooth[last] || len(cuts) < 2 { // two close: the stronger stays
				continue
			}
			cuts = cuts[:len(cuts)-1]
		}
		last = t
		// the middle of the change, and the quietest hop (about 6 ms) within 40 ms of it
		mid := t*hop + size/2 - hop/2
		cuts = append(cuts, quietest(mono, mid, int(0.04*sr), int(sr*0.006)))
	}
	cuts = append(cuts, int64(total))
	return snapCuts(snd, cuts)
}

// quietest returns the start of the quietest stretch of n frames within reach of a frame
func quietest(x []float64, at, reach, n int) int64 {
	best, bestE := at, math.MaxFloat64
	for s := max(0, at-reach); s <= min(len(x)-n, at+reach); s += n / 2 {
		e := 0.0
		for _, v := range x[s : s+n] {
			e += v * v
		}
		if e < bestE {
			best, bestE = s+n/2, e
		}
	}
	return int64(best)
}

// snapCuts moves each cut but the first and last to the nearest zero crossing of the sound within five milliseconds
func snapCuts(snd Sound, cuts []int64) []int64 {
	reach := int64(snd.SampleRate() * 0.005)
	total := snd.NumFrames()
	buf := make([]float32, total)
	for c := range snd.NumChannels() {
		for i, v := range snd.Buffer(c, 0)[:total] {
			buf[i] += v
		}
	}
	out := append([]int64(nil), cuts...)
	for i := 1; i < len(out)-1; i++ {
		best, bestD := out[i], int64(math.MaxInt64)
		for f := max(1, out[i]-reach); f <= min(total-1, out[i]+reach); f++ {
			if (buf[f-1] <= 0) != (buf[f] <= 0) {
				if d := abs64(f - out[i]); d < bestD {
					best, bestD = f, d
				}
			}
		}
		if best > out[i-1] && best < out[len(out)-1] {
			out[i] = best
		}
	}
	return out
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// sliceFade is the seconds a piece fades in at its start and out at its end (see Slices)
const sliceFade = 0.005

// Slices returns a region for every piece between two cuts (see EvenCuts and OnsetCuts), the first on firstKey, the next on the key above
// it and so on (at most up to key 127). Each plays its piece to the end at the pitch it was recorded, whatever the length of the note
// (OneShot). The cuts do not click, and the pieces played one after the other are the recording again, with no gaps: a piece fades in over
// five milliseconds, and goes on for five milliseconds past its end, fading out, which is where the next one fades in. Tune, envelope and the
// rest of a region can be changed on the result.
func Slices(snd Sound, firstKey int, cuts []int64) []Region {
	var regions []Region
	over := int64(sliceFade * snd.SampleRate())
	for i := 0; i+1 < len(cuts) && firstKey+i <= 127; i++ {
		key := firstKey + i
		r := NewRegion(snd).Keys(key, key, float64(key))
		r.Offset, r.End = cuts[i], min(cuts[i+1]-1+over, snd.NumFrames()-1)
		r.Loop = OneShot
		r.Env = Envelope{Attack: sliceFade, Sustain: 1, Release: 0.01}
		r.EndFade = sliceFade
		r.Name = "slice"
		regions = append(regions, r)
	}
	return regions
}
