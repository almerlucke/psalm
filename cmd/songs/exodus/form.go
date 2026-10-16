package main

import (
	"fmt"

	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/tools/harmony"
)

// the sections of the story
const (
	egypt      = "Egypt"
	night      = "The night"
	exodus     = "The exodus"
	pursuit    = "Pharaoh pursues"
	sea        = "The sea"
	waters     = "The waters return"
	miriam     = "Miriam's song"
	wilderness = "The wilderness"
	sinai      = "Sinai"
	years      = "Forty years"
	land       = "The promised land"
)

// a chapter is a section of the story: its bars (a chord each), its tempo and its verse (KJV)
type chapter struct {
	name   string
	chords string
	bpm    float64
	verse  string
}

var chapters = []chapter{
	{egypt, "Dm Dm Gm Dm Bb A Dm Gm Eb A Dm A", 58,
		"And the Egyptians made the children of Israel to serve with rigour: and they made their lives bitter with hard bondage, in morter, and in brick, and in all manner of service in the field. (Exodus 1:13-14)"},
	{night, "Dm Bb Gm A Dm Bb A A", 72,
		"And it came to pass, that at midnight the LORD smote all the firstborn in the land of Egypt. And he called for Moses and Aaron by night, and said, Rise up, and get you forth from among my people. (Exodus 12:29, 31)"},
	{exodus, "Dm Dm F F Bb F C C Dm Gm F C Bb F C Dm", 108,
		"And the LORD went before them by day in a pillar of a cloud, to lead them the way; and by night in a pillar of fire, to give them light; to go by day and night. (Exodus 13:21)"},
	{pursuit, "Dm Dm Bb Bb Gm A Dm A", 126,
		"But the Egyptians pursued after them, all the horses and chariots of Pharaoh, and overtook them encamping by the sea. And the children of Israel cried out unto the LORD. (Exodus 14:9-10)"},
	{sea, "Gm Eb Bb F Gm Eb F A Dm Bb F A", 54,
		"And the LORD caused the sea to go back by a strong east wind all that night, and made the sea dry land, and the waters were divided. And the children of Israel went into the midst of the sea upon the dry ground: and the waters were a wall unto them on their right hand, and on their left. (Exodus 14:21-22)"},
	{waters, "Dm Dm A A", 132,
		"And the waters returned, and covered the chariots, and the horsemen, and all the host of Pharaoh: there remained not so much as one of them. (Exodus 14:28)"},
	{miriam, "F C Bb F Dm Gm C F F Bb C F", 120,
		"And Miriam the prophetess took a timbrel in her hand; and all the women went out after her with timbrels and with dances. And Miriam answered them, Sing ye to the LORD, for he hath triumphed gloriously. (Exodus 15:20-21)"},
	{wilderness, "Dm Am C Dm Gm Dm Bb C Dm Am Bb A", 52,
		"Then said the LORD unto Moses, Behold, I will rain bread from heaven for you. (Exodus 16:4)"},
	{sinai, "Dm Bb Dm Gm Bb A Dm A", 66,
		"And it came to pass on the third day in the morning, that there were thunders and lightnings, and a thick cloud upon the mount, and the voice of the trumpet exceeding loud. (Exodus 19:16)"},
	{years, "Dm Am Bb A Dm Gm A Dm", 50,
		"And thou shalt remember all the way which the LORD thy God led thee these forty years in the wilderness. (Deuteronomy 8:2)"},
	{land, "Dm Gm A A D G A D Bm Em A D G A D D", 60,
		"For the LORD thy God bringeth thee into a good land, a land of brooks of water, of fountains and depths that spring out of valleys and hills. And all the Israelites passed over on dry ground, until all the people were passed clean over Jordan. (Deuteronomy 8:7, Joshua 3:17)"},
}

const bpm = 58 // the tempo the song starts in

// how loud every chapter plays (1 when it has no entry)
var dynamics = map[string]float64{
	egypt: 0.7, night: 0.85, exodus: 1, pursuit: 1.1, sea: 0.9, waters: 1.2, miriam: 1,
	wilderness: 0.55, sinai: 1.3, years: 0.5, land: 1.25,
}

// the form: every chapter is a section of as many bars as it has chords, four beats to the bar
var form = func() *arrange.Form {
	var secs []arrange.Section
	for _, c := range chapters {
		secs = append(secs, arrange.Section{Name: c.name, Bars: len(harmony.MustProgression(c.chords)), Params: map[string]any{"verse": c.verse}})
	}
	for i := range secs {
		secs[i].Dynamics = dynamics[secs[i].Name]
	}
	f := arrange.NewForm(secs...)
	f.Loop = false
	return f
}()

// bar holds the chord and the voicings of a bar of the song
type bar struct {
	chord harmony.Chord
	high  []float64 // upper voices (harp, vibraphone)
	mid   []float64 // middle voices (organ, piano)
	root  float64   // the bass
}

var bars = func() []bar {
	var bs []bar
	var high, mid []float64
	for _, s := range form.Sections {
		for _, c := range harmony.MustProgression(chapter_(s.Name).chords) {
			if high == nil {
				high, mid = c.Close(65), c.Close(53)
			} else {
				high, mid = harmony.VoiceLead(high, c), harmony.VoiceLead(mid, c)
			}
			bs = append(bs, bar{chord: c, high: high, mid: mid, root: c.BassPitch(33)})
		}
	}
	return bs
}()

func chapter_(name string) chapter {
	for _, c := range chapters {
		if c.name == name {
			return c
		}
	}
	panic(fmt.Sprintf("no chapter %q", name))
}

// at returns a bar of a section
func at(s arrange.Section, b int) bar { return bars[s.StartBar+b] }

// timeline lays out the chord of every bar, for the harmony check
func timeline() []harmony.TimedChord {
	var tl []harmony.TimedChord
	for i, b := range bars {
		tl = append(tl, harmony.TimedChord{Beat: float64(i) * form.BeatsPerBar, Beats: form.BeatsPerBar, Chord: b.chord})
	}
	tl[len(tl)-1].Beats += 12 // the last chord rings on
	return tl
}
