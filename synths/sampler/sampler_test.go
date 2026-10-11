package sampler

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
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
