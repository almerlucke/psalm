// Package render renders patches offline: to a sound file with its reverb tail, to stems (every
// mixer strip in its own file, in one pass), or with a profile of the time every module takes.
// Create the patch without an audio engine (psalm.New(nil)) and set psalm.Seed for renders that
// come out the same every time
package render

import (
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime/pprof"
	"slices"
	"strings"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/meter"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/sndfile/writer"
)

// Options set how long to render
type Options struct {
	Seconds float64 // length to render
	Tail    float64 // up to this many seconds more while the output still sounds (reverb tails), stopping once it is silent
	Silence float64 // dB below which the tail counts as silent (0 is -80)

	// FadeOut fades what is written to the files (Stems, ToFile) out over this many seconds before Seconds, along a raised cosine, so the piece ends in silence
	// (0 is no fade). The stems fade with the master
	FadeOut float64

	// Until keeps rendering after Seconds until it returns true (a playlist that is done: its
	// length isn't known before), then the tail. It is asked once per buffer, at most 10 minutes on
	Until func() bool
}

// Stats describe a render
type Stats struct {
	Seconds  float64       // audio rendered
	Elapsed  time.Duration // time it took
	Peak     float64       // dB, of the output
	RMS      float64       // dB, of the output
	Loudness float64       // LUFS integrated, of the output

	// the time every buffer took to make: the 99th percentile, the longest, and how many took longer than the buffer lasts
	// (live, those are dropouts: the average of a whole render hides them)
	P99, Worst time.Duration
	Late       int
	Buffers    int
	Budget     time.Duration // the length of a buffer
}

// Speed returns how many times faster than real time the render ran
func (s Stats) Speed() float64 {
	if s.Elapsed <= 0 {
		return math.Inf(1)
	}
	return s.Seconds / s.Elapsed.Seconds()
}

// Load returns the part of one CPU core playing the patch live would take
func (s Stats) Load() float64 {
	return 1 / s.Speed()
}

func (s Stats) String() string {
	ms := func(d time.Duration) float64 { return d.Seconds() * 1000 }
	return fmt.Sprintf("rendered %.1f s in %.1f s: %.1fx real time (%.0f%% of a core live), peak %.1f dB, rms %.1f dB, %.1f LUFS\n"+
		"buffers: 99%% under %.1f ms, the longest %.1f ms, %d of %d over the %.1f ms a buffer lasts",
		s.Seconds, s.Elapsed.Seconds(), s.Speed(), 100*s.Load(), s.Peak, s.RMS, s.Loudness,
		ms(s.P99), ms(s.Worst), s.Late, s.Buffers, ms(s.Budget))
}

// Run renders a patch offline, calling each (when not nil) with the output after every buffer
func Run(p *psalm.Psalm, opts Options, each func(out psalm.Channels)) Stats {
	var (
		cfg        = p.Config()
		bs         = cfg.BufferSize
		sr         = cfg.SampleRate
		buffers    = int(math.Ceil(opts.Seconds * sr / float64(bs)))
		maxTail    = int(math.Ceil(opts.Tail * sr / float64(bs)))
		quietNeed  = int(math.Ceil(0.25 * sr / float64(bs))) // a quarter second of silence ends the tail
		silence    = opts.Silence
		m          = meter.New()
		start      = time.Now()
		rendered   = 0
		quietCount = 0
		took       []time.Duration // every buffer
	)
	if silence == 0 {
		silence = -80
	}
	threshold := math.Pow(10, silence/20)

	tailFrom, waiting := buffers, opts.Until != nil
	limit := buffers + int(math.Ceil(600*sr/float64(bs)))
	for b := 0; b < tailFrom+maxTail; b++ {
		if waiting && b >= buffers {
			if opts.Until() || b >= limit {
				waiting, tailFrom = false, b
			} else {
				tailFrom = b + 1
			}
		}
		p.ResetRun()
		t0 := time.Now()
		p.Run()
		took = append(took, time.Since(t0))
		out := p.Output(0).Channels
		m.Input(0).Channels.Copy(out)
		m.Synthesize()
		if each != nil {
			each(out)
		}
		rendered++

		if b >= tailFrom && !waiting {
			peak := 0.0
			for _, ch := range out {
				for _, x := range ch {
					peak = math.Max(peak, math.Abs(float64(x)))
				}
			}
			if peak < threshold {
				quietCount++
				if quietCount >= quietNeed {
					break
				}
			} else {
				quietCount = 0
			}
		}
	}

	st := Stats{
		Seconds:  float64(rendered*bs) / sr,
		Elapsed:  time.Since(start),
		Peak:     m.Peak(),
		RMS:      m.RMS(),
		Loudness: m.Loudness().Integrated(),
		Buffers:  len(took),
		Budget:   time.Duration(float64(bs) / sr * float64(time.Second)),
	}
	if len(took) > 0 {
		for _, d := range took {
			if d > st.Budget {
				st.Late++
			}
		}
		slices.Sort(took)
		st.P99, st.Worst = took[len(took)*99/100], took[len(took)-1]
	}
	return st
}

