// Package shapeview is a plugin of the panel (modules/controls/panel): the editor of the chains of the shaper synth (synths/shaper). Four chains (A to D) are
// each a rack of cards, one for every stage, with knobs for their numbers (a number may be a macro m1 to m4 with a range), drawn as plots: the wave of the
// chain, the curve of the stage you pick, the harmonics, and the four chains along the morph; the envelopes (amp, morph, m1, pitch) are curves to drag, with
// buttons for fades; the text of a chain stays the truth (the rack and the text are two views of one string).
package shapeview

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/almerlucke/psalm/blocks/shapechain"
	"github.com/almerlucke/psalm/modules/controls/panel"
)

//go:embed shapeview.js
var script string

//go:embed shapeview.css
var styles string

// Plugin is the plugin (Add uses it)
var Plugin = panel.Plugin{Name: "shapeview", JS: script, CSS: styles}

// Host is what the view works on: the texts of the chains and the settings of the synth
type Host struct {
	ChainKeys [4]string // the keys of the texts (chainA...)
	Text      func(key string) string
	TextErr   func(key string) string
	SetText   func(key, value string) error
	Values    func() map[string]float64
	Apply     func(values map[string]float64)
	Play      func(pitch, seconds float64)
}

const (
	wavePoints  = 256
	curvePoints = 96
	harmonics   = 32
)

// stagePlot is what one card shows: the wave after it (the chain up to it, over a cycle) and its own curve over -1 to 1
type stagePlot struct {
	Text  string    `json:"text"`
	Wave  []float64 `json:"wave,omitempty"`
	Curve []float64 `json:"curve,omitempty"`
	Err   string    `json:"err,omitempty"`
}

// chainPlot is what the page draws of a chain
type chainPlot struct {
	Text     string      `json:"text"`
	Err      string      `json:"err,omitempty"`
	Wave     []float64   `json:"wave,omitempty"`
	Spectrum []float64   `json:"spectrum,omitempty"`
	Stages   []stagePlot `json:"stages"`
	Harm     int         `json:"harm,omitempty"` // the harmonics that matter; shapechain.Wide for an edge
	Macros   [4]bool     `json:"macros"`
}

// state is what the page gets as things change
type state struct {
	Chains   [4]chainPlot       `json:"chains"`
	Used     int                `json:"used"` // the chains in use (A first)
	Blend    []float64          `json:"blend,omitempty"`
	Spectrum []float64          `json:"spectrum,omitempty"`
	Values   map[string]float64 `json:"values"`
}

// View is the editor
type View struct {
	h Host

	mu    sync.Mutex
	key   string
	cache state
}

// catalog is what the page is given once
type catalog struct {
	Stages []shapechain.StageInfo `json:"stages"`
	Keys   [4]string              `json:"keys"`
}

// Add adds the editor to a panel (and the plugin to its page)
func Add(page *panel.Panel, name string, h Host) *View {
	page.Use(Plugin)
	v := &View{h: h}
	page.Custom("shapeview", name+" shapes", nil, panel.NoDefault(),
		panel.Data(catalog{Stages: shapechain.Catalog, Keys: h.ChainKeys}),
		panel.Live(v.state),
		panel.OnMessage(func(raw json.RawMessage) { go v.message(raw) }))
	return v
}

// message is what the page sends: {"text": {"key": "chainA", "value": "..."}}, {"set": {"morph": 0.3}}, {"play": 60}
func (v *View) message(raw json.RawMessage) {
	var m struct {
		Text *struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"text"`
		Set  map[string]float64 `json:"set"`
		Play *float64           `json:"play"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return
	}
	switch {
	case m.Text != nil:
		for _, k := range v.h.ChainKeys {
			if k == m.Text.Key {
				_ = v.h.SetText(k, m.Text.Value)
			}
		}
	case m.Set != nil:
		v.h.Apply(m.Set)
	case m.Play != nil:
		v.h.Play(*m.Play, 1.5)
	}
}

