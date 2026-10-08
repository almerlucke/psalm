package main

import (
	"fmt"

	"github.com/almerlucke/psalm/modules/controls/arrange"
)

// arrangement follows the days: the light, the voices of creation coming in, the piano's pedal,
// the rest of the seventh day
func (o *orchestra) arrangement() *arrange.Arrangement {
	day := form.Span(day1) // every day but the seventh: 12 bars
	rest := form.Span(day7)

	return arrange.NewArrangement(form).
		// the seventh day rests: slower, and slower still as the last chord rings
		Tempo(arrange.Sections{day7: rest.Last(4, arrange.Smooth(restBPM, 48))}).

		// "Let there be light": the darkness lifts in the first bar of the first morning
		AutomateMessage(arrange.Sections{day1: day.Between(evening, evening+1, arrange.Exp(300, 18000))}.Else(18000), o.darkness, "cutoff").

		// the Word is spoken twice into the darkness of the first evening
		At(day1, 2, o.word.Bang).
		At(day1, 4, o.word.Bang).

		// the deep, the waters, the seas gathered together (carried on from the second day into the third)
		AutomateMessage(arrange.Sections{
			day1: day.Fade(-14, 0, evening),
			day2: day.Fade(-8, 0, 13),
			day3: day.Fade(-8, -1, evening),
		}.Else(arrange.Silent), o.waves, "volume").
		// the creatures of the sea and the birds
		AutomateMessage(arrange.Sections{day5: day.Fade(-8, 0, 12)}.Else(arrange.Silent), o.seaBirds, "volume").
		AutomateMessage(arrange.Sections{day5: day.Fade(-16, evening, 12)}.Else(arrange.Silent), o.gulls, "volume").
		// the beasts of the earth
		AutomateMessage(arrange.Sections{day6: day.Fade(-10, 0, 6.6)}.Else(arrange.Silent), o.animals, "volume").
		// "be fruitful, and multiply" (Genesis 1:28): the children laugh through the last four bars of
		// the sixth day
		At(day6, 8, o.laughter.Bang).
		AutomateMessage(arrange.Sections{day6: day.Fade(-3, 8, 12)}.Else(arrange.Silent), o.children, "volume").

		// the piano's sustain pedal changes at every bar of the sixth morning
		AutomateMessage(arrange.Sections{day6: day.Pedal(evening, 12)}, o.grand, "sustain").
		OnSection(func(pos arrange.Position) {
			fmt.Printf("\n%s\n%s\n", pos.Section.Name, pos.Section.Params["verse"])
		})
}
