package sampler

import (
	"math"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/modules/poly"
)

// Config is a sampler playing an instrument
type Config struct {
	NumChannels int
	Instrument  *Instrument
	Level       float64 // output level (0 is treated as 1)
	// Layers is how many regions one note can play at once, the regions of its key down and the
	// release regions of its key up together (0 is 8)
	Layers int
}

// DefaultConfig plays an instrument in stereo
func DefaultConfig(inst *Instrument) Config {
	return Config{NumChannels: psalm.Stereo, Instrument: inst, Level: 1}
}

// shared is what the voices of one sampler share: the round robin turns of the regions, the random
// numbers, and the voices themselves, for choke groups
type shared struct {
	turns  []int // per region: how often its notes played
	rng    *rand.Rand
	voices []*Voice
	cc     [128]float64 // the controllers, 0-127
	sw     int          // the last keyswitch pressed (-1 none)
	starts int64        // counts the regions started: the oldest let go first
}

// ccGain returns the gain the controllers give a region now: its fades by controller and its
// controller gains
func (sh *shared) ccGain(r *Region) float64 {
	g, db := 1.0, 0.0
	for _, f := range r.Fades {
		if f.From >= 0 && int(f.From) < len(sh.cc) {
			g *= f.gain(sh.cc[f.From])
		}
	}
	for _, c := range r.CCGain {
		if c.CC >= 0 && c.CC < len(sh.cc) {
			db += c.DB * sh.cc[c.CC] / 127
		}
	}
	if db != 0 {
		g *= math.Pow(10, db/20)
	}
	return g
}

// Voice plays one note of a sampler: its regions, and the release regions when the key goes up.
// Notes take "pitch" (or "frequency") and an optional "velocity" (0-1, default 1)
type Voice struct {
	*psalm.Module
	cfg       Config
	envOver   envOverride // the envelope set by messages, over the regions' own
	shared    *shared
	layers    []layer
	pitch     float64
	velocity  float64
	amp       float64
	held      int                   // samples since the note started, for the decay of release regions
	noteOff   int                   // samples until the note of Note ends, 0 when none
	released  bool                  // the key is up
	sustained bool                  // the key is up, the sustain pedal holds the note
	expr      expression.Expression // the pitch bend (the regions' bend_up and bend_down); the mod wheel is controller 1
	bendFrom  float64               // the bend at the start of the buffer
	down      bool                  // the key is down (the pedal aside): for trigger=first and legato
	others    int                   // the other keys down when the note started
	waitSamps int
}

// layer is one region playing
type layer struct {
	r         *Region
	index     int     // of the region in the instrument
	pos       float64 // the frame playing
	step      float64 // frames per sample
	depth     int     // the band limited copy of the sound
	gainL     float64
	gainR     float64
	looping   bool // inside the loop: wraps at the loop end
	env       envelope
	filters   [2]*svf.SVF
	filterOn  bool
	filterMod svf.Mode
	active    bool
	cc        float64 // the gain of the controllers, gliding to theirs over a buffer
	key       int     // the key it plays
	started   int64   // when it started: the oldest let go first
	envSet    Envelope
	wait      int     // samples before it starts (Delay, DelayRandom)
	stepBase  float64 // the step without the vibrato
	lfos      []lfoRun
	eq        [maxEQ][2]biquad
	eqN       int
	width     float64 // the side's gain of a stereo sound
}

// maxEQ is the most EQ bands a region has
const maxEQ = 3

// lfoRun is an LFO of a layer playing
type lfoRun struct {
	set   *LFO
	phase float64
	inc   float64
	wait  int     // samples of delay left
	fade  float64 // per sample, the growth to full depth
	depth float64 // 0 to 1
}

