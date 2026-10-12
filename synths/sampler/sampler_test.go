package sampler

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*Voice)(nil)

const sr = 44100.0

// sine returns a mono sound of a sine
func sine(freq, seconds float64) []float32 {
	buf := make([]float32, int(seconds*sr))
	for i := range buf {
		buf[i] = float32(math.Sin(2 * math.Pi * freq * float64(i) / sr))
	}
	return buf
}

func constant(v float32, seconds float64) []float32 {
	buf := make([]float32, int(seconds*sr))
	for i := range buf {
		buf[i] = v
	}
	return buf
}

// play holds a note on a voice for some buffers and returns the left channel
func play(v *Voice, msg psalm.Message, buffers int) []float64 {
	v.Clear()
	v.NoteOn(1, msg, psalm.CtrlTick{}, v.Config())
	return run(v, buffers)
}

// run renders buffers of a voice, the left channel
func run(v *Voice, buffers int) []float64 {
	var out []float64
	for range buffers {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			out = append(out, float64(s))
		}
	}
	return out
}

func buffers(seconds float64) int {
	return int(math.Ceil(seconds * sr / float64(psalm.CurrentEnv().Config().BufferSize)))
}

// frequency counts the zero crossings of a stretch of sound
func frequency(x []float64) float64 {
	n := 0
	for i := 1; i < len(x); i++ {
		if (x[i-1] < 0) != (x[i] < 0) {
			n++
		}
	}
	return float64(n) / 2 / (float64(len(x)) / sr)
}

func peak(x []float64) float64 {
	p := 0.0
	for _, s := range x {
		p = math.Max(p, math.Abs(s))
	}
	return p
}

// group returns voices sharing round robins and choke groups, as the voices of NewPoly
func group(cfg Config, n int) []*Voice {
	var vs []*Voice
	var sh *shared
	for range n {
		v := newVoice(cfg, sh)
		sh = v.shared
		vs = append(vs, v)
	}
	return vs
}

func voice(regions ...Region) *Voice {
	return New(Config{Instrument: MustInstrument(regions...)})
}

func TestPitchFollowsTheKey(t *testing.T) {
	r := NewRegion(NewSound(sr, sine(220, 2))).Keys(0, 127, 57)
	v := voice(r)
	for _, c := range []struct{ pitch, want float64 }{{57, 220}, {69, 440}, {45, 110}, {64, 220 * math.Pow(2, 7.0/12)}} {
		x := play(v, psalm.Message{"pitch": c.pitch}, buffers(1))
		if got := frequency(x[2000:40000]); math.Abs(got-c.want) > c.want*0.01 {
			t.Errorf("pitch %g: %.1f Hz, want %.1f", c.pitch, got, c.want)
		}
	}
	// a region at one pitch (drums) plays the same on every key, a sound recorded at another
	// sample rate plays in tune
	fixed := NewRegion(NewSound(sr/2, sine(220, 2))).Keys(0, 127, 57) // 110 Hz at its rate
	fixed.KeyTrack = 0
	v = voice(fixed)
	if got := frequency(play(v, psalm.Message{"pitch": 80.0}, buffers(1))[2000:]); math.Abs(got-110) > 1 {
		t.Errorf("a fixed region at %.1f Hz, want 110", got)
	}
}

func TestZonesAndVelocityLayers(t *testing.T) {
	low := NewRegion(NewSound(sr, constant(0.1, 1))).Keys(0, 59, 48)
	highSoft := NewRegion(NewSound(sr, constant(0.2, 1))).Keys(60, 127, 72).Velocities(1, 63)
	highHard := NewRegion(NewSound(sr, constant(0.4, 1))).Keys(60, 127, 72).Velocities(64, 127)
	for _, r := range []*Region{&low, &highSoft, &highHard} {
		r.KeyTrack, r.VelTrack = 0, 0
	}
	v := voice(low, highSoft, highHard)
	for _, c := range []struct {
		pitch, vel, want float64
	}{{40, 1, 0.1}, {59, 0.3, 0.1}, {60, 0.3, 0.2}, {70, 0.49, 0.2}, {70, 0.51, 0.4}, {100, 1, 0.4}} {
		x := play(v, psalm.Message{"pitch": c.pitch, "velocity": c.vel}, 4)
		if got := x[1000]; math.Abs(got-c.want) > 1e-6 {
			t.Errorf("key %g velocity %g: %g, want the region of %g", c.pitch, c.vel, got, c.want)
		}
	}
	// layered regions play together
	a := NewRegion(NewSound(sr, constant(0.1, 1)))
	b := NewRegion(NewSound(sr, constant(0.25, 1)))
	a.VelTrack, b.VelTrack = 0, 0
	if got := play(voice(a, b), psalm.Message{"pitch": 60.0}, 4)[1000]; math.Abs(got-0.35) > 1e-6 {
		t.Errorf("two layers: %g, want 0.35", got)
	}
}

func TestVelocityAndLevel(t *testing.T) {
	r := NewRegion(NewSound(sr, constant(0.5, 1)))
	r.Volume = -6
	v := voice(r)
	full := play(v, psalm.Message{"pitch": 60.0, "velocity": 1.0}, 4)[1000]
	half := play(v, psalm.Message{"pitch": 60.0, "velocity": 0.5}, 4)[1000]
	if want := 0.5 * math.Pow(10, -6.0/20); math.Abs(full-want) > 1e-4 {
		t.Errorf("full velocity at -6 dB: %g, want %g", full, want)
	}
	if math.Abs(half/full-0.25) > 0.01 { // the velocity squared
		t.Errorf("half velocity is %g of full, want 0.25", half/full)
	}
	// panning a mono sound is equal power
	r.Volume, r.Pan = 0, 1
	v = voice(r)
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	v.Synthesize()
	out := v.Output(0).Channels
	if l, rr := out[0][100], out[1][100]; math.Abs(float64(l)) > 1e-6 || math.Abs(float64(rr)-0.5*math.Sqrt2) > 1e-4 {
		t.Errorf("panned right: %g %g", l, rr)
	}
}

