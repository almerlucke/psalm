// Package sampler plays multi-sampled instruments: recordings of an instrument spread over its keys
// and played at several strengths. Each recording is a region: the keys and velocities it answers,
// the key it was recorded at, how it loops and how it fades. A note plays every region that
// matches it, so regions can be layered, and alternate recordings of the same note can take turns
// (round robin) or be picked at random.
//
// Instruments can be built in code or loaded from SFZ, the plain text format most free sample
// libraries come in:
//
//	inst, err := sampler.LoadSFZ("Salamander/piano.sfz")
//	piano := sampler.NewPoly(sampler.Config{Instrument: inst}, 32).AddTo(p)
package sampler

import (
	"fmt"
	"math"
	"slices"
)

// Sound is the audio of a region: deinterleaved channels, with optional band limited copies for
// playing it faster (depth 0 is the sound itself). Every sndfile.SoundFiler[float32] is a Sound;
// NewSound makes one from samples in memory
type Sound interface {
	NumChannels() int
	SampleRate() float64
	NumFrames() int64
	Depth() int
	Buffer(channel int, depth int) []float32
}

// memSound is a Sound in memory
type memSound struct {
	sr       float64
	channels [][]float32
}

// NewSound returns a sound of channels (all the same length) recorded at a sample rate
func NewSound(sampleRate float64, channels ...[]float32) Sound {
	return &memSound{sr: sampleRate, channels: channels}
}

func (s *memSound) NumChannels() int    { return len(s.channels) }
func (s *memSound) SampleRate() float64 { return s.sr }
func (s *memSound) Depth() int          { return 1 }
func (s *memSound) Buffer(channel int, _ int) []float32 {
	return s.channels[channel]
}
func (s *memSound) NumFrames() int64 {
	if len(s.channels) == 0 {
		return 0
	}
	return int64(len(s.channels[0]))
}

// LoopMode says how a region plays its sound
type LoopMode int

const (
	NoLoop         LoopMode = iota // plays the sound once, a note off fades it out with the release
	OneShot                        // plays the sound to its end, note offs are ignored (drums)
	LoopContinuous                 // loops between LoopStart and LoopEnd until the release has faded it out
	LoopSustain                    // loops while the key is down, then plays on to the end of the sound
)

// Trigger says when a region plays
type Trigger int

const (
	OnAttack  Trigger = iota // when the key goes down
	OnRelease                // when the key goes up (the damper of a piano, the release of a string)
	OnFirst                  // when the key goes down and no other key is held (the start of a phrase)
	OnLegato                 // when the key goes down while another is held (a slur, a legato transition)
)

// OffMode says how a region choked by another (OffBy) stops
type OffMode int

const (
	OffFast   OffMode = iota // in a few milliseconds
	OffNormal                // with its own release
)

// Envelope is the loudness of a region over a note, the times are in seconds and Sustain is a
// level 0-1. The decay and release fall exponentially, -60 dB over their time
type Envelope struct {
	Delay, Attack, Hold, Decay float64
	Sustain                    float64
	Release                    float64
}

// Region is a sound and the notes it plays. Start from NewRegion, which fills in the defaults
// (all keys, all velocities, root key 60, key tracking, full velocity tracking, a short release)
type Region struct {
	Sound Sound
	Name  string // the file of the sound, for messages

	LoKey, HiKey int // the keys it plays, MIDI 0-127
	LoVel, HiVel int // the velocities it plays, 1-127

	Root      float64 // the key the sound was recorded at
	KeyTrack  float64 // cents per key, 100 plays the sound in tune over the keys, 0 at one pitch (drums)
	Transpose float64 // semitones
	Tune      float64 // cents

	Volume   float64 // dB
	Amp      float64 // linear gain on top of Volume, 0-1 (SFZ amplitude)
	Pan      float64 // -1 (left) to 1 (right); a stereo sound is balanced
	VelTrack float64 // 0-1: how much the velocity sets the loudness, 0 plays every velocity as loud

	Offset int64 // the frame it starts from
	End    int64 // the last frame it plays, 0 is the end of the sound

	Loop          LoopMode
	LoopStart     int64   // the first frame of the loop
	LoopEnd       int64   // the last frame of the loop, 0 is the end of the sound
	LoopCrossfade float64 // seconds the end of the loop fades into its start, for smooth loops

	Trigger Trigger
	RtDecay float64 // dB per second a release region is softer the longer the key was down

	Group   int     // a choke group, 0 is none
	OffBy   int     // a region of this group starting stops this one (an open hihat by a closed one)
	OffMode OffMode // how it stops

	SeqLength   int     // round robin: the region plays every SeqLength-th time its notes play...
	SeqPosition int     // ...at this turn, 1 to SeqLength
	LoRand      float64 // random choice: plays when a random number for the note is in [LoRand, HiRand)
	HiRand      float64

	Env Envelope

	Cutoff      float64    // Hz, a filter on the region, 0 is none
	Resonance   float64    // dB the filter peaks at its cutoff, 0 is none
	Filter      FilterType // the kind of filter
	FilVelTrack float64    // cents the cutoff rises at full velocity

	// Keyswitches: the keys SwLoKey to SwHiKey choose an articulation instead of playing. The region
	// plays only while the last of them pressed is SwLast (-1: whichever), SwDefault (-1 none) is
	// the one chosen before any is pressed, SwLabel its name
	SwLoKey, SwHiKey int
	SwLast           int
	SwDefault        int
	SwLabel          string

	Fades  []Fade   // crossfades by velocity, key or a controller
	CCGain []CCGain // dB a controller adds at its top (gain_ccN)
	CCEnv  []CCEnv  // seconds (sustain: 0-1) a controller adds to the envelope at its top
	// EnvDynamic makes the envelope follow the controllers while the note sounds (SFZ
	// ampeg_dynamic), instead of taking them when it starts
	EnvDynamic bool

	// Every note a little different: up to half these louder or softer (dB), sharper or flatter
	// (cents), and up to DelayRandom seconds later
	AmpRandom, PitchRandom, DelayRandom float64
	Delay                               float64  // seconds the region starts after the note
	VelEnv                              Envelope // added to the envelope at full velocity (seconds; sustain 0-1)
	LFOs                                []LFO    // vibrato, tremolo
	EQ                                  []EQBand // peaking filters
	Width                               float64  // a stereo sound's width in percent: 100 as it is, 0 mono, -100 swapped
	BendUp, BendDown                    float64  // cents the pitch bend moves it all the way up, down (SFZ: 200, -200)
	EndFade                             float64  // seconds the sound fades out in a straight line before its end (frame End), so a piece cut out of a recording does not click; 0: it just ends
	// Polyphony is how many notes of its group (of the region itself without one) sound at once,
	// NotePolyphony how many of the same key: past it the oldest let go, quickly (0 is no limit)
	Polyphony, NotePolyphony int
}

