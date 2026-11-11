package waveguide

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/almerlucke/psalm"
)

// The friction bow plays a little off pitch: the stick and the slip change the time the wave takes
// round the loop by an amount that depends on the weight of the bow, its speed and where it is on
// the string. So the bow is run on a string of its own, a short while, the period of the motion is
// measured and the lines are given the difference.
//
// That takes tens of milliseconds, so never on the audio thread. A note looks its tuning up in a cache
// that is published whole (a lookup, nothing locked or allocated); on a miss it takes an estimate from
// the nearest pitch that is cached (or the offset a bow of that position has, if none is) and asks the
// one worker goroutine to calibrate it for the next time. A voice with a friction bow asks for all
// the pitches of its bow when it is made (see WarmBow).

// fricKey identifies a calibration: a bow (the table bow when table is set, its slope in force) at a
// pitch
type fricKey struct {
	freq, pos, force, speed, attack, only, sr float64
	follow, table                             bool
}

// family is the key without the pitch
func (k fricKey) family() fricKey {
	k.freq = 0
	return k
}

// fricScales are the weights a bow tries, as a share of its own, when the one it has does not hold
// the string in the motion of its pitch: a player leans on or lets up
var fricScales = [...]float64{1, 1.15, 0.87, 1.3, 0.76}

// fricFit is what a bow is given at the start of a note
type fricFit struct {
	d, scale float64
	holds    bool // the bow can be tuned to the pitch
}

type famEntry struct {
	freq float64
	fit  fricFit
}

// fricStore is a published set of calibrations: it is never changed, a new one replaces it
type fricStore struct {
	exact map[fricKey]fricFit
	fam   map[fricKey][]famEntry // by pitch
}

var (
	fricCur     atomic.Pointer[fricStore]
	fricWriteMu sync.Mutex
	fricQueue   = make(chan fricKey, 16384)
	fricPending atomic.Int64
	fricOnce    sync.Once
	fricWarmed  sync.Map
)

// fricLookup is the calibration of a bow at a pitch, when it has been made
func fricLookup(k fricKey) (fricFit, bool) {
	s := fricCur.Load()
	if s == nil {
		return fricFit{}, false
	}
	f, ok := s.exact[k]
	return f, ok
}

// fricPublish adds a calibration to the published set
func fricPublish(k fricKey, fit fricFit) {
	fricWriteMu.Lock()
	defer fricWriteMu.Unlock()
	old := fricCur.Load()
	n := &fricStore{exact: map[fricKey]fricFit{}, fam: map[fricKey][]famEntry{}}
	if old != nil && len(old.exact) < 16384 {
		for key, f := range old.exact {
			n.exact[key] = f
		}
		for key, l := range old.fam {
			n.fam[key] = l
		}
	}
	n.exact[k] = fit
	fk := k.family()
	l := append([]famEntry(nil), n.fam[fk]...)
	i := sort.Search(len(l), func(i int) bool { return l[i].freq >= k.freq })
	l = append(l, famEntry{})
	copy(l[i+1:], l[i:])
	l[i] = famEntry{k.freq, fit}
	n.fam[fk] = l
	fricCur.Store(n)
}

// fricEstimate is a calibration for a pitch that has none yet: from the nearest pitches of the same
// bow that have, or from what a bow at its position adds, if there are none
func fricEstimate(k fricKey) fricFit {
	if s := fricCur.Load(); s != nil {
		l := s.fam[k.family()]
		if len(l) > 0 {
			i := sort.Search(len(l), func(i int) bool { return l[i].freq >= k.freq })
			switch {
			case i == 0:
				return l[0].fit
			case i == len(l):
				return l[len(l)-1].fit
			}
			lo, hi := l[i-1], l[i]
			t := math.Log(k.freq/lo.freq) / math.Log(hi.freq/lo.freq)
			near := lo.fit
			if t > 0.5 {
				near = hi.fit
			}
			if !lo.fit.holds || !hi.fit.holds {
				return near
			}
			return fricFit{lo.fit.d + t*(hi.fit.d-lo.fit.d), near.scale, true}
		}
	}
	if k.table {
		return fricFit{0, 1, true}
	}
	// near the bridge the bow plays a good half sample sharp, in the middle hardly
	d := 0.1 + 0.4*max(0, min((0.4-k.pos)/0.273, 1))
	return fricFit{d, 1, k.sr/k.freq >= 32}
}

