package dx

import (
	"math"
	"sort"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

// renderNote plays a note and returns the left channel: held for hold seconds, then released, for total seconds
func renderNote(p Patch, pitch, velocity, hold, total float64) []float64 {
	v := New(p, 2)
	v.NoteOn(velocity, psalm.Message{"pitch": pitch, "velocity": velocity}, psalm.CtrlTick{}, v.Config())
	var out []float64
	released := false
	for len(out) < int(total*sr) {
		if !released && len(out) >= int(hold*sr) {
			v.NoteOff()
			released = true
		}
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			out = append(out, float64(s))
		}
	}
	return out
}

func TestDXPresetsRender(t *testing.T) {
	if len(dxPresets) < 60 {
		t.Fatalf("%d presets, expected 60 or more", len(dxPresets))
	}
	for name, f := range dxPresets {
		if Presets[name] == nil {
			t.Errorf("%s is not in Presets", name)
		}
		p := f()
		if len(p.Name) == 0 || len(p.Name) > 10 {
			t.Errorf("%s: name %q is not 1 to 10 characters", name, p.Name)
		}
		if p.Level <= 0 {
			t.Errorf("%s: not calibrated", name)
		}
		for _, pitch := range []float64{36, 57, 72, 96} {
			for _, vel := range []float64{0.2, 0.8, 1} {
				out := renderNote(p, pitch, vel, 0.4, 0.9)
				peak := 0.0
				for _, s := range out {
					if math.IsNaN(s) || math.IsInf(s, 0) {
						t.Fatalf("%s pitch %v velocity %v: %v", name, pitch, vel, s)
					}
					peak = math.Max(peak, math.Abs(s))
				}
				if peak > 8 {
					t.Errorf("%s pitch %v velocity %v: peak %v", name, pitch, vel, peak)
				}
				if vel >= 0.8 && pitch <= 72 && peak < 0.02 {
					t.Errorf("%s pitch %v velocity %v: silent, peak %v", name, pitch, vel, peak)
				}
			}
		}
	}
}

// a held note ends after it is released
func TestDXPresetsRelease(t *testing.T) {
	for name, f := range dxPresets {
		v := New(f(), 2)
		v.NoteOn(0.8, psalm.Message{"pitch": 57.0, "velocity": 0.8}, psalm.CtrlTick{}, v.Config())
		for range 60 {
			v.Synthesize()
		}
		v.NoteOff()
		ended := false
		for range int(40*sr) / v.Config().BufferSize {
			v.Synthesize()
			if !v.IsActive() {
				ended = true
				break
			}
		}
		if !ended {
			t.Errorf("%s never ends after its release", name)
		}
	}
}

// the values of the DX7 survive a single voice dump: the 0-99 values of every preset, the waves,
// the filter, the glide and the matrix aren't part of it
func TestDXPresetsSysEx(t *testing.T) {
	for name, f := range dxPresets {
		p := f()
		parsed, err := ParseSysEx(EncodeSingle(p))
		if err != nil || len(parsed) != 1 {
			t.Fatalf("%s: %v", name, err)
		}
		q := parsed[0]
		want := p
		want.Level, want.Routing, want.Mods, want.Ext, want.LFO2Rate = 0, nil, [modmatrix.Slots]modmatrix.Slot{}, [modmatrix.Exts]float64{}, 0
		want.Glide, want.VibratoRate, want.VibratoDepth, want.VibratoDelay, want.FreeFeedback, want.ModDepth = 0, 0, 0, 0, 0, 0
		want.Filter, want.FilterSet = Filter{}, false
		for i := range want.Ops {
			want.Ops[i].Wave = 0
		}
		q.Level = 0
		if q != want {
			t.Errorf("%s differs after a round trip:\n%+v\n%+v", name, want, q)
		}
		if p.Name != q.Name {
			t.Errorf("%s: name %q came back as %q", name, p.Name, q.Name)
		}
		// and in a bank of 32
		bank, err := ParseSysEx(EncodeBank([]Patch{p}))
		if err != nil || bank[0].Ops[0].EGRates != p.Ops[0].EGRates || bank[0].Algorithm != p.Algorithm {
			t.Errorf("%s: bank %v", name, err)
		}
	}
}

