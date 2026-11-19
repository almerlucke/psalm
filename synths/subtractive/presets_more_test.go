package subtractive

import (
	"math"
	"math/cmplx"
	"sort"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

// morePresets are the presets of presets_more.go
var morePresets = func() map[string]func() Config {
	m := map[string]func() Config{}
	for _, n := range []string{"moogBass", "pulseBass", "growlBass", "bass808", "acidSquare", "dubWobble", "pluckBass", "wideReese", "monoLead", "squareLead", "pwmLead", "tranceLead", "portaLead", "screamLead", "sineLead", "brassLead", "warmPad", "glassPad", "stringMachine", "evolvingPad", "ahChoir", "vowelPad", "darkPad", "brightPad", "hugePad", "clavinet", "harpsichord", "wurli", "marimba", "pizzicato", "tubularBell", "analogSeq", "junoBrass", "jupiterBrass", "stringSection", "percOrgan", "rockOrgan", "unoOrgan", "droneFifth", "darkDrone", "gale", "laser", "sirenFx", "noiseTexture", "kick808", "clap", "rimShot", "cowbell", "junoPad", "obStab", "jupiterPad"} {
		m[n] = Presets[n]
	}
	return m
}()

// renderNote plays a note of the voice and returns the left channel: a held note of holdSeconds, then released
func renderNote(cfg Config, pitch, amp, holdSeconds, seconds float64, extra psalm.Message) []float64 {
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
	v := New(cfg)
	msg := psalm.Message{"pitch": pitch}
	for k, x := range extra {
		msg[k] = x
	}
	v.Note(holdSeconds, amp, msg, psalm.CtrlTick{}, v.Config())
	n := int(seconds * v.Config().SampleRate / float64(v.Config().BufferSize))
	out := make([]float64, 0, n*v.Config().BufferSize)
	for range n {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			out = append(out, float64(s))
		}
	}
	return out
}

func TestMorePresetsRender(t *testing.T) {
	for name, f := range morePresets {
		t.Run(name, func(t *testing.T) {
			for _, pitch := range []float64{33, 45, 57, 69, 84} {
				for _, amp := range []float64{0.25, 0.8, 1} {
					x := renderNote(f(), pitch, amp, 0.6, 1.2, nil)
					peak := 0.0
					for _, s := range x {
						if math.IsNaN(s) || math.IsInf(s, 0) {
							t.Fatalf("pitch %v amp %v: invalid sample", pitch, amp)
						}
						peak = max(peak, math.Abs(s))
					}
					if peak < 0.005 || peak > 4 {
						t.Fatalf("pitch %v amp %v: peak %v", pitch, amp, peak)
					}
				}
			}
			// a note held with NoteOn and released ends
			v := New(f())
			v.NoteOn(0.8, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
			for range 100 {
				v.Synthesize()
			}
			v.NoteOff()
			for range 3000 {
				v.Synthesize()
			}
			if v.IsActive() {
				t.Errorf("still sounding long after the release")
			}
		})
	}
}

// fft is a radix 2 fast Fourier transform
func fft(x []complex128) {
	n := len(x)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}
	for l := 2; l <= n; l <<= 1 {
		w := cmplx.Exp(complex(0, -2*math.Pi/float64(l)))
		for i := 0; i < n; i += l {
			wk := complex(1, 0)
			for k := 0; k < l/2; k++ {
				u, v := x[i+k], x[i+k+l/2]*wk
				x[i+k], x[i+k+l/2] = u+v, u-v
				wk *= w
			}
		}
	}
}

const (
	fpBands = 24
	fpSegs  = 8
)

// fingerprint describes how a sound sounds: the energy in 24 log spaced bands from 50 Hz to 16 kHz, in
// dB relative to the total; the energy over eight segments of time, the same way; the attack time and
// the spectral centroid
type fingerprint struct {
	bands    [fpBands]float64
	segs     [fpSegs]float64
	attackMs float64
	centroid float64
}

