package main

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/dxeditor"
	"github.com/almerlucke/psalm/modules/controls/panel/editor"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/synths/drumsynth"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/pd"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/sampler"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/supersaw"
	"github.com/almerlucke/psalm/synths/vocal"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/westcoast"
)

// describer describes a synth at its presets: their names, and one described by name
type describer struct {
	presets  []string
	describe func(preset string) params.Set
}

func describing[T any](presets map[string]T, describe func(preset string) params.Set) describer {
	return describer{presets: slices.Sorted(maps.Keys(presets)), describe: describe}
}

// describers are the synths the editor of a channel describes (the DX7 has its own editor)
var describers = map[string]describer{
	"piano":       describing(piano.Presets, func(n string) params.Set { return piano.Describe(piano.Presets[n]()) }),
	"fm":          describing(fm.Presets, func(n string) params.Set { return fm.Describe(fm.Presets[n]()) }),
	"organ":       describing(organ.Presets, func(n string) params.Set { return organ.Describe(organ.Presets[n]()) }),
	"waveguide":   describing(waveguide.Presets, func(n string) params.Set { return waveguide.Describe(waveguide.Presets[n]()) }),
	"modal":       describing(modal.Presets, func(n string) params.Set { return modal.Describe(modal.Presets[n]()) }),
	"westcoast":   describing(westcoast.Presets, func(n string) params.Set { return westcoast.Describe(westcoast.Presets[n]()) }),
	"subtractive": describing(subtractive.Presets, func(n string) params.Set { return subtractive.Describe(subtractive.Presets[n]()) }),
	"supersaw":    describing(supersaw.Presets, func(n string) params.Set { return supersaw.Describe(supersaw.Presets[n]()) }),
	"wavetable": describing(wavetable.Presets, func(n string) params.Set {
		return wavetable.Describe(wavetable.Presets[n](wavetableTable))
	}),
	"pd":     describing(pd.Presets, func(n string) params.Set { return pd.Describe(pd.Presets[n]()) }),
	"pulsar": describing(pulsar.Presets, func(n string) params.Set { return pulsar.Describe(pulsar.Presets[n]()) }),
	"vocal":  describing(vocal.Presets, func(n string) params.Set { return vocal.Describe(vocal.Presets[n]()) }),
	"drumsynth": describing(drumsynth.Kits, func(n string) params.Set {
		return drumsynth.Describe(drumsynth.Config{Kit: drumsynth.Kits[n]()})
	}),
}

// later is a patch that keeps what is added to it, to add it to the real one later (between two
// buffers, with Psalm.Do): a page made off the audio thread
type later struct {
	psalm.Patcher
	added []psalm.Modular
}

func (l *later) Add(m psalm.Modular) psalm.Patcher {
	l.added = append(l.added, m)
	return l
}

// editPage makes the page of a sound of a channel: its editor (the DX editor for the DX7), its
// presets and yours (kept in dir/<synth>, the same as the editors example's). The page isn't in the
// patch yet: add what the returned patcher kept (p.Add) where it is safe
func editPage(s *sound, theme panel.Theme, dir string) (*panel.Panel, []psalm.Modular) {
	synth, preset, _ := strings.Cut(s.name, "/")
	l := &later{}
	page := panel.New(l, s.name)
	page.SetTheme(theme)
	switch {
	case synth == "dx":
		patch := dx.Presets[preset]()
		patch.Name = preset
		var bank []dx.Patch
		for _, n := range slices.Sorted(maps.Keys(dx.Presets)) {
			b := dx.Presets[n]()
			b.Name = n
			bank = append(bank, b)
		}
		ed := dxeditor.Add(page, s.name, s.synth.(*poly.Poly[*dx.Voice]), patch, dxeditor.Bank(bank))
		_ = ed.Keep(filepath.Join(dir, synth))
	case synth == "sfz":
		ed := editor.Add(page, s.name, s.synth, sampler.Describe(sampler.DefaultConfig(nil)), panel.NoDefault())
		_ = ed.Keep(filepath.Join(dir, "sampler"))
	default:
		d, ok := describers[synth]
		if !ok {
			return nil, nil
		}
		ed := editor.Add(page, s.name, s.synth, d.describe(preset), panel.NoDefault())
		ed.Factory(d.presets, d.describe)
		_ = ed.Keep(filepath.Join(dir, synth))
	}
	return page, l.added
}

// editing is a mount showing the page of the sound a channel plays, the channel chosen
type editing struct {
	parts []*part
	now   atomic.Int64
}

func (e *editing) Mounted() *panel.Panel {
	if i := int(e.now.Load()); i >= 0 && i < len(e.parts) {
		return e.parts[i].page.Load()
	}
	return nil
}