// fingerprint is how a patch sounds: 24 bands over the note, how fast it comes in, where its weight is
// and how fast it fades
type fingerprint struct {
	bands    [24]float64 // dB below the loudest band
	attack   float64     // seconds to 90% of the peak
	centroid float64     // Hz
	decay    float64     // dB from the first 200 ms to the last 500 ms
}

func fingerprintOf(p Patch) fingerprint {
	out := renderNote(p, 57, 0.8, 1.2, 1.5)
	var fp fingerprint
	const n = 4096
	f := fft.New(n)
	w := fft.HannWindow(n)
	re, im := make([]float64, n), make([]float64, n)
	var energy [24]float64
	lo, hi := 80.0, 12000.0
	var cs, cw float64
	for start := 0; start+n <= len(out); start += n / 2 {
		for i := range n {
			re[i], im[i] = out[start+i]*w[i], 0
		}
		f.Forward(re, im)
		for k := 1; k < n/2; k++ {
			freq := float64(k) * sr / n
			if freq < lo || freq >= hi {
				continue
			}
			m := re[k]*re[k] + im[k]*im[k]
			b := int(24 * math.Log(freq/lo) / math.Log(hi/lo))
			energy[b] += m
			cs += freq * m
			cw += m
		}
	}
	top := 0.0
	for _, e := range energy {
		top = math.Max(top, e)
	}
	for i, e := range energy {
		fp.bands[i] = 10 * math.Log10((e+1e-12*top)/top)
	}
	fp.centroid = cs / math.Max(cw, 1e-30)
	// the envelope in blocks of 5 ms
	block := 220
	var env []float64
	peak := 0.0
	for i := 0; i+block <= len(out); i += block {
		e := 0.0
		for _, s := range out[i : i+block] {
			e += s * s
		}
		e = math.Sqrt(e / float64(block))
		env = append(env, e)
		peak = math.Max(peak, e)
	}
	for i, e := range env {
		if e >= 0.9*peak {
			fp.attack = float64(i) * 0.005
			break
		}
	}
	mean := func(a, b int) float64 {
		s := 0.0
		for _, e := range env[a:b] {
			s += e * e
		}
		return s / float64(b-a)
	}
	fp.decay = 10 * math.Log10((mean(len(env)*3/10-0, len(env)*3/10+40)+1e-12)/(mean(0, 40)+1e-12))
	return fp
}

// distance between two fingerprints, in about dB: the spectrum, plus what the envelope and the weight add
func (a fingerprint) distance(b fingerprint) float64 {
	var s float64
	for i := range a.bands {
		d := a.bands[i] - b.bands[i]
		s += d * d
	}
	spectral := math.Sqrt(s / 24)
	attack := 6 * math.Abs(math.Log2((a.attack+0.01)/(b.attack+0.01)))
	centroid := 6 * math.Abs(math.Log2(a.centroid/b.centroid))
	decay := 0.5 * math.Abs(a.decay-b.decay)
	return spectral + attack + centroid + decay
}