func TestRoundRobinAndRandom(t *testing.T) {
	var regions []Region
	for i := range 3 {
		r := NewRegion(NewSound(sr, constant(float32(i+1)*0.1, 1)))
		r.VelTrack, r.SeqLength, r.SeqPosition = 0, 3, i+1
		regions = append(regions, r)
	}
	vs := group(Config{Instrument: MustInstrument(regions...)}, 2)
	var got []float64
	for range 6 {
		v := vs[0]
		v.Clear()
		v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
		v.Synthesize()
		got = append(got, math.Round(float64(v.Output(0).Channels[0][10])*10)/10)
	}
	want := []float64{0.1, 0.2, 0.3, 0.1, 0.2, 0.3}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("round robin %v, want %v", got, want)
		}
	}
	// random: two halves, each takes about half the notes
	a, b := NewRegion(NewSound(sr, constant(0.1, 1))), NewRegion(NewSound(sr, constant(0.2, 1)))
	a.VelTrack, b.VelTrack = 0, 0
	a.HiRand, b.LoRand = 0.5, 0.5
	v := voice(a, b)
	n := 0
	for range 200 {
		if play(v, psalm.Message{"pitch": 60.0}, 1)[10] > 0.15 {
			n++
		}
	}
	if n < 70 || n > 130 {
		t.Errorf("random: %d of 200 notes played the upper half", n)
	}
}

func TestLoops(t *testing.T) {
	short := sine(441, 0.1) // 10 whole cycles, a loop over the last 5
	loop := func(mode LoopMode) Region {
		r := NewRegion(NewSound(sr, short))
		r.KeyTrack, r.Loop = 0, mode
		r.LoopStart, r.LoopEnd = 2205, 4409
		r.Env.Release = 0.1
		return r
	}
	// no loop: the sound ends, and so does the voice
	r := loop(NoLoop)
	v := voice(r)
	x := play(v, psalm.Message{"pitch": 60.0}, buffers(0.5))
	if peak(x[int(0.2*sr):]) != 0 || v.IsActive() {
		t.Errorf("no loop: still sounds after the sound")
	}
	// a continuous loop sounds while held, and fades out with the release
	v = voice(loop(LoopContinuous))
	x = play(v, psalm.Message{"pitch": 60.0}, buffers(1))
	if p := peak(x[int(0.9*sr):]); p < 0.99 {
		t.Errorf("a loop held for a second: peak %g", p)
	}
	if got := frequency(x[int(0.5*sr):]); math.Abs(got-441) > 2 {
		t.Errorf("the loop plays %g Hz, want 441", got)
	}
	v.NoteOff()
	x = run(v, buffers(0.3))
	if p := peak(x[int(0.05*sr):int(0.06*sr)]); p > 0.02 || p < 0.003 { // -80 dB over the release
		t.Errorf("halfway the release: peak %g", p)
	}
	if v.IsActive() {
		t.Errorf("a released loop still plays")
	}
	// a sustain loop plays on to the end after the key goes up
	r = NewRegion(NewSound(sr, sine(441, 0.5)))
	r.KeyTrack, r.Loop, r.LoopStart, r.LoopEnd, r.Env.Release = 0, LoopSustain, 2205, 4409, 2
	v = voice(r)
	play(v, psalm.Message{"pitch": 60.0}, buffers(0.5))
	v.NoteOff()
	run(v, buffers(0.3))
	if !v.IsActive() {
		t.Errorf("a sustain loop stopped at once")
	}
	run(v, buffers(0.2))
	if v.IsActive() {
		t.Errorf("a sustain loop let go plays on after the end of its sound")
	}
	// a one shot ignores the key going up, and plays the sound to its end
	r = loop(OneShot)
	r.Env.Release = 0.001
	v = voice(r)
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	v.NoteOff()
	x = run(v, buffers(0.2))
	if p := peak(x[int(0.08*sr):int(0.095*sr+0.5)]); p < 0.99 {
		t.Errorf("a one shot let go: peak %g near its end", p)
	}
}

func TestLoopCrossfade(t *testing.T) {
	// a ramp: looping it jumps from 1 back to 0, unless the end fades into the start
	ramp := make([]float32, 20000)
	for i := range ramp {
		ramp[i] = float32(i) / 20000
	}
	jump := func(xf float64) float64 {
		r := NewRegion(NewSound(sr, ramp))
		r.KeyTrack, r.Loop, r.LoopStart, r.LoopEnd, r.LoopCrossfade = 0, LoopContinuous, 10000, 19999, xf
		x := play(voice(r), psalm.Message{"pitch": 60.0}, buffers(1))
		j := 0.0
		for i := 1; i < len(x); i++ {
			j = math.Max(j, math.Abs(x[i]-x[i-1]))
		}
		return j
	}
	if j := jump(0); j < 0.4 {
		t.Errorf("without a crossfade the loop jumps only %g", j)
	}
	if j := jump(0.05); j > 0.01 {
		t.Errorf("with a crossfade the loop jumps %g", j)
	}
}

func TestEnvelope(t *testing.T) {
	r := NewRegion(NewSound(sr, constant(1, 2)))
	r.KeyTrack, r.VelTrack = 0, 0
	r.Env = Envelope{Delay: 0.01, Attack: 0.1, Hold: 0.05, Decay: 0.2, Sustain: 0.5, Release: 0.2}
	x := play(voice(r), psalm.Message{"pitch": 60.0}, buffers(1))
	at := func(s float64) float64 { return x[int(s*sr)] }
	if at(0.005) != 0 {
		t.Errorf("during the delay: %g", at(0.005))
	}
	if got := at(0.06); math.Abs(got-0.5) > 0.01 {
		t.Errorf("halfway the attack: %g", got)
	}
	if got := at(0.15); math.Abs(got-1) > 1e-3 {
		t.Errorf("in the hold: %g", got)
	}
	if got := at(0.8); math.Abs(got-0.5) > 1e-3 {
		t.Errorf("sustained: %g", got)
	}
	// sustain 0: the sound ends after the decay (a drum)
	r.Env = Envelope{Decay: 0.1}
	v := voice(r)
	play(v, psalm.Message{"pitch": 60.0}, buffers(0.2))
	if v.IsActive() {
		t.Errorf("a decay to 0 still plays")
	}
}

