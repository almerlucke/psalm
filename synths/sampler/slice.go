package sampler

import (
	"math"
	"slices"

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

// PauseCuts returns the cuts at the pauses of a recording, for speech and phrases: where the level stays more than depth dB (20 to 40) under
// the loud parts for minPause seconds or longer, the cut is in the middle of the pause. A piece that is still longer than maxLen seconds
// (0: no limit) is divided into equal parts, each cut moved to the quietest place near it. The first cut is 0 and the last the end.
func PauseCuts(snd Sound, depth, minPause, maxLen float64) []int64 {
	total := snd.NumFrames()
	sr := snd.SampleRate()
	hop := max(16, int(sr*0.01))
	nh := int(total) / hop
	if nh < 2 {
		return []int64{0, total}
	}
	level := make([]float64, nh)
	for h := range level {
		sum, count := 0.0, 0
		for c := range snd.NumChannels() {
			for _, v := range snd.Buffer(c, 0)[h*hop : (h+1)*hop] {
				sum += float64(v) * float64(v)
				count++
			}
		}
		level[h] = 10 * math.Log10(sum/float64(count)+1e-12)
	}
	// the loud level: the 95th percentile
	sorted := append([]float64(nil), level...)
	slices.Sort(sorted)
	threshold := sorted[int(0.95*float64(nh-1))] - depth
	cuts := []int64{0}
	need := max(1, int(minPause*sr)/hop)
	start := -1
	flush := func(end int) {
		if start >= 0 && end-start >= need {
			if at := int64((start+end)/2) * int64(hop); at > cuts[len(cuts)-1] && at < total {
				cuts = append(cuts, at)
			}
		}
		start = -1
	}
	for h := range level {
		if level[h] < threshold {
			if start < 0 {
				start = h
			}
		} else {
			flush(h)
		}
	}
	flush(nh)
	cuts = append(cuts, total)
	// the pieces that are too long
	if maxLen > 0 {
		var out []int64
		for i := 0; i+1 < len(cuts); i++ {
			out = append(out, cuts[i])
			length := float64(cuts[i+1]-cuts[i]) / sr
			if length <= maxLen {
				continue
			}
			parts := int(math.Ceil(length / maxLen))
			for j := 1; j < parts; j++ {
				target := cuts[i] + (cuts[i+1]-cuts[i])*int64(j)/int64(parts)
				reach := (cuts[i+1] - cuts[i]) / int64(parts) / 6
				best, bestL := target, math.MaxFloat64
				for h := int(max(0, target-reach)) / hop; h <= int(min(total-1, target+reach))/hop && h < nh; h++ {
					if level[h] < bestL {
						best, bestL = int64(h*hop+hop/2), level[h]
					}
				}
				out = append(out, best)
			}
		}
		cuts = append(out, total)
	}
	return snapCuts(snd, cuts)
}

// VerseCuts returns the cuts of a recitation of `verses` verses, each followed by a refrain (a psalm said with its response): 2*verses+1
// cuts, so the pieces are the first verse, its refrain, the second verse, its refrain and so on. The verses are about equally long, so
// their ends are found among the pauses (a level under the loud parts by depth dB, 20 to 30, of 0.3 seconds or more) by the sequence of
// them that fits best: nearly equal verses, ending in long pauses. Each verse is then divided at the pause nearest its middle, or where it is
// quietest there when it has none. The first cut is 0 and the last the end.
func VerseCuts(snd Sound, verses int, depth float64) []int64 {
	total := snd.NumFrames()
	sr := snd.SampleRate()
	hop := max(16, int(sr*0.01))
	nh := int(total) / hop
	verses = max(1, verses)
	if nh < 2*verses {
		return EvenCuts(snd, 2*verses)
	}
	level := make([]float64, nh)
	for h := range level {
		sum, count := 0.0, 0
		for c := range snd.NumChannels() {
			for _, v := range snd.Buffer(c, 0)[h*hop : (h+1)*hop] {
				sum += float64(v) * float64(v)
				count++
			}
		}
		level[h] = 10 * math.Log10(sum/float64(count)+1e-12)
	}
	sorted := append([]float64(nil), level...)
	slices.Sort(sorted)
	threshold := sorted[int(0.95*float64(nh-1))] - depth
	// the pauses: where they are (in hops) and how long they last (seconds)
	type pause struct {
		at  int
		dur float64
	}
	var pauses []pause
	start := -1
	for h := 0; h <= nh; h++ {
		if h < nh && level[h] < threshold {
			if start < 0 {
				start = h
			}
			continue
		}
		if start >= 0 {
			if d := float64(h-start) * float64(hop) / sr; d >= 0.08 {
				pauses = append(pauses, pause{(start + h) / 2, d})
			}
			start = -1
		}
	}
	// the ends of the verses: the long pauses, with the start and the end of the recording; the cheapest way through them in `verses` steps
	// (the cost of a verse is its length against the mean, squared, and a pause is worth its length up to a second)
	cand := []pause{{0, 0}}
	for _, p := range pauses {
		if p.dur >= 0.3 && p.at > 0 && p.at < nh-1 {
			cand = append(cand, p)
		}
	}
	cand = append(cand, pause{nh, 0})
	m := len(cand)
	mean := float64(nh) / float64(verses)
	inf := math.MaxFloat64
	cost := make([][]float64, verses+1)
	from := make([][]int, verses+1)
	for k := range cost {
		cost[k] = make([]float64, m)
		from[k] = make([]int, m)
		for j := range cost[k] {
			cost[k][j] = inf
		}
	}
	cost[0][0] = 0
	for k := 1; k <= verses; k++ {
		for j := 1; j < m; j++ {
			for i := 0; i < j; i++ {
				if cost[k-1][i] == inf {
					continue
				}
				dev := (float64(cand[j].at-cand[i].at) - mean) / mean
				c := cost[k-1][i] + dev*dev - 0.3*min(cand[j].dur, 1)
				if c < cost[k][j] {
					cost[k][j], from[k][j] = c, i
				}
			}
		}
	}
	ends := make([]int, verses+1)
	j := m - 1
	if cost[verses][j] == inf { // fewer pauses than verses: equal verses
		return EvenCuts(snd, 2*verses)
	}
	for k := verses; k >= 1; k-- {
		ends[k] = cand[j].at
		j = from[k][j]
	}
	// each verse in two, at the pause near its middle
	cuts := []int64{0}
	for k := 1; k <= verses; k++ {
		a, b := ends[k-1], ends[k]
		mid := (a + b) / 2
		best, bestScore := -1, -math.MaxFloat64
		for _, p := range pauses {
			if p.at <= a+(b-a)/5 || p.at >= b-(b-a)/5 {
				continue
			}
			if score := min(p.dur, 0.6) - math.Abs(float64(p.at-mid))/float64(b-a)*2; score > bestScore {
				best, bestScore = p.at, score
			}
		}
		if best < 0 || bestScore < -0.5 {
			best = mid
			quiet := math.MaxFloat64
			for h := a + (b-a)*3/10; h <= b-(b-a)*3/10 && h < nh; h++ {
				if level[h] < quiet {
					best, quiet = h, level[h]
				}
			}
		}
		cuts = append(cuts, int64(best*hop+hop/2), int64(min(b*hop, int(total))))
	}
	cuts[len(cuts)-1] = total
	return snapCuts(snd, cuts)
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

// Window keeps the cuts that lie in the stretch of the recording from `from` to `to` seconds (to <= 0: to the end), and ends the last piece at `to`,
// so a long recording of many kinds of sounds can be used a part at a time.
func Window(cuts []int64, sampleRate, from, to float64, frames int64) []int64 {
	end := frames
	if to > 0 {
		end = min(frames, int64(to*sampleRate))
	}
	var out []int64
	for _, c := range cuts {
		if c >= int64(from*sampleRate) && c < end {
			out = append(out, c)
		}
	}
	if len(out) > 0 {
		out = append(out, end)
	}
	return out
}

// Notes makes a region for each played note of a recording (sung notes, a phrase on an instrument): from the second `from` to `to`, at the
// pitch it is sung or played at (a MIDI note with a fraction, which tunes it). Note i is on the key firstKey+spacing*i and the keys two
// either side, so that a key a little off plays it a little higher or lower; a spacing of five keeps the notes apart. Notes begin and end
// softly (a fade in of 40 ms, a release of 0.4 s), and go on as long as the key is held: the second half of the note goes round with a crossfade.
func Notes(snd Sound, firstKey, spacing int, notes [][3]float64) []Region {
	var regions []Region
	sr := snd.SampleRate()
	for i, n := range notes {
		key := firstKey + spacing*i
		if key+2 > 127 {
			break
		}
		r := NewRegion(snd).Keys(key-2, key+2, float64(key)+n[2]-math.Round(n[2]))
		r.Offset, r.End = int64(n[0]*sr), min(int64(n[1]*sr), snd.NumFrames()-1)
		// the note goes on as long as the key is down: its second half goes round, the end fading into the start
		dur := n[1] - n[0]
		r.Loop = LoopContinuous
		r.LoopStart, r.LoopEnd = int64((n[0]+0.4*dur)*sr), r.End-int64(0.1*sr)
		r.LoopCrossfade = min(0.25, 0.4*dur)
		r.Env = Envelope{Attack: 0.04, Sustain: 1, Release: 0.4}
		r.Name = "note"
		regions = append(regions, r)
	}
	return regions
}