// value returns the LFO's next value, -1 to 1 (times its depth so far)
func (l *lfoRun) value() float64 {
	if l.wait > 0 {
		l.wait--
		return 0
	}
	l.depth = math.Min(1, l.depth+l.fade)
	p := l.phase
	l.phase += l.inc
	l.phase -= math.Floor(l.phase)
	var y float64
	switch l.set.Wave {
	case LFOSine:
		y = math.Sin(2 * math.Pi * p)
	case LFOPulse75:
		y = pulse(p, 0.75)
	case LFOSquare:
		y = pulse(p, 0.5)
	case LFOPulse25:
		y = pulse(p, 0.25)
	case LFOPulse12:
		y = pulse(p, 0.125)
	case LFOSawUp:
		y = 2*p - 1
	case LFOSawDown:
		y = 1 - 2*p
	default: // triangle
		y = 1 - 4*math.Abs(p-0.5)
	}
	return y * l.depth
}

func pulse(p, width float64) float64 {
	if p < width {
		return 1
	}
	return -1
}

// biquad is a filter of two poles and two zeros (transposed direct form II)
type biquad struct {
	b0, b1, b2, a1, a2 float64
	z1, z2             float64
}

// peaking sets a peaking filter: gain dB around freq, over a bandwidth in octaves (RBJ)
func (b *biquad) peaking(freq, octaves, gain, sr float64) {
	a := math.Pow(10, gain/40)
	w := 2 * math.Pi * math.Min(freq, sr*0.45) / sr
	alpha := math.Sin(w) * math.Sinh(math.Ln2/2*math.Max(0.01, octaves)*w/math.Sin(w))
	a0 := 1 + alpha/a
	*b = biquad{
		b0: (1 + alpha*a) / a0, b1: -2 * math.Cos(w) / a0, b2: (1 - alpha*a) / a0,
		a1: -2 * math.Cos(w) / a0, a2: (1 - alpha/a) / a0,
	}
}

func (b *biquad) process(x float64) float64 {
	y := b.b0*x + b.z1
	b.z1 = b.b1*x - b.a1*y + b.z2
	b.z2 = b.b2*x - b.a2*y
	return y
}

// New returns a voice. Voices made one by one don't share round robin turns or choke groups; use
// NewPoly for a playable sampler
func New(cfg Config) *Voice {
	return newVoice(cfg, nil)
}

func newVoice(cfg Config, sh *shared) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	if cfg.Level <= 0 {
		cfg.Level = 1
	}
	if cfg.Layers <= 0 {
		cfg.Layers = 8
	}
	v := &Voice{
		Module: psalm.NewModule(nil, []int{cfg.NumChannels}),
		cfg:    cfg,
		layers: make([]layer, cfg.Layers),
		pitch:  60,
	}
	sr := v.Config().SampleRate
	for i := range v.layers {
		v.layers[i].filters = [2]*svf.SVF{svf.New(1000, 0, sr), svf.New(1000, 0, sr)}
	}
	if sh == nil {
		sh = &shared{rng: v.Rand(), sw: -1}
		if inst := cfg.Instrument; inst != nil {
			sh.turns = make([]int, len(inst.Regions))
			for cc, val := range inst.CC {
				if cc >= 0 && cc < len(sh.cc) {
					sh.cc[cc] = val
				}
			}
			_, sh.sw = inst.Switches()
		}
	}
	v.shared = sh
	sh.voices = append(sh.voices, v)
	v.SetSelf(v)
	return v
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	v.envOver.receive(msg)
	msg.IfFltVal("pitch", func(p float64) { v.pitch = p })
	msg.IfFltVal("frequency", func(f float64) {
		if f > 0 {
			v.pitch = 69 + 12*math.Log2(f/440)
		}
	})
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
	// the pitch bend, and the mod wheel as controller 1 (what SFZ instruments listen to)
	v.expr.Receive(msg)
	msg.IfFltVal("modWheel", func(f float64) { v.shared.cc[1] = math.Max(0, math.Min(f, 1)) * 127 })
	// the sustain pedal, 0 up to 1 down: its controller
	if inst := v.cfg.Instrument; inst != nil {
		msg.IfFltVal("sustain", func(f float64) {
			cc, _ := inst.pedal()
			v.shared.cc[cc] = math.Max(0, math.Min(f, 1)) * 127
		})
	}
	defer func() {
		if v.sustained && !v.pedalDown() {
			v.release() // the pedal came up
		}
	}()
	// controllers: "cc1" (the mod wheel) to "cc127", 0-1
	for key := range msg {
		if n, ok := strings.CutPrefix(key, "cc"); ok {
			if cc, err := strconv.Atoi(n); err == nil && cc >= 0 && cc < len(v.shared.cc) {
				msg.IfFltVal(key, func(f float64) { v.shared.cc[cc] = math.Max(0, math.Min(f, 1)) * 127 })
			}
		}
	}
}

