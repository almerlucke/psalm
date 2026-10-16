package main

import (
	"fmt"
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/synths/sampler"
)

// the verses are spoken: a clip for every chapter (see assets/sounds/exodus), a synthesized voice
// reading the King James text the song prints. The clip of a chapter starts a number of bars into it

// voiceDip is how far the music gives way (dB) while the voice speaks
const voiceDip = 6.0

// voiceRMS is the loudness every clip is brought to (the rms of the sound where it speaks)
const voiceRMS = 0.1

type reading struct {
	chapter string
	file    string
	start   float64 // the bar of the chapter the voice starts at
	gain    float64 // dB the voice is louder (or softer) in this chapter than the strip's volume
	dip     float64 // dB the music gives way in this chapter, 0 is voiceDip
}

var readings = []reading{
	{egypt, "00-egypt", 2, 0, 0},
	{night, "01-night", 1, 0, 0},
	{exodus, "02-exodus", 1, 0, 0},
	{pursuit, "03-pursuit", 1, 0, 0},
	{sea, "04-sea", 1, 3, 12},
	{waters, "05-waters", 0, 3, 12},
	{miriam, "06-miriam", 2, 0, 0},
	{wilderness, "07-wilderness", 2, 5, 10},
	{sinai, "08-sinai", 1, 0, 0},
	{years, "09-years", 2, 5, 14},
	{land, "10-land", 2, 0, 0},
}

// voiceKey is the key of a chapter's clip
func voiceKey(i int) int { return 36 + i }

// voiceLength is the length of every clip in seconds, by chapter
var voiceLength = map[string]float64{}

// spokenRMS is the rms of the sound where it isn't a pause
func spokenRMS(snd sampler.Sound) float64 {
	sum, n := 0.0, 0
	for _, x := range snd.Buffer(0, 0) {
		if math.Abs(float64(x)) > 0.01 {
			sum += float64(x) * float64(x)
			n++
		}
	}
	if n == 0 {
		return voiceRMS
	}
	return math.Sqrt(sum / float64(n))
}

// newVoice is the sampler that holds the clips, one on each key, played to their end
func newVoice(p *psalm.Psalm) psalm.Modular {
	var regions []sampler.Region
	for i, r := range readings {
		snd, err := sampler.LoadFile(fmt.Sprintf("assets/sounds/exodus/%s.wav", r.file))
		if err != nil {
			panic(err)
		}
		voiceLength[r.chapter] = float64(snd.NumFrames()) / snd.SampleRate()
		reg := sampler.NewRegion(snd).Keys(voiceKey(i), voiceKey(i), float64(voiceKey(i)))
		reg.Volume = 20*math.Log10(voiceRMS/spokenRMS(snd)) + r.gain
		reg.KeyTrack, reg.VelTrack, reg.Loop = 0, 0, sampler.OneShot
		reg.Env = sampler.Envelope{Attack: 0.08, Sustain: 1, Release: 0.3}
		regions = append(regions, reg)
	}
	cfg := sampler.DefaultConfig(sampler.MustInstrument(regions...))
	cfg.NumChannels = psalm.Stereo
	return sampler.NewPoly(cfg, 2).AddTo(p)
}

// voicePlays is the voice's part: a note on the chapter's key, a bar in
func voicePlays() plays {
	m := plays{}
	for i, r := range readings {
		m[r.chapter] = every(within(int(r.start), int(r.start)+1, func(b int, br bar) []N {
			return []N{{Pitches: []float64{float64(voiceKey(i))}, Beats: 4, Hold: 4, Velocity: 1}}
		}))
	}
	return m
}

// voiceDips is the dip of the music (dB) over every chapter: down a little before the voice starts, up
// again a little after it ends
func voiceDips() arrange.Sections {
	s := arrange.Sections{}
	for _, r := range readings {
		c := chapter_(r.chapter)
		n := form.Span(r.chapter)
		end := r.start + voiceLength[r.chapter]*c.bpm/240 // the bars it speaks (the tempo of the chapter)
		smooth := func(x float64) float64 {
			x = max(0, min(x, 1))
			return x * x * (3 - 2*x)
		}
		length := float64(n)
		start := r.start
		dip := voiceDip
		if r.dip != 0 {
			dip = r.dip
		}
		s[r.chapter] = func(progress float64) float64 {
			b := progress * length
			return -dip * (smooth((b-(start-0.6))/0.6) - smooth((b-end)/0.6))
		}
	}
	return s
}