func TestNoteStartsOnItsSample(t *testing.T) {
	r := NewRegion(NewSound(sr, constant(1, 1)))
	v := voice(r)
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{SampleOffset: 37}, v.Config())
	v.Synthesize()
	out := v.Output(0).Channels[0]
	if out[36] != 0 || out[37] == 0 {
		t.Errorf("the note starts at %v..., want sample 37", out[35:39])
	}
	// a note of a duration lets go by itself
	v.Clear()
	v.Note(0.1, 1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	run(v, buffers(0.2))
	if v.IsActive() {
		t.Errorf("a note of 0.1 s still plays after 0.2 s")
	}
}

func TestReleaseRegions(t *testing.T) {
	body := NewRegion(NewSound(sr, constant(0.5, 2)))
	body.VelTrack, body.Env.Release = 0, 0.01
	rel := NewRegion(NewSound(sr, constant(0.2, 0.2)))
	rel.VelTrack, rel.Trigger, rel.RtDecay = 0, OnRelease, 6
	v := voice(body, rel)
	x := play(v, psalm.Message{"pitch": 60.0}, buffers(0.1))
	if got := x[100]; math.Abs(got-0.5) > 1e-6 {
		t.Errorf("held: %g, the release region plays early", got)
	}
	v.NoteOff()
	x = run(v, buffers(0.1))
	held := 0.1 // and a bit: whole buffers
	want := 0.2 * math.Pow(10, -6*held/20)
	if got := x[int(0.05*sr)]; math.Abs(got-want) > 0.01 {
		t.Errorf("the release region after %g s: %g, want about %g", held, got, want)
	}
	run(v, buffers(0.3))
	if v.IsActive() {
		t.Errorf("still plays after the release region")
	}
}

func TestChokeGroups(t *testing.T) {
	open := NewRegion(NewSound(sr, constant(0.3, 1))).Keys(46, 46, 46)
	closed := NewRegion(NewSound(sr, constant(0.1, 0.05))).Keys(42, 42, 42)
	open.Group, open.OffBy, open.Loop = 1, 2, OneShot
	closed.Group, closed.Loop = 2, OneShot
	open.VelTrack, closed.VelTrack = 0, 0
	vs := group(Config{Instrument: MustInstrument(open, closed)}, 2)
	vs[0].NoteOn(1, psalm.Message{"pitch": 46.0}, psalm.CtrlTick{}, vs[0].Config())
	run(vs[0], 4)
	vs[1].NoteOn(1, psalm.Message{"pitch": 42.0}, psalm.CtrlTick{}, vs[1].Config())
	run(vs[0], buffers(0.02))
	if vs[0].IsActive() {
		t.Errorf("the open hihat plays on after the closed one")
	}
	if !vs[1].IsActive() {
		t.Errorf("the closed hihat stopped")
	}
}

func TestPolyPlaysNotes(t *testing.T) {
	r := NewRegion(NewSound(sr, sine(220, 1))).Keys(0, 127, 57)
	p := NewPoly(DefaultConfig(MustInstrument(r)), 4)
	p.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "duration": 0.2, "amplitude": 0.5,
		"message": psalm.Message{"pitch": 69.0}})
	var x []float64
	for range buffers(0.15) {
		p.Synthesize()
		for _, s := range p.Output(0).Channels[0] {
			x = append(x, float64(s))
		}
	}
	if got := frequency(x[1000:]); math.Abs(got-440) > 5 {
		t.Errorf("the poly plays %g Hz, want 440", got)
	}
	if pk := peak(x); math.Abs(pk-0.5) > 0.02 {
		t.Errorf("the poly at amplitude 0.5: peak %g", pk)
	}
}