func (v *Voice) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
	}
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(0, amplitude, message, tick)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(duration, amplitude, message, tick)
}

func (v *Voice) note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	v.velocity = 1
	v.ReceiveMessage(message)
	v.waitSamps = tick.SampleOffset
	v.amp = amplitude
	v.held = 0
	v.released, v.sustained = false, false
	// the other keys down decide the regions of trigger=first and legato
	v.others = 0
	for _, o := range v.shared.voices {
		if o != v && o.down {
			v.others++
		}
	}
	v.down = true
	v.noteOff = 0
	if duration > 0 {
		v.noteOff = max(1, int(duration*v.Config().SampleRate))
	}
	v.start(OnAttack)
}

// key and vel return the MIDI key and velocity of the note
func (v *Voice) key() int { return int(math.Round(v.pitch)) }
func (v *Voice) vel() int { return max(1, min(127, int(math.Round(v.velocity*127)))) }

// start plays the regions of a trigger that match the note
func (v *Voice) start(trigger Trigger) {
	inst := v.cfg.Instrument
	if inst == nil {
		return
	}
	sh := v.shared
	key, vel := v.key(), v.vel()
	if trigger == OnAttack && inst.isSwitch(key) {
		sh.sw = key // a keyswitch: the articulation from now on
	}
	random := sh.rng.Float64()
	var playing [16]int
	matched := playing[:0]
	for i := range inst.Regions {
		r := &inst.Regions[i]
		if !triggers(r.Trigger, trigger, v.others) || !r.plays(key, vel) || (r.SwLast >= 0 && r.SwLast != sh.sw) {
			continue
		}
		if fadeAt(r, key, vel) == 0 {
			continue // faded out at this velocity or key
		}
		if r.SeqLength > 0 {
			turn := sh.turns[i]
			sh.turns[i]++
			if turn%r.SeqLength != r.SeqPosition-1 {
				continue
			}
		}
		if random < r.LoRand || random >= r.HiRand {
			continue
		}
		matched = append(matched, i)
	}
	// the groups starting stop what they turn off, then the regions start
	for _, i := range matched {
		if g := inst.Regions[i].Group; g != 0 {
			sh.choke(g)
		}
	}
	for _, i := range matched {
		v.play(&inst.Regions[i], i)
	}
}

// triggers says whether a region of a trigger plays at a moment (the key going down or up, with
// others held or not)
func triggers(region, moment Trigger, others int) bool {
	switch region {
	case OnFirst:
		return moment == OnAttack && others == 0
	case OnLegato:
		return moment == OnAttack && others > 0
	}
	return region == moment
}

// fadeAt returns the gain of a region's crossfades by velocity and key for a note
func fadeAt(r *Region, key, vel int) float64 {
	g := 1.0
	for _, f := range r.Fades {
		switch f.From {
		case FromVelocity:
			g *= f.gain(float64(vel))
		case FromKey:
			g *= f.gain(float64(key))
		}
	}
	return g
}

// polyGroup returns the polyphony group of a region: its group, or the region itself without one
func polyGroup(r *Region, index int) int {
	if r.Group != 0 {
		return r.Group
	}
	return -1 - index
}

