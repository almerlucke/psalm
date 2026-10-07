package piano

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/meter"
)

const sr = 44100.0

// play plays a note of a duration (and maybe a sustain pedal), returns the mono output and
// whether the voice still sounds at the end
func play(cfg Config, msg psalm.Message, duration, seconds float64) ([]float64, bool) {
	v := New(cfg)
	v.Note(duration, 1, msg, psalm.CtrlTick{}, v.Config())
	var out []float64
	for range int(seconds * sr / float64(v.Config().BufferSize)) {
		v.Synthesize()
		ch := v.Output(0).Channels
		for i := range ch[0] {
			out = append(out, float64(ch[0][i])+float64(ch[1][i]))
		}
	}
	return out, v.IsActive()
}

// peakFrequency finds the strongest frequency within span (relative) of f
func peakFrequency(x []float64, f, span float64) float64 {
	n := len(x)
	best, bestF := 0.0, f
	for fq := f * (1 - span); fq <= f*(1+span); fq += f * 0.0002 {
		var s complex128
		rot := cmplx.Exp(complex(0, -2*math.Pi*fq/sr))
		ph := complex(1, 0)
		for i, v := range x {
			s += complex(v*(0.5-0.5*math.Cos(2*math.Pi*float64(i)/float64(n))), 0) * ph
			ph *= rot
		}
		if a := cmplx.Abs(s); a > best {
			best, bestF = a, fq
		}
	}
	return bestF
}

func rmsDB(x []float64, from, to float64) float64 {
	s := 0.0
	seg := x[int(from*sr):min(int(to*sr), len(x))]
	for _, v := range seg {
		s += v * v
	}
	return 10 * math.Log10(s/float64(len(seg))+1e-30)
}

// grand is the grand without variation, so renders can be compared
func grand() Config {
	cfg := GrandPreset()
	cfg.Variation = 0
	return cfg
}

func single() Config {
	cfg := grand()
	cfg.Strings = 1
	return cfg
}

func TestTuning(t *testing.T) {
	for _, p := range []float64{33, 57, 81} {
		x, _ := play(single(), psalm.Message{"pitch": p}, 2, 1.5)
		cfg := single()
		f0 := 440 * math.Exp2((stretched(p, &cfg)-69)/12) // tuned with the stretch
		f := peakFrequency(x[int(0.3*sr):int(1.3*sr)], f0, 0.01)
		if c := 1200 * math.Log2(f/f0); math.Abs(c) > 1.5 {
			t.Errorf("key %v is %.1f cents out of tune", p, c)
		}
	}
}

// The partials of a stiff string are stretched: the 7th of A2 sits about 6 cents sharp
func TestInharmonicity(t *testing.T) {
	p := 45.0
	x, _ := play(single(), psalm.Message{"pitch": p}, 2, 1.5)
	f0 := 440 * math.Exp2((p-69)/12)
	b := inharmonicity(p)
	seg := x[int(0.3*sr):int(1.3*sr)]
	f1 := peakFrequency(seg, f0, 0.01)
	f7 := peakFrequency(seg, 7*f1, 0.01)
	got := 1200 * math.Log2(f7/(7*f1))
	want := 600 * math.Log2((1+49*b)/(1+b))
	if math.Abs(got-want) > 3 || got < 3 {
		t.Fatalf("partial 7 is %.1f cents sharp, expected %.1f", got, want)
	}
}

// Released keys are damped, the sustain pedal and the keys without dampers ring on
func TestDampers(t *testing.T) {
	held, _ := play(grand(), psalm.Message{"pitch": 60.0}, 3, 1.5)
	released, _ := play(grand(), psalm.Message{"pitch": 60.0}, 0.3, 1.5)
	pedal, _ := play(grand(), psalm.Message{"pitch": 60.0, "sustain": 1.0}, 0.3, 1.5)
	if d := rmsDB(held, 1, 1.5) - rmsDB(released, 1, 1.5); d < 30 {
		t.Errorf("a released key should be damped, only %.1f dB quieter", d)
	}
	if d := math.Abs(rmsDB(held, 1, 1.5) - rmsDB(pedal, 1, 1.5)); d > 1 {
		t.Errorf("with the pedal down a released key should ring on, %.1f dB apart", d)
	}
	topHeld, _ := play(grand(), psalm.Message{"pitch": 100.0}, 3, 1)
	topReleased, _ := play(grand(), psalm.Message{"pitch": 100.0}, 0.1, 1)
	if d := math.Abs(rmsDB(topHeld, 0.5, 1) - rmsDB(topReleased, 0.5, 1)); d > 1 {
		t.Errorf("the top keys have no dampers, %.1f dB apart", d)
	}
}

