package render

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/granulator"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/sndfile"
)

// song builds a small patch: a pulsar note (random pans and masking) through a granulator and a
// reverb, in a mixer
func song() (*psalm.Psalm, *mixer.Mixer) {
	p := psalm.New(nil)
	mix := mixer.New(p)
	cfg := fdnreverb.DefaultConfig()
	cfg.Decay = 0.8
	cfg.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(cfg).AddTo(p))

	crackle := pulsar.NewPoly(pulsar.CracklePreset(), 2).AddTo(p)
	crackle.ValueIn(score.NewSequencer([]score.Note{score.N(50, 2)}, score.Config{Amp: 0.5, Key: "frequency", Feel: score.Feel{Timing: 20}}).AddTo(p))
	gcfg := granulator.DefaultConfig()
	gran := granulator.New(gcfg)
	gran.AddTo(p)
	gran.In(crackle)
	mix.Channel("crackle", gran, mixer.StripConfig{Sends: map[string]float64{"hall": 0.5}})

	tone := pulsar.NewPoly(pulsar.VowelLeadPreset(), 2).AddTo(p)
	tone.ValueIn(score.NewSequencer([]score.Note{score.N(57, 2)}, score.Config{Amp: 0.3}).AddTo(p))
	mix.Channel("tone", tone, mixer.StripConfig{Pan: 0.3})

	p.Out(mix.Master())
	return p, mix
}

func TestToFileWithTail(t *testing.T) {
	p, _ := song()
	path := filepath.Join(t.TempDir(), "song.wav")
	stats, err := ToFile(p, path, Options{Seconds: 1, Tail: 20})
	if err != nil {
		t.Fatal(err)
	}
	// the notes last about 2 s and the reverb rings for about a second after: the tail ends
	// well before the 20 s allowed
	if stats.Seconds < 2 || stats.Seconds > 8 {
		t.Fatalf("expected the render to stop once the tail was silent, rendered %.1f s", stats.Seconds)
	}
	sf, err := sndfile.NewSoundFile[float32](path)
	if err != nil {
		t.Fatal(err)
	}
	if sf.NumChannels() != 2 || math.Abs(float64(sf.NumFrames())/44100-stats.Seconds) > 0.01 {
		t.Fatalf("file has %d channels and %d frames, rendered %.2f s", sf.NumChannels(), sf.NumFrames(), stats.Seconds)
	}
	if stats.Peak < -40 || stats.Loudness < -60 || stats.Speed() <= 0 {
		t.Fatalf("unexpected stats %v", stats)
	}
	t.Log(stats)
}

func TestStems(t *testing.T) {
	p, mix := song()
	path := filepath.Join(t.TempDir(), "out", "song.wav")
	stats, err := Stems(p, path, Options{Seconds: 1}, MixerStems(mix))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"crackle", "tone", "hall"} {
		f := filepath.Join(filepath.Dir(path), "song-stems", name+".wav")
		sf, err := sndfile.NewSoundFile[float32](f)
		if err != nil {
			t.Fatalf("stem %s: %v", name, err)
		}
		if math.Abs(float64(sf.NumFrames())/44100-stats.Seconds) > 0.01 {
			t.Fatalf("stem %s has %d frames, the render %.2f s", name, sf.NumFrames(), stats.Seconds)
		}
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("the master file is missing")
	}
}

// capture renders a patch and returns the left output
func capture(p *psalm.Psalm, seconds float64) []float32 {
	var out []float32
	Run(p, Options{Seconds: seconds}, func(ch psalm.Channels) {
		out = append(out, ch[0]...)
	})
	return out
}

// With a seed the whole render repeats exactly: grains, humanized timing, random pans, masking
func TestSeedRepeats(t *testing.T) {
	psalm.Seed(42)
	p, _ := song()
	a := capture(p, 1.5)
	psalm.Seed(42)
	p, _ = song()
	b := capture(p, 1.5)
	psalm.Seed(43)
	p, _ = song()
	c := capture(p, 1.5)

	same, differ := true, false
	for i := range a {
		if a[i] != b[i] {
			same = false
		}
		if a[i] != c[i] {
			differ = true
		}
	}
	if !same {
		t.Fatal("the same seed should render the same audio")
	}
	if !differ {
		t.Fatal("another seed should render other audio")
	}
}

func TestProfile(t *testing.T) {
	p, _ := song()
	pr := RunProfiled(p, Options{Seconds: 1})
	if pr.Total <= 0 || len(pr.ByType) < 4 {
		t.Fatalf("expected time per module type, got %+v", pr.ByType)
	}
	report := pr.String()
	for _, want := range []string{"poly.Poly[pulsar.Voice]", "fdnreverb.FDNReverb", "granulator.Granulator", "crackle (mixer.Strip)"} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in the report:\n%s", want, report)
		}
	}
	for i := 1; i < len(pr.ByType); i++ {
		if pr.ByType[i].Time > pr.ByType[i-1].Time {
			t.Fatal("types should be sorted by time")
		}
	}
	t.Log("\n" + report)
}

// tempoSong is a patch whose arrangement slows it down halfway
func tempoSong() *psalm.Psalm {
	p := psalm.New(nil)
	form := arrange.NewForm(arrange.Section{Name: "a", Bars: 1}, arrange.Section{Name: "b", Bars: 1})
	tone := pulsar.NewPoly(pulsar.BuzzBassPreset(), 4).AddTo(p)
	form.Part(p, tone, "bass", score.Config{Amp: 0.4, Feel: score.Feel{Timing: 10, Velocity: 0.2}}, func(s arrange.Section) []score.Note {
		return score.Notes(0.5, 40, 47, 52, 47, 40, 47, 52, 55)
	})
	arrange.NewArrangement(form).Tempo(arrange.Lane{Sections: map[string]arrange.Curve{"b": arrange.Linear(120, 60)}}).AddTo(p)
	p.Out(tone)
	return p
}

// Two patches render in parallel, each exactly as when it renders alone: every patch has its own
// tempo and random streams (run with -race to check for shared state)
func TestParallelRenders(t *testing.T) {
	build := func() (*psalm.Psalm, *psalm.Psalm) {
		psalm.Seed(11)
		a, _ := song()
		psalm.Seed(12)
		b := tempoSong()
		return a, b
	}

	a, b := build()
	soloA, soloB := capture(a, 3), capture(b, 3)

	a, b = build()
	var parA, parB []float32
	done := make(chan bool)
	go func() { parA = capture(a, 3); done <- true }()
	go func() { parB = capture(b, 3); done <- true }()
	<-done
	<-done

	for i := range soloA {
		if soloA[i] != parA[i] {
			t.Fatalf("patch a differs at sample %d when rendered in parallel", i)
		}
	}
	for i := range soloB {
		if soloB[i] != parB[i] {
			t.Fatalf("patch b differs at sample %d when rendered in parallel", i)
		}
	}
	// the tempo change of b stayed in b
	if bpm := a.Env().BPM(); bpm != 120 {
		t.Fatalf("patch a should keep its tempo, it is %v", bpm)
	}
	if bpm := b.Env().BPM(); bpm > 100 {
		t.Fatalf("patch b should have slowed down, it is %v", bpm)
	}
}