// fricRequest asks for a calibration to be made in the background (it never waits)
func fricRequest(k fricKey) bool {
	fricOnce.Do(func() { go fricWorker() })
	fricPending.Add(1)
	select {
	case fricQueue <- k:
		return true
	default:
		fricPending.Add(-1)
		return false
	}
}

// fricWorker makes the calibrations that are asked for, one at a time
func fricWorker() {
	for k := range fricQueue {
		if _, ok := fricLookup(k); !ok {
			fricPublish(k, fricCalibrate(k))
		}
		fricPending.Add(-1)
	}
}

// WaitBow waits until the calibrations that are asked for have been made (for tests and for a patch
// that wants every pitch of its bows tuned before it plays)
func WaitBow() {
	for fricPending.Load() > 0 {
		time.Sleep(2 * time.Millisecond)
	}
}

// WarmBow asks for the calibrations of a friction bow for every pitch of its range (the lowest
// string of a cello to the top of a violin, at a full note), so that a note finds its tuning ready. A
// voice with a friction bow does this itself when it is made; nothing is done twice
func WarmBow(cfg Config) {
	if cfg.Model != Bow || cfg.BowModel != 1 {
		return
	}
	warmBow(&cfg, psalm.CurrentEnv().Config().SampleRate)
}

func (v *Voice) warmBow() {
	if v.cfg.Model == Bow && v.cfg.BowModel == 1 {
		warmBow(&v.cfg, v.Config().SampleRate)
	}
}

func warmBow(cfg *Config, sr float64) {
	pos := max(0.02, min(cfg.BowPosition, 0.5))
	speed := tableSpeedOf(1, 0.8)
	if cfg.BowSpeed > 0 {
		speed = bowSpeedOf(cfg.BowSpeed)
	}
	follow := cfg.BowSpeed <= 0
	force := bowForceOf(cfg.BowForce)
	if follow {
		force *= speed / bowSpeedRef
	}
	slope := bowSlope(cfg.Pressure)
	tspeed := tableSpeedOf(1, 0.8)
	id := [8]float64{pos, force, speed, cfg.BowAttack, slope, tspeed, sr, 0}
	if follow {
		id[7] = 1
	}
	if _, done := fricWarmed.LoadOrStore(id, true); done {
		return
	}
	queued := true
	for p := 36.0; p <= 96; p++ {
		f := 440 * math.Exp2((p-69)/12)
		a := fricRequest(fricKey{freq: f, pos: pos, force: force, speed: speed, attack: cfg.BowAttack, sr: sr, follow: follow})
		b := fricRequest(fricKey{freq: f, pos: pos, force: slope, speed: tspeed, attack: cfg.BowAttack, sr: sr, table: true})
		queued = queued && a && b
	}
	if !queued {
		fricWarmed.Delete(id) // the queue was full: the next voice asks again
	}
}

// bowSlope is the slope of the friction of the bow of the table for a pressure
func bowSlope(pressure float64) float64 { return 5 - 4*max(0, min(pressure, 1)) }

// fricCalibrate runs the bow of a key on a string of its own and makes its calibration: the samples to
// add to the lines, and the share of its force the bow needs to hold the string in the motion of its
// pitch (a bow is leaned on or let up until it does, 1 when its own weight does), and whether any
// weight does (holds): at the highest notes the stick and slip fall on whole samples and none does.
// It uses nothing but its own state.
func fricCalibrate(k fricKey) fricFit {
	base := bowDelayFor(k.freq, k.sr)
	target := k.sr / k.freq
	if k.table {
		d, found := tuneLines(target, func(try float64) (float64, bool) {
			return linePeriod(base+try, k.pos, 0, k.speed, k.attack, false, k.force, k.sr, target)
		})
		if found > max(0.1, 0.003*target) {
			d = 0
		}
		return fricFit{d, 1, true}
	}
	best, bestErr := fricFit{0, 1, false}, math.Inf(1)
	scales := fricScales[:]
	if k.only > 0 {
		scales = []float64{k.only} // a glide keeps the weight it has
	}
	for _, scale := range scales {
		d, found := tuneLines(target, func(try float64) (float64, bool) {
			return fricPeriod(base+try, k.pos, k.force*scale, k.speed, k.attack, k.follow, k.sr, target)
		})
		if scale == 1 && math.IsInf(found, 1) && !k.follow {
			break // a bow with a speed of its own, set to be too heavy or too light, is left so
		}
		if found < bestErr {
			best, bestErr = fricFit{d, scale, found < max(0.03, 0.002*target)}, found
		}
		if found < max(0.03, 0.0015*target) {
			break
		}
	}
	return best
}

