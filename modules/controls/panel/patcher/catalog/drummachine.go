package catalog

import (
	"hash/fnv"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/stepseq"
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/sndfile"
)

// the kit of the drum machine: a row each, by its sample in the folder of drums
var drumKit = []struct{ name, file string }{
	{"kick", "kick/Cymatics - Humble Friday Kick - E.wav"},
	{"snare", "snare/Cymatics - Humble King Snare 1 - A.wav"},
	{"clap", "clap/Cymatics - Humble Stars Clap.wav"},
	{"hat", "hihat/Cymatics - Humble Closed Hihat 4.wav"},
	{"open", "hihat/Cymatics - Humble Open Hihat 1.wav"},
	{"perc", "percussion/Cymatics - Odyssey Percussion 3.wav"},
	{"808", "808/Cymatics - Humble 808 1 - D.wav"},
}

// the drum machine has patterns of up to drumSteps steps (16 to start with), drumPatterns of them
const (
	drumSteps    = 32
	drumPatterns = 16
)

// collect gathers what a step sequencer adds to a patch
type collect struct {
	psalm.Patcher
	added []psalm.Modular
}

func (c *collect) Add(m psalm.Modular) psalm.Patcher {
	c.added = append(c.added, m)
	return c
}

// samples is a folder of drum samples: the files (by their path in the folder, sorted) and the
// sounds loaded from them, kept for the modules that use them
type samples struct {
	dir   string
	files []string // slash separated, from dir
	mu    sync.Mutex
	cache map[string]sndfile.SoundFiler[float32]
}

// listSamples lists the sound files in a folder and its folders
func listSamples(dir string) *samples {
	s := &samples{dir: dir, cache: map[string]sndfile.SoundFiler[float32]{}}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".wav", ".aif", ".aiff", ".flac":
			if rel, err := filepath.Rel(dir, path); err == nil {
				s.files = append(s.files, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	slices.Sort(s.files)
	return s
}

// load reads a sample (once)
func (s *samples) load(rel string) (sndfile.SoundFiler[float32], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sf, ok := s.cache[rel]; ok {
		return sf, nil
	}
	sf, err := sndfile.NewMipMapSoundFile[float32](filepath.Join(s.dir, filepath.FromSlash(rel)), 4)
	if err != nil {
		return nil, err
	}
	s.cache[rel] = sf
	return sf, nil
}

// id is a number for a sample that is kept in a patch instead of its place in the list (which
// changes when files come and go): a hash of its path, small enough to be exact as a float
func sampleID(rel string) float64 {
	h := fnv.New64a()
	h.Write([]byte(rel))
	return float64(h.Sum64() & (1<<52 - 1))
}

// drumSettings keeps a page's controls and grids with a patch; the samples chosen by their id
type drumSettings struct {
	page    *panel.Panel
	samples *samples
	picks   map[string]*panel.Param // "kick sample" ...
}

func (s drumSettings) Values() map[string]float64 {
	v := s.page.Snapshot()
	for name, par := range s.picks {
		delete(v, name)
		if i := int(par.Value()); i >= 0 && i < len(s.samples.files) {
			v[name+" id"] = sampleID(s.samples.files[i])
		}
	}
	return v
}

func (s drumSettings) Apply(v map[string]float64) {
	v = maps.Clone(v)
	for name := range s.picks {
		id, ok := v[name+" id"]
		delete(v, name+" id")
		if !ok {
			continue
		}
		if i := slices.IndexFunc(s.samples.files, func(f string) bool { return sampleID(f) == id }); i >= 0 {
			v[name] = float64(i) // a sample that is gone: the one it has stays
		}
	}
	s.page.Restore(v)
}

// drumMachineKind is the drum machine: a kit of samples (any of the folder's, chosen for each
// row in the side panel) played by a step sequencer (seven rows, sixteen patterns of up to 32
// steps), its grid in the side panel; it follows the tempo of the patch. It is left out (false)
// when the samples aren't in the folder
func drumMachineKind(dir string) (patcher.Kind, bool) {
	if dir == "" {
		dir = "assets/sounds/drums"
	}
	if _, err := os.Stat(filepath.Join(dir, drumKit[0].file)); err != nil {
		return patcher.Kind{}, false
	}
	lib := listSamples(dir)
	return patcher.Kind{Name: "drummachine", Group: "Synths", Outs: outs, Make: func(string) (*patcher.Made, error) {
		var rows []string
		var start []sndfile.SoundFiler[float32]
		bank := sndfile.SoundBank[float32]{}
		for _, k := range drumKit {
			sf, err := lib.load(k.file)
			if err != nil {
				return nil, err
			}
			rows = append(rows, k.name)
			start = append(start, sf)
		}
		kit := drums.NewDrums(bank, 24, 2)
		var added collect
		return &patcher.Made{Module: kit,
			Added: func() []psalm.Modular { return added.added },
			Edit: func(page *panel.Panel, name, _ string) patcher.Settings {
				page.Section("Beat")
				grid := stepseq.NewGrid(page, "Beat", rows, drumSteps, drumPatterns)
				grid.DefaultSteps(16) // a pattern starts with 16, up to drumSteps
				seq := stepseq.New(&added, grid, stepseq.Config{StepsPerBeat: 4, Swing: 0.5, Humanize: 4, HumanizeVelocity: 0.06})
				targets := map[string]*stepseq.SampleSound{}
				for i, r := range rows {
					var opts []stepseq.RowOption
					if r == "hat" || r == "open" {
						opts = append(opts, stepseq.Choke(1)) // the closed hihat cuts the open one
					}
					targets[r] = stepseq.Sample(kit, r, start[i])
					seq.Row(r, targets[r], opts...)
				}
				startBeat(grid)
				// the sample of every row, from the folder
				page.Section("Samples")
				labels := make([]string, len(lib.files))
				for i, f := range lib.files {
					labels[i] = strings.TrimSuffix(f, filepath.Ext(f))
				}
				picks := map[string]*panel.Param{}
				for _, k := range drumKit {
					row, def := k.name, slices.Index(lib.files, filepath.ToSlash(k.file))
					picks[row+" sample"] = page.Select(row+" sample", labels, func(n int) {
						if n < 0 || n >= len(lib.files) {
							return
						}
						go func() { // reading a file takes a while: not on the audio thread
							if sf, err := lib.load(lib.files[n]); err == nil {
								targets[row].Set(sf)
							}
						}()
					}, panel.Label(row), panel.Default(float64(max(def, 0))))
				}
				seq.Controls(page)
				return drumSettings{page, lib, picks}
			}}, nil
	}}, true
}

// startBeat writes a beat in the first pattern, to hear it at once
func startBeat(g *panel.Grid) {
	for row, text := range map[string]string{
		"kick": "X... ..x. X... .o..", "snare": ".... X... .... X..x", "hat": "x.x. xox. x.x. xoxx",
		"open": ".... .... .... ..x.", "808": "X... ..x. x... ....",
	} {
		i := 0
		for _, c := range text {
			if c == ' ' {
				continue
			}
			if v := map[rune]float64{'X': 1, 'x': 0.75, 'o': 0.4}[c]; v > 0 {
				g.Set(0, g.Row(row), i, stepseq.Velocity, v)
			}
			i++
		}
	}
}
