package piano

import (
	"math"
	"sync"
)

// Stretch tuning. The partials of a stiff string are sharp, so a piano tuned to exact equal
// temperament has octaves that beat: the second partial of the lower note is above the upper
// note. Tuners widen the octaves until they are clean, matching partials of the notes: the 2nd of
// the lower note with the fundamental of the upper one in the treble, the 4th with the 2nd in the
// middle, the 6th with the 3rd in the bass and the 8th with the 4th in the lowest octaves (to the
// ear the higher partials of the bass are the pitch). Starting from A4, that gives the Railsback
// curve: the low bass some 20 cents flat, the top some 30 cents sharp

var stretchTables sync.Map // stiffness -> *[128]float64

// stretch returns how many cents every key is tuned from equal temperament for a stiffness (the
// inharmonicity is what the tuner hears)
func stretch(stiffness float64) *[128]float64 {
	if t, ok := stretchTables.Load(stiffness); ok {
		return t.(*[128]float64)
	}
	b := func(key int) float64 { return inharmonicity(float64(key)) * stiffness }
	// cents partial m of a key is above m times its fundamental
	partial := func(key, m int) float64 {
		k := float64(m)
		return 600 * math.Log2((1+b(key)*k*k)/(1+b(key)))
	}
	// how much wider than 1200 cents the octave from key up is tuned
	octave := func(key int) float64 {
		switch {
		case key >= 69:
			return partial(key, 2) // 2:1
		case key >= 50:
			return partial(key, 4) - partial(key+12, 2) // 4:2
		case key >= 33:
			return partial(key, 6) - partial(key+12, 3) // 6:3
		default:
			return partial(key, 8) - partial(key+12, 4) // 8:4, the lowest octaves
		}
	}

	var t [128]float64
	// a semitone is a twelfth of the octave above it
	for key := 70; key < 128; key++ {
		t[key] = t[key-1] + octave(min(key-1, 108))/12
	}
	for key := 68; key >= 0; key-- {
		t[key] = t[key+1] - octave(max(key, 21))/12
	}
	stored, _ := stretchTables.LoadOrStore(stiffness, &t)
	return stored.(*[128]float64)
}

// stretched returns a pitch tuned with the stretch of the piano, scaled by the Stretch setting
func stretched(pitch float64, cfg *Config) float64 {
	s := math.Max(0, cfg.Stretch)
	if s == 0 {
		return pitch
	}
	t := stretch(math.Max(0, cfg.Stiffness))
	key := max(0, min(int(math.Floor(pitch)), 126))
	frac := math.Max(0, math.Min(pitch-float64(key), 1))
	cents := t[key] + (t[key+1]-t[key])*frac
	return pitch + s*cents/100
}
