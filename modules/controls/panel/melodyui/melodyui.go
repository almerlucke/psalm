// Package melodyui is the editor of a melodist: what it plays now (a small piano roll of the pass, with the chords over it and a playhead), the settings in plain words (sliders from
// one thing to its opposite, the shape of a line as pictures, the range on a keyboard), where in the song it plays (a chip for every section), how much it changes from pass to pass,
// presets, and the buttons that draw it again, freeze it and keep it in a note editor. The editor is a plugin of the panel (modules/controls/panel); the module it edits gives it its
// state and takes what the page asks (Controller).
package melodyui

import (
	_ "embed"
	"encoding/json"

	"github.com/almerlucke/psalm/modules/controls/panel"
)

//go:embed melodyui.js
var script string

//go:embed melodyui.css
var styles string

// Plugin is the plugin (Add uses it)
var Plugin = panel.Plugin{Name: "melodist", JS: script, CSS: styles}

// Kind is the kind of its controls
const Kind panel.Kind = "melodist"

// Controller is what the editor edits: View is what the page gets (anything that turns into JSON; asked as the page updates, so quick), Message what it asks (the raw JSON of the
// page's message)
type Controller interface {
	View() any
	Message(raw json.RawMessage)
}

// Add adds the editor of a melodist to a panel (and the plugin to its page)
func Add(ctl *panel.Panel, name string, c Controller) *panel.Param {
	ctl.Use(Plugin)
	return ctl.Custom(Kind, name, nil, panel.NoDefault(),
		panel.Live(func() any { return c.View() }),
		panel.OnMessage(func(raw json.RawMessage) { c.Message(raw) }))
}