// LFOWave is the shape of an LFO
type LFOWave int

const (
	LFOTriangle LFOWave = iota
	LFOSine
	LFOPulse75 // high three quarters of the cycle
	LFOSquare
	LFOPulse25
	LFOPulse12
	LFOSawUp
	LFOSawDown
)

// LFO is a slow oscillator moving the pitch (vibrato) or the level (tremolo) of a region
type LFO struct {
	Freq   float64 // Hz
	Wave   LFOWave
	Delay  float64 // seconds before it starts
	Fade   float64 // seconds it takes to grow to its depth after the delay
	Pitch  float64 // cents up and down
	Volume float64 // dB up and down
}

// EQBand is a peaking filter: a gain around a frequency, over a bandwidth in octaves
type EQBand struct {
	Freq      float64
	Bandwidth float64
	Gain      float64 // dB
}

// Source is what a crossfade follows: the velocity, the key, or a controller (0-127)
type Source int

const (
	FromVelocity Source = -1
	FromKey      Source = -2
)

// Fade fades a region in (from Lo to Hi) or out (Out) over a source: the velocity (1-127), the key
// or a controller's value (0-127). Fades by velocity and key are set when the note starts, by a
// controller they follow it while the note sounds (the mod wheel blending the dynamic layers)
type Fade struct {
	From   Source
	Lo, Hi float64
	Out    bool
	Linear bool // the gain follows the source in a straight line (SFZ xf_*curve=gain), not as power
}

// gain returns the gain of the fade at a value
func (f Fade) gain(v float64) float64 {
	t := 1.0
	switch {
	case v <= f.Lo:
		t = 0
	case v < f.Hi:
		t = (v - f.Lo) / (f.Hi - f.Lo)
	}
	if f.Out {
		t = 1 - t
	}
	if !f.Linear {
		t = math.Sqrt(t)
	}
	return t
}

// CCGain is the dB a controller adds at its top, in proportion below
type CCGain struct {
	CC int
	DB float64
}

// CCEnv is what a controller adds to the envelope at its top, in proportion below, when a note starts
type CCEnv struct {
	CC  int
	Env Envelope
}

// FilterType is the filter of a region
type FilterType int

const (
	LowPass FilterType = iota
	HighPass
	BandPass
)

// NewRegion returns a region playing a sound on every key and velocity, in tune around key 60
func NewRegion(sound Sound) Region {
	return Region{
		Sound:    sound,
		LoKey:    0,
		HiKey:    127,
		LoVel:    1,
		HiVel:    127,
		Root:     60,
		KeyTrack: 100,
		Amp:      1,
		VelTrack: 1,
		HiRand:   1,
		Env:      Envelope{Sustain: 1, Release: 0.05},
		SwLoKey:  -1, SwHiKey: -1, SwLast: -1, SwDefault: -1,
		Width: 100, BendUp: 200, BendDown: -200,
	}
}

// Keys returns the region for the keys lo to hi, recorded at root
func (r Region) Keys(lo, hi int, root float64) Region {
	r.LoKey, r.HiKey, r.Root = lo, hi, root
	return r
}

// Velocities returns the region for the velocities lo to hi (1-127)
func (r Region) Velocities(lo, hi int) Region {
	r.LoVel, r.HiVel = lo, hi
	return r
}