// Harder strikes are brighter (and louder)
func TestVelocity(t *testing.T) {
	brightness := func(vel float64) (float64, float64) {
		x, _ := play(grand(), psalm.Message{"pitch": 60.0, "velocity": vel}, 1, 0.5)
		d := make([]float64, len(x)-1)
		for i := range d {
			d[i] = x[i+1] - x[i]
		}
		return rmsDB(d, 0, 0.4) - rmsDB(x, 0, 0.4), rmsDB(x, 0, 0.4)
	}
	softB, softL := brightness(0.2)
	hardB, hardL := brightness(1)
	if hardB < softB+3 || hardL < softL+6 {
		t.Fatalf("hard should be brighter and louder: brightness %.1f vs %.1f, level %.1f vs %.1f", hardB, softB, hardL, softL)
	}
}

// The keys sound like a recorded concert grand (the level of the first second at mezzo forte):
// about equally loud up to C5, a few dB softer from C6, the top octave some 13-16 dB softer
func TestKeysEven(t *testing.T) {
	level := func(p float64) float64 {
		x := playPiano(grand(), []psalm.Message{{"pitch": p, "velocity": 0.6}}, 2, 1.1, nil)
		return rmsDB(x, 0.003, 1.003)
	}
	ref := level(60)
	for p := 21.0; p <= 108; p += 7 {
		d := level(p) - ref
		switch {
		case p < 78 && math.Abs(d) > 5.5:
			t.Errorf("key %v is %.1f dB off", p, d)
		case p >= 78 && p < 94 && (d > 2 || d < -10):
			t.Errorf("key %v should be a few dB softer, %.1f dB", p, d)
		case p >= 94 && (d > -8 || d < -19):
			t.Errorf("key %v should be about 13-16 dB softer, %.1f dB", p, d)
		}
	}
}

// Every preset on every key, played hard: stable, and the voice rests once it has died away
func TestStableAndRests(t *testing.T) {
	for name, preset := range Presets {
		for p := 21.0; p <= 108; p += 5 {
			x, active := play(preset(), psalm.Message{"pitch": p, "velocity": 1.0}, 0.2, 4)
			for _, v := range x {
				if math.IsNaN(v) || math.Abs(v) > 20 { // loud when played hard, but bounded
					t.Fatalf("%s key %v: unstable %v", name, p, v)
				}
			}
			if active && p <= highestDamper {
				t.Errorf("%s key %v: still sounding 4 seconds after the release", name, p)
			}
		}
	}
}

// playPiano plays notes on the whole instrument, returns the output (the mean of the sides)
func playPiano(cfg Config, notes []psalm.Message, duration, seconds float64, setup psalm.Message) []float64 {
	p := NewPoly(cfg, 16)
	if setup != nil {
		p.ReceiveMessage(setup)
	}
	for _, m := range notes {
		p.Note(duration, 1, m, psalm.CtrlTick{}, p.Config())
	}
	var out []float64
	for range int(seconds*sr/float64(p.Config().BufferSize)) + 1 {
		p.Synthesize()
		ch := p.Output(0).Channels
		for i := range ch[0] {
			out = append(out, (float64(ch[0][i])+float64(ch[1][i]))/2)
		}
	}
	return out
}

var cMajor = []psalm.Message{{"pitch": 60.0}, {"pitch": 64.0}, {"pitch": 67.0}}

// With the sustain pedal down the other strings ring along: a halo that blooms as the notes die
// away, without it only the undamped top strings do, very quietly
func TestSympatheticResonance(t *testing.T) {
	without := grand()
	without.Resonance = 0
	halo := func(setup psalm.Message) (float64, float64) {
		a := playPiano(grand(), cMajor, 0.3, 3, setup)
		b := playPiano(without, cMajor, 0.3, 3, setup)
		d := make([]float64, len(a))
		for i := range a {
			d[i] = a[i] - b[i]
		}
		return rmsDB(d, 0, 0.5) - rmsDB(b, 0, 0.5), rmsDB(d, 2, 3) - rmsDB(b, 2, 3)
	}
	early, late := halo(psalm.Message{"sustain": 1.0})
	if early > -15 || late < -20 || late > 0 {
		t.Errorf("with the pedal the halo should start quiet and bloom: %.1f dB, then %.1f dB", early, late)
	}
	if early, _ := halo(nil); early > -30 {
		t.Errorf("without the pedal only the top strings ring along, %.1f dB", early)
	}
}