// limit lets go of the oldest sounding regions of a region's group (of one key, or of any key with
// key -1) until fewer than limit sound, so the region starting makes limit
func (sh *shared) limit(r *Region, index, key, limit int) {
	g := polyGroup(r, index)
	for {
		var oldest *layer
		count := 0
		for _, o := range sh.voices {
			for i := range o.layers {
				l := &o.layers[i]
				if !l.active || l.env.stage >= stageRelease || polyGroup(l.r, l.index) != g || (key >= 0 && l.key != key) {
					continue
				}
				count++
				if oldest == nil || l.started < oldest.started {
					oldest = l
				}
			}
		}
		if count < limit || oldest == nil {
			return
		}
		oldest.env.releaseIn(0.03)
	}
}

// choke stops the regions that a region of a group starting turns off
func (sh *shared) choke(group int) {
	for _, o := range sh.voices {
		for i := range o.layers {
			l := &o.layers[i]
			if !l.active || l.r.OffBy != group {
				continue
			}
			if l.r.OffMode == OffNormal {
				l.env.release()
			} else {
				l.env.releaseIn(0.006)
			}
		}
	}
}

// play starts a region in a free layer (the quietest one when all are playing)
func (v *Voice) play(r *Region, index int) {
	// past its polyphony, the oldest of its group (or of its key) let go
	if r.Polyphony > 0 {
		v.shared.limit(r, index, -1, r.Polyphony)
	}
	if r.NotePolyphony > 0 {
		v.shared.limit(r, index, v.key(), r.NotePolyphony)
	}
	var l *layer
	for i := range v.layers {
		if !v.layers[i].active {
			l = &v.layers[i]
			break
		}
	}
	if l == nil {
		l = &v.layers[0]
		for i := range v.layers {
			if v.layers[i].env.level < l.env.level {
				l = &v.layers[i]
			}
		}
	}
	sr := v.Config().SampleRate
	semis := r.Transpose + r.Tune/100
	if r.KeyTrack != 0 {
		semis += (v.pitch - r.Root) * r.KeyTrack / 100
	}
	rng := v.shared.rng
	spread := func(width float64) float64 { return (rng.Float64() - 0.5) * width } // -width/2 to width/2
	if r.PitchRandom != 0 {
		semis += spread(r.PitchRandom) / 100
	}
	*l = layer{r: r, index: index, filters: l.filters, lfos: l.lfos[:0], active: true, pos: float64(r.Offset),
		key: v.key(), started: v.shared.starts}
	v.shared.starts++
	l.step = math.Exp2(semis/12) * r.Sound.SampleRate() / sr
	l.stepBase = l.step
	// the deepest vibrato plays faster: the band limited copy for its top
	top := 0.0
	for i := range r.LFOs {
		top += math.Abs(r.LFOs[i].Pitch)
	}
	l.depth = min(depthFor(l.step*math.Exp2(top/1200)), r.Sound.Depth()-1)
	l.wait = int((math.Max(0, r.Delay) + rng.Float64()*math.Max(0, r.DelayRandom)) * sr)
	for i := range r.LFOs {
		f := &r.LFOs[i]
		run := lfoRun{set: f, inc: math.Max(0, f.Freq) / sr, wait: int(math.Max(0, f.Delay) * sr), depth: 1}
		if f.Fade > 0 {
			run.depth, run.fade = 0, 1/(f.Fade*sr)
		}
		l.lfos = append(l.lfos, run)
	}
	for _, b := range r.EQ {
		if b.Gain != 0 && l.eqN < maxEQ {
			for c := range 2 {
				l.eq[l.eqN][c].peaking(b.Freq, b.Bandwidth, b.Gain, sr)
			}
			l.eqN++
		}
	}
	l.width = r.Width / 100

	// the loudness: the velocity, squared, as much as the region follows it
	vel := v.velocity
	gain := v.amp * v.cfg.Level * r.Amp * math.Pow(10, r.Volume/20) * (1 - r.VelTrack + r.VelTrack*vel*vel) *
		fadeAt(r, v.key(), v.vel())
	if r.Trigger == OnRelease && r.RtDecay > 0 {
		gain *= math.Pow(10, -r.RtDecay*float64(v.held)/sr/20)
	}
	if r.AmpRandom != 0 {
		gain *= math.Pow(10, spread(r.AmpRandom)/20)
	}
	pan := math.Max(-1, math.Min(r.Pan, 1))
	if r.Sound.NumChannels() == 1 {
		a := (pan + 1) * math.Pi / 4
		l.gainL, l.gainR = gain*math.Cos(a)*math.Sqrt2, gain*math.Sin(a)*math.Sqrt2
	} else { // balance
		l.gainL, l.gainR = gain*math.Min(1, 1-pan), gain*math.Min(1, 1+pan)
	}
	if r.Cutoff > 0 {
		fc := r.Cutoff * math.Exp2(r.FilVelTrack*vel/1200)
		fc = math.Min(fc, sr*0.45)
		for _, f := range l.filters {
			f.Clear()
			f.SetFrequencyQ(fc, math.Max(math.Sqrt2/2, math.Pow(10, r.Resonance/20)))
		}
		l.filterOn = true
		l.filterMod = map[FilterType]svf.Mode{LowPass: svf.LowPass, HighPass: svf.HighPass, BandPass: svf.BandPass}[r.Filter]
	}
	// the envelope, with what the velocity and the controllers add to it
	l.envSet = v.envFor(r)
	l.cc = v.shared.ccGain(r)
	l.env.start(&l.envSet, sr)
}

