package catalog

import (
	"encoding/base64"
	"fmt"
	"slices"
	"sync"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/clap"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/params"
)

// the CLAP plugins on this computer, searched for once (see clap.SearchPath)
var scanned = sync.OnceValue(func() []clap.Descriptor { return clap.Scan() })

// clapKinds are the kinds of the CLAP plugins in dirs (the usual places when there are none): "clap" for the instruments (a synth of the patcher: it takes notes)
// and "clap fx" for the effects, each with the plugins as its presets. A kind with no plugin is left out
func clapKinds(dirs ...string) []patcher.Kind {
	ds := scanned()
	if len(dirs) > 0 {
		ds = clap.Scan(dirs...)
	}
	var kinds []patcher.Kind
	for _, k := range []struct {
		name, group string
		ins         []patcher.Port
		wants       func(clap.Descriptor) bool
	}{
		{"clap", "Synths", note, clap.Descriptor.Instrument},
		{"clap fx", "Effects", ins, clap.Descriptor.Effect},
	} {
		byName := map[string]clap.Descriptor{}
		for _, d := range ds {
			if !k.wants(d) {
				continue
			}
			name := d.Name
			if _, dup := byName[name]; dup {
				name = fmt.Sprintf("%s (%s)", d.Name, d.ID)
			}
			byName[name] = d
		}
		if len(byName) == 0 {
			continue
		}
		presets := make([]string, 0, len(byName))
		for n := range byName {
			presets = append(presets, n)
		}
		slices.Sort(presets)
		kinds = append(kinds, patcher.Kind{Name: k.name, Group: k.group, Presets: presets, Ins: k.ins, Outs: outs,
			Make: func(preset string) (*patcher.Made, error) {
				d, ok := byName[preset]
				if !ok {
					return nil, fmt.Errorf("no plugin %q", preset)
				}
				pl, err := clap.New(d)
				if err != nil {
					return nil, err
				}
				made := &patcher.Made{Module: pl, Describe: func(string) params.Set { return pl.Describe() }, Close: pl.Close,
					State: func() (string, error) {
						st, err := pl.SaveState()
						return base64.StdEncoding.EncodeToString(st), err
					},
					SetState: func(state string) error {
						st, err := base64.StdEncoding.DecodeString(state)
						if err != nil {
							return err
						}
						return pl.LoadState(st)
					},
					Bind: pl.Bind,
					Status: func() string {
						if sig := pl.Crashed(); sig != 0 {
							return fmt.Sprintf("crashed (signal %d): take it away and add it again", sig)
						}
						if l := pl.Latency(); l > 0 {
							return fmt.Sprintf("latency %d samples (%.1f ms)", l, float64(l)/psalm.CurrentEnv().Config().SampleRate*1000)
						}
						return ""
					}}
				if pl.HasWindow() { // what a double click opens: its own window (Alt: our settings)
					made.Open = pl.ShowWindow
				}
				// a plugin with more audio ports than one has them as ports of the module: its outputs (a sampler with one for every part) and a side chain
				insN, outsN := pl.PortNames()
				if len(outsN) > 1 {
					for i, name := range outsN {
						if i == 0 {
							name = "out"
						}
						made.Outs = append(made.Outs, patcher.Port{Name: name, Type: patcher.Audio})
					}
				}
				if pl.NumInputs() > 1 && len(insN) > 1 && k.wants(d) && d.Effect() {
					made.Ins = []patcher.Port{{Name: "in", Type: patcher.Audio}, {Name: insN[1], Type: patcher.Audio}}
				}
				return made, nil
			}})
	}
	return kinds
}