// A key held down silently rings along with a struck note it shares partials with
func TestSilentKey(t *testing.T) {
	ring := func(silent bool) float64 {
		p := NewPoly(grand(), 8)
		if silent {
			p.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "amplitude": 0.0, "message": psalm.Message{"pitch": 48.0}, "noteOn": "c3"})
		}
		p.Note(0.2, 1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, p.Config()) // C4: the second partial of C3
		var out []float64
		for range 87 {
			p.Synthesize()
			out = append(out, toFloat(p.Output(0).Channels[0])...)
		}
		return rmsDB(out, 1, 1.9)
	}
	// (above what the undamped top strings ring along anyway)
	if d := ring(true) - ring(false); d < 4 {
		t.Fatalf("the silently held C3 should ring on after the C4 is damped, %.1f dB", d)
	}
}

// The soft pedal: quieter and darker
func TestSoftPedal(t *testing.T) {
	brightness := func(x []float64) float64 {
		d := make([]float64, len(x)-1)
		for i := range d {
			d[i] = x[i+1] - x[i]
		}
		return rmsDB(d, 0, 0.5) - rmsDB(x, 0, 0.5)
	}
	normal := playPiano(grand(), cMajor[:1], 2, 1, nil)
	soft := playPiano(grand(), cMajor[:1], 2, 1, psalm.Message{"soft": 1.0})
	drop := rmsDB(normal, 0, 0.5) - rmsDB(soft, 0, 0.5)
	if drop < 2 || drop > 7 || brightness(soft) > brightness(normal)-1 {
		t.Fatalf("soft should be a few dB quieter (%.1f) and darker (%.1f vs %.1f)", drop, brightness(soft), brightness(normal))
	}
}

func toFloat(x []float32) []float64 {
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = float64(v)
	}
	return out
}

// A key struck again while it sounds is struck on its own strings: one voice, nothing cut off
func TestRestrike(t *testing.T) {
	p := NewPoly(grand(), 4)
	p.ReceiveMessage(psalm.Message{"sustain": 1.0})
	var out []float64
	for b := range 60 {
		if b%10 == 0 {
			p.Note(0.1, 1, psalm.Message{"pitch": 60.0, "velocity": 0.3}, psalm.CtrlTick{SampleOffset: 300}, p.Config())
		}
		p.Synthesize()
		out = append(out, toFloat(p.Output(0).Channels[0])...)
	}
	active := 0
	for _, v := range p.voices {
		if v.IsActive() {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("the key should sound on one voice, %d", active)
	}
	// just before every new strike the note is still ringing at its level
	for b := 10; b < 60; b += 10 {
		at := b*1024 + 300
		before := rmsDB(out, float64(at-2000)/sr, float64(at-1000)/sr)
		edge := rmsDB(out, float64(at-200)/sr, float64(at)/sr)
		if edge < before-3 {
			t.Fatalf("the ringing dropped before strike %d: %.1f to %.1f dB", b/10, before, edge)
		}
	}
}

// With all voices sounding a new key takes the quietest voice, a released key before a held one
func TestStealsQuietest(t *testing.T) {
	p := NewPoly(grand(), 2)
	p.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "amplitude": 1.0, "message": psalm.Message{"pitch": 48.0}, "noteOn": "held"})
	p.Note(0.05, 1, psalm.Message{"pitch": 72.0}, psalm.CtrlTick{}, p.Config())
	for range 20 {
		p.Synthesize()
	}
	p.Note(1, 1, psalm.Message{"pitch": 64.0}, psalm.CtrlTick{}, p.Config())
	keys := map[int]bool{}
	for _, v := range p.voices {
		if v.IsActive() {
			keys[v.Key()] = true
		}
	}
	if !keys[48] || !keys[64] {
		t.Fatalf("the released key should make way, sounding %v", keys)
	}
}

// The soundboard response in every strike fades out: a response that stops while it still sounds
// clicks a quarter second into every note
func TestSoundboardEndsSilently(t *testing.T) {
	b := soundboard(sr)
	peak, end := 0.0, 0.0
	for i, x := range b {
		peak = math.Max(peak, math.Abs(x))
		if i >= len(b)-200 {
			end = math.Max(end, math.Abs(x))
		}
	}
	if db := 20 * math.Log10(end/peak); db > -80 {
		t.Fatalf("the soundboard response ends at %.1f dB", db)
	}
}

