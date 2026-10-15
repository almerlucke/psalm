// Package organ is a tonewheel organ: one manual of 91 wheels spinning all the time, which the
// keys tap at the pitches of the nine drawbars. Like the real one it is a whole instrument, not a
// set of voices: keys sharing a wheel share its phase, the percussion sounds on the first key of a
// phrase only (single trigger), and the vibrato scanner moves the whole sound. Put a rotary speaker
// after it (modules/effects/rotary) for the full sound.
//
//	org := organ.New(organ.Jazz()).AddTo(p)
package organ

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
)

// Percussion is the harmonic the percussion adds to the first key of a phrase
type Percussion int

const (
	PercOff    Percussion = iota
	PercSecond            // the 4' an octave up
	PercThird             // the 2 2/3' an octave and a fifth up
)

// Vibrato is the setting of the vibrato scanner: vibrato (V) or chorus (C, the vibrato mixed with
// the dry sound), 1 to 3 deep
type Vibrato int

const (
	VibratoOff Vibrato = iota
	V1
	V2
	V3
	C1
	C2
	C3
)

// Config is the registration of the organ
type Config struct {
	NumChannels int
	// Drawbars are the levels 0-8 of the footages 16', 5 1/3', 8', 4', 2 2/3', 2', 1 3/5', 1 1/3'
	// and 1'; each step is 3 dB
	Drawbars   [9]float64
	Percussion Percussion
	PercFast   bool    // a short percussion decay instead of a long one
	PercSoft   bool    // the percussion 6 dB softer
	Vibrato    Vibrato //
	VibratoHz  float64 // the scanner speed, 0 is 6.9 Hz
	Click      float64 // 0-1, the click of the key contacts
	ClickRand  float64 // 0-1, how much the click differs from key to key (loudness and brightness)
	Leakage    float64 // 0-1, the faint crosstalk of other tonewheels into a key, different on every key press
	Drive      float64 // 0-1, the preamp driven into saturation
	Level      float64 // output level (0 is treated as 1)
	// Filter is one filter on the sum of the drawbars (an organ has no voices), its envelope
	// starting with the first key of a phrase, like the percussion (off unless its Type is set)
	Filter voicefilter.Config
	Mods   [modmatrix.Slots]modmatrix.Slot // the matrix: a source to a destination (see Dests) over an amount
	Ext    [modmatrix.Exts]float64         // the values 0-1 of the matrix's sources Ext 1 to 4 (cabled in on InExt1...)
}

// DefaultConfig is the first three drawbars out, the classic 888000000
func DefaultConfig() Config {
	return Config{NumChannels: psalm.Stereo, Drawbars: Bars("888000000"), Click: 0.5, Level: 1}
}

// Bars reads a registration the way organists write it, "888000000": a digit per drawbar
func Bars(s string) [9]float64 {
	var b [9]float64
	i := 0
	for _, c := range s {
		if c >= '0' && c <= '8' && i < 9 {
			b[i] = float64(c - '0')
			i++
		}
	}
	return b
}

// footages are the drawbar pitches in semitones from the key: 16' 5 1/3' 8' 4' 2 2/3' 2' 1 3/5'
// 1 1/3' 1'
var footages = [9]int{-12, 7, 0, 12, 19, 24, 28, 31, 36}

const (
	lowWheel  = 24 // the pitches of the wheels, MIDI: C1 to F#8
	numWheels = 91
	tableSize = 4096
	// the gain of the sum: a key with all drawbars out is about 4 times full scale
	busGain = 0.1
)

var sineTable = func() []float64 {
	t := make([]float64, tableSize+1)
	for i := range t {
		t[i] = math.Sin(2 * math.Pi * float64(i) / tableSize)
	}
	return t
}()

// wheelOf returns the wheel a key plays at a pitch: wheels beyond the ends fold back an octave
func wheelOf(pitch int) int {
	for pitch < lowWheel {
		pitch += 12
	}
	for pitch >= lowWheel+numWheels {
		pitch -= 12
	}
	return pitch - lowWheel
}

