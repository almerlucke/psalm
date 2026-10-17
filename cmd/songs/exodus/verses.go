package main

import (
	"fmt"
	"strings"
)

// the languages the verses are printed and spoken in (-lang): the text of every chapter, its title, and
// the clips of the voice (assets/sounds/exodus/<code>)
type language struct {
	code, name string
	titles     map[string]string // the chapter's title, "" is the section's name
	verses     map[string]string // the chapter's verse, "" is the verse of the chapter (English)
}

var languages = []language{
	{code: "en", name: "English (King James)"},
	{code: "nl", name: "Nederlands (Canisius)", titles: titlesNL, verses: versesNL},
}

// lang is the language asked for
var lang = languages[0]

// setLanguage chooses the language by its code
func setLanguage(code string) error {
	var codes []string
	for _, l := range languages {
		if l.code == strings.ToLower(strings.TrimSpace(code)) {
			lang = l
			return nil
		}
		codes = append(codes, l.code+" "+l.name)
	}
	return fmt.Errorf("-lang %q: no such language, the languages are: %s", code, strings.Join(codes, ", "))
}

// verse is the verse of a chapter in the language
func (l language) verse(chapter string) string {
	if v, ok := l.verses[chapter]; ok {
		return v
	}
	return chapter_(chapter).verse
}

// title is the title of a chapter in the language
func (l language) title(chapter string) string {
	if t, ok := l.titles[chapter]; ok {
		return t
	}
	return chapter
}

// the Dutch: the Canisius translation (1939), the verses of the chapter cut where the English is cut; the
// name of God, Jahweh in this translation, is read as "de HEER"
var titlesNL = map[string]string{
	egypt: "Egypte", night: "De nacht", exodus: "De uittocht", pursuit: "Farao zet hen na", sea: "De zee",
	waters: "De wateren keren terug", miriam: "Het lied van Mirjam", wilderness: "De woestijn", sinai: "Sinaï",
	years: "Veertig jaren", land: "Het beloofde land",
}

var versesNL = map[string]string{
	egypt:      "Zo maakten de Egyptenaren de kinderen Israëls met geweld tot hun slaven. (Exodus 1:13-14)",
	night:      "In het holst van de nacht sloeg de HEER al de eerstgeborenen. Daarna ontbood hij Moses en sprak trekt weg met mijn volk. (Exodus 12:29, 31)",
	exodus:     "En de HEER ging voor hen uit, overdag in een wolkkolom, en des nachts in een vuurzuil, om hen voor te lichten. (Exodus 13:21)",
	pursuit:    "De Egyptenaren joegen hen na en bereikten hen bij de zee. Nu werden de kinderen Israëls zeer beangst, en riepen de HEER aan. (Exodus 14:9-10)",
	sea:        "En de HEER wierp de zee terug door een sterke wind. Hij maakte de zee droog land. (Exodus 14:21-22)",
	waters:     "De wateren stroomden terug en overstelpten het leger van Farao. (Exodus 14:28)",
	miriam:     "En Mirjam de profetes, nam de tamboerijn ter hand. Laat ons zingen voor de HEER, want hoog is Hij verheven! (Exodus 15:20-21)",
	wilderness: "Toen sprak de HEER tot Moses: Zie, Ik zal voor u brood uit de hemel doen regenen. (Exodus 16:4)",
	sinai:      "En op de derde dag in de morgen: donderslagen en bliksemflitsen; over de berg een donkere wolk en schetteren van bazuingeschal. (Exodus 19:16)",
	years:      "Denk eens terug aan heel de tocht, die de HEER, uw God, u deze veertig jaren door de woestijn liet maken. (Deuteronomium 8:2)",
	land:       "Waarachtig, de HEER, uw God, brengt u naar een heerlijk land, een land van waterbeken, bronnen en stromen, die in de dalen en op de bergen ontspringen. (Deuteronomium 8:7)",
}