func difference(a, b []float64) []float64 {
	d := make([]float64, min(len(a), len(b)))
	for i := range d {
		d[i] = a[i] - b[i]
	}
	return d
}

// The bark of the bass: phantom partials that grow with the velocity (from the square of the
// motion), gone in the treble
func TestBark(t *testing.T) {
	bark := func(pitch, vel float64) float64 {
		none := grand()
		none.Bark = 0
		a := playPiano(grand(), []psalm.Message{{"pitch": pitch, "velocity": vel}}, 2, 0.6, nil)
		b := playPiano(none, []psalm.Message{{"pitch": pitch, "velocity": vel}}, 2, 0.6, nil)
		return rmsDB(difference(a, b), 0, 0.5) - rmsDB(b, 0, 0.5)
	}
	soft, hard := bark(40, 0.4), bark(40, 1)
	if hard < -25 || hard < soft+8 {
		t.Errorf("the bass should bark when played hard: %.1f dB, soft %.1f dB", hard, soft)
	}
	if top := bark(80, 1); top > -60 {
		t.Errorf("the treble should not bark, %.1f dB", top)
	}
}

// The strings of a key share the bridge: what they do together dies fast, what they do against
// each other lasts, so a note falls fast and then lingers
func TestTwoStageDecay(t *testing.T) {
	x := playPiano(grand(), []psalm.Message{{"pitch": 60.0}}, 6, 6, nil)
	early := rmsDB(x, 0.1, 0.3) - rmsDB(x, 1.1, 1.3)  // dB per second, the first second
	late := (rmsDB(x, 2, 2.5) - rmsDB(x, 5, 5.5)) / 3 // dB per second, later
	if early < 2*late {
		t.Fatalf("a note should fall fast first (%.1f dB/s) and then linger (%.1f dB/s)", early, late)
	}
}

// The action is heard: the dampers and the key on release, the pedal
func TestMechanics(t *testing.T) {
	quiet := grand()
	quiet.Mechanics = 0
	a := playPiano(grand(), []psalm.Message{{"pitch": 48.0}}, 0.5, 0.8, nil)
	b := playPiano(quiet, []psalm.Message{{"pitch": 48.0}}, 0.5, 0.8, nil)
	noise := difference(a, b)
	if before, at := rmsDB(noise, 0.2, 0.45), rmsDB(noise, 0.5, 0.6); at < before+20 {
		t.Errorf("the release should make a little noise: %.1f dB, before %.1f dB", at, before)
	}
	p := NewPoly(grand(), 2)
	var out []float64
	for blk := range 40 {
		if blk == 10 {
			p.ReceiveMessage(psalm.Message{"sustain": 1.0})
		}
		if blk == 25 {
			p.ReceiveMessage(psalm.Message{"sustain": 0.0})
		}
		p.Synthesize()
		out = append(out, toFloat(p.Output(0).Channels[0])...)
	}
	if rmsDB(out, 0.58, 0.68) < -90 || rmsDB(out, 0, 0.2) > -200 {
		t.Errorf("the pedal should be heard, and nothing before it: %.1f, %.1f dB", rmsDB(out, 0.58, 0.68), rmsDB(out, 0, 0.2))
	}
}

// The soundboard colors the sound
func TestSoundboard(t *testing.T) {
	none := grand()
	none.Body = 0
	a := playPiano(grand(), cMajor, 1, 1, nil)
	b := playPiano(none, cMajor, 1, 1, nil)
	if d := rmsDB(difference(a, b), 0, 1) - rmsDB(b, 0, 1); d < -10 {
		t.Fatalf("the soundboard should change the sound, %.1f dB", d)
	}
}

// The stretch makes the octaves clean: the second partial of A5 is the fundamental of A6
func TestStretchedOctave(t *testing.T) {
	lower, _ := play(single(), psalm.Message{"pitch": 81.0}, 2, 1.5)
	upper, _ := play(single(), psalm.Message{"pitch": 93.0}, 2, 1.5)
	f := 440 * math.Exp2((81.0-69)/12)
	second := peakFrequency(lower[int(0.2*sr):int(1.2*sr)], 2*f, 0.02)
	fundamental := peakFrequency(upper[int(0.2*sr):int(1.2*sr)], 2*f, 0.02)
	if c := 1200 * math.Log2(fundamental/second); math.Abs(c) > 2 {
		t.Errorf("the octave beats: %.1f cents apart", c)
	}
	equal := single()
	equal.Stretch = 0
	upper, _ = play(equal, psalm.Message{"pitch": 93.0}, 2, 1.5)
	fundamental = peakFrequency(upper[int(0.2*sr):int(1.2*sr)], 2*f, 0.02)
	if c := 1200 * math.Log2(fundamental/second); math.Abs(c) < 3 {
		t.Errorf("without the stretch the octave should beat, %.1f cents", c)
	}
}

