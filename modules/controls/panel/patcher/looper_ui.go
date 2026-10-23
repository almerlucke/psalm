package patcher

import (
	"encoding/json"
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/editor"
	"github.com/almerlucke/psalm/params"

	_ "embed"
)

//go:embed looper.js
var looperScript string

//go:embed looper.css
var looperStyles string

// LooperPlugin is the page of a looper: the bars with the beat it is at, and the layers, each to take away
var LooperPlugin = panel.Plugin{Name: "looper", JS: looperScript, CSS: looperStyles}

// looperView is what the page shows of the layers: notes as [start, length, pitch] in beats, copied when it changes, so the page reads it from another goroutine
type looperView struct {
	Layers    []layerView  `json:"layers"`
	Cur       [][3]float64 `json:"cur"` // the layer being recorded
	Recording bool         `json:"recording"`
}

type layerView struct {
	ID    int          `json:"id"`
	Notes [][3]float64 `json:"notes"`
}

func (l *Looper) beats(notes []loopNote) [][3]float64 {
	ppq := float64(l.Config().PPQ)
	out := make([][3]float64, len(notes))
	for i, n := range notes {
		out[i] = [3]float64{math.Round(float64(n.start)/ppq*1000) / 1000, math.Round(float64(n.length)/ppq*1000) / 1000, n.pitch}
	}
	return out
}

// publish makes the view of the layers (on the audio thread, when they change)
func (l *Looper) publish() {
	v := &looperView{Layers: make([]layerView, len(l.layers)), Cur: l.beats(l.cur), Recording: l.recording}
	for i, layer := range l.layers {
		v.Layers[i] = layerView{ID: l.ids[i], Notes: l.beats(layer)}
	}
	l.view.Store(v)
}

// looperState is what the page gets as the loop plays
type looperState struct {
	*looperView
	Bars  int     `json:"bars"`
	Pos   float64 `json:"pos"` // beats into the loop
	BPS   float64 `json:"bps"` // beats a second
	Armed bool    `json:"armed"`
}

func (l *Looper) state() any {
	v := l.view.Load()
	if v == nil {
		v = &looperView{Layers: []layerView{}, Cur: [][3]float64{}}
	}
	return looperState{looperView: v, Bars: max(1, int(l.shape.Load())), BPS: l.BPM() / 60, Armed: l.armed.Load(),
		Pos: math.Round(float64(l.where.Load())/float64(l.Config().PPQ)*1000) / 1000}
}

// Drop asks for a layer to be taken away, by its number (done in the next tick)
func (l *Looper) Drop(id int) {
	l.mu.Lock()
	l.drops = append(l.drops, id)
	l.mu.Unlock()
}

// AddView adds the page of the looper to a panel: the bars and the layers
func (l *Looper) AddView(page *panel.Panel, name string) *panel.Param {
	page.Use(LooperPlugin)
	return page.Custom("looper", name+" loop", nil, panel.NoDefault(),
		panel.Live(l.state),
		panel.OnMessage(func(raw json.RawMessage) {
			var m struct {
				Drop *int `json:"drop"`
			}
			if json.Unmarshal(raw, &m) == nil && m.Drop != nil {
				l.Drop(*m.Drop)
			}
		}))
}

// edit makes the page of the looper: its settings, then the loop with its layers
func (l *Looper) edit(page *panel.Panel, name string, set params.Set) Settings {
	ed := editor.Add(page, name, l, set, panel.NoDefault())
	l.released = func(key string) {
		if c := ed.Setting(key); c != nil {
			c.Place(0)
		}
	}
	page.Section("Loop")
	l.AddView(page, name)
	return ed
}

var _ psalm.Receiver = (*Looper)(nil)
