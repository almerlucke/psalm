package sampler

import (
	"math"
	"math/rand/v2"

	"github.com/almerlucke/psalm"
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
}

// Voice plays one note of a sampler: its regions, and the release regions when the key goes up.
// Notes take "pitch" (or "frequency") and an optional "velocity" (0-1, default 1)
type Voice struct {
	*psalm.Module
	cfg       Config
	shared    *shared
	layers    []layer
	pitch     float64
	velocity  float64
	amp       float64
	held      int  // samples since the note started, for the decay of release regions
	noteOff   int  // samples until the note of Note ends, 0 when none
	released  bool // the key is up
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
		sh = &shared{rng: v.Rand()}
		if cfg.Instrument != nil {
			sh.turns = make([]int, len(cfg.Instrument.Regions))
		}
	}
	v.shared = sh
	sh.voices = append(sh.voices, v)
	v.SetSelf(v)
	return v
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("pitch", func(p float64) { v.pitch = p })
	msg.IfFltVal("frequency", func(f float64) {
		if f > 0 {
			v.pitch = 69 + 12*math.Log2(f/440)
		}
	})
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
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
	v.released = false
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
	random := sh.rng.Float64()
	var playing [16]int
	matched := playing[:0]
	for i := range inst.Regions {
		r := &inst.Regions[i]
		if r.Trigger != trigger || !r.plays(key, vel) {
			continue
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
	*l = layer{r: r, index: index, filters: l.filters, active: true, pos: float64(r.Offset)}
	l.step = math.Exp2(semis/12) * r.Sound.SampleRate() / sr
	l.depth = min(depthFor(l.step), r.Sound.Depth()-1)

	// the loudness: the velocity, squared, as much as the region follows it
	vel := v.velocity
	gain := v.amp * v.cfg.Level * r.Amp * math.Pow(10, r.Volume/20) * (1 - r.VelTrack + r.VelTrack*vel*vel)
	if r.Trigger == OnRelease && r.RtDecay > 0 {
		gain *= math.Pow(10, -r.RtDecay*float64(v.held)/sr/20)
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
	l.env.start(&r.Env, sr)
}

// depthFor returns the band limited copy of a sound for playing it step times as fast: copy d is
// filtered for speeds up to d+1
func depthFor(step float64) int {
	return max(0, int(math.Ceil(step-1e-4))-1)
}

func (v *Voice) NoteOff() {
	if v.released {
		return
	}
	v.released = true
	v.noteOff = 0
	for i := range v.layers {
		l := &v.layers[i]
		if !l.active || l.r.Trigger != OnAttack {
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
	for i := range v.layers {
		if l := &v.layers[i]; l.active {
			l.render(channels, v.waitSamps, n, v.released)
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
func (l *layer) render(channels psalm.Channels, from, to int, released bool) {
	r := l.r
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

	for i := from; i < to; i++ {
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
		e := l.env.next()
		if l.filterOn {
			xl = l.filters[0].Process(xl, l.filterMod)
			if snd.NumChannels() > 1 {
				xr = l.filters[1].Process(xr, l.filterMod)
			} else {
				xr = xl
			}
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