// Variation: the same seed plays the same, two strikes of a key are never exactly alike
func TestVariation(t *testing.T) {
	render := func(seed uint64, strikes int) []float64 {
		psalm.Seed(seed)
		psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
		p := NewPoly(GrandPreset(), 4)
		var out []float64
		for b := range 20 * strikes {
			if b%20 == 0 {
				p.Note(0.3, 1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, p.Config())
			}
			p.Synthesize()
			out = append(out, toFloat(p.Output(0).Channels[0])...)
		}
		return out
	}
	a, b := render(5, 2), render(5, 2)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("the same seed should play the same, sample %d", i)
		}
	}
	first, second := a[:20*1024], a[20*1024:]
	if d := rmsDB(difference(first, second), 0, 0.3) - rmsDB(first, 0, 0.3); d < -40 {
		t.Fatalf("two strikes should differ a little, %.1f dB", d)
	}
}

// Where the piano is heard from: about as loud everywhere, wide close by, narrower and darker from
// the audience
func TestPositions(t *testing.T) {
	type heard struct{ lufs, correlation, brightness float64 }
	listen := func(pos Position) heard {
		cfg := grand()
		cfg.Position = pos
		p := NewPoly(cfg, 16)
		for _, k := range []float64{36, 48, 60, 64, 67, 72, 84} {
			p.Note(2, 0.7, psalm.Message{"pitch": k}, psalm.CtrlTick{}, p.Config())
		}
		l := meter.NewLoudness(sr)
		var ll, rr, lr, d, e, prev float64
		for range 129 {
			p.Synthesize()
			ch := p.Output(0).Channels
			for i := range ch[0] {
				a, b := float64(ch[0][i]), float64(ch[1][i])
				l.Add(a, b)
				ll, rr, lr = ll+a*a, rr+b*b, lr+a*b
				m := (a + b) / 2
				d, e, prev = d+(m-prev)*(m-prev), e+m*m, m
			}
		}
		return heard{l.Integrated(), lr / math.Sqrt(ll*rr), 10 * math.Log10(d/e)}
	}
	player, studio, hall := listen(Player), listen(Studio), listen(Hall)
	for _, h := range []heard{studio, hall} {
		if math.Abs(h.lufs-player.lufs) > 2 {
			t.Errorf("the positions should be about as loud: %.1f vs %.1f LUFS", h.lufs, player.lufs)
		}
	}
	if !(studio.correlation < player.correlation && player.correlation < hall.correlation) {
		t.Errorf("studio should be widest, the hall narrowest: %.2f, %.2f, %.2f", studio.correlation, player.correlation, hall.correlation)
	}
	if hall.brightness > player.brightness-0.3 {
		t.Errorf("the hall should be darker: %.1f vs %.1f dB", hall.brightness, player.brightness)
	}
}

// Half pedaling: the dampers touch the strings lightly, a released note rings on shorter than with
// the pedal down, the bass more than the treble
func TestHalfPedal(t *testing.T) {
	ring := func(pitch, pedal float64) float64 {
		x := playPiano(grand(), []psalm.Message{{"pitch": pitch}}, 0.3, 1.2, psalm.Message{"sustain": pedal})
		return rmsDB(x, 0.9, 1.1)
	}
	share := func(pitch float64) float64 {
		none, half, full := ring(pitch, 0), ring(pitch, 0.5), ring(pitch, 1)
		if !(none < half && half < full) {
			t.Errorf("key %v: half pedal should ring between: %.1f, %.1f, %.1f dB", pitch, none, half, full)
		}
		return (half - none) / (full - none)
	}
	bass, treble := share(36), share(84)
	t.Logf("half pedal rings %.0f%% of the way to full in the bass, %.0f%% in the treble", 100*bass, 100*treble)
	if bass < treble {
		t.Errorf("half pedaling should let the bass ring more than the treble: %.2f vs %.2f", bass, treble)
	}
}