// key is a key held (or still letting go)
type key struct {
	pitch  int
	leak   [3]leak
	wheels [9]int
	perc   [2]int // the percussion wheels: second, third
	gain   float64
	target float64
	amp    float64
	notes  int // how many notes hold it
}

// leak is a wheel that bleeds into a key, and how much
type leak struct {
	wheel int
	amp   float64
}

// timer lets go of a key of a note of Note
type timer struct {
	pitch int
	left  int
}

type event struct {
	at    int // sample of the buffer
	pitch int
	amp   float64
	down  bool
	timed int // for a note of a duration: its length in samples
}

// Organ is the instrument
type Organ struct {
	*psalm.Module
	cfg       Config
	phase     [numWheels]float64
	inc       [numWheels]float64
	keys      []*key
	ids       map[string]int // notes on: their key
	events    []event
	timers    []timer
	bars      [9]float64 // the drawbar gains, gliding to their settings
	perc      float64    // the percussion envelope
	percDec   float64
	sustain   bool
	sustained map[int]int // keys the sustain pedal holds: how many notes
	click     struct{ env, lp, hp, noise, coef float64 }
	rng       uint32
	rng2      uint32 // for what differs from key to key, apart from the click's noise
	filter    *voicefilter.Filter
	mods      *modmatrix.Matrix
	lastPitch int
	velocity  float64
	random    float64
	delay     []float64
	dpos      int
	lfoPhase  float64
	silent    int                   // samples it has been silent: it rests once the scanner's delay is empty
	expr      expression.Expression // the pitch bend (all the wheels); the mod wheel brings in the scanner vibrato
}

// New returns an organ
func New(cfg Config) *Organ {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	o := &Organ{
		Module:    psalm.NewModule([]int{psalm.Mono}, []int{cfg.NumChannels}),
		cfg:       cfg,
		ids:       map[string]int{},
		sustained: map[int]int{},
		rng:       uint32(cfg.NumChannels*7919 + 12345),
		rng2:      uint32(cfg.NumChannels*104729 + 777),
		lastPitch: 60,
		delay:     make([]float64, 1024),
	}
	sr := o.Config().SampleRate
	for w := range numWheels {
		o.inc[w] = 440 * math.Exp2(float64(w+lowWheel-69)/12) / sr
		o.phase[w] = math.Mod(float64(w)*0.618034, 1) // the wheels are not in phase
	}
	o.bars = o.barGains()
	o.click.coef = 0.5
	o.filter = voicefilter.New(cfg.Filter, sr, 1)
	o.mods = modmatrix.New(Dests)
	for i, sl := range cfg.Mods {
		o.mods.Set(i, sl)
	}
	for i, e := range cfg.Ext {
		o.mods.SetExt(i, e)
	}
	o.SetSelf(o)
	return o
}

// barGains returns the gains of the drawbars: 3 dB a step, the 1' off with the percussion on
func (o *Organ) barGains() [9]float64 {
	var g [9]float64
	for i, b := range o.cfg.Drawbars {
		b = math.Max(0, math.Min(b, 8))
		if b > 0 {
			g[i] = math.Pow(10, -3*(8-b)/20)
		}
	}
	if o.cfg.Percussion != PercOff {
		g[8] = 0
	}
	return g
}

// AllNotesOff lets go of every key, and the sustain pedal
func (o *Organ) AllNotesOff() {
	o.sustain = false
	clear(o.sustained)
	clear(o.ids)
	o.events = o.events[:0]
	o.timers = o.timers[:0]
	for _, k := range o.keys {
		k.notes, k.target = 0, 0
	}
	o.filter.Release()
}

// random2 returns a random number from 0 to 1 that doesn't disturb the noise of the click
func (o *Organ) random2() float64 {
	o.rng2 = o.rng2*1664525 + 1013904223
	return float64(o.rng2>>8) / (1 << 24)
}