// fileWriter writes stereo 32 bit float WAV files
type fileWriter struct {
	w   *writer.Writer
	buf [][]float32
}

func newFileWriter(path string, sr float64, bs int) (*fileWriter, error) {
	w, err := writer.New(path, writer.WAV, 2, sr, writer.NewChannelConverter[float32](bs, 2))
	if err != nil {
		return nil, err
	}
	return &fileWriter{w: w, buf: [][]float32{make([]float32, bs), make([]float32, bs)}}, nil
}

// write writes a buffer, mono plays on both sides; gain returns the gain for each frame of the buffer (nil is 1)
func (f *fileWriter) write(ch psalm.Channels, gain func(i int) float64) error {
	copy(f.buf[0], ch[0])
	if len(ch) > 1 {
		copy(f.buf[1], ch[1])
	} else {
		copy(f.buf[1], ch[0])
	}
	if gain != nil {
		for i := range f.buf[0] {
			g := float32(gain(i))
			f.buf[0][i] *= g
			f.buf[1][i] *= g
		}
	}
	return f.w.Write(f.buf, false)
}

// fadeGain is the gain for each frame of the buffer that begins at frame `from`: 1 before the fade, a raised cosine down to 0 at opts.Seconds, 0 after. nil
// when the buffer is not in the fade
func fadeGain(from, bufferSize int, sampleRate float64, opts Options) func(i int) float64 {
	if opts.FadeOut <= 0 {
		return nil
	}
	end, start := opts.Seconds*sampleRate, (opts.Seconds-opts.FadeOut)*sampleRate
	if float64(from+bufferSize) <= start {
		return nil
	}
	return func(i int) float64 {
		x := (float64(from+i) - start) / (end - start)
		switch {
		case x <= 0:
			return 1
		case x >= 1:
			return 0
		}
		return 0.5 * (1 + math.Cos(math.Pi*x))
	}
}

// ToFile renders a patch to a stereo 32 bit float WAV file
func ToFile(p *psalm.Psalm, path string, opts Options) (Stats, error) {
	return Stems(p, path, opts, nil)
}

// Stems renders a patch to a file (the master) and the output of every source module to its own
// file in the same pass, in a directory next to the master file named after it ("song.wav" puts
// the stems in "song-stems/"). Sources are named by their key, like "violins" becomes
// "song-stems/violins.wav"
func Stems(p *psalm.Psalm, path string, opts Options, sources map[string]psalm.Modular) (Stats, error) {
	var (
		cfg     = p.Config()
		master  *fileWriter
		writers = map[string]*fileWriter{}
		names   = slices.Sorted(maps.Keys(sources))
		err     error
	)
	if dir := filepath.Dir(path); dir != "" {
		if err = os.MkdirAll(dir, 0o755); err != nil {
			return Stats{}, err
		}
	}
	if master, err = newFileWriter(path, cfg.SampleRate, cfg.BufferSize); err != nil {
		return Stats{}, err
	}
	if len(sources) > 0 {
		dir := strings.TrimSuffix(path, filepath.Ext(path)) + "-stems"
		if err = os.MkdirAll(dir, 0o755); err != nil {
			return Stats{}, err
		}
		for _, name := range names {
			w, err := newFileWriter(filepath.Join(dir, fileName(name)+".wav"), cfg.SampleRate, cfg.BufferSize)
			if err != nil {
				return Stats{}, err
			}
			writers[name] = w
		}
	}

	var writeErr error
	frames := 0 // written so far
	stats := Run(p, opts, func(out psalm.Channels) {
		if writeErr != nil {
			return
		}
		gain := fadeGain(frames, cfg.BufferSize, cfg.SampleRate, opts)
		frames += cfg.BufferSize
		if writeErr = master.write(out, gain); writeErr != nil {
			return
		}
		for _, name := range names {
			if writeErr = writers[name].write(sources[name].Output(0).Channels, gain); writeErr != nil {
				return
			}
		}
	})

	for _, w := range writers {
		if err := w.w.Close(); err != nil && writeErr == nil {
			writeErr = err
		}
	}
	if err := master.w.Close(); err != nil && writeErr == nil {
		writeErr = err
	}
	return stats, writeErr
}

