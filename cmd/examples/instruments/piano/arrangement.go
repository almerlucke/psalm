package main

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/arrange"
)

// newArrangement sets the tempo of every section and plays the pedals: the sustain pedal changed
// at every bar line (up for a moment to clear the chord, down for the bar), half pedaling, the
// sostenuto and the soft pedal
func newArrangement(form *arrange.Form, pianos map[string]psalm.Modular) *arrange.Arrangement {
	grand := pianos["grand"]
	a := arrange.NewArrangement(form).
		Tempo(arrange.Sections{
			keyboard: arrange.Const(72),
			pedal:    arrange.Const(80),
			sost:     arrange.Const(72),
			soft:     arrange.Const(66),
			pop:      arrange.Const(100),
			hymn:     arrange.Const(70),
			boogie:   arrange.Const(126),
			ambient:  arrange.Const(60),
			musicBox: arrange.Const(96),
		}.Else(66)).
		AutomateMessage(arrange.Sections{
			nocturne: form.Span(nocturne).Pedal(0, 8),
			keyboard: arrange.Const(1),
			pedal:    form.Span(pedal).Steps(0, 0, 0.5, 1), // dry, half pedal, down
			soft:     arrange.Span(2).Pedal(0, 2),          // changed at every phrase of two bars
		}, grand, "sustain").
		AutomateMessage(arrange.Sections{soft: form.Span(soft).Steps(0, 0, 1)}, grand, "soft"). // the phrase again, una corda
		// the sostenuto goes down while the low C is held (its first beat) and up at the end
		AutomateMessage(arrange.Sections{sost: form.Span(sost).On(0.12, 3.88)}, grand, "sostenuto").
		AutomateMessage(arrange.Sections{hymn: form.Span(hymn).Pedal(0, 4)}, pianos["upright"], "sustain").
		AutomateMessage(arrange.Sections{ambient: arrange.Const(1)}, pianos["felt"], "sustain").
		PrintSections(true)
	for _, pr := range parade {
		a.AutomateMessage(arrange.Sections{pr.section: form.Span(pr.section).Pedal(0, 4)}, pianos[pr.name], "sustain")
	}
	return a
}
