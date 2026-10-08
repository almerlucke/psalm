package score

// Tuning turns a MIDI pitch (a fraction is a part of a semitone) into a frequency in Hz. A
// sequencer with a tuning (Config.Tuning) sends its notes as frequencies in it: write the notes in
// MIDI pitches as always, the tuning decides how they sound. harmony.EqualTemperament and the Just
// tuning of a key (harmony.MustKey("Eb major").Just(440)) are tunings
type Tuning func(pitch float64) float64
