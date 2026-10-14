package catalog

import (
	"os"
	"path/filepath"

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

// collect gathers what a step sequencer adds to a patch
type collect struct {
	psalm.Patcher
	added []psalm.Modular
}

func (c *collect) Add(m psalm.Modular) psalm.Patcher {
	c.added = append(c.added, m)
	return c
}

// pageSettings keeps a page's controls and grids with a patch
type pageSettings struct{ page *panel.Panel }

func (s pageSettings) Values() map[string]float64 { return s.page.Snapshot() }
func (s pageSettings) Apply(v map[string]float64)  { s.page.Restore(v) }

// drumMachineKind is the drum machine: a kit of samples played by a step sequencer (seven rows,
// sixteen steps, four patterns), its grid in the side panel; it follows the tempo of the patch.
// It is left out (false) when the samples aren't in the folder
func drumMachineKind(dir string) (patcher.Kind, bool) {
	if dir == "" {
		dir = "assets/sounds/drums"
	}
	if _, err := os.Stat(filepath.Join(dir, drumKit[0].file)); err != nil {
		return patcher.Kind{}, false
	}
	return patcher.Kind{Name: "drummachine", Group: "Synths", Outs: outs, Make: func(string) (*patcher.Made, error) {
		bank := sndfile.SoundBank[float32]{}
		var rows []string
		for _, k := range drumKit {
			f, err := sndfile.NewMipMapSoundFile[float32](filepath.Join(dir, k.file), 4)
			if err != nil {
				return nil, err
			}
			bank[k.name] = f
			rows = append(rows, k.name)
		}
		kit := drums.NewDrums(bank, 24, 2)
		var added collect
		return &patcher.Made{Module: kit,
			Added: func() []psalm.Modular { return added.added },
			Edit: func(page *panel.Panel, name, _ string) patcher.Settings {
				page.Section("Beat")
				grid := stepseq.NewGrid(page, "Beat", rows, 16, 4)
				seq := stepseq.New(&added, grid, stepseq.Config{StepsPerBeat: 4, Swing: 0.5, Humanize: 4, HumanizeVelocity: 0.06})
				for _, r := range rows {
					var opts []stepseq.RowOption
					if r == "hat" || r == "open" {
						opts = append(opts, stepseq.Choke(1)) // the closed hihat cuts the open one
					}
					seq.Row(r, stepseq.Sound(kit, r), opts...)
				}
				startBeat(grid)
				seq.Controls(page)
				return pageSettings{page}
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
