package drumsynth

import (
	"slices"
	"strings"

	"github.com/almerlucke/psalm/params"
)

// soundOrder is the order the sounds of a kit are described in: the drums of a drum machine, then
// the percussion (the sounds of a kit of your own after them, by name)
var soundOrder = []string{"kick", "snare", "clap", "rim", "hat", "open", "tomLow", "tomMid", "tomHigh",
	"cymbal", "crash", "ride", "rideBell", "cowbell", "tambourine", "shaker", "bongoHigh", "bongoLow",
	"congaMute", "congaHigh", "congaLow", "timbaleHigh", "timbaleLow", "agogoHigh", "agogoLow",
	"clave", "woodblockHigh", "woodblockLow", "triangle", "triangleMute"}

// Describe describes the settings of a drum synth set as cfg (see params): the level of the whole,
// and every sound of its kit, each its own group (its pitch, decay, sweep, tone, snap, drive, level
// and pan, by the messages "<sound><Setting>")
func Describe(cfg Config) params.Set {
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	s := params.Set{Name: "drumsynth", Params: []params.Param{
		params.K("Output", "level", 0, 4, level),
	}}
	var names []string
	for _, n := range soundOrder {
		if _, ok := cfg.Kit.Sounds[n]; ok {
			names = append(names, n)
		}
	}
	var own []string
	for n := range cfg.Kit.Sounds {
		if !slices.Contains(names, n) {
			own = append(own, n)
		}
	}
	slices.Sort(own)
	for _, n := range append(names, own...) {
		snd := cfg.Kit.Sounds[n]
		group := strings.ToUpper(n[:1]) + n[1:]
		metal := snd.Kind == Hat || snd.Kind == Cymbal || snd.Kind == Shaker
		pitch := snd.Pitch
		if metal {
			if pitch <= 0 {
				pitch = 1
			}
			s.Params = append(s.Params, params.K(group, n+"Pitch", 0.25, 4, min(4, max(0.25, pitch)), params.Exp(), params.Label("pitch")))
		} else {
			if pitch <= 0 {
				pitch = 100
			}
			s.Params = append(s.Params, params.K(group, n+"Pitch", 20, 8000, min(8000, max(20, pitch)), params.Exp(), params.Unit("Hz"), params.Label("pitch")))
		}
		s.Params = append(s.Params,
			params.K(group, n+"Decay", 0.005, 10, min(10, max(0.005, snd.Decay)), params.Exp(), params.Unit("s"), params.Label("decay")),
			params.K(group, n+"Sweep", 0, 48, min(48, max(0, snd.Sweep)), params.Unit("st"), params.Label("sweep")),
			params.K(group, n+"SweepTime", 0.001, 0.5, min(0.5, max(0.001, snd.SweepTime)), params.Exp(), params.Unit("s"), params.Label("sweep time")),
			params.K(group, n+"Tone", 0, 1, min(1, max(0, snd.Tone)), params.Label("tone")),
			params.K(group, n+"Snap", 0, 1, min(1, max(0, snd.Snap)), params.Label("snap")),
			params.K(group, n+"Drive", 0, 1, min(1, max(0, snd.Drive)), params.Label("drive")),
			params.K(group, n+"Level", 0, 1.5, levelOf(snd), params.Label("level")),
			params.K(group, n+"Pan", -1, 1, min(1, max(-1, snd.Pan)), params.Label("pan")),
		)
	}
	return s
}

// levelOf returns the level of a sound (0 is 1)
func levelOf(snd Sound) float64 {
	if snd.Level <= 0 {
		return 1
	}
	return min(1.5, snd.Level)
}