// envFor returns the envelope of a region for the note: its own, with what the velocity and the
// controllers (as they are now) add to it
func (v *Voice) envFor(r *Region) Envelope {
	e := r.Env
	vel := v.velocity
	add := func(d Envelope, k float64) {
		e.Delay += d.Delay * k
		e.Attack += d.Attack * k
		e.Hold += d.Hold * k
		e.Decay += d.Decay * k
		e.Sustain += d.Sustain * k
		e.Release += d.Release * k
	}
	add(r.VelEnv, vel)
	for _, c := range r.CCEnv {
		if c.CC >= 0 && c.CC < len(v.shared.cc) {
			add(c.Env, v.shared.cc[c.CC]/127)
		}
	}
	v.envOver.apply(&e)
	e.Delay, e.Attack, e.Hold = max(0, e.Delay), max(0, e.Attack), max(0, e.Hold)
	e.Decay, e.Release = max(0, e.Decay), max(0.001, e.Release)
	return e
}

// envOverride is an envelope set by messages ("ampAttack", "ampDecay", "ampRelease" in
// milliseconds, "ampSustain" 0-1) over the envelopes of the regions: each part set replaces theirs
type envOverride struct {
	attack, decay, sustain, release float64
	set                             [4]bool
}

func (o *envOverride) receive(msg psalm.Message) {
	for i, name := range []string{"ampAttack", "ampDecay", "ampSustain", "ampRelease"} {
		msg.IfFltVal(name, func(f float64) {
			o.set[i] = true
			switch i {
			case 0:
				o.attack = math.Max(0, f) / 1000
			case 1:
				o.decay = math.Max(0, f) / 1000
			case 2:
				o.sustain = math.Max(0, math.Min(f, 1))
			case 3:
				o.release = math.Max(0.1, f) / 1000
			}
		})
	}
}

func (o *envOverride) apply(e *Envelope) {
	if o.set[0] {
		e.Attack = o.attack
	}
	if o.set[1] {
		e.Decay = o.decay
	}
	if o.set[2] {
		e.Sustain = o.sustain
	}
	if o.set[3] {
		e.Release = o.release
	}
}

// depthFor returns the band limited copy of a sound for playing it step times as fast: copy d is
// filtered for speeds up to d+1
func depthFor(step float64) int {
	return max(0, int(math.Ceil(step-1e-4))-1)
}

