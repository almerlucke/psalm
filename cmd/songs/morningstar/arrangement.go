package main

import "github.com/almerlucke/psalm/modules/controls/arrange"

// arrangement automates the piece: dawn and dusk on the strings, the frozen harp of the interlude,
// the piano's pedal and a ritardando over the last four bars
func (o *orchestra) arrangement() *arrange.Arrangement {
	coda := form.Span("Coda")
	interlude := form.Span("Interlude")

	return arrange.NewArrangement(form).
		Tempo(arrange.Sections{"Coda": coda.From(4, arrange.Smooth(bpm, 52))}).
		AutomateMessage(arrange.Sections{
			"Prelude": arrange.Exp(250, 12000), // dawn
			"Coda":    arrange.Exp(12000, 400), // dusk
		}.Else(12000), o.stringFilter, "cutoff").
		// the granulator mixes into the harp in the interlude and freezes halfway; it mixes in at
		// equal power, the harp strip takes back the level it adds
		AutomateMessage(arrange.Sections{"Interlude": arrange.Const(0.63)}, o.gran, "mix").
		AutomateMessage(arrange.Sections{"Interlude": interlude.On(4, 8)}, o.gran, "freeze").
		AutomateMessage(arrange.Sections{"Interlude": arrange.Const(-5.8)}.Else(-3), o.harpStrip, "volume").
		// the piano's sustain pedal changes at every bar line where it plays, up where it is silent
		AutomateMessage(arrange.Sections{
			"Prelude": form.Span("Prelude").Pedal(0, 4),
			"Chorale": form.Span("Chorale").Pedal(0, 8),
			"Coda":    coda.Pedal(0, 8),
		}, o.grand, "sustain").
		PrintSections(true)
}
