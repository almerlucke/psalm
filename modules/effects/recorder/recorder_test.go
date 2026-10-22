package recorder

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestRecordsAFileAndPassesTheSound(t *testing.T) {
	dir := t.TempDir()
	r := New(Config{Dir: dir, Name: "take"})
	done := make(chan float64, 1)
	var path string
	r.OnSaved(func(p string, seconds float64, dropped int64, err error) {
		if err != nil || dropped != 0 {
			t.Errorf("saved: %v, %d dropped", err, dropped)
		}
		path = p
		done <- seconds
	})
	bs := r.Config().BufferSize
	run := func(n int) {
		for range n {
			for i := range bs {
				r.Input(0).Channels[0][i] = 0.25
				r.Input(0).Channels[1][i] = -0.25
			}
			r.Synthesize()
		}
	}
	run(3) // not recording: nothing is written
	if out := r.Output(0).Channels; math.Abs(float64(out[0][0])-0.25) > 1e-6 || math.Abs(float64(out[1][0])+0.25) > 1e-6 {
		t.Fatalf("the sound does not pass: %v %v", out[0][0], out[1][0])
	}
	r.SetRecord(true)
	run(50)
	r.SetRecord(false)
	run(1)
	select {
	case s := <-done:
		if want := float64(50*bs) / r.Config().SampleRate; math.Abs(s-want) > 1e-9 {
			t.Errorf("%v seconds, want %v", s, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the file was not closed")
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() < int64(50*bs*2*4) {
		t.Errorf("file %s: %v %v", path, fi, err)
	}
	if m, _ := filepath.Glob(filepath.Join(dir, "take-*.wav")); len(m) != 1 {
		t.Errorf("files %v", m)
	}
}

func TestDescribe(t *testing.T) {
	r := New(Config{Dir: t.TempDir()})
	paramstest.Check(t, func() params.Set { return Describe(r.Settings()) }, r.ReceiveMessage)
}