// MixerStems returns every strip of a mixer (channels, groups and bus returns) as stem sources:
// the strip outputs after volume and pan, so the stems of the channels add up to the mix
func MixerStems(mix *mixer.Mixer) map[string]psalm.Modular {
	sources := map[string]psalm.Modular{}
	for _, s := range mix.Strips() {
		sources[s.Name()] = s
	}
	return sources
}

// fileName makes a file name of a stem name
func fileName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		case r == ' ', r == '/', r == '.':
			return '-'
		}
		return -1
	}, name)
}

// ModuleTime is the time spent in one module, or all modules of a type
type ModuleTime struct {
	Name  string
	Count int
	Time  time.Duration
}

// Profile is the time spent per module during a render
type Profile struct {
	Stats   Stats
	Total   time.Duration // all modules
	ByType  []ModuleTime  // most time first
	Modules []ModuleTime  // most time first
}

// String reports the render and the modules taking the most time
func (pr Profile) String() string {
	var b strings.Builder
	fmt.Fprintln(&b, pr.Stats)
	share := func(d time.Duration) float64 { return 100 * d.Seconds() / pr.Stats.Elapsed.Seconds() }
	ms := func(d time.Duration) float64 { return d.Seconds() * 1000 }
	fmt.Fprintf(&b, "\n%-48s %5s %12s %7s\n", "module type", "count", "time", "share")
	for _, t := range pr.ByType {
		fmt.Fprintf(&b, "%-48s %5d %9.1f ms %6.1f%%\n", t.Name, t.Count, ms(t.Time), share(t.Time))
	}
	fmt.Fprintf(&b, "\n%-48s %18s %7s\n", "slowest modules", "time", "share")
	for _, t := range pr.Modules[:min(len(pr.Modules), 15)] {
		fmt.Fprintf(&b, "%-48s %15.1f ms %6.1f%%\n", t.Name, ms(t.Time), share(t.Time))
	}
	fmt.Fprintf(&b, "\n%-48s %15.1f ms %6.1f%%\n", "all modules", ms(pr.Total), share(pr.Total))
	return b.String()
}

// timed is a module that measures its time
type timed interface {
	psalm.Modular
	CPUTime() time.Duration
	ResetCPUTime()
}

// walk calls f for every module in a patch and its sub patches
func walk(mods []psalm.Modular, f func(m psalm.Modular)) {
	for _, m := range mods {
		f(m)
		if sub, ok := m.(interface{ Modules() []psalm.Modular }); ok {
			walk(sub.Modules(), f)
		}
	}
}

// RunProfiled renders a patch measuring the time every module spends synthesizing (its own work,
// not the modules feeding it). A poly synth includes its voices
func RunProfiled(p *psalm.Psalm, opts Options) Profile {
	walk(p.Modules(), func(m psalm.Modular) {
		if t, ok := m.(timed); ok {
			t.ResetCPUTime()
		}
	})
	psalm.SetProfiling(true)
	stats := Run(p, opts, nil)
	psalm.SetProfiling(false)

	pr := Profile{Stats: stats}
	types := map[string]*ModuleTime{}
	walk(p.Modules(), func(m psalm.Modular) {
		t, ok := m.(timed)
		if !ok || t.CPUTime() == 0 {
			return
		}
		typ := typeName(m)
		name := typ
		if id := m.ID(); id != "" {
			name = id + " (" + typ + ")"
		}
		pr.Modules = append(pr.Modules, ModuleTime{Name: name, Count: 1, Time: t.CPUTime()})
		if types[typ] == nil {
			types[typ] = &ModuleTime{Name: typ}
		}
		types[typ].Count++
		types[typ].Time += t.CPUTime()
		pr.Total += t.CPUTime()
	})
	for _, t := range types {
		pr.ByType = append(pr.ByType, *t)
	}
	byTime := func(a, b ModuleTime) int { return int(b.Time - a.Time) }
	slices.SortFunc(pr.ByType, byTime)
	slices.SortFunc(pr.Modules, byTime)
	return pr
}

// packagePath matches the package path in front of a type name
var packagePath = regexp.MustCompile(`[\w.-]+(/[\w.-]+)*/`)

// typeName returns a short type name like "fdnreverb.FDNReverb" or "poly.Poly[wavetable.Voice]"
func typeName(m any) string {
	return strings.ReplaceAll(packagePath.ReplaceAllString(fmt.Sprintf("%T", m), ""), "*", "")
}

// CPUProfile writes a Go CPU profile of f to a file, for go tool pprof
func CPUProfile(path string, f func()) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := pprof.StartCPUProfile(file); err != nil {
		return err
	}
	f()
	pprof.StopCPUProfile()
	return nil
}