// writeWAV writes a mono 16 bit WAV
func writeWAV(t *testing.T, path string, samples []float32) {
	t.Helper()
	data := make([]byte, 44+2*len(samples))
	copy(data, "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(36+2*len(samples)))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1) // PCM
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], uint32(sr))
	binary.LittleEndian.PutUint32(data[28:], uint32(sr*2))
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], uint32(2*len(samples)))
	for i, s := range samples {
		binary.LittleEndian.PutUint16(data[44+2*i:], uint16(int16(math.Round(float64(s)*32767))))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSFZ(t *testing.T) {
	dir := t.TempDir()
	writeWAV(t, filepath.Join(dir, "samples", "low A.wav"), sine(220, 0.5))
	writeWAV(t, filepath.Join(dir, "samples", "high.wav"), sine(880, 0.5))
	writeWAV(t, filepath.Join(dir, "samples", "rel.wav"), constant(0.25, 0.1))
	inc := `<group> trigger=release rt_decay=3 amp_veltrack=0
<region> sample=rel.wav lokey=c2 hikey=c6
`
	sfz := `// a test instrument
<control> default_path=samples\
#define $LOW 57
/* the sustain:
   two zones */
<global> ampeg_release=0.3 amp_veltrack=50
<group> loop_mode=loop_continuous loop_start=100 loop_end=22049 seq_length=1
<region> sample=low A.wav lokey=c2 hikey=b3 pitch_keycenter=$LOW volume=-3
<region> sample=high.wav lokey=c4 hikey=c6 pitch_keycenter=a5 tune=-10 pan=-50 unknown_op=1
#include "release.sfz"
`
	if err := os.WriteFile(filepath.Join(dir, "release.sfz"), []byte(inc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test.sfz"), []byte(sfz), 0o644); err != nil {
		t.Fatal(err)
	}
	inst, err := LoadSFZ(filepath.Join(dir, "test.sfz"))
	if err != nil {
		t.Fatal(err)
	}
	if len(inst.Regions) != 3 {
		t.Fatalf("%d regions, want 3", len(inst.Regions))
	}
	low, high, rel := inst.Regions[0], inst.Regions[1], inst.Regions[2]
	if low.Name != "low A.wav" || low.LoKey != 36 || low.HiKey != 59 || low.Root != 57 || low.Volume != -3 {
		t.Errorf("the low region: %s keys %d-%d root %g volume %g", low.Name, low.LoKey, low.HiKey, low.Root, low.Volume)
	}
	if high.Root != 81 || high.Tune != -10 || high.Pan != -0.5 || high.VelTrack != 0.5 || high.Env.Release != 0.3 {
		t.Errorf("the high region: root %g tune %g pan %g veltrack %g release %g", high.Root, high.Tune, high.Pan, high.VelTrack, high.Env.Release)
	}
	if high.Loop != LoopContinuous || high.LoopStart != 100 || high.LoopEnd != 22049 {
		t.Errorf("the loop: %v %d-%d", high.Loop, high.LoopStart, high.LoopEnd)
	}
	if rel.Trigger != OnRelease || rel.RtDecay != 3 || rel.VelTrack != 0 || rel.Loop != NoLoop || rel.Env.Release != 0.3 {
		t.Errorf("the release region (a new group): %+v", rel)
	}
	if inst.Unsupported["unknown_op"] != 1 {
		t.Errorf("unsupported: %v", inst.Unsupported)
	}
	if lo, hi := inst.Keys(); lo != 36 || hi != 84 {
		t.Errorf("keys %d-%d", lo, hi)
	}

	// it plays: the high region in tune, 10 cents flat
	v := New(Config{Instrument: inst})
	x := play(v, psalm.Message{"pitch": 81.0}, buffers(1))
	if got, want := frequency(x[int(0.6*sr):]), 880*math.Pow(2, -10.0/1200); math.Abs(got-want) > 2 {
		t.Errorf("the loaded instrument plays %g Hz, want %g", got, want)
	}
}

func TestParseKey(t *testing.T) {
	for in, want := range map[string]int{"60": 60, "c4": 60, "C4": 60, "c#4": 61, "db4": 61, "a4": 69, "c-1": 0, "b3": 59, "bb3": 58} {
		if got, err := parseKey(in); err != nil || got != want {
			t.Errorf("%s: %d %v, want %d", in, got, err, want)
		}
	}
	if _, err := parseKey("h4"); err == nil {
		t.Errorf("h4 is a key")
	}
}

func TestBadRegions(t *testing.T) {
	snd := NewSound(sr, constant(1, 0.1))
	bad := []func(r *Region){
		func(r *Region) { r.Sound = nil },
		func(r *Region) { r.Offset = 100000 },
		func(r *Region) { r.Loop, r.LoopStart, r.LoopEnd = LoopContinuous, 3000, 2000 },
		func(r *Region) { r.SeqLength, r.SeqPosition = 2, 3 },
		func(r *Region) { r.LoRand, r.HiRand = 0.5, 0.5 },
	}
	for i, f := range bad {
		r := NewRegion(snd)
		f(&r)
		if _, err := NewInstrument(r); err == nil {
			t.Errorf("bad region %d is accepted", i)
		}
	}
}

// level plays a note on a voice and returns its level once it settled
func level(v *Voice, msg psalm.Message) float64 {
	x := play(v, msg, 4)
	return x[len(x)-1]
}

func TestKeyswitches(t *testing.T) {
	art := func(value float32, sw int, label string) Region {
		r := NewRegion(NewSound(sr, constant(value, 1))).Keys(48, 96, 60)
		r.VelTrack, r.KeyTrack = 0, 0
		r.SwLoKey, r.SwHiKey, r.SwLast, r.SwDefault, r.SwLabel = 36, 37, sw, 36, label
		return r
	}
	inst := MustInstrument(art(0.1, 36, "sustain"), art(0.3, 37, "pizzicato"))
	if sw, def := inst.Switches(); len(sw) != 2 || sw[1] != (Switch{37, "pizzicato"}) || def != 36 {
		t.Errorf("the switches: %v, default %d", sw, def)
	}
	vs := group(Config{Instrument: inst}, 2)
	if got := level(vs[0], psalm.Message{"pitch": 60.0}); math.Abs(got-0.1) > 1e-6 {
		t.Errorf("by default the first articulation: %g", got)
	}
	// the keyswitch itself is silent, and the next notes play the other articulation
	if got := level(vs[1], psalm.Message{"pitch": 37.0}); got != 0 {
		t.Errorf("a keyswitch sounds: %g", got)
	}
	if got := level(vs[0], psalm.Message{"pitch": 60.0}); math.Abs(got-0.3) > 1e-6 {
		t.Errorf("after the keyswitch: %g, want the second articulation", got)
	}
}

func TestVelocityAndKeyFades(t *testing.T) {
	soft := NewRegion(NewSound(sr, constant(1, 1)))
	soft.VelTrack = 0
	soft.Fades = []Fade{{From: FromVelocity, Lo: 40, Hi: 80, Out: true}}
	loud := soft
	loud.Fades = []Fade{{From: FromVelocity, Lo: 40, Hi: 80}}
	v := voice(soft, loud)
	// halfway the fade the two meet at equal power: sqrt(0.5) each
	if got := level(v, psalm.Message{"pitch": 60.0, "velocity": 60.0 / 127}); math.Abs(got-2*math.Sqrt(0.5)) > 0.02 {
		t.Errorf("halfway the velocity fade: %g, want %g", got, 2*math.Sqrt(0.5))
	}
	if got := level(v, psalm.Message{"pitch": 60.0, "velocity": 0.2}); math.Abs(got-1) > 1e-6 {
		t.Errorf("below the fade: %g, want the soft layer alone", got)
	}
	// a linear fade by key
	r := soft
	r.Fades = []Fade{{From: FromKey, Lo: 60, Hi: 70, Linear: true}}
	if got := level(voice(r), psalm.Message{"pitch": 65.0}); math.Abs(got-0.5) > 1e-6 {
		t.Errorf("halfway a linear key fade: %g", got)
	}
}

func TestModWheel(t *testing.T) {
	// two dynamic layers blended by the mod wheel while the note sounds
	piano := NewRegion(NewSound(sr, constant(0.2, 2)))
	piano.VelTrack = 0
	piano.Fades = []Fade{{From: 1, Lo: 0, Hi: 127, Out: true, Linear: true}}
	forte := NewRegion(NewSound(sr, constant(0.6, 2)))
	forte.VelTrack = 0
	forte.Fades = []Fade{{From: 1, Lo: 0, Hi: 127, Linear: true}}
	inst := MustInstrument(piano, forte)
	inst.CC = map[int]float64{1: 0}
	v := New(Config{Instrument: inst})
	if got := level(v, psalm.Message{"pitch": 60.0}); math.Abs(got-0.2) > 1e-6 {
		t.Errorf("the wheel down: %g, want the soft layer", got)
	}
	v.ReceiveMessage(psalm.Message{"cc1": 1.0})
	x := run(v, 2)
	if got := x[len(x)-1]; math.Abs(got-0.6) > 1e-6 {
		t.Errorf("the wheel up: %g, want the loud layer", got)
	}
	// it glides there: no jump between two samples
	for i := 1; i < len(x); i++ {
		if math.Abs(x[i]-x[i-1]) > 0.01 {
			t.Fatalf("the wheel jumps at sample %d: %g to %g", i, x[i-1], x[i])
		}
	}
	// a controller's gain, and its time added to the attack
	r := NewRegion(NewSound(sr, constant(1, 2)))
	r.VelTrack = 0
	r.CCGain = []CCGain{{CC: 1, DB: -12}}
	r.CCEnv = []CCEnv{{CC: 1, Env: Envelope{Attack: 0.5}}}
	inst = MustInstrument(r)
	inst.CC = map[int]float64{1: 127}
	v = New(Config{Instrument: inst})
	x = play(v, psalm.Message{"pitch": 60.0}, buffers(1))
	if got := x[int(0.25*sr)]; math.Abs(got-0.5*math.Pow(10, -12.0/20)) > 0.01 {
		t.Errorf("halfway the attack the wheel lengthened, 12 dB down: %g", got)
	}
}

func TestLoadSFZControls(t *testing.T) {
	dir := t.TempDir()
	writeWAV(t, filepath.Join(dir, "a.wav"), constant(0.5, 0.2))
	sfz := `<control> set_cc1=64
<group> sw_lokey=c2 sw_hikey=d2 sw_default=c2 sw_last=c#2 sw_label=Tremolo
xfin_locc1=32 xfin_hicc1=127 xf_cccurve=gain gain_cc1=-15 ampeg_attackcc1=0.5
xfout_lovel=40 xfout_hivel=80 xfin_lokey=c4 xfin_hikey=c5
<region> sample=a.wav
`
	if err := os.WriteFile(filepath.Join(dir, "c.sfz"), []byte(sfz), 0o644); err != nil {
		t.Fatal(err)
	}
	inst, err := LoadSFZ(filepath.Join(dir, "c.sfz"))
	if err != nil {
		t.Fatal(err)
	}
	r := inst.Regions[0]
	if inst.CC[1] != 64 || r.SwLoKey != 36 || r.SwHiKey != 38 || r.SwDefault != 36 || r.SwLast != 37 || r.SwLabel != "Tremolo" {
		t.Errorf("keyswitches and controllers: cc %v, %d-%d default %d last %d %q", inst.CC, r.SwLoKey, r.SwHiKey, r.SwDefault, r.SwLast, r.SwLabel)
	}
	want := []Fade{
		{From: FromKey, Lo: 60, Hi: 72},
		{From: FromVelocity, Lo: 40, Hi: 80, Out: true},
		{From: 1, Lo: 32, Hi: 127, Linear: true},
	}
	if len(r.Fades) != len(want) {
		t.Fatalf("fades %+v", r.Fades)
	}
	for i := range want {
		if r.Fades[i] != want[i] {
			t.Errorf("fade %d: %+v, want %+v", i, r.Fades[i], want[i])
		}
	}
	if len(r.CCGain) != 1 || r.CCGain[0] != (CCGain{1, -15}) || len(r.CCEnv) != 1 || r.CCEnv[0].Env.Attack != 0.5 {
		t.Errorf("controller gain %+v, envelope %+v", r.CCGain, r.CCEnv)
	}
	if len(inst.Unsupported) != 0 {
		t.Errorf("unsupported: %v", inst.Unsupported)
	}
}

func TestRandomAndDelay(t *testing.T) {
	r := NewRegion(NewSound(sr, constant(0.5, 1)))
	r.VelTrack, r.AmpRandom = 0, 6
	v := voice(r)
	lo, hi := math.Inf(1), 0.0
	for range 100 {
		got := level(v, psalm.Message{"pitch": 60.0})
		lo, hi = math.Min(lo, got), math.Max(hi, got)
	}
	// 6 dB of spread around the level: 3 dB louder at most, 3 softer
	if d := 20 * math.Log10(hi/lo); d < 4.5 || d > 6.01 || hi > 0.5*math.Pow(10, 3.01/20) || lo < 0.5*math.Pow(10, -3.01/20) {
		t.Errorf("amp_random 6: from %g to %g (%.1f dB)", lo, hi, d)
	}
	// the pitch: up to 50 cents either way
	p := NewRegion(NewSound(sr, sine(440, 1))).Keys(0, 127, 69)
	p.PitchRandom = 100
	v = voice(p)
	flo, fhi := math.Inf(1), 0.0
	for range 30 {
		f := preciseFrequency(play(v, psalm.Message{"pitch": 69.0}, buffers(0.3))[:int(0.25*sr)])
		flo, fhi = math.Min(flo, f), math.Max(fhi, f)
	}
	if c := 1200 * math.Log2(fhi/flo); c < 50 || c > 101 {
		t.Errorf("pitch_random 100: the notes spread %.0f cents", c)
	}
	// a delay, and a random one on top
	d := NewRegion(NewSound(sr, constant(1, 1)))
	d.Delay, d.DelayRandom = 0.01, 0.01
	v = voice(d)
	for range 10 {
		x := play(v, psalm.Message{"pitch": 60.0}, buffers(0.05))
		first := slices.IndexFunc(x, func(s float64) bool { return s != 0 })
		if at := float64(first) / sr; at < 0.0099 || at > 0.0201 {
			t.Errorf("a delay of 0.01 and up to 0.01 at random starts at %.4f s", at)
		}
	}
}

func TestVelocityEnvelope(t *testing.T) {
	r := NewRegion(NewSound(sr, constant(1, 1)))
	r.VelTrack = 0
	r.Env.Attack, r.VelEnv.Attack = 0.01, 0.2 // harder: slower
	v := voice(r)
	at := func(vel float64) float64 {
		return play(v, psalm.Message{"pitch": 60.0, "velocity": vel}, buffers(0.3))[int(0.05*sr)]
	}
	if soft, hard := at(0.1), at(1); soft < 0.99 || hard > 0.3 {
		t.Errorf("50 ms in: a soft note %g (attack 0.03 s), a hard one %g (0.21 s)", soft, hard)
	}
}

func TestLFOs(t *testing.T) {
	// vibrato: 50 cents up and down at 5 Hz
	r := NewRegion(NewSound(sr, sine(440, 2))).Keys(0, 127, 69)
	r.LFOs = []LFO{{Freq: 5, Wave: LFOSine, Pitch: 50}}
	x := play(voice(r), psalm.Message{"pitch": 69.0}, buffers(1.5))
	lo, hi := math.Inf(1), 0.0
	for at := 0.1; at < 1.3; at += 0.01 {
		f := preciseFrequency(x[int(at*sr) : int(at*sr)+int(0.02*sr)])
		lo, hi = math.Min(lo, f), math.Max(hi, f)
	}
	if c := 1200 * math.Log2(hi/lo); c < 70 || c > 130 {
		t.Errorf("a vibrato of 50 cents swings %.0f cents", c)
	}
	// tremolo: 6 dB up and down, after a delay and a fade
	r = NewRegion(NewSound(sr, constant(0.5, 2)))
	r.VelTrack = 0
	r.LFOs = []LFO{{Freq: 4, Wave: LFOSquare, Volume: 6, Delay: 0.2, Fade: 0.1}}
	x = play(voice(r), psalm.Message{"pitch": 60.0}, buffers(1.2))
	if p := peakIn(x[:int(0.19*sr)]); math.Abs(p-0.5) > 1e-6 {
		t.Errorf("before the tremolo's delay: %g", p)
	}
	if p := peakIn(x[int(0.4*sr):]); math.Abs(p-0.5*math.Pow(10, 6.0/20)) > 0.01 {
		t.Errorf("the tremolo at its depth peaks at %g", p)
	}
}

func peakIn(x []float64) float64 { return peak(x) }

// preciseFrequency measures between the first and the last upward zero crossing, interpolated
func preciseFrequency(x []float64) float64 {
	first, last, n := -1.0, 0.0, 0
	for i := 1; i < len(x); i++ {
		if x[i-1] < 0 && x[i] >= 0 {
			at := float64(i-1) + x[i-1]/(x[i-1]-x[i])
			if first < 0 {
				first = at
			} else {
				last = at
				n++
			}
		}
	}
	if n == 0 {
		return 0
	}
	return float64(n) * sr / (last - first)
}

func TestEQAndWidth(t *testing.T) {
	r := NewRegion(NewSound(sr, sine(1000, 1))).Keys(0, 127, 60)
	r.VelTrack = 0
	plain := peakIn(play(voice(r), psalm.Message{"pitch": 60.0}, buffers(0.5))[int(0.2*sr):])
	r.EQ = []EQBand{{Freq: 1000, Bandwidth: 1, Gain: 6}}
	boosted := peakIn(play(voice(r), psalm.Message{"pitch": 60.0}, buffers(0.5))[int(0.2*sr):])
	if d := 20 * math.Log10(boosted/plain); math.Abs(d-6) > 0.2 {
		t.Errorf("an EQ band of 6 dB at the tone: %.2f dB", d)
	}
	r.EQ = []EQBand{{Freq: 8000, Bandwidth: 0.5, Gain: 12}}
	if far := peakIn(play(voice(r), psalm.Message{"pitch": 60.0}, buffers(0.5))[int(0.2*sr):]); math.Abs(20*math.Log10(far/plain)) > 0.5 {
		t.Errorf("an EQ band 3 octaves away changes the tone %.2f dB", 20*math.Log10(far/plain))
	}
	// a stereo sound made mono
	s := NewRegion(NewSound(sr, constant(0.4, 1), constant(0, 1)))
	s.VelTrack, s.Width = 0, 0
	v := voice(s)
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	v.Synthesize()
	out := v.Output(0).Channels
	if l, rr := out[0][100], out[1][100]; math.Abs(float64(l)-0.2) > 1e-6 || l != rr {
		t.Errorf("width 0: %g %g, want 0.2 both", l, rr)
	}
}

func TestLoadSFZModulation(t *testing.T) {
	dir := t.TempDir()
	writeWAV(t, filepath.Join(dir, "a.wav"), constant(0.5, 0.2))
	sfz := `<region> sample=a.wav amp_random=1.5 pitch_random=12 delay_random=0.012 delay=0.05 width=50
ampeg_vel2attack=-0.18 ampeg_vel2release=0.4 ampeg_vel2sustain=-20
pitchlfo_freq=5.5 pitchlfo_depth=15 pitchlfo_delay=0.4 pitchlfo_fade=0.3
amplfo_freq=4 amplfo_depth=1.2
lfo01_freq=6 lfo01_wave=1 lfo01_volume=-2 lfo01_fade=1
eq1_gain=3 eq2_freq=1200 eq2_bw=2 eq2_gain=-4 eq3_freq=8000
`
	if err := os.WriteFile(filepath.Join(dir, "m.sfz"), []byte(sfz), 0o644); err != nil {
		t.Fatal(err)
	}
	inst, err := LoadSFZ(filepath.Join(dir, "m.sfz"))
	if err != nil {
		t.Fatal(err)
	}
	r := inst.Regions[0]
	if r.AmpRandom != 1.5 || r.PitchRandom != 12 || r.DelayRandom != 0.012 || r.Delay != 0.05 || r.Width != 50 {
		t.Errorf("random %g %g %g, delay %g, width %g", r.AmpRandom, r.PitchRandom, r.DelayRandom, r.Delay, r.Width)
	}
	if r.VelEnv.Attack != -0.18 || r.VelEnv.Release != 0.4 || r.VelEnv.Sustain != -0.2 {
		t.Errorf("the velocity envelope %+v", r.VelEnv)
	}
	wantLFOs := []LFO{
		{Freq: 6, Wave: LFOSine, Fade: 1, Volume: -2},
		{Freq: 4, Wave: LFOSine, Volume: 1.2},
		{Freq: 5.5, Wave: LFOSine, Delay: 0.4, Fade: 0.3, Pitch: 15},
	}
	if !slices.Equal(r.LFOs, wantLFOs) {
		t.Errorf("LFOs %+v, want %+v", r.LFOs, wantLFOs)
	}
	wantEQ := []EQBand{{50, 1, 3}, {1200, 2, -4}} // the third has no gain
	if !slices.Equal(r.EQ, wantEQ) {
		t.Errorf("EQ %+v, want %+v", r.EQ, wantEQ)
	}
	if len(inst.Unsupported) != 0 {
		t.Errorf("unsupported: %v", inst.Unsupported)
	}
}

func TestSustainPedal(t *testing.T) {
	r := NewRegion(NewSound(sr, constant(0.5, 3)))
	r.VelTrack, r.Env.Release = 0, 0.01
	rel := NewRegion(NewSound(sr, constant(0.2, 0.5)))
	rel.VelTrack, rel.Trigger = 0, OnRelease
	inst := MustInstrument(r, rel)
	inst.SustainLo = 64
	vs := group(Config{Instrument: inst}, 2)
	v := vs[0]
	play(v, psalm.Message{"pitch": 60.0}, 2)
	// half way is not down (64 and up is), all the way is
	v.ReceiveMessage(psalm.Message{"sustain": 0.4})
	if v.pedalDown() {
		t.Errorf("the pedal at 0.4 (51) is down with sustain_lo 64")
	}
	v.ReceiveMessage(psalm.Message{"cc64": 1.0})
	v.NoteOff()
	x := run(v, buffers(0.2))
	if got := x[len(x)-1]; math.Abs(got-0.5) > 1e-6 {
		t.Errorf("let go with the pedal down: %g, want it held at 0.5", got)
	}
	// the pedal up: the note lets go, its release region plays
	v.ReceiveMessage(psalm.Message{"cc64": 0.0})
	x = run(v, buffers(0.1))
	if got := x[len(x)-1]; math.Abs(got-0.2) > 1e-3 {
		t.Errorf("after the pedal came up: %g, want the release region (0.2)", got)
	}
	// without a pedal of its own the instrument ignores it
	inst.NoSustain = true
	v = New(Config{Instrument: inst})
	v.ReceiveMessage(psalm.Message{"sustain": 1.0})
	play(v, psalm.Message{"pitch": 60.0}, 2)
	v.NoteOff()
	if !v.released {
		t.Errorf("NoSustain: the pedal still holds the note")
	}
}

func TestDynamicEnvelope(t *testing.T) {
	// the mod wheel lengthens the attack: with ampeg_dynamic, moving it mid attack slows what is left
	mk := func(dynamic bool) *Voice {
		r := NewRegion(NewSound(sr, constant(1, 6)))
		r.VelTrack, r.Env.Attack, r.EnvDynamic = 0, 0.2, dynamic
		r.CCEnv = []CCEnv{{CC: 1, Env: Envelope{Attack: 1.8, Sustain: -0.5}}}
		return New(Config{Instrument: MustInstrument(r)})
	}
	at := func(dynamic bool) (mid, end float64) {
		v := mk(dynamic)
		play(v, psalm.Message{"pitch": 60.0}, buffers(0.1)) // 0.1 s: halfway the attack of 0.2 s
		v.ReceiveMessage(psalm.Message{"cc1": 1.0})         // now 2 s, the sustain half
		x := run(v, buffers(0.5))
		y := run(v, buffers(3))
		return x[len(x)-1], y[len(y)-1]
	}
	still, stillEnd := at(false)
	moved, movedEnd := at(true)
	if still < 0.99 || stillEnd < 0.99 {
		t.Errorf("not dynamic: the wheel changes the playing note (%g, %g)", still, stillEnd)
	}
	// dynamic: the half left of the attack now takes half of 2 s
	if moved > 0.85 || moved < 0.6 {
		t.Errorf("dynamic: 0.5 s after the wheel moved %g, want about 0.75", moved)
	}
	if math.Abs(movedEnd-0.5) > 0.01 {
		t.Errorf("dynamic: the sustain %g, want 0.5", movedEnd)
	}
}

func TestLoadSFZPedalAndDynamic(t *testing.T) {
	dir := t.TempDir()
	writeWAV(t, filepath.Join(dir, "a.wav"), constant(0.5, 0.2))
	sfz := "<group> sustain_lo=63 sustain_cc=66 ampeg_dynamic=1 ampeg_attackcc1=0.5\n<region> sample=a.wav\n"
	if err := os.WriteFile(filepath.Join(dir, "p.sfz"), []byte(sfz), 0o644); err != nil {
		t.Fatal(err)
	}
	inst, err := LoadSFZ(filepath.Join(dir, "p.sfz"))
	if err != nil {
		t.Fatal(err)
	}
	if inst.SustainLo != 63 || inst.SustainCC != 66 || !inst.Regions[0].EnvDynamic || len(inst.Unsupported) != 0 {
		t.Errorf("pedal %d from %g, dynamic %v, unsupported %v", inst.SustainCC, inst.SustainLo, inst.Regions[0].EnvDynamic, inst.Unsupported)
	}
}

func TestPolyphony(t *testing.T) {
	// note_polyphony 1: striking a key again lets the old note of it go
	r := NewRegion(NewSound(sr, constant(0.3, 3)))
	r.VelTrack, r.NotePolyphony = 0, 1
	vs := group(Config{Instrument: MustInstrument(r)}, 4)
	vs[0].NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, vs[0].Config())
	run(vs[0], 2)
	vs[1].NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, vs[1].Config())
	vs[2].NoteOn(1, psalm.Message{"pitch": 62.0}, psalm.CtrlTick{}, vs[2].Config())
	run(vs[0], buffers(0.1))
	if vs[0].IsActive() {
		t.Errorf("note_polyphony 1: the first note of the key plays on")
	}
	run(vs[2], 1)
	if !vs[2].IsActive() {
		t.Errorf("note_polyphony 1: another key stopped")
	}
	// polyphony 2 over the group: the third note lets the oldest go
	g := NewRegion(NewSound(sr, constant(0.3, 3)))
	g.VelTrack, g.Group, g.Polyphony = 0, 3, 2
	vs = group(Config{Instrument: MustInstrument(g)}, 4)
	for i, p := range []float64{60, 62, 64} {
		vs[i].NoteOn(1, psalm.Message{"pitch": p}, psalm.CtrlTick{}, vs[i].Config())
		run(vs[i], 1)
	}
	run(vs[0], buffers(0.1))
	if vs[0].IsActive() || !vs[1].IsActive() {
		t.Errorf("polyphony 2: the oldest (%v) should stop, the second (%v) play", vs[0].IsActive(), vs[1].IsActive())
	}
}

