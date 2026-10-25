package streamer

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
)

// writeWAV writes a 32 bit float wav of the channels
func writeWAV(t *testing.T, path string, sr int, channels ...[]float32) {
	t.Helper()
	n, c := len(channels[0]), len(channels)
	data := make([]byte, 0, n*c*4)
	for i := range n {
		for _, ch := range channels {
			data = binary.LittleEndian.AppendUint32(data, math.Float32bits(ch[i]))
		}
	}
	var b []byte
	b = append(b, "RIFF"...)
	b = binary.LittleEndian.AppendUint32(b, uint32(36+len(data)))
	b = append(b, "WAVEfmt "...)
	b = binary.LittleEndian.AppendUint32(b, 16)
	b = binary.LittleEndian.AppendUint16(b, 3) // ieee float
	b = binary.LittleEndian.AppendUint16(b, uint16(c))
	b = binary.LittleEndian.AppendUint32(b, uint32(sr))
	b = binary.LittleEndian.AppendUint32(b, uint32(sr*c*4))
	b = binary.LittleEndian.AppendUint16(b, uint16(c*4))
	b = binary.LittleEndian.AppendUint16(b, 32)
	b = append(b, "data"...)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(data)))
	b = append(b, data...)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// ramp is a sound whose every frame is its own: a wrong frame is found at once
func ramp(n int) []float32 {
	x := make([]float32, n)
	for i := range x {
		x[i] = 0.9 * float32(math.Sin(float64(i)*0.013)) * float32(1-float64(i)/float64(n)/2)
	}
	return x
}

// run renders a voice for some frames, the left channel, waiting before every buffer until the reader has read as far as the buffer needs
func run(t *testing.T, v *Voice, frames int) []float32 {
	t.Helper()
	var out []float32
	bs := v.Config().BufferSize
	for len(out) < frames {
		deadline := time.Now().Add(5 * time.Second)
		for v.active && v.e != nil {
			w := v.fill.Load()
			avail := int64(w&(1<<genBits-1)) + int64(len(v.e.head)/2)
			need := int64(v.pos) + int64(bs)*2 + 2
			if w>>genBits != v.gen || avail >= need || (v.total >= 0 && avail >= v.total) {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("the reader does not get on")
			}
			time.Sleep(time.Millisecond)
		}
		v.Synthesize()
		out = append(out, v.Output(0).Channels[0]...)
	}
	return out[:frames]
}

func sr() int { return int(psalm.CurrentEnv().Config().SampleRate) }

func near(t *testing.T, name string, got, want []float32, from int) {
	t.Helper()
	for i := from; i < len(want) && i < len(got); i++ {
		if math.Abs(float64(got[i]-want[i])) > 1e-5 {
			t.Fatalf("%s: frame %d is %v, want %v", name, i, got[i], want[i])
		}
	}
}

// a sound longer than its head plays whole from the disk, frame for frame
func TestStreamsFromDisk(t *testing.T) {
	src := ramp(sr() * 3)
	path := filepath.Join(t.TempDir(), "long.wav")
	writeWAV(t, path, sr(), src)
	inst, err := Open(path, 60)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(inst.Entries[0].head) / 2; got != int(HeadSeconds*float64(sr())) {
		t.Errorf("%d frames in memory, want the head of %v s", got, HeadSeconds)
	}
	v := New(DefaultConfig(inst))
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	out := run(t, v, 3*sr())
	near(t, "from disk", out, src, 200) // after the fade in
	if !v.IsActive() && len(out) < len(src) {
		t.Error("stopped early")
	}
	for range 4 {
		v.Synthesize()
	}
	if v.IsActive() {
		t.Error("still active after the end")
	}
}

// a loop goes round between its points while the key is down, and a note off lets it fade out
func TestLoopAndRelease(t *testing.T) {
	src := ramp(sr() * 2 / 3) // shorter than the head, a loop in memory and from the disk both
	long := ramp(sr() * 2)
	for name, s := range map[string][]float32{"short": src, "long": long} {
		path := filepath.Join(t.TempDir(), name+".wav")
		writeWAV(t, path, sr(), s)
		inst, _, err := ParseInstrument("60 " + path + " loop release=0.05")
		if err != nil {
			t.Fatal(err)
		}
		v := New(DefaultConfig(inst))
		v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
		out := run(t, v, len(s)*5/2)
		want := make([]float32, len(out))
		for i := range want {
			want[i] = s[i%len(s)]
		}
		near(t, name+" loop", out, want, 200)
		v.NoteOff()
		tail := run(t, v, sr()/10)
		if v.IsActive() || math.Abs(float64(tail[len(tail)-1])) > 1e-6 {
			t.Errorf("%s: the loop goes on after a note off (%v)", name, tail[len(tail)-1])
		}
	}
}