// tuneLines finds the samples to add to lines so that a motion that period measures (the samples of
// its period, and whether it is steady) has the period target: what the period lacks is added a few
// times, and the length that came nearest is kept, with how near (infinite when never steady)
func tuneLines(target float64, period func(float64) (float64, bool)) (d, found float64) {
	try, found := 0.0, math.Inf(1)
	for range 3 {
		p, steady := period(try)
		if !steady {
			break
		}
		e := target - p
		if math.Abs(e) < found {
			d, found = try, math.Abs(e)
		}
		if found < max(0.02, 0.0005*target) {
			break
		}
		try = max(-1.5, min(try+e, 3))
	}
	return d, found
}

// fricPeriod runs a bow on a string with lines of length total and returns the period of the motion
// in samples, and whether it is steady
func fricPeriod(total, pos, force, speed, attack float64, follow bool, sr, target float64) (float64, bool) {
	return linePeriod(total, pos, force, speed, attack, follow, 0, sr, target)
}

// linePeriod is the same for the bow of the table when slope is above 0
func linePeriod(total, pos, force, speed, attack float64, follow bool, slope, sr, target float64) (float64, bool) {
	size := int(total) + 8
	neck := delay{buf: make([]float64, size)}
	bridge := delay{buf: make([]float64, size)}
	bridge.len = max(total*pos, minBridge)
	neck.len = total - bridge.len
	v := &Voice{heatRate: 1 / (frictionCool * sr)}
	ramp := max(0.005, min(attack, 1.5)) * sr
	settle := int(ramp + 0.1*sr)
	k := max(1, int(0.03*sr/target))
	window := max(int(0.04*sr), int(3*target))
	n := settle + window + int(float64(k)*(target+5)) + int(0.1*float64(k)) + 12
	x := make([]float64, n)
	p := bowPole(sr)
	var filt, env float64
	step := 1 / max(1, max(0.005, attack)*sr)
	for i := range n {
		// the bow speeds up as the voice's does (to the end of the attack the sample counts are the same)
		env = min(1, env+step)
		nut := -neck.out
		filt = (1-p)*bridge.out*0.95 + p*filt
		br := -filt
		dv := speed*env - (nut + br)
		push := 0.0
		if env > 0 {
			f := force
			if follow {
				f *= env
			}
			if slope > 0 {
				push = dv * bowTable(dv, slope)
			} else {
				push = v.friction(dv, f, 1)
			}
		}
		neck.tick(br + push)
		bridge.tick(nut + push)
		x[i] = bridge.out * 1.2
	}
	// the period: where the motion best matches itself, found with one period and then k periods on
	corr := func(l int) float64 {
		var a, b, c float64
		for i := settle; i < settle+window; i++ {
			a += x[i] * x[i+l]
			b += x[i] * x[i]
			c += x[i+l] * x[i+l]
		}
		return a / math.Sqrt(b*c+1e-30)
	}
	find := func(centre float64, span int) (float64, float64) {
		c0 := int(math.Round(centre))
		best, bl := -2.0, c0
		for l := c0 - span; l <= c0+span; l++ {
			if c := corr(l); c > best {
				best, bl = c, l
			}
		}
		a, b, c := corr(bl-1), corr(bl), corr(bl+1)
		off := 0.0
		if den := a - 2*b + c; den < 0 {
			off = 0.5 * (a - c) / den
		}
		return float64(bl) + off, best
	}
	coarse, best := find(target, 4)
	if best < 0.9 {
		return target, false // no steady motion to measure
	}
	// a motion of half the period (the string jumped to its octave) matches itself at the period too
	half := -2.0
	for l := int(coarse/2) - 2; l <= int(coarse/2)+2; l++ {
		half = max(half, corr(l))
	}
	if half > 0.8 {
		return target, false
	}
	if k == 1 {
		return coarse, true
	}
	fine, best := find(float64(k)*coarse, int(math.Ceil(float64(k)*0.1))+2)
	if best < 0.9 {
		return coarse, true
	}
	return fine / float64(k), true
}
