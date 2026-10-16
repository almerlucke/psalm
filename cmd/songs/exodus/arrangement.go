package main

import (
	"fmt"

	"github.com/almerlucke/psalm/modules/controls/arrange"
)

// arrangement follows the story: the tempo of every chapter, the piano's pedal, and the slowing
// of the last bars
func (o *orchestra) arrangement() *arrange.Arrangement {
	tempo := arrange.Sections{}
	for _, c := range chapters {
		tempo[c.name] = arrange.Const(c.bpm)
	}
	// the sea draws back slowly, the years drag, the promised land is reached with a breath
	tempo[egypt] = form.Span(egypt).Between(0, 12, arrange.Linear(58, 54))
	tempo[wilderness] = form.Span(wilderness).Between(0, 12, arrange.Linear(52, 48))
	tempo[land] = form.Span(land).Last(5, arrange.Smooth(60, 40))
	tempo[waters] = form.Span(waters).Last(1, arrange.Smooth(132, 90))

	pedal := func(name string, from, to float64) arrange.Curve { return form.Span(name).Pedal(from, to) }
	return arrange.NewArrangement(form).
		Tempo(tempo).
		AutomateMessage(arrange.Sections{
			egypt:  pedal(egypt, 0, 12),
			night:  pedal(night, 0, 8),
			sea:    pedal(sea, 0, 12),
			waters: pedal(waters, 0, 3),
			sinai:  pedal(sinai, 0, 8),
			years:  pedal(years, 4, 8),
			land:   pedal(land, 0, 16),
		}, o.grand, "sustain").
		// the granulators bloom where the story opens out (the sea, the desert, the years, the land) and
		// stay at a whisper elsewhere
		AutomateMessage(arrange.Sections{
			night: arrange.Const(0.25), sea: arrange.Const(0.5), waters: arrange.Const(0.3), wilderness: arrange.Const(0.5),
			sinai: arrange.Const(0.3), years: arrange.Const(0.6), land: arrange.Const(0.4),
		}.Else(0.15), o.glassCloud, "mix").
		AutomateMessage(arrange.Sections{
			sea: arrange.Const(0.3), wilderness: arrange.Const(0.4), years: arrange.Const(0.45), land: arrange.Const(0.3),
		}.Else(0), o.shakuCloud, "mix").
		OnSection(func(pos arrange.Position) {
			fmt.Printf("\n%s\n%s\n", pos.Section.Name, pos.Section.Params["verse"])
		})
}