// clickOn adds the click of a key contact: its size varies with ClickRand, and so does its colour
func (o *Organ) clickOn(size float64) {
	r := o.cfg.ClickRand
	if r > 0 {
		size *= 1 + r*0.8*(o.random2()*2-1)
		o.click.coef = 0.5 + r*0.35*(o.random2()*2-1)
	} else {
		o.click.coef = 0.5
	}
	o.click.env += size
}

func (o *Organ) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		o.ReceiveMessage(msg)
		return
	}
	if index >= 1 && index <= 9 {
		o.cfg.Drawbars[index-1] = psalm.FltVal[float64](value)
	} else if index >= InExt1 && index < InExt1+modmatrix.Exts {
		o.mods.SetExt(index-InExt1, psalm.FltVal[float64](value))
		o.cfg.Ext[index-InExt1] = o.mods.Ext(index - InExt1)
	}
}

// ReceiveMessage plays the triggers of a poly synth (with a duration, or noteOn and noteOff) and
// takes the settings: "drawbars" ("888000000"), "drawbar1" to "drawbar9" (0-8), "percussion"
// (0 off, 1 second, 2 third), "percFast", "percSoft", "vibrato" (0 off, 1-3 V1-V3, 4-6 C1-C3),
// "vibratoHz", "click", "clickRand", "leakage", "drive", "level", the "sustain" pedal (above 0.5 is
// down), the filter (voicefilter: "filterType", "cutoff", "resonance", ...) and the matrix
// (modmatrix: "mod1Src", "mod1Dest", "mod1Amount", ..., "ext1" to "ext4")
func (o *Organ) ReceiveMessage(msg psalm.Message) {
	if command, ok := msg.StrVal("command"); ok && command == "trigger" {
		tick := msg.MustTickVal("tick")
		if id, ok := msg.StrVal("noteOff"); ok {
			if p, ok := o.ids[id]; ok {
				delete(o.ids, id)
				o.events = append(o.events, event{at: tick.SampleOffset, pitch: p})
			}
			return
		}
		amp := msg.MustFltVal("amplitude")
		note := msg.MustMsg("message")
		if id, ok := msg.StrVal("noteOn"); ok {
			p := pitchOf(note)
			o.ids[id] = p
			o.events = append(o.events, event{at: tick.SampleOffset, pitch: p, amp: amp, down: true})
			return
		}
		o.Note(msg.MustFltVal("duration"), amp, note, tick, o.Config())
		return
	} else if ok && command != "voice" {
		return
	}
	o.expr.Receive(msg)
	o.filter.Receive(msg)
	o.cfg.Filter = o.filter.Config()
	o.mods.Receive(msg)
	for i := range o.cfg.Mods {
		o.cfg.Mods[i] = o.mods.Slot(i)
	}
	for i := range o.cfg.Ext {
		o.cfg.Ext[i] = o.mods.Ext(i)
	}
	msg.IfStrVal("drawbars", func(s string) { o.cfg.Drawbars = Bars(s) })
	for i := range 9 {
		msg.IfFltVal("drawbar"+strconv.Itoa(i+1), func(f float64) { o.cfg.Drawbars[i] = f })
	}
	msg.IfIntVal("percussion", func(i int) { o.cfg.Percussion = Percussion(i) })
	msg.IfBoolVal("percFast", func(b bool) { o.cfg.PercFast = b })
	msg.IfBoolVal("percSoft", func(b bool) { o.cfg.PercSoft = b })
	msg.IfIntVal("vibrato", func(i int) { o.cfg.Vibrato = Vibrato(i) })
	msg.IfFltVal("vibratoHz", func(f float64) { o.cfg.VibratoHz = f })
	msg.IfFltVal("click", func(f float64) { o.cfg.Click = f })
	msg.IfFltVal("clickRand", func(f float64) { o.cfg.ClickRand = f })
	msg.IfFltVal("leakage", func(f float64) { o.cfg.Leakage = f })
	msg.IfFltVal("drive", func(f float64) { o.cfg.Drive = f })
	msg.IfFltVal("level", func(f float64) { o.cfg.Level = f })
	msg.IfFltVal("sustain", func(f float64) {
		o.sustain = f > 0.5
		if !o.sustain {
			for p, n := range o.sustained {
				for range n {
					o.events = append(o.events, event{pitch: p})
				}
			}
			clear(o.sustained)
		}
	})
}