// state works out what the page shows; the plots are only made again when the chains or the macros or the morph changed
func (v *View) state() any {
	vals := v.h.Values()
	var texts [4]string
	var errs [4]string
	key := strings.Builder{}
	for i, k := range v.h.ChainKeys {
		texts[i], errs[i] = v.h.Text(k), v.h.TextErr(k)
		fmt.Fprintf(&key, "%s\x00%s\x01", texts[i], errs[i])
	}
	morph := vals["morph"]
	macros := shapechain.Macros{vals["macro1"], vals["macro2"], vals["macro3"], vals["macro4"]}
	fmt.Fprintf(&key, "%.3f %.3f %.3f %.3f %.3f", morph, macros[0], macros[1], macros[2], macros[3])
	v.mu.Lock()
	defer v.mu.Unlock()
	if key.String() != v.key {
		v.key = key.String()
		v.cache = plot(texts, errs, morph, macros)
	}
	out := v.cache
	out.Values = vals
	return out
}

// plot works out the plots of the four chains and the blend of them at a morph
func plot(texts, errs [4]string, morph float64, macros shapechain.Macros) state {
	var st state
	var chains [4]*shapechain.Chain
	for i, t := range texts {
		cp := chainPlot{Text: t, Err: errs[i], Stages: []stagePlot{}}
		if strings.TrimSpace(t) == "" {
			st.Chains[i] = cp
			continue
		}
		c, err := shapechain.Parse(t)
		if err != nil {
			if cp.Err == "" {
				cp.Err = err.Error()
			}
			// the stages that read are still shown
			cp.Stages = stagePlots(t, macros)
			st.Chains[i] = cp
			continue
		}
		chains[i] = c
		w := c.Wave(wavePoints, macros)
		cp.Spectrum = round(shapechain.Spectrum(w, harmonics), 1e4)
		cp.Wave = round(w, 1e4)
		cp.Harm = c.Bandwidth()
		for m := 1; m <= 4; m++ {
			cp.Macros[m-1] = c.Uses(m)
		}
		cp.Stages = stagePlots(t, macros)
		st.Chains[i] = cp
	}
	for i := range chains {
		if chains[i] != nil && (i == 0 || chains[i-1] != nil) {
			st.Used = i + 1
		}
	}
	if st.Used > 0 {
		f := max(0, min(morph, 1)) * float64(st.Used-1)
		i := min(int(f), st.Used-1)
		frac := f - float64(i)
		a := st.Chains[i].Wave
		blend := make([]float64, len(a))
		copy(blend, a)
		if i+1 < st.Used && frac > 0 {
			b := st.Chains[i+1].Wave
			for k := range blend {
				blend[k] += (b[k] - blend[k]) * frac
			}
		}
		st.Blend = round(blend, 1e4)
		st.Spectrum = round(shapechain.Spectrum(blend, harmonics), 1e4)
	}
	return st
}

// stagePlots is the plots of the cards of a chain: the wave after each stage, and the curve of the stage alone
func stagePlots(text string, macros shapechain.Macros) []stagePlot {
	stages := shapechain.SplitStages(text)
	out := make([]stagePlot, len(stages))
	for i, s := range stages {
		sp := stagePlot{Text: s}
		if c, err := shapechain.Parse(strings.Join(stages[:i+1], " | ")); err == nil {
			sp.Wave = decimate(c.Wave(wavePoints, macros), 64)
		} else {
			sp.Err = err.Error()
		}
		if c, err := shapechain.Parse(s); err == nil {
			sp.Curve = c.Curve(curvePoints, -1, 1, macros)
			for k, y := range sp.Curve { // keep the numbers short
				sp.Curve[k] = math.Round(y*1000) / 1000
			}
		}
		out[i] = sp
	}
	return out
}

func decimate(w []float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = math.Round(w[i*len(w)/n]*1000) / 1000
	}
	return out
}

// round keeps the numbers of a plot short (a page gets twenty a second): to a multiple of 1/scale, in place
func round(w []float64, scale float64) []float64 {
	for i, v := range w {
		w[i] = math.Round(v*scale) / scale
	}
	return w
}