// no two patches of this file sound the same, and none sounds like a patch before it
func TestDXPresetsDistinct(t *testing.T) {
	var names []string
	for n := range dxPresets {
		names = append(names, n)
	}
	for n := range Presets {
		if _, ok := dxPresets[n]; !ok {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	prints := map[string]fingerprint{}
	for _, n := range names {
		prints[n] = fingerprintOf(Presets[n]())
	}
	const minDistance = 4.0
	type pair struct {
		a, b string
		d    float64
	}
	var pairs []pair
	for i, a := range names {
		for _, b := range names[i+1:] {
			_, ina := dxPresets[a]
			_, inb := dxPresets[b]
			if !ina && !inb {
				continue
			}
			pairs = append(pairs, pair{a, b, prints[a].distance(prints[b])})
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].d < pairs[j].d })
	for _, p := range pairs[:min(12, len(pairs))] {
		t.Logf("%-16s %-16s %.2f", p.a, p.b, p.d)
		if p.d < minDistance {
			t.Errorf("%s and %s sound alike: distance %.2f", p.a, p.b, p.d)
		}
	}
}

// the patches of the matrix and the filter respond to what they are made to follow
func TestDXPresetsRespond(t *testing.T) {
	roughness := func(p Patch) float64 {
		out := renderNote(p, 57, 0.8, 1, 1.2)
		var e, prev float64
		for _, s := range out {
			e += (s - prev) * (s - prev)
			prev = s
		}
		return e
	}
	energy := func(p Patch, vel float64) float64 {
		var e float64
		for _, s := range renderNote(p, 57, vel, 1, 1.2) {
			e += s * s
		}
		return e
	}
	// the wheel opens the modulators of wheelKeys and the filter of wheelPad: with the wheel up they
	// sound unlike the same patch without the matrix
	for _, name := range []string{"wheelKeys", "wheelPad"} {
		p := dxPresets[name]()
		with, without := wheelRoughness(p, 1), wheelRoughness(withoutMatrix(p), 1)
		if math.Abs(with-without) < without*0.02 {
			t.Errorf("%s: roughness %g with the wheel on its matrix, %g without", name, with, without)
		}
		if still, up := wheelRoughness(p, 0), wheelRoughness(p, 1); math.Abs(still-up) < still*0.02 {
			t.Errorf("%s: the wheel changes nothing (%g, %g)", name, still, up)
		}
	}
	// velocity opens velocityFM, and the filter of filterPluck is on
	p := dxPresets["velocityFM"]()
	if soft, hard := roughness(p), roughness(withoutMatrix(p)); math.Abs(soft-hard) < soft*0.02 {
		t.Errorf("velocityFM: roughness %g with the matrix, %g without", soft, hard)
	}
	for _, name := range []string{"acidBass", "filterPluck", "sweepStrings", "wheelPad", "riserSweep", "windGust", "hatFM"} {
		p := dxPresets[name]()
		if !p.FilterSet || p.Filter.Type == 0 {
			t.Errorf("%s has no filter", name)
			continue
		}
		open := p
		open.Filter.Type = 0
		if r, o := roughness(p), roughness(open); math.Abs(r-o) < o*0.02 {
			t.Errorf("%s: the filter changes nothing (%g, %g)", name, r, o)
		}
	}
	// the random patch plays another sound each note
	cm := dxPresets["chanceMallet"]()
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
	v := New(cm, 2)
	energies := map[int]bool{}
	for range 6 {
		v.NoteOn(0.8, psalm.Message{"pitch": 57.0, "velocity": 0.8}, psalm.CtrlTick{}, v.Config())
		var e float64
		for range 30 {
			v.Synthesize()
			for _, s := range v.Output(0).Channels[0] {
				e += float64(s) * float64(s)
			}
		}
		energies[int(e*1000)] = true
		v.NoteOff()
		for range 400 {
			v.Synthesize()
		}
	}
	if len(energies) < 3 {
		t.Errorf("chanceMallet sounds the same every note: %d different", len(energies))
	}
	// the glide and the vibrato are in glideLead, a velocity makes a difference to most
	gl := dxPresets["glideLead"]()
	if gl.Glide <= 0 || gl.VibratoDepth <= 0 {
		t.Errorf("glideLead: glide %v vibrato %v", gl.Glide, gl.VibratoDepth)
	}
	if a, b := energy(dxPresets["slapFM"](), 0.3), energy(dxPresets["slapFM"](), 1); a >= b {
		t.Errorf("slapFM is not louder when played harder: %g, %g", a, b)
	}
}

// wheelRoughness is the roughness of a note with the mod wheel at a position
func wheelRoughness(p Patch, wheel float64) float64 {
	v := New(p, 2)
	v.ReceiveMessage(psalm.Message{"modWheel": wheel})
	v.NoteOn(0.8, psalm.Message{"pitch": 57.0, "velocity": 0.8}, psalm.CtrlTick{}, v.Config())
	var e, prev float64
	for range 100 {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			e += (float64(s) - prev) * (float64(s) - prev)
			prev = float64(s)
		}
	}
	return e
}

func withoutMatrix(p Patch) Patch {
	p.Mods = [modmatrix.Slots]modmatrix.Slot{}
	return p
}
