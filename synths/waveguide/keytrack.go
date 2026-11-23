package waveguide

import "math"

// KeyTrack makes how a plucked string dies depend on its pitch, as a real string does: how long it
// rings and how fast its high partials go. It holds the decay and the damping of a string a semitone
// apart, from pitch From up; the settings of the string (Config.Decay, Config.Damping) are what they are
// at pitch Ref, and the other pitches have them in the same ratio (the decay) and the same distance
// (the damping) as the table has them to Ref. A string is plucked at a pitch, so a chord has each of its
// strings die its own way, and the settings still turn the whole of it
type KeyTrack struct {
	From    int       // the MIDI pitch of the first of the tables
	Ref     int       // the pitch the settings of the string hold at
	Decay   []float64 // seconds the string rings at 220 Hz at each pitch
	Damping []float64 // the damping at each pitch
	// Edge is the Edge of the pluck at each pitch (see Config.Edge), as the other tables from From up; none (or
	// empty) leaves the pluck as the settings have it
	Edge []float64
}

// at returns the table between its semitones (in the logarithm of the decay), and clamps outside it
func (k *KeyTrack) at(table []float64, pitch float64, logarithmic bool) float64 {
	x := pitch - float64(k.From)
	i := int(math.Floor(x))
	if i < 0 {
		return table[0]
	}
	if i >= len(table)-1 {
		return table[len(table)-1]
	}
	f := x - float64(i)
	if logarithmic {
		return math.Exp(math.Log(table[i])*(1-f) + math.Log(table[i+1])*f)
	}
	return table[i]*(1-f) + table[i+1]*f
}

// DecayAt is the seconds a string rings at 220 Hz at a pitch (MIDI, fractions allowed)
func (k *KeyTrack) DecayAt(pitch float64) float64 { return k.at(k.Decay, pitch, true) }

// DampingAt is the damping at a pitch
func (k *KeyTrack) DampingAt(pitch float64) float64 { return k.at(k.Damping, pitch, false) }

// EdgeAt is the Edge of the pluck at a pitch (0 without the table)
func (k *KeyTrack) EdgeAt(pitch float64) float64 {
	if k == nil || len(k.Edge) == 0 {
		return 0
	}
	return k.at(k.Edge, pitch, false)
}

// of returns what the settings of the string at the reference pitch become at a pitch: the distance to
// add to the damping and the factor for the decay. A track without tables changes nothing
func (k *KeyTrack) of(pitch float64) (damping, decay float64) {
	if k == nil || len(k.Decay) == 0 || len(k.Damping) == 0 {
		return 0, 1
	}
	ref := float64(k.Ref)
	return k.DampingAt(pitch) - k.DampingAt(ref), k.DecayAt(pitch) / k.DecayAt(ref)
}

// keyTrack is what the track of the string makes of the damping and the decay for the note it plays
func (v *Voice) keyTrack() (damping, decay float64) {
	if v.cfg.Track == nil {
		return 0, 1
	}
	return v.cfg.Track.of(69 + 12*math.Log2(v.freq/440))
}

// keyEdge is the distance to add to the Edge of the pluck for the note it plays: the table's at the pitch against
// its value at the reference pitch
func (v *Voice) keyEdge() float64 {
	k := v.cfg.Track
	if k == nil || len(k.Edge) == 0 {
		return 0
	}
	return k.EdgeAt(69+12*math.Log2(v.freq/440)) - k.EdgeAt(float64(k.Ref))
}