func makePrint(x []float64, sr float64) fingerprint {
	var fp fingerprint
	const n = 1 << 16
	buf := make([]complex128, n)
	for i := 0; i < n && i < len(x); i++ {
		h := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(n-1))
		buf[i] = complex(x[i]*h, 0)
	}
	fft(buf)
	var total, cw float64
	var e [fpBands]float64
	lo, hi := math.Log(50.0), math.Log(16000.0)
	for k := 1; k < n/2; k++ {
		f := float64(k) * sr / n
		p := real(buf[k])*real(buf[k]) + imag(buf[k])*imag(buf[k])
		total += p
		cw += p * f
		if f < 50 || f >= 16000 {
			continue
		}
		b := int((math.Log(f) - lo) / (hi - lo) * fpBands)
		e[b] += p
	}
	for b := range e {
		fp.bands[b] = 10 * math.Log10((e[b]+1e-12*total)/(total+1e-30))
	}
	fp.centroid = cw / (total + 1e-30)
	// time segments and attack
	seg := len(x) / fpSegs
	var se [fpSegs]float64
	var all float64
	for i, s := range x {
		if k := i / max(seg, 1); k < fpSegs {
			se[k] += s * s
		}
		all += s * s
	}
	for k := range se {
		fp.segs[k] = 10 * math.Log10((se[k]+1e-12*all)/(all+1e-30))
	}
	win := int(0.002 * sr)
	var rms []float64
	peak := 0.0
	for i := 0; i+win <= len(x); i += win {
		var s float64
		for _, v := range x[i : i+win] {
			s += v * v
		}
		r := math.Sqrt(s / float64(win))
		rms = append(rms, r)
		peak = max(peak, r)
	}
	for i, r := range rms {
		if r >= 0.9*peak {
			fp.attackMs = float64(i+1) * 2
			break
		}
	}
	return fp
}

// distance is how far apart two sounds are: the spectral and temporal shapes in dB, plus the octaves
// between their centroids and between their attack times
func (a fingerprint) distance(b fingerprint) float64 {
	var sb, ss float64
	for i := range a.bands {
		d := a.bands[i] - b.bands[i]
		sb += d * d
	}
	for i := range a.segs {
		d := a.segs[i] - b.segs[i]
		ss += d * d
	}
	return math.Sqrt(sb/fpBands) + 0.5*math.Sqrt(ss/fpSegs) +
		4*math.Abs(math.Log2(a.centroid/b.centroid)) + 2*math.Abs(math.Log2((a.attackMs+4)/(b.attackMs+4)))
}

// minDistinct is how far apart two presets sound at least
const minDistinct = 7

func TestPresetsAreDistinct(t *testing.T) {
	sr := psalm.CurrentEnv().Config().SampleRate
	var names []string
	for n := range Presets {
		names = append(names, n)
	}
	sort.Strings(names)
	prints := map[string]fingerprint{}
	for _, n := range names {
		prints[n] = makePrint(renderNote(Presets[n](), 57, 0.8, 1.0, 1.5, nil), sr)
	}
	type pair struct {
		a, b string
		d    float64
	}
	var pairs []pair
	for i, a := range names {
		for _, b := range names[i+1:] {
			pairs = append(pairs, pair{a, b, prints[a].distance(prints[b])})
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].d < pairs[j].d })
	for _, p := range pairs[:min(12, len(pairs))] {
		t.Logf("%-14s %-14s %.2f", p.a, p.b, p.d)
	}
	for _, p := range pairs {
		if p.d >= minDistinct {
			break
		}
		_, na := morePresets[p.a]
		_, nb := morePresets[p.b]
		if na || nb { // the older presets are left as they were
			t.Errorf("%s and %s sound alike (%.2f)", p.a, p.b, p.d)
		}
	}
}

