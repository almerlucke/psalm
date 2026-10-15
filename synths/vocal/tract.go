package vocal

import "math"

// Vowels are the vowels a voice can sing by name (Config.Phoneme, 1 to 12), in an order where
// neighbours sound alike, so a fraction between two morphs between them
var Vowels = []string{"i", "ɪ", "e", "ɛ", "æ", "a", "ɔ", "o", "ʊ", "u", "ʌ", "ɜ"}

// the vowels a e i o u of the 0-4 scale, as indices of Vowels
var scaleRows = [5]int{5, 2, 0, 7, 9}

// the formants F1 to F3 (Hz) of each vowel for a man, a woman and a child (after Peterson and Barney,
// with e and o filled in between their neighbours and u and ʊ lowered a little, as they are sung)
var vowelBase = [12][3][3]float64{
	{{270, 2290, 3010}, {310, 2790, 3310}, {370, 3200, 3730}},  // i
	{{390, 1990, 2550}, {430, 2480, 3070}, {530, 2730, 3600}},  // ɪ
	{{400, 2150, 2700}, {440, 2650, 3100}, {500, 2900, 3650}},  // e
	{{530, 1840, 2480}, {610, 2330, 2990}, {690, 2610, 3570}},  // ɛ
	{{660, 1720, 2410}, {860, 2050, 2850}, {1010, 2320, 3320}}, // æ
	{{730, 1090, 2440}, {850, 1220, 2810}, {1030, 1370, 3170}}, // a
	{{570, 840, 2410}, {590, 920, 2710}, {680, 1060, 3180}},    // ɔ
	{{420, 800, 2400}, {440, 860, 2650}, {520, 1000, 3150}},    // o
	{{440, 960, 2300}, {470, 1160, 2680}, {560, 1410, 3310}},   // ʊ
	{{300, 760, 2240}, {370, 950, 2670}, {430, 1370, 3170}},    // u
	{{640, 1190, 2390}, {760, 1400, 2780}, {850, 1590, 3360}},  // ʌ
	{{490, 1350, 1690}, {500, 1640, 1960}, {560, 1820, 2160}},  // ɜ
}

// the bandwidths B1 to B3 (Hz) of each vowel: the closed vowels ring longer in the first formant
var vowelBW = [12][3]float64{
	{45, 90, 130}, {55, 85, 130}, {55, 90, 130}, {65, 85, 130}, {75, 85, 130}, {85, 80, 130},
	{70, 75, 120}, {55, 70, 120}, {55, 75, 120}, {45, 70, 120}, {75, 80, 130}, {55, 90, 120},
}

// a voice type: which of the three bases it takes, how much it scales the formants (the length of
// its tract), the widths, F4 and F5, and the singer's formant of a trained voice
type voiceType struct {
	base       int
	scale      float64
	bw         float64
	f4, f5     float64
	singerForm float64
}

var voiceTypes = [numRanges]voiceType{
	Auto:       {0, 1, 1, 3400, 4400, 2800},
	Bass:       {0, 0.92, 0.95, 3400, 4400, 2500},
	Tenor:      {0, 1, 1, 3400, 4400, 2800},
	Alto:       {1, 1, 1.1, 4000, 4900, 3000},
	Soprano:    {1, 1.06, 1.15, 4000, 4900, 3200},
	ChildVoice: {2, 1, 1.25, 4500, 5400, 3300},
}

// row is the formants and bandwidths of a vowel
type row struct{ f, bw [5]float64 }

var table [numRanges][12]row

func init() {
	for r := range table {
		t := voiceTypes[r]
		for i := range 12 {
			var x row
			for k := range 3 {
				x.f[k] = vowelBase[i][t.base][k] * t.scale
				x.bw[k] = vowelBW[i][k] * t.bw
			}
			x.f[3] = math.Max(t.f4*t.scale, x.f[2]+450)
			x.f[4] = math.Max(t.f5*t.scale, x.f[3]+500)
			x.bw[3], x.bw[4] = 150*t.bw, 250*t.bw
			table[r][i] = x
		}
	}
}

// Formants returns the five formants (Hz) and their bandwidths of vowel (1 to 12, see Vowels) as sung
// by a voice of the range r (Auto is a tenor)
func Formants(r Range, vowel int) (freq, bw [5]float64) {
	if r < 0 || r >= numRanges {
		r = Tenor
	}
	x := table[r][max(0, min(vowel-1, 11))]
	return x.f, x.bw
}

// tractAt returns the formants and widths of a position: on the scale a e i o u (0-4) when phoneme
// is 0, else along Vowels (0-11); fractions morph in between
func tractAt(r Range, phoneme int, pos float64) (f, bw [5]float64) {
	var ia, ib int
	var frac float64
	if phoneme == 0 {
		pos = math.Max(0, math.Min(pos, 4))
		i := min(int(pos), 3)
		frac = pos - float64(i)
		ia, ib = scaleRows[i], scaleRows[i+1]
	} else {
		pos = math.Max(0, math.Min(pos, 11))
		i := min(int(pos), 10)
		frac = pos - float64(i)
		ia, ib = i, i+1
	}
	if r < 0 || r >= numRanges {
		r = Tenor
	}
	a, b := &table[r][ia], &table[r][ib]
	for k := range 5 {
		f[k] = a.f[k] * math.Pow(b.f[k]/a.f[k], frac)
		bw[k] = a.bw[k] + (b.bw[k]-a.bw[k])*frac
	}
	return
}

// reson is a resonator (Klatt): a pole pair with a gain of 1 at 0 Hz
type reson struct{ a, b, c, y1, y2 float64 }

func (r *reson) set(f, bw, sr float64) {
	f = math.Max(20, math.Min(f, sr*0.45))
	bw = math.Max(10, bw)
	r.c = -math.Exp(-2 * math.Pi * bw / sr)
	r.b = 2 * math.Exp(-math.Pi*bw/sr) * math.Cos(2*math.Pi*f/sr)
	r.a = 1 - r.b - r.c
}

func (r *reson) tick(x float64) float64 {
	y := r.a*x + r.b*r.y1 + r.c*r.y2
	r.y2, r.y1 = r.y1, y
	return y
}

// anti is an anti-resonator: a zero pair, the other way round
type anti struct{ a, b, c, x1, x2 float64 }

func (r *anti) set(f, bw, sr float64) {
	var p reson
	p.set(f, bw, sr)
	r.a, r.b, r.c = 1/p.a, -p.b/p.a, -p.c/p.a
}

func (r *anti) tick(x float64) float64 {
	y := r.a*x + r.b*r.x1 + r.c*r.x2
	r.x2, r.x1 = r.x1, x
	return y
}

// the tract of a channel: the nasal pole and zero, then the five formants in cascade
type tractFilter struct {
	np    reson
	nz    anti
	f     [5]reson
	nasal float64 // how much the nose is in the path (0-1)
}

func (t *tractFilter) clear() {
	t.np.y1, t.np.y2, t.nz.x1, t.nz.x2 = 0, 0, 0, 0
	for k := range t.f {
		t.f[k].y1, t.f[k].y2 = 0, 0
	}
}

func (t *tractFilter) tick(x float64) float64 {
	x += t.nasal * (t.nz.tick(t.np.tick(x)) - x) // always run, so it is warm when the nose opens
	for k := range t.f {
		x = t.f[k].tick(x)
	}
	return x
}