// a note again starts the sound from its start, from the head and the disk once more; a note off lets a sound that is not held play on
func TestRetriggerAndOneShot(t *testing.T) {
	src := ramp(sr() * 2)
	path := filepath.Join(t.TempDir(), "s.wav")
	writeWAV(t, path, sr(), src)
	inst, _, err := ParseInstrument("60 " + path)
	if err != nil {
		t.Fatal(err)
	}
	v := New(DefaultConfig(inst))
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	run(t, v, sr()*3/4) // into the part from the disk
	v.NoteOff()
	if !v.IsActive() {
		t.Fatal("a note off stopped a sound that plays to its end")
	}
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	out := run(t, v, sr()*3/2)
	near(t, "again", out, src, 200)
}

// stereo files keep their channels, a part of a file plays, a pan and a volume and the speed count
func TestStereoPartsAndSpeed(t *testing.T) {
	l, r := ramp(sr()*2), make([]float32, sr()*2)
	for i := range r {
		r[i] = -l[i] / 2
	}
	path := filepath.Join(t.TempDir(), "st.wav")
	writeWAV(t, path, sr(), l, r)
	inst, _, err := ParseInstrument("60 " + path + " start=0.8 end=1.8")
	if err != nil {
		t.Fatal(err)
	}
	v := New(DefaultConfig(inst))
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	var left, right []float32
	bs := v.Config().BufferSize
	for len(left) < sr() {
		for range 200 {
			if w := v.fill.Load(); int64(w&(1<<genBits-1)) > int64(v.pos)+int64(bs)*2 || !v.active {
				break
			}
			time.Sleep(time.Millisecond)
		}
		v.Synthesize()
		left = append(left, v.Output(0).Channels[0]...)
		right = append(right, v.Output(0).Channels[1]...)
	}
	from := int(0.8 * float64(sr()))
	near(t, "left", left[:sr()-bs], l[from:], 200)
	near(t, "right", right[:sr()-bs], r[from:], 200)

	// twice as fast: every second frame, an octave up
	fast, _, err := ParseInstrument("60 " + path + " speed=2 vol=-6 pan=-1")
	if err != nil {
		t.Fatal(err)
	}
	w := New(DefaultConfig(fast))
	w.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, w.Config())
	out := run(t, w, sr()/2)
	want := make([]float32, len(out))
	for i := range want {
		want[i] = l[2*i] * float32(math.Pow(10, -6.0/20)) // a stereo sound panned all the way to the left: the right side silent, the left as it is
	}
	near(t, "fast", out, want, 200)
	if peak := maxAbs(w.Output(0).Channels[1]); peak != 0 {
		t.Errorf("the right side sounds (%v) with the pan all the way to the left", peak)
	}
}

func maxAbs(x []float32) float64 {
	m := 0.0
	for _, v := range x {
		m = math.Max(m, math.Abs(float64(v)))
	}
	return m
}

func TestParseInstrument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.wav")
	writeWAV(t, path, sr(), ramp(sr()))
	inst, report, err := ParseInstrument("# a comment\n36 " + path + " loop name=rain vol=-3\n38 " + path + " hold\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(inst.Entries) != 2 || inst.Lookup(36).Name != "rain" || inst.Lookup(38) == nil || inst.Lookup(40) != nil {
		t.Errorf("entries %+v", inst.Entries)
	}
	if len(report) != 2 || !strings.Contains(report[0], "from disk") || !strings.Contains(report[0], "loops") || !strings.Contains(report[1], "stops at a note off") {
		t.Errorf("report %q", report)
	}
	one, _ := Open(path, 36)
	if one.Lookup(90) == nil {
		t.Error("a single sound is not on every key")
	}
	for text, want := range map[string]string{
		"":                           "no sounds",
		"x " + path:                  "not a key",
		"36":                         "a key and a file",
		"36 " + path + " loopy":      "loop, hold",
		"36 " + path + " speed=fast": "not a number",
		"36 " + path + " pitch=2":    "speed, start",
		"36 /no/such/file.wav":       "line 1",
		"36 " + path + " start=2":    "before it starts",
	} {
		if _, _, err := ParseInstrument(text); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: error %v, want %q", text, err, want)
		}
	}
}
