// Package pianoroll is a plugin of the panel (modules/controls/panel): the notes of a MIDI file
// scrolling past a playhead as a player plays them, coloured by channel, the notes sounding lit,
// the channels muted dimmed. A click or a drag on it moves the player there; the mouse wheel zooms.
// It follows the theme of the page.
//
//	player := midi.NewPlayer(file)
//	ctl.Section("Notes")
//	pianoroll.Add(ctl, "Notes", player)
//
// It is also an example of a plugin: a script registering a kind of control, the notes sent once
// as its data, the position as its live state, the clicks coming back as messages.
package pianoroll

import (
	_ "embed"
	"encoding/json"
	"math"

	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
)

//go:embed pianoroll.js
var script string

//go:embed pianoroll.css
var styles string

// Plugin is the plugin (Add uses it)
var Plugin = panel.Plugin{Name: "pianoroll", JS: script, CSS: styles}

// Kind is the kind of its controls
const Kind panel.Kind = "pianoroll"

// data is what the page gets once: the notes as [start, end, key, channel, velocity]
type data struct {
	Duration float64      `json:"duration"`
	Notes    [][5]float64 `json:"notes"`
	Seconds  float64      `json:"seconds"` // the seconds shown at first
}

// state is what it gets as it plays
type state struct {
	Position float64 `json:"pos"`
	Rate     float64 `json:"rate"` // file seconds a second, 0 stopped
	Muted    []int   `json:"muted"`
}

// Option sets something of the piano roll
type Option func(*data)

// Seconds is how much of the file it shows at once (10 without it)
func Seconds(s float64) Option { return func(d *data) { d.Seconds = s } }

// Add adds a piano roll of the file a player plays to a panel (and the plugin to its page)
func Add(ctl *panel.Panel, name string, player *midi.Player, opts ...Option) *panel.Param {
	ctl.Use(Plugin)
	d := data{Duration: player.Duration(), Seconds: 10, Notes: [][5]float64{}}
	for _, o := range opts {
		o(&d)
	}
	round := func(x float64) float64 { return math.Round(x*1000) / 1000 }
	for _, n := range player.File().Notes() {
		d.Notes = append(d.Notes, [5]float64{round(n.Start), round(n.End), float64(n.Key), float64(n.Channel), float64(n.Velocity)})
	}
	return ctl.Custom(Kind, name, nil, panel.Data(d),
		panel.Live(func() any {
			st := state{Position: math.Round(player.Position()*1000) / 1000, Rate: player.Rate(), Muted: []int{}}
			for ch := 1; ch <= 16; ch++ {
				if player.ChannelMuted(ch) {
					st.Muted = append(st.Muted, ch)
				}
			}
			return st
		}),
		panel.OnMessage(func(raw json.RawMessage) {
			var m struct {
				Seek *float64 `json:"seek"`
			}
			if json.Unmarshal(raw, &m) == nil && m.Seek != nil {
				player.Seek(*m.Seek)
			}
		}))
}