// the presets that use the matrix sound different from the same without it, and the sources do their work
func TestMorePresetsMatrixResponds(t *testing.T) {
	sr := psalm.CurrentEnv().Config().SampleRate
	for name, f := range morePresets {
		cfg := f()
		var used []modmatrix.Source
		for _, s := range cfg.Mods {
			if s.Src != modmatrix.None && s.Dest != "" && s.Amount != 0 {
				used = append(used, s.Src)
			}
		}
		if len(used) == 0 {
			continue
		}
		t.Run(name, func(t *testing.T) {
			bare := cfg
			bare.Mods = [modmatrix.Slots]modmatrix.Slot{}
			with := makePrint(renderNote(cfg, 57, 0.7, 1.0, 1.5, psalm.Message{"modWheel": 0.0}), sr)
			without := makePrint(renderNote(bare, 57, 0.7, 1.0, 1.5, psalm.Message{"modWheel": 0.0}), sr)
			if d := with.distance(without); d < 0.1 {
				// a source that rests at zero (wheel) shows when it is moved
				moved := makePrint(renderNote(cfg, 57, 0.7, 1.0, 1.5, psalm.Message{"modWheel": 1.0}), sr)
				movedBare := makePrint(renderNote(bare, 57, 0.7, 1.0, 1.5, psalm.Message{"modWheel": 1.0}), sr)
				if d2 := moved.distance(movedBare); d2 < 0.1 {
					t.Errorf("the matrix changes nothing (%.3f, %.3f)", d, d2)
				}
			}
		})
	}
}

// the velocity and the wheel move the sound as the presets name them
func TestVelocityAndWheelSlots(t *testing.T) {
	sr := psalm.CurrentEnv().Config().SampleRate
	for _, tc := range []struct {
		name string
		src  modmatrix.Source
	}{{"moogBass", modmatrix.Velocity}, {"pulseBass", modmatrix.Velocity}, {"screamLead", modmatrix.Velocity},
		{"clavinet", modmatrix.Velocity}, {"growlBass", modmatrix.Wheel}, {"dubWobble", modmatrix.Wheel},
		{"percOrgan", modmatrix.Wheel}, {"wideReese", modmatrix.Wheel}} {
		cfg := morePresets[tc.name]()
		var lo, hi []float64
		if tc.src == modmatrix.Velocity {
			lo = renderNote(cfg, 57, 0.3, 1.0, 1.0, nil)
			hi = renderNote(cfg, 57, 1.0, 1.0, 1.0, nil)
		} else {
			lo = renderNote(cfg, 57, 0.8, 1.0, 1.0, psalm.Message{"modWheel": 0.0})
			hi = renderNote(cfg, 57, 0.8, 1.0, 1.0, psalm.Message{"modWheel": 1.0})
		}
		a, b := makePrint(lo, sr), makePrint(hi, sr)
		if tc.src == modmatrix.Velocity { // the loudness is not the point, the colour is
			if d := a.distance(b); d < 0.5 {
				t.Errorf("%s: the velocity doesn't change the colour (%.2f)", tc.name, d)
			}
			continue
		}
		if d := a.distance(b); d < 0.5 {
			t.Errorf("%s: the wheel doesn't change the sound (%.2f)", tc.name, d)
		}
	}
}