// The sostenuto pedal keeps the dampers of the keys held when it goes down up, until it comes up;
// keys played after it are damped as always
func TestSostenuto(t *testing.T) {
	trigger := func(p *Piano, msg psalm.Message) {
		msg["command"], msg["tick"] = "trigger", psalm.CtrlTick{}
		p.ReceiveMessage(msg)
	}
	// C3 held, the sostenuto (or not), C3 released, a short E5, the sostenuto up
	cfg := grand()
	cfg.Resonance = 0 // the dampers of the C3, not the strings ringing along
	render := func(sostenuto, bass bool) []float64 {
		p := NewPoly(cfg, 8)
		var out []float64
		for b := range 130 {
			switch b {
			case 0:
				if bass {
					trigger(p, psalm.Message{"amplitude": 1.0, "message": psalm.Message{"pitch": 48.0}, "noteOn": "c3"})
				}
			case 5:
				if sostenuto {
					p.ReceiveMessage(psalm.Message{"sostenuto": 1.0})
				}
			case 10:
				trigger(p, psalm.Message{"noteOff": "c3"})
			case 40:
				p.Note(0.1, 1, psalm.Message{"pitch": 76.0}, psalm.CtrlTick{}, p.Config())
			case 90:
				p.ReceiveMessage(psalm.Message{"sostenuto": 0.0})
			}
			p.Synthesize()
			out = append(out, toFloat(p.Output(0).Channels[0])...)
		}
		return out
	}
	at := func(x []float64, block int) float64 { // the level just before a block
		return rmsDB(x, float64(block*1024-4096)/sr, float64(block*1024)/sr)
	}
	with, without := render(true, true), render(false, true)
	if d := at(with, 40) - at(without, 40); d < 20 {
		t.Errorf("the held C3 should ring on after its release, %.1f dB", d)
	}
	if d := at(with, 88) - at(with, 70); d < -12 {
		t.Errorf("the C3 should keep ringing while the pedal is down, %.1f dB", d)
	}
	if d := at(with, 130) - at(with, 90); d > -30 {
		t.Errorf("the C3 should be damped when the pedal comes up, %.1f dB", d)
	}
	// the E5 alone, played after the pedal went down: damped as without the pedal
	e5with, e5without := render(true, false), render(false, false)
	if d := math.Abs(at(e5with, 70) - at(e5without, 70)); d > 1 {
		t.Errorf("a key played after the pedal went down should be damped as always, %.1f dB apart", d)
	}
}

// A string played hard starts sharp and settles: the harder, the more
func TestPitchSettles(t *testing.T) {
	sharp := func(vel float64) float64 {
		v := New(single())
		v.Note(1, 1, psalm.Message{"pitch": 33.0, "velocity": vel}, psalm.CtrlTick{}, v.Config())
		start := v.settle
		for range 40 {
			v.Synthesize()
		}
		if v.settle > start/10 {
			t.Errorf("velocity %v: should settle within a second, %.2f of %.2f cents left", vel, v.settle, start)
		}
		return start
	}
	if soft, hard := sharp(0.3), sharp(1); hard < 3 || hard < 4*soft {
		t.Errorf("a hard strike should start sharp (%.1f cents), more than a soft one (%.1f)", hard, soft)
	}
}

// The irregular losses don't detune: with variation the keys stay in tune
func TestTuningWithVariation(t *testing.T) {
	cfg := single()
	cfg.Variation = 1
	for _, p := range []float64{33, 57} {
		x, _ := play(cfg, psalm.Message{"pitch": p, "velocity": 0.5}, 3, 2.5)
		f0 := 440 * math.Exp2((stretched(p, &cfg)-69)/12)
		f := peakFrequency(x[int(1.2*sr):int(2.4*sr)], f0, 0.01)
		if c := 1200 * math.Log2(f/f0); math.Abs(c) > 2 {
			t.Errorf("key %v is %.1f cents out of tune", p, c)
		}
	}
}

// The knock of the soundboard is an impact: in the treble, where the strings die fast, it must not
// outlast the note (a long thud sounds like a slow hammer)
func TestKnockIsShort(t *testing.T) {
	for _, p := range []float64{96, 104} {
		without := grand()
		without.Soundboard = 0
		a := playPiano(grand(), []psalm.Message{{"pitch": p}}, 2, 0.4, nil)
		b := playPiano(without, []psalm.Message{{"pitch": p}}, 2, 0.4, nil)
		if d := rmsDB(difference(a, b), 0.1, 0.3) - rmsDB(b, 0.1, 0.3); d > -12 {
			t.Errorf("key %v: the knock should be gone after the attack, %.1f dB against the note", p, d)
		}
	}
}