// Instrument is the regions of a multi-sampled instrument. It doesn't change once built, so one
// instrument can play in many synths
type Instrument struct {
	Regions []Region
	// Unsupported lists the SFZ opcodes the file uses that the sampler doesn't know (they are left
	// out), with how often
	Unsupported map[string]int
	// CC are the values (0-127) the controllers start at (SFZ set_ccN), the others start at 0
	CC map[int]float64
	// The sustain pedal: the controller it is (0 is 64), the value from which it is down (0 is 1,
	// as SFZ has it: only 0 is up), or none at all (NoSustain: the keyboard holds the notes)
	SustainCC int
	SustainLo float64
	NoSustain bool
}

// pedal returns the controller of the sustain pedal and the value from which it is down
func (inst *Instrument) pedal() (int, float64) {
	cc, lo := inst.SustainCC, inst.SustainLo
	if cc <= 0 || cc > 127 {
		cc = 64
	}
	if lo <= 0 {
		lo = 1
	}
	return cc, lo
}

// Switch is a keyswitch: its key and its name
type Switch struct {
	Key   int
	Label string
}

// Switches returns the keyswitches of the instrument (the articulations), lowest key first, and
// the one chosen at the start (-1 none)
func (inst *Instrument) Switches() ([]Switch, int) {
	seen := map[int]string{}
	def := -1
	for i := range inst.Regions {
		r := &inst.Regions[i]
		if r.SwLast >= 0 && (seen[r.SwLast] == "" || r.SwLabel != "") {
			seen[r.SwLast] = r.SwLabel
		}
		if def < 0 && r.SwDefault >= 0 {
			def = r.SwDefault
		}
	}
	var out []Switch
	for k, l := range seen {
		out = append(out, Switch{k, l})
	}
	slices.SortFunc(out, func(a, b Switch) int { return a.Key - b.Key })
	return out, def
}

// isSwitch says whether a key is a keyswitch
func (inst *Instrument) isSwitch(key int) bool {
	for i := range inst.Regions {
		if r := &inst.Regions[i]; r.SwLoKey >= 0 && key >= r.SwLoKey && key <= r.SwHiKey {
			return true
		}
	}
	return false
}

// NewInstrument checks the regions and returns them as an instrument
func NewInstrument(regions ...Region) (*Instrument, error) {
	for i := range regions {
		if err := regions[i].check(); err != nil {
			return nil, err
		}
	}
	return &Instrument{Regions: regions}, nil
}

// MustInstrument is NewInstrument that panics on a bad region
func MustInstrument(regions ...Region) *Instrument {
	inst, err := NewInstrument(regions...)
	if err != nil {
		panic(err)
	}
	return inst
}

// check fixes what can be fixed and says what can't
func (r *Region) check() error {
	name := r.Name
	if name == "" {
		name = "a region"
	}
	if r.Sound == nil || r.Sound.NumFrames() == 0 || r.Sound.NumChannels() == 0 {
		return fmt.Errorf("sampler: %s has no sound", name)
	}
	n := r.Sound.NumFrames()
	if r.End <= 0 || r.End >= n {
		r.End = n - 1
	}
	if r.Offset < 0 || r.Offset > r.End {
		return fmt.Errorf("sampler: %s starts at frame %d, after its end %d", name, r.Offset, r.End)
	}
	if r.Loop == LoopContinuous || r.Loop == LoopSustain {
		if r.LoopEnd <= 0 || r.LoopEnd > r.End {
			r.LoopEnd = r.End
		}
		if r.LoopStart < 0 || r.LoopStart >= r.LoopEnd {
			return fmt.Errorf("sampler: %s loops from frame %d to %d", name, r.LoopStart, r.LoopEnd)
		}
		// the crossfade reads before the loop start, and can't be longer than the loop
		xf := r.LoopCrossfade * r.Sound.SampleRate()
		xf = math.Min(xf, math.Min(float64(r.LoopStart), float64(r.LoopEnd-r.LoopStart)))
		r.LoopCrossfade = math.Max(0, xf/r.Sound.SampleRate())
	}
	if r.SeqLength > 0 && (r.SeqPosition < 1 || r.SeqPosition > r.SeqLength) {
		return fmt.Errorf("sampler: %s has round robin turn %d of %d", name, r.SeqPosition, r.SeqLength)
	}
	if r.HiRand <= r.LoRand {
		return fmt.Errorf("sampler: %s plays for random numbers [%g, %g)", name, r.LoRand, r.HiRand)
	}
	return nil
}

// plays says whether the region answers a key and velocity (1-127)
func (r *Region) plays(key, vel int) bool {
	return key >= r.LoKey && key <= r.HiKey && vel >= r.LoVel && vel <= r.HiVel
}

// Keys returns the lowest and highest key the instrument plays (-1, -1 when none)
func (inst *Instrument) Keys() (lo, hi int) {
	lo, hi = -1, -1
	for i := range inst.Regions {
		r := &inst.Regions[i]
		if lo < 0 || r.LoKey < lo {
			lo = r.LoKey
		}
		if r.HiKey > hi {
			hi = r.HiKey
		}
	}
	return lo, hi
}