// Note holds a key for a duration in seconds
func (o *Organ) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	o.events = append(o.events, event{at: tick.SampleOffset, pitch: pitchOf(message), amp: amplitude, down: true,
		timed: max(1, int(duration*o.Config().SampleRate))})
}

func pitchOf(note psalm.Message) int {
	pitch := 60.0
	note.IfFltVal("pitch", func(f float64) { pitch = f })
	note.IfFltVal("frequency", func(f float64) {
		if f > 0 {
			pitch = 69 + 12*math.Log2(f/440)
		}
	})
	return int(math.Round(pitch))
}

// apply plays an event: a key down or up
func (o *Organ) apply(e event) {
	var k *key
	for _, x := range o.keys {
		if x.pitch == e.pitch {
			k = x
		}
	}
	if !e.down {
		if k == nil || k.notes == 0 {
			return
		}
		if o.sustain {
			o.sustained[e.pitch]++
			return
		}
		k.notes--
		if k.notes == 0 {
			k.target = 0
			o.clickOn(o.clickLevel() * 0.5)
			if !o.held() {
				o.filter.Release()
			}
		}
		return
	}
	// single trigger: the percussion starts when no other key is down
	if !o.held() {
		o.perc = 1
		o.filter.Trigger(0, adsr.NoteOff)
	}
	o.lastPitch, o.velocity, o.random = e.pitch, e.amp, o.random2()*2-1
	if k == nil {
		k = &key{pitch: e.pitch}
		for i, f := range footages {
			k.wheels[i] = wheelOf(e.pitch + f)
		}
		k.perc = [2]int{wheelOf(e.pitch + 12), wheelOf(e.pitch + 19)}
		// the crosstalk: three wheels near the key's own, a little different every time
		for i := range k.leak {
			k.leak[i] = leak{wheel: int(o.random2() * numWheels), amp: (0.3 + 0.7*o.random2()) * (1 - 2*float64(i&1))}
		}
		o.keys = append(o.keys, k)
	}
	k.notes++
	k.target, k.amp = 1, e.amp
	if e.timed > 0 {
		o.timers = append(o.timers, timer{pitch: e.pitch, left: e.timed})
	}
	o.clickOn(o.clickLevel())
}

// held says whether any key is down
func (o *Organ) held() bool {
	for _, x := range o.keys {
		if x.notes > 0 {
			return true
		}
	}
	return false
}

// clickLevel is the size of the click of a key contact: the setting, and what the matrix adds
func (o *Organ) clickLevel() float64 {
	return math.Max(0, o.cfg.Click+o.mods.Mod(dClick))
}