func TestFirstAndLegato(t *testing.T) {
	first := NewRegion(NewSound(sr, constant(0.1, 1)))
	legato := NewRegion(NewSound(sr, constant(0.4, 1)))
	first.VelTrack, legato.VelTrack = 0, 0
	first.Trigger, legato.Trigger = OnFirst, OnLegato
	vs := group(Config{Instrument: MustInstrument(first, legato)}, 3)
	// the first key of a phrase plays the first region, a key while it is held the legato one
	if got := level(vs[0], psalm.Message{"pitch": 60.0}); math.Abs(got-0.1) > 1e-6 {
		t.Errorf("alone: %g, want the first region", got)
	}
	if got := level(vs[1], psalm.Message{"pitch": 62.0}); math.Abs(got-0.4) > 1e-6 {
		t.Errorf("over a held key: %g, want the legato region", got)
	}
	vs[0].NoteOff()
	vs[1].NoteOff()
	if got := level(vs[2], psalm.Message{"pitch": 64.0}); math.Abs(got-0.1) > 1e-6 {
		t.Errorf("after letting go: %g, want the first region again", got)
	}
}

// writeLoopedWAV writes a mono 16 bit WAV with a smpl chunk holding one loop
func writeLoopedWAV(t *testing.T, path string, samples []float32, start, end uint32) {
	t.Helper()
	writeWAV(t, path, samples)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	smpl := make([]byte, 8+36+24)
	copy(smpl, "smpl")
	binary.LittleEndian.PutUint32(smpl[4:], 36+24)
	binary.LittleEndian.PutUint32(smpl[8+28:], 1) // one loop
	binary.LittleEndian.PutUint32(smpl[8+36+8:], start)
	binary.LittleEndian.PutUint32(smpl[8+36+12:], end)
	data = append(data, smpl...)
	binary.LittleEndian.PutUint32(data[4:], uint32(len(data)-8))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWAVLoopsAndBend(t *testing.T) {
	dir := t.TempDir()
	writeLoopedWAV(t, filepath.Join(dir, "loop.wav"), sine(441, 0.5), 4410, 8819)
	writeWAV(t, filepath.Join(dir, "plain.wav"), sine(441, 0.5))
	sfz := `<region> sample=loop.wav key=60 bend_up=1200 bend_down=-100
<region> sample=loop.wav key=62 loop_mode=no_loop
<region> sample=plain.wav key=64
`
	if err := os.WriteFile(filepath.Join(dir, "l.sfz"), []byte(sfz), 0o644); err != nil {
		t.Fatal(err)
	}
	inst, err := LoadSFZ(filepath.Join(dir, "l.sfz"))
	if err != nil {
		t.Fatal(err)
	}
	r := inst.Regions
	if r[0].Loop != LoopContinuous || r[0].LoopStart != 4410 || r[0].LoopEnd != 8819 {
		t.Errorf("the WAV's loop: %v %d-%d", r[0].Loop, r[0].LoopStart, r[0].LoopEnd)
	}
	if r[1].Loop != NoLoop || r[2].Loop != NoLoop {
		t.Errorf("no_loop set, or no loop in the file: %v %v", r[1].Loop, r[2].Loop)
	}
	if r[0].BendUp != 1200 || r[0].BendDown != -100 || r[2].BendUp != 200 {
		t.Errorf("bend up %g down %g, the default %g", r[0].BendUp, r[0].BendDown, r[2].BendUp)
	}
	// the looped one sounds on past its end; bent all the way up an octave
	v := New(Config{Instrument: inst})
	x := play(v, psalm.Message{"pitch": 60.0}, buffers(1))
	if p := peak(x[int(0.8*sr):]); p < 0.5 {
		t.Errorf("the WAV's loop: silent after the sound (%g)", p)
	}
	v.ReceiveMessage(psalm.Message{"pitchBend": 1.0})
	run(v, 2)
	x = run(v, buffers(0.3))
	if f := frequency(x[1000:]); math.Abs(f-882) > 6 {
		t.Errorf("bent up by bend_up 1200: %.0f Hz, want 882", f)
	}
}
