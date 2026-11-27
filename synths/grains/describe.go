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
		params.K("Source", "stretch", -16, 16, cfg.Stretch, params.Label("stretch"),
			params.Hint("Time stretch: the recording moves this many times slower than it was (4 is four times as long) while the pitch stays. Negative goes backwards, 0 is off and the scan is used")),
		params.C("Source", "slices", SliceChoices, cfg.Slices,
			params.Hint("Markers in a recording: the onsets (where a new sound starts) or equal parts. The position is then a place inside one slice")),
		params.K("Source", "slice", 0, 63, float64(cfg.Slice), params.Step(1),
			params.Hint("Which slice the grains play (0 is the first), round and round; a script can sequence them with sliceSeq")),

		params.K("Grains", "density", 1, 200, cfg.Density, params.Exp(), params.Unit("Hz"),
			params.Hint("How many grains start every second: few is a stutter, many is a smooth cloud")),
		params.K("Grains", "grainSize", 5, 1000, cfg.GrainSize, params.Exp(), params.Unit("ms"), params.Label("size"),
			params.Hint("How long a grain is: short is rough and percussive, long is soft")),
		params.K("Grains", "sync", 0, 4, cfg.Sync, params.Unit("beats"),
			params.Hint("Grains on the tempo: the beats between two grains (0.25 is a sixteenth). 0 is off: the density is used")),
		params.K("Grains", "overlap", 0, 16, cfg.Overlap,
			params.Hint("How many grains sound at once: the density follows the size of the grains. 0 is off: the density or sync is used. Good for time stretching")),
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

		params.C("Filter", "filter", FilterNames, cfg.Filter,
			params.Hint("A filter on every grain: each grain gets its own cutoff (from the modulation, a script's cutoffSeq or the pitch)")),
		params.K("Filter", "cutoff", 20, 18000, cfg.Cutoff, params.Exp(), params.Unit("Hz"),
			params.Hint("Where the filter of a grain cuts")),
		params.K("Filter", "resonance", 0, 1, cfg.Resonance,
			params.Hint("How much the filter rings at its cutoff")),
		params.K("Filter", "keyTrack", 0, 1, cfg.KeyTrack, params.Label("key track"),
			params.Hint("How much the cutoff follows the pitch of the grain: 1 moves it an octave for an octave")),

		params.K("Feedback", "feedback", 0, 0.95, cfg.Feedback,
			params.Hint("The chance that a grain plays what the voice itself played a moment ago: echoes and swells that feed on themselves")),
		params.K("Feedback", "feedbackTime", 0.01, 3.8, cfg.FeedbackTime, params.Exp(), params.Unit("s"), params.Label("time"),
			params.Hint("How far back those grains look")),
		params.K("Feedback", "feedbackPitch", -12, 12, cfg.FeedbackPitch, params.Unit("st"), params.Label("pitch"),
			params.Hint("Semitones added to those grains: 12 climbs an octave with every round, -12 sinks")),

		params.C("Modulation", "mod1Source", ModSourceNames, cfg.Mods[0].Source, params.Label("1 source"),
			params.Hint("What moves the grains: an lfo, a random value for each grain, a rise over the note, the velocity or the note volume")),
		params.C("Modulation", "mod1Dest", ModDestNames, cfg.Mods[0].Dest, params.Label("1 to"),
			params.Hint("What it moves: the position, the pitch (12 semitones), the size, the pan, the level, the cutoff or the density")),
		params.K("Modulation", "mod1Amount", -1, 1, cfg.Mods[0].Amount, params.Label("1 amount"),
			params.Hint("How much; negative turns it round")),
		params.K("Modulation", "mod1Rate", 0.01, 50, cfg.Mods[0].Rate, params.Exp(), params.Unit("Hz"), params.Label("1 rate"),
			params.Hint("The speed of the lfo, and of the rise (1/rate seconds)")),

		params.C("Modulation", "mod2Source", ModSourceNames, cfg.Mods[1].Source, params.Label("2 source"),
			params.Hint("What moves the grains: an lfo, a random value for each grain, a rise over the note, the velocity or the note volume")),
		params.C("Modulation", "mod2Dest", ModDestNames, cfg.Mods[1].Dest, params.Label("2 to"),
			params.Hint("What it moves: the position, the pitch (12 semitones), the size, the pan, the level, the cutoff or the density")),
		params.K("Modulation", "mod2Amount", -1, 1, cfg.Mods[1].Amount, params.Label("2 amount"),
			params.Hint("How much; negative turns it round")),
		params.K("Modulation", "mod2Rate", 0.01, 50, cfg.Mods[1].Rate, params.Exp(), params.Unit("Hz"), params.Label("2 rate"),
			params.Hint("The speed of the lfo, and of the rise (1/rate seconds)")),

		params.C("Modulation", "mod3Source", ModSourceNames, cfg.Mods[2].Source, params.Label("3 source"),
			params.Hint("What moves the grains: an lfo, a random value for each grain, a rise over the note, the velocity or the note volume")),
		params.C("Modulation", "mod3Dest", ModDestNames, cfg.Mods[2].Dest, params.Label("3 to"),
			params.Hint("What it moves: the position, the pitch (12 semitones), the size, the pan, the level, the cutoff or the density")),
		params.K("Modulation", "mod3Amount", -1, 1, cfg.Mods[2].Amount, params.Label("3 amount"),
			params.Hint("How much; negative turns it round")),
		params.K("Modulation", "mod3Rate", 0.01, 50, cfg.Mods[2].Rate, params.Exp(), params.Unit("Hz"), params.Label("3 rate"),
			params.Hint("The speed of the lfo, and of the rise (1/rate seconds)")),

		params.C("Modulation", "mod4Source", ModSourceNames, cfg.Mods[3].Source, params.Label("4 source"),
			params.Hint("What moves the grains: an lfo, a random value for each grain, a rise over the note, the velocity or the note volume")),
		params.C("Modulation", "mod4Dest", ModDestNames, cfg.Mods[3].Dest, params.Label("4 to"),
			params.Hint("What it moves: the position, the pitch (12 semitones), the size, the pan, the level, the cutoff or the density")),
		params.K("Modulation", "mod4Amount", -1, 1, cfg.Mods[3].Amount, params.Label("4 amount"),
			params.Hint("How much; negative turns it round")),
		params.K("Modulation", "mod4Rate", 0.01, 50, cfg.Mods[3].Rate, params.Exp(), params.Unit("Hz"), params.Label("4 rate"),
			params.Hint("The speed of the lfo, and of the rise (1/rate seconds)")),

		params.K("Output", "stereoSpread", 0, 1, cfg.StereoSpread, params.Label("stereo"),
			params.Hint("A random place in the stereo field for each grain")),
		params.K("Output", "level", 0, 16, level,
			params.Hint("The volume of the module")),
	}, Envelopes: []params.Envelope{
		{Prefix: "amp", Label: "Note volume", Group: "Output", Stages: "ADSR",
			Attack: cfg.AmpEnv.AttackDuration, Decay: cfg.AmpEnv.DecayDuration, Sustain: cfg.AmpEnv.DecayLevel, Release: cfg.AmpEnv.ReleaseDuration,
			Hint: "How the volume of a whole note goes: the attack when it starts, the decay to the sustain level while the key is held, the release when it is let go"},
	}, GroupHints: map[string]string{
		"Source":     "What is played, and where in it the grains are taken from.",
		"Grains":     "How the grains are made: how often they start, how long they are, what shape.",
		"Pitch":      "The pitch of each grain: kept in a scale, stacked into chords, or scattered.",
		"Filter":     "A filter for every grain, each with its own cutoff.",
		"Feedback":   "Grains of what the voice itself played: echoes that feed on themselves.",
		"Modulation": "Four slots: a source moves a setting of every grain that starts.",
		"Output":     "The volume, the width and the envelope of a note.",
	}}
}
