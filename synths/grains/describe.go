package grains

import (
	"github.com/almerlucke/psalm/params"
)

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params)
func Describe(cfg Config) params.Set {
	level := cfg.Level
	if level == 0 {
		level = 1
	}
	kind := max(0, min(cfg.SourceKind, numSourceKinds-1))
	return params.Set{Name: "grains", Params: []params.Param{
		params.C("Source", "source", SourceNames, kind,
			params.Hint("What the grains play: a recording (the sounds box), a single cycle wave, or the sound coming into a grain buffer module (live)")),
		params.C("Source", "wave", WaveChoices, max(0, min(cfg.Wave, len(WaveChoices)-1)),
			params.Hint("The wave of the source wave: sine, triangle, saw or square, or morph (the position picks)")),
		params.K("Source", "rootNote", 0, 127, cfg.RootNote, params.Label("root note"),
			params.Hint("The pitch the recording (or the live sound) is at: this key plays it as it was")),
		params.K("Source", "position", 0, 1, cfg.Position,
			params.Hint("Where the grains are taken from: the place in a recording, how far back in the live sound (0 is now), or which wave (morph)")),
		params.K("Source", "scan", -2, 2, cfg.Scan,
			params.Hint("How fast the place moves on while a note is held: 0 holds one moment, 1 is the speed of the recording, negative goes backwards")),
		params.K("Source", "spray", 0, 1, cfg.Spray, params.Unit("s"),
			params.Hint("A random spread around the place: the grains are taken from this many seconds around it")),

		params.K("Grains", "density", 1, 200, cfg.Density, params.Exp(), params.Unit("Hz"),
			params.Hint("How many grains start every second: few is a stutter, many is a smooth cloud")),
		params.K("Grains", "grainSize", 5, 1000, cfg.GrainSize, params.Exp(), params.Unit("ms"), params.Label("size"),
			params.Hint("How long a grain is: short is rough and percussive, long is soft")),
		params.K("Grains", "sync", 0, 4, cfg.Sync, params.Unit("beats"),
			params.Hint("Grains on the tempo: the beats between two grains (0.25 is a sixteenth). 0 is off: the density is used")),
		params.C("Grains", "window", WindowNames, cfg.Window,
			params.Hint("The shape of a grain: hann is smooth, gauss a bell, tukey has a flat middle (heard as notes), triangle is sharper")),
		params.K("Grains", "shape", 0, 1, cfg.Shape,
			params.Hint("Turns the window into a percussive one: a fast attack and a decay")),
		params.K("Grains", "jitter", 0, 1, cfg.Jitter,
			params.Hint("How uneven the time between grains is: 0 is a machine, 1 is a random scatter")),
		params.K("Grains", "reverse", 0, 1, cfg.Reverse,
			params.Hint("The chance that a grain plays backwards")),

		params.C("Pitch", "scale", ScaleChoices, scaleIndex(cfg.Scale),
			params.Hint("Holds the pitch of every grain to a scale, so a cloud of grains is always in key")),
		params.C("Pitch", "scaleRoot", NoteNames, int(cfg.ScaleRoot+1200)%12, params.Label("scale root"),
			params.Hint("The note the scale starts on")),
		params.C("Pitch", "stack", StackChoices, stackIndex(cfg.Stack),
			params.Hint("Every grain is a chord: a grain for each note of the chord")),
		params.K("Pitch", "pitchSpread", 0, 12, cfg.PitchSpread, params.Unit("st"), params.Label("spread"),
			params.Hint("A random detune of each grain, up to this many semitones: a chorus, a smear")),
		params.K("Pitch", "octave", 0, 1, cfg.Octave,
			params.Hint("The chance that a grain plays an octave higher: sparkle")),

		params.K("Output", "stereoSpread", 0, 1, cfg.StereoSpread, params.Label("stereo"),
			params.Hint("A random place in the stereo field for each grain")),
		params.K("Output", "level", 0, 16, level,
			params.Hint("The volume of the module")),
	}, Envelopes: []params.Envelope{
		{Prefix: "amp", Label: "Note volume", Group: "Output", Stages: "ADSR",
			Attack: cfg.AmpEnv.AttackDuration, Decay: cfg.AmpEnv.DecayDuration, Sustain: cfg.AmpEnv.DecayLevel, Release: cfg.AmpEnv.ReleaseDuration,
			Hint: "How the volume of a whole note goes: the attack when it starts, the decay to the sustain level while the key is held, the release when it is let go"},
	}, GroupHints: map[string]string{
		"Source": "What is played, and where in it the grains are taken from.",
		"Grains": "How the grains are made: how often they start, how long they are, what shape.",
		"Pitch":  "The pitch of each grain: kept in a scale, stacked into chords, or scattered.",
		"Output": "The volume, the width and the envelope of a note.",
	}}
}