func (o *Organ) Synthesize() {
	out := o.Output(0).Channels
	n := o.Config().BufferSize
	sr := o.Config().SampleRate
	cfg := &o.cfg

	// no key sounding, nothing left in the scanner: the wheels turn on, nothing else to do
	if len(o.keys) == 0 && len(o.events) == 0 && len(o.timers) == 0 && o.click.env <= 1e-5 && o.silent > len(o.delay) {
		for w := range numWheels {
			o.phase[w] += o.inc[w] * float64(n)
			o.phase[w] -= math.Floor(o.phase[w])
		}
		o.bars = o.barGains()
		for _, ch := range out {
			clear(ch)
		}
		return
	}
	if len(o.keys) == 0 && o.click.env <= 1e-5 {
		o.silent += n
	} else {
		o.silent = 0
	}

	// the drawbars glide to their settings over the buffer
	target := o.barGains()
	var barInc [9]float64
	for i := range 9 {
		barInc[i] = (target[i] - o.bars[i]) / float64(n)
	}
	mods := o.mods
	modded := mods.Active()
	percBase := 0.0
	percWheel := 0
	if cfg.Percussion != PercOff {
		percBase = 1.0
		if cfg.PercSoft {
			percBase = 0.5
		}
		if cfg.Percussion == PercThird {
			percWheel = 1
		}
	}
	percLevel := percBase
	filtered := o.filter.On()
	envUsed := modded && mods.Uses(modmatrix.FilterEnv)
	lfoUsed := modded && mods.Uses(modmatrix.LFO1)
	if modded {
		mods.Values[modmatrix.Velocity] = o.velocity
		mods.Values[modmatrix.Random] = o.random
		mods.Values[modmatrix.Key] = math.Max(-1, math.Min(float64(o.lastPitch-60)/60, 1))
	}
	lastFreq := 440 * math.Exp2(float64(o.lastPitch-69)/12)
	decay := 1.0
	if cfg.PercFast {
		decay = 0.3
	}
	percCoef := math.Exp(math.Log(1e-3) / (decay * sr))
	attack := 1 - math.Exp(-1/(0.0008*sr))
	release := 1 - math.Exp(-1/(0.003*sr))
	clickCoef := math.Exp(-1 / (0.002 * sr))
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	drive := 1 + math.Max(0, cfg.Drive)*8

	// the vibrato scanner
	depth := 0.0
	chorus := false
	if v := cfg.Vibrato; v != VibratoOff {
		depth = [...]float64{0, 0.00045, 0.0009, 0.00135}[(int(v)-1)%3+1] * sr
		chorus = v >= C1
	}
	// the mod wheel brings in the vibrato, up to V3, whatever the scanner is set to
	if wheel := o.expr.Wheel() * 0.00135 * sr; wheel > depth {
		depth, chorus = wheel, false
	}
	hz := cfg.VibratoHz
	if hz <= 0 {
		hz = 6.9
	}
	lfoInc := hz / sr

	slices.SortStableFunc(o.events, func(a, b event) int { return a.at - b.at })
	ev := 0
	for i := range n {
		for ev < len(o.events) && o.events[ev].at <= i {
			o.apply(o.events[ev])
			ev++
		}
		for t := 0; t < len(o.timers); {
			if o.timers[t].left--; o.timers[t].left > 0 {
				t++
				continue
			}
			o.apply(event{pitch: o.timers[t].pitch})
			o.timers = slices.Delete(o.timers, t, t+1)
		}
		o.expr.Tick(sr)
		bend := o.expr.Ratio()
		var leakage, vibMod, levelMod float64
		cutMod, resMod := 0.0, 0.0
		if modded {
			loud := 0.0
			for _, k := range o.keys {
				loud = math.Max(loud, k.gain*k.amp)
			}
			mv := &mods.Values
			mv[modmatrix.FilterEnv], mv[modmatrix.AmpEnv] = o.filter.Env(), math.Min(loud, 1)
			mv[modmatrix.LFO1] = math.Sin(2 * math.Pi * o.lfoPhase)
			mv[modmatrix.Wheel] = o.expr.Wheel()
			mods.Eval(sr)
			bend *= math.Exp2(mods.Mod(dPitch) / 12)
			leakage, vibMod, levelMod = mods.Mod(dLeakage), mods.Mod(dVibrato), mods.Mod(dLevel)
			cutMod, resMod = mods.Mod(dCutoff), mods.Mod(dResonance)
			percLevel = math.Max(0, percBase+mods.Mod(dPerc))
			drive = 1 + math.Max(0, cfg.Drive+mods.Mod(dDrive))*8
		}
		leakage += cfg.Leakage
		for w := range numWheels {
			o.phase[w] += o.inc[w] * bend
			if o.phase[w] >= 1 {
				o.phase[w] -= math.Floor(o.phase[w])
			}
		}
		for b := range 9 {
			o.bars[b] += barInc[b]
		}
		s := 0.0
		ps := 0.0
		for _, k := range o.keys {
			if k.target > k.gain {
				k.gain += (k.target - k.gain) * attack
			} else {
				k.gain += (k.target - k.gain) * release
			}
			g := k.gain * k.amp
			if g < 1e-7 {
				continue
			}
			ks := 0.0
			for b, w := range k.wheels {
				if o.bars[b] > 0 {
					ks += o.bars[b] * o.wheel(w)
				}
			}
			if leakage > 0 {
				for _, l := range k.leak {
					ks += leakage * 0.04 * l.amp * o.wheel(l.wheel)
				}
			}
			s += ks * g
			if percLevel > 0 {
				ps += o.wheel(k.perc[percWheel]) * g
			}
		}
		s += ps * o.perc * percLevel * 1.5
		o.perc *= percCoef

		// the key click: a burst of noise as the contacts close and open
		if o.click.env > 1e-5 {
			o.rng = o.rng*1664525 + 1013904223
			noise := float64(int32(o.rng)) / 2147483648
			o.click.lp += (noise - o.click.lp) * o.click.coef
			hp := o.click.lp - o.click.hp
			o.click.hp += hp * 0.05
			s += hp * o.click.env * 0.4
			o.click.env *= clickCoef
		}
		s *= busGain

		// the scanner: a delay swept by a sine, mixed with the dry sound for the chorus
		depthNow := depth
		if vibMod != 0 {
			depthNow = math.Max(0, depth+vibMod*0.00135*sr)
		}
		if depthNow <= 0 && lfoUsed { // the matrix's LFO is the scanner's, running without its vibrato
			if o.lfoPhase += lfoInc; o.lfoPhase >= 1 {
				o.lfoPhase--
			}
		}
		if depthNow > 0 {
			o.delay[o.dpos] = s
			d := 2 + depthNow*(0.5+0.5*math.Sin(2*math.Pi*o.lfoPhase))
			o.lfoPhase += lfoInc
			if o.lfoPhase >= 1 {
				o.lfoPhase--
			}
			pos := float64(o.dpos) - d
			if pos < 0 {
				pos += float64(len(o.delay))
			}
			j := int(pos)
			f := pos - float64(j)
			v := o.delay[j]*(1-f) + o.delay[(j+1)%len(o.delay)]*f
			o.dpos = (o.dpos + 1) % len(o.delay)
			if chorus {
				s = (s + v) * 0.5
			} else {
				s = v
			}
		}
		if drive > 1 {
			s = math.Tanh(s*drive) / drive
		}
		if filtered || envUsed {
			o.filter.Tick(lastFreq, cutMod, resMod)
		}
		if filtered {
			s = o.filter.Process(0, s)
		}
		lv := level
		if levelMod != 0 {
			lv = math.Max(0, level+levelMod)
		}
		val := float32(s * lv)
		for _, ch := range out {
			ch[i] = val
		}
	}
	o.events = o.events[:0]
	o.bars = target
	// keys that have let go and are silent leave
	keep := o.keys[:0]
	for _, k := range o.keys {
		if k.notes > 0 || k.gain > 1e-6 {
			keep = append(keep, k)
		}
	}
	clear(o.keys[len(keep):])
	o.keys = keep
}

// wheel returns the sound of a wheel now
func (o *Organ) wheel(w int) float64 {
	p := o.phase[w] * tableSize
	i := int(p)
	f := p - float64(i)
	return sineTable[i] + (sineTable[i+1]-sineTable[i])*f
}

// Registration returns the drawbars as organists write them, "888000000"
func (cfg Config) Registration() string {
	var b strings.Builder
	for _, d := range cfg.Drawbars {
		b.WriteByte(byte('0' + int(math.Round(math.Max(0, math.Min(d, 8))))))
	}
	return b.String()
}
