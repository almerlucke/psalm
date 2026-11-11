package tune

import "math"

// fill gives the frames without a pitch the pitch of the nearest frame with one, so a track is a pitch everywhere (a pause holds the last one)
func fill(t Track) Track {
	out := Track{Hop: t.Hop, SampleRate: t.SampleRate, F0: make([]float64, len(t.F0))}
	last := 0.0
	for i, f := range t.F0 {
		if f > 0 {
			last = f
		}
		out.F0[i] = last
	}
	next := 0.0
	for i := len(t.F0) - 1; i >= 0; i-- {
		if t.F0[i] > 0 {
			next = t.F0[i]
		}
		if out.F0[i] == 0 {
			out.F0[i] = next
		}
	}
	for i := range out.F0 {
		if out.F0[i] == 0 {
			out.F0[i] = 100
		}
	}
	return out
}

// Marks finds the pitch marks of a voice: a sample for each period, on the peak of the wave in the voiced parts and evenly spaced (at the pitch held from the voice) where there is
// no pitch, so that the marks go through the whole sound
func Marks(x []float64, t Track) []int {
	held := fill(t)
	var marks []int
	n := len(x)
	// the next mark after m, n if there is none: the peak of the wave a period on in a voiced part, else a period on
	next := func(m int, sign float64, voiced bool) int {
		p := t.SampleRate / held.At(float64(m))
		if !voiced {
			return m + max(int(math.Round(p)), 8)
		}
		lo, hi := m+int(0.75*p), m+int(1.25*p)
		best, bestScore := m+int(math.Round(p)), math.Inf(-1)
		for i := lo; i <= hi && i < n; i++ {
			// the loudest peak, with a liking for the one a whole period on
			score := sign * x[i] * (1 - 0.3*math.Abs(float64(i-m)-p)/(0.25*p))
			if score > bestScore {
				best, bestScore = i, score
			}
		}
		return best
	}
	voicedAt := func(i int) bool {
		f := i / t.Hop
		return f >= 0 && f < len(t.F0) && t.F0[f] > 0
	}
	// the wave is turned so that the loudest sample of the first period of a voiced run is positive: the marks are on the peaks of that side
	signAt := func(i int) float64 {
		p := int(t.SampleRate / held.At(float64(i)))
		sign, top := 1.0, 0.0
		for j := i; j < min(i+p, n); j++ {
			if math.Abs(x[j]) > top {
				top, sign = math.Abs(x[j]), math.Copysign(1, x[j])
			}
		}
		return sign
	}
	m, sign, was := 0, 1.0, false
	marks = append(marks, 0)
	for m < n {
		v := voicedAt(m)
		if v && !was {
			sign = signAt(m)
		}
		was = v
		m = next(m, sign, v)
		if m < n {
			marks = append(marks, m)
		}
	}
	return marks
}

// Shift moves the pitch of a voice to the pitch in the target track (the same frames as t, 0 where it is left as it was), PSOLA: a grain of two periods round every mark of the voice
// is put down again at the periods of the new pitch, so the voice has the new pitch and the same vowels, and lasts as long. The whole sound goes through it, the parts without a pitch
// (the consonants, the breaths) with periods that are not changed, so they come out as they went in and there is no moment where one kind of sound turns into the other
func Shift(x []float64, t, target Track) []float64 {
	n := len(x)
	held := fill(t)
	// the pitch to go to everywhere: the target where the voice has a pitch, else the pitch it has
	goal := Track{Hop: t.Hop, SampleRate: t.SampleRate, F0: make([]float64, len(t.F0))}
	for i := range goal.F0 {
		goal.F0[i] = held.F0[i]
		if target.F0[i] > 0 && t.F0[i] > 0 {
			goal.F0[i] = target.F0[i]
		}
	}
	marks := Marks(x, t)
	wet := make([]float64, n)
	weight := make([]float64, n)
	k := 0
	for ts := 0.0; ts < float64(n); {
		for k+1 < len(marks) && math.Abs(float64(marks[k+1])-ts) <= math.Abs(float64(marks[k])-ts) {
			k++
		}
		m := marks[k]
		half := int(math.Round(t.SampleRate / held.At(float64(m))))
		if k > 0 && k+1 < len(marks) {
			half = (marks[k+1] - marks[k-1]) / 2
		}
		half = max(half, 8)
		c := int(math.Round(ts))
		for j := -half; j <= half; j++ {
			a, o := m+j, c+j
			if a < 0 || a >= n || o < 0 || o >= n {
				continue
			}
			g := 0.5 + 0.5*math.Cos(math.Pi*float64(j)/float64(half+1)) // Hann
			wet[o] += g * x[a]
			weight[o] += g
		}
		// the next grain goes a period of the voice on, shortened or lengthened by how far the pitch is moved: with no change it is the next mark, exactly, so what is not changed comes
		// out as it went in
		spacing := t.SampleRate / held.At(ts)
		if k+1 < len(marks) {
			spacing = float64(marks[k+1] - marks[k])
		}
		ts += spacing * held.At(ts) / max(goal.At(ts), 1)
	}
	out := make([]float64, n)
	for i := range out {
		if weight[i] < 1e-6 {
			out[i] = x[i]
			continue
		}
		out[i] = wet[i] / weight[i]
	}
	return out
}