// the glided presets slide: the second note starts at the pitch of the first
func TestGlidingPresetsSlide(t *testing.T) {
	for _, name := range []string{"monoLead", "portaLead", "acidSquare", "brassLead", "sineLead"} {
		cfg := morePresets[name]()
		if cfg.Glide <= 0 {
			t.Errorf("%s has no glide", name)
		}
		v := New(cfg)
		v.NoteOn(0.8, psalm.Message{"pitch": 45.0}, psalm.CtrlTick{}, v.Config())
		for range 20 {
			v.Synthesize()
		}
		v.NoteOn(0.8, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
		v.Synthesize()
		if v.glideFreq > v.frequency*0.9 {
			t.Errorf("%s: the pitch jumped to %.1f Hz at once (target %.1f)", name, v.glideFreq, v.frequency)
		}
	}
}

// crossings estimates the pitch of a stretch by the rate of its sign changes
func crossings(x []float64, sr float64) float64 {
	n := 0
	for i := 1; i < len(x); i++ {
		if (x[i-1] < 0) != (x[i] < 0) {
			n++
		}
	}
	return float64(n) / 2 / (float64(len(x)) / sr)
}

// the kicks and the laser fall in pitch within the note
func TestFallingPresetsFall(t *testing.T) {
	sr := psalm.CurrentEnv().Config().SampleRate
	for _, name := range []string{"kick808", "bass808", "laser"} {
		x := renderNote(morePresets[name](), 45, 1, 1.0, 1.0, nil)
		early, late := crossings(x[int(0.002*sr):int(0.04*sr)], sr), crossings(x[int(0.3*sr):int(0.4*sr)], sr)
		if early < late*1.3 {
			t.Errorf("%s: %.0f Hz at the start, %.0f Hz later", name, early, late)
		}
	}
}

func TestPresetVoices(t *testing.T) {
	if n := Voices("unoOrgan", 12); n != 3 {
		t.Errorf("unoOrgan plays with %d voices, want 3", n)
	}
	if n := Voices("lead", 12); n != 12 {
		t.Errorf("lead plays with %d voices, want the default 12", n)
	}
	for name := range PresetVoices {
		if _, ok := Presets[name]; !ok {
			t.Errorf("PresetVoices has %q, not a preset", name)
		}
	}
}

// a voice the poly steals (cleared while it sounds) slides to its new note with the legato glide, a free one jumps
func TestGlideLegatoStolenVoice(t *testing.T) {
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
	cfg := UnoOrganPreset()
	play := func(v *Voice, pitch float64) {
		v.NoteOn(0.8, psalm.Message{"pitch": pitch}, psalm.CtrlTick{}, v.Config())
	}
	v := New(cfg)
	play(v, 57)
	for range 20 {
		v.Synthesize()
	}
	from := v.glideFreq
	v.Clear() // what the poly does to a stolen voice
	play(v, 69)
	if !(v.glideFreq < v.frequency && v.glideFreq == from) {
		t.Errorf("stolen voice starts at %v, want its old pitch %v (target %v)", v.glideFreq, from, v.frequency)
	}
	// a voice that had finished jumps
	w := New(cfg)
	play(w, 57)
	for range 20 {
		w.Synthesize()
	}
	w.NoteOff()
	for range 3000 {
		w.Synthesize()
	}
	w.Clear()
	play(w, 69)
	if w.glideFreq != w.frequency {
		t.Errorf("free voice glides from %v to %v, want a jump", w.glideFreq, w.frequency)
	}
}

// glideSeconds plays a voice at one pitch, moves it to another and says how long the pitch takes to get there (to the buffer; -1 if it never does)
func glideSeconds(mode GlideMode, glide, from, to float64) float64 {
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
	cfg := OrganPreset()
	cfg.Glide, cfg.GlideMode = glide, mode
	cfg.AmpEnv = adsrEnv(1, 10, 1, 50)
	v := New(cfg)
	v.NoteOn(0.8, psalm.Message{"pitch": from}, psalm.CtrlTick{}, v.Config())
	v.glideFreq = v.frequency
	v.SetPitch(to)
	buffer := float64(v.Config().BufferSize) / v.Config().SampleRate
	for i := 1; i < 4000; i++ {
		v.Synthesize()
		if v.glideFreq == v.frequency {
			return float64(i) * buffer
		}
	}
	return -1
}

func TestGlideModes(t *testing.T) {
	const glide = 400.0 // ms
	near := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 0.03 {
			t.Errorf("%s: %.3f s, want about %.3f s", name, got, want)
		}
	}
	// consistent time: an octave and two semitones take the same time
	a := glideSeconds(GlideTime, glide, 57, 69)
	b := glideSeconds(GlideTime, glide, 57, 59)
	near("time, an octave", a, 0.4)
	near("time, two semitones", b, 0.4)
	// consistent speed: the glide is the time of an octave, two semitones take a sixth of it
	c := glideSeconds(GlideRate, glide, 57, 69)
	d := glideSeconds(GlideRate, glide, 57, 59)
	e := glideSeconds(GlideRate, glide, 69, 57) // down as well
	near("rate, an octave", c, 0.4)
	near("rate, two semitones", d, 0.4/6)
	near("rate, an octave down", e, 0.4)
	// the exponential glide has not changed: it is still on its way after the time constant
	if x := glideSeconds(GlideExponential, glide, 57, 69); x != -1 && x < 0.8 {
		t.Errorf("the exponential glide arrived after %.2f s, it should take several time constants", x)
	}
}
