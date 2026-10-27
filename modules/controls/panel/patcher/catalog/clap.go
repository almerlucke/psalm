package catalog

import (
	"encoding/base64"
	"fmt"
	"slices"
	"sync"

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
				return &patcher.Made{Module: pl, Describe: func(string) params.Set { return pl.Describe() }, Close: pl.Close,
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
					Bind: pl.Bind}, nil
			}})
	}
	return kinds
}
