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
	arr := arrange.NewArrangement(form).
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
		// the granulators bloom where the story opens out and stay lower elsewhere
		AutomateMessage(arrange.Sections{
			night: arrange.Const(0.4), sea: arrange.Const(0.7), waters: arrange.Const(0.5), wilderness: arrange.Const(0.6),
			sinai: arrange.Const(0.4), years: arrange.Const(0.7), land: arrange.Const(0.55),
		}.Else(0.25), o.glassCloud, "mix").
		AutomateMessage(arrange.Sections{
			sea: arrange.Const(0.45), wilderness: arrange.Const(0.55), years: arrange.Const(0.65), land: arrange.Const(0.45),
		}.Else(0), o.shakuCloud, "mix")
	// the shimmer: the heavens open on the sea, the waters, the desert, the years and the land
	for name, v := range map[string]struct {
		base float64
		in   map[string]float64
	}{
		"glass harmonica":  {0.2, map[string]float64{sea: 0.8, waters: 0.7, wilderness: 0.6, years: 0.6, land: 0.7}},
		"bowed vibraphone": {0.2, map[string]float64{sea: 0.8, wilderness: 0.6, years: 0.6}},
		"space tine":       {0.45, map[string]float64{sea: 0.85, waters: 0.95}},
		"alto flute":       {0, map[string]float64{years: 0.3, land: 0.5}},
		"bansuri":          {0.1, map[string]float64{sea: 0.5, wilderness: 0.45, land: 0.5}},
		"shakuhachi":       {0.15, map[string]float64{years: 0.6, land: 0.4}},
		"harp":             {0.1, map[string]float64{sea: 0.5, land: 0.5}},
		"piano":            {0, map[string]float64{sea: 0.25, waters: 0.3, land: 0.3}},
		"glockenspiel":     {0.3, map[string]float64{sea: 0.6, land: 0.6}},
		"singing bowl":     {0.3, map[string]float64{wilderness: 0.6, years: 0.5}},
	} {
		secs := arrange.Sections{}
		for sec, x := range v.in {
			secs[sec] = arrange.Const(x)
		}
		arr = arr.AutomateMessage(secs.Else(v.base), o.mix.Strip(name), "send.heavens")
	}
	// the shimmer draws back over the last bars, so that what rings out at the end is the bells and the gong
	arr = arr.AutomateMessage(arrange.Sections{land: form.Span(land).Last(6, arrange.Linear(0, -26))}.Else(0), o.mix.Strip("heavens"), "volume")
	return arr.
		OnSection(func(pos arrange.Position) {
			fmt.Printf("\n%s\n%s\n", pos.Section.Name, pos.Section.Params["verse"])
		})
}