// NoteOff lets go of the note, or keeps it sounding while the sustain pedal is down
func (v *Voice) NoteOff() {
	v.down = false
	if v.released {
		return
	}
	if v.pedalDown() {
		v.sustained = true
		return
	}
	v.release()
}

// pedalDown says whether the instrument's sustain pedal is down
func (v *Voice) pedalDown() bool {
	inst := v.cfg.Instrument
	if inst == nil || inst.NoSustain {
		return false
	}
	cc, lo := inst.pedal()
	return v.shared.cc[cc] >= lo
}

// release lets go of the note: the regions fade out with their release, the release regions play
func (v *Voice) release() {
	v.sustained = false
	if v.released {
		return
	}
	v.released = true
	v.noteOff = 0
	for i := range v.layers {
		l := &v.layers[i]
		if !l.active || l.r.Trigger == OnRelease {
			continue
		}
		switch l.r.Loop {
		case OneShot:
		case LoopSustain:
			l.looping = false // plays on to the end of the sound
			l.env.release()
		default:
			l.env.release()
		}
	}
	v.start(OnRelease)
}

func (v *Voice) Clear() {
	v.sustained, v.down = false, false
	for i := range v.layers {
		v.layers[i].active = false
	}
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool {
	for i := range v.layers {
		if v.layers[i].active {
			return true
		}
	}
	return false
}

func (v *Voice) Synthesize() {
	out := v.Output(0)
	out.Clear()
	channels := out.Channels
	n := v.Config().BufferSize
	v.expr.Settle()
	bendTo := v.expr.Bend()
	defer func() { v.bendFrom = bendTo }()
	for i := range v.layers {
		if l := &v.layers[i]; l.active && l.r.EnvDynamic {
			// the envelope follows the controllers
			if e := v.envFor(l.r); e != l.envSet {
				old := l.envSet
				l.envSet = e
				l.env.retime(old)
			}
		}
		if l := &v.layers[i]; l.active {
			l.render(channels, v.waitSamps, n, v.released, v.shared.ccGain(l.r), v.bendFrom, bendTo)
		}
	}
	// the note of Note ends in this buffer
	if v.noteOff > 0 {
		if left := n - v.waitSamps; v.noteOff > left {
			v.noteOff -= left
		} else {
			v.NoteOff()
		}
	}
	v.held += n - v.waitSamps
	v.waitSamps = 0
}

// render adds the samples from..to of a layer to the channels
func (l *layer) render(channels psalm.Channels, from, to int, released bool, cc, bendFrom, bendTo float64) {
	r := l.r
	// the controllers' gain glides to where they are now over the buffer: a wheel moved never clicks
	ccInc := (cc - l.cc) / float64(max(1, to-from))
	defer func() { l.cc = cc }()
	snd := r.Sound
	left := snd.Buffer(0, l.depth)
	right := left
	if snd.NumChannels() > 1 {
		right = snd.Buffer(1, l.depth)
	}
	looped := r.Loop == LoopContinuous || (r.Loop == LoopSustain && !released)
	loopStart, loopEnd := float64(r.LoopStart), float64(r.LoopEnd)
	loopLen := loopEnd - loopStart + 1
	xf := r.LoopCrossfade * snd.SampleRate()
	end := float64(r.End)
	stereo := len(channels) > 1

	stereoSound := snd.NumChannels() > 1
	for i := from; i < to; i++ {
		if l.wait > 0 { // it starts later
			l.wait--
			l.cc += ccInc
			continue
		}
		// the LFOs and the bend: the vibrato and the bend set the speed, the tremolo the level
		lfoGain := 1.0
		bend := bendFrom
		if bendTo != bendFrom { // the bend glides over the buffer
			bend += (bendTo - bendFrom) * float64(i-from) / float64(max(1, to-from))
		}
		if len(l.lfos) > 0 || bend != 0 {
			cents, db := 0.0, 0.0
			for k := range l.lfos {
				y := l.lfos[k].value()
				cents += y * l.lfos[k].set.Pitch
				db += y * l.lfos[k].set.Volume
			}
			if bend > 0 {
				cents += bend * r.BendUp
			} else if bend < 0 {
				cents -= bend * r.BendDown
			}
			l.step = l.stepBase
			if cents != 0 {
				l.step *= math.Exp2(cents / 1200)
			}
			if db != 0 {
				lfoGain = math.Pow(10, db/20)
			}
		}
		// the loop: once in it, the sound wraps at its end
		if looped && !l.looping && l.pos >= loopStart {
			l.looping = true
		}
		if !looped {
			l.looping = false
		}
		if l.looping {
			for l.pos >= loopEnd+1 {
				l.pos -= loopLen
			}
		} else if l.pos > end {
			l.active = false
			return
		}
		limit := int64(end)
		if l.looping {
			limit = r.LoopEnd
		}
		xl := hermite(left, l.pos, limit, l.looping, r.LoopStart)
		xr := xl
		if snd.NumChannels() > 1 {
			xr = hermite(right, l.pos, limit, l.looping, r.LoopStart)
		}
		// the end of the loop fades into what comes before its start
		if l.looping && xf > 0 {
			if into := l.pos - (loopEnd + 1 - xf); into > 0 {
				t := into / xf
				a, b := math.Cos(t*math.Pi/2), math.Sin(t*math.Pi/2)
				pre := l.pos - loopLen
				xl = xl*a + hermite(left, pre, limit, false, 0)*b
				if snd.NumChannels() > 1 {
					xr = xr*a + hermite(right, pre, limit, false, 0)*b
				} else {
					xr = xl
				}
			}
		}
		e := l.env.next() * l.cc * lfoGain
		l.cc += ccInc
		if l.filterOn {
			xl = l.filters[0].Process(xl, l.filterMod)
			if snd.NumChannels() > 1 {
				xr = l.filters[1].Process(xr, l.filterMod)
			} else {
				xr = xl
			}
		}
		for k := range l.eqN {
			xl = l.eq[k][0].process(xl)
			if stereoSound {
				xr = l.eq[k][1].process(xr)
			} else {
				xr = xl
			}
		}
		if stereoSound && l.width != 1 {
			mid, side := (xl+xr)/2, (xl-xr)/2*l.width
			xl, xr = mid+side, mid-side
		}
		if stereo {
			channels[0][i] += float32(xl * e * l.gainL)
			channels[1][i] += float32(xr * e * l.gainR)
		} else {
			channels[0][i] += float32((xl*l.gainL + xr*l.gainR) * 0.5 * e)
		}
		if l.env.stage == stageDone {
			l.active = false
			return
		}
		l.pos += l.step
	}
}

// hermite reads a buffer between frames with a cubic (4 points). Frames after limit wrap to
// loopStart when looping, and hold the last frame when not
func hermite(buf []float32, pos float64, limit int64, looping bool, loopStart int64) float64 {
	i := int64(pos)
	t := pos - float64(i)
	at := func(j int64) float64 {
		if j > limit {
			if !looping {
				return float64(buf[limit])
			}
			j = loopStart + (j-limit-1)%(limit-loopStart+1)
		}
		if j < 0 {
			j = 0
		}
		return float64(buf[j])
	}
	y0, y1, y2, y3 := at(i-1), at(i), at(i+1), at(i+2)
	c1 := 0.5 * (y2 - y0)
	c2 := y0 - 2.5*y1 + 2*y2 - 0.5*y3
	c3 := 0.5*(y3-y0) + 1.5*(y1-y2)
	return ((c3*t+c2)*t+c1)*t + y1
}

// NewPoly returns a sampler of numVoices voices, which share round robin turns and choke groups
func NewPoly(cfg Config, numVoices int) *poly.Poly[*Voice] {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	voices := make([]*Voice, numVoices)
	var sh *shared
	for i := range numVoices {
		voices[i] = newVoice(cfg, sh)
		sh = voices[i].shared
	}
	return poly.New[*Voice](cfg.NumChannels, voices)
}
