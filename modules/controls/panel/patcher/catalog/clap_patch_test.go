package catalog

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/sndfile"
)

// the patch "clap demo" (patches/): a clock, a sequence and notes play Surge XT through Surge XT Effects, an LFO moves the cutoff; it plays for a few bars
func TestClapDemoPatch(t *testing.T) {
	dir := os.Getenv("CLAP_TEST_DIR")
	if dir == "" {
		t.Skip("CLAP_TEST_DIR is not set")
	}
	t.Setenv("CLAP_PATH", dir)
	t.Chdir("../../../../..")
	psalm.PushConfig(psalm.Config{BPM: 100, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	ir := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", Kinds(Config{IR: ir}), patcher.Keep(filepath.Join(".", "patches")))
	if err := pt.Load("clap demo"); err != nil {
		t.Fatal(err)
	}
	p.Out(pt.Out())
	var peak float64
	for range 44100 * 8 / 256 {
		p.ResetRun()
		p.Run()
		for _, s := range pt.Out().Output(0).Channels[0] {
			peak = max(peak, math.Abs(float64(s)))
		}
	}
	t.Logf("peak %.3f", peak)
	if peak < 0.02 || math.IsNaN(peak) {
		t.Errorf("silent: %v", peak)
	}
}
