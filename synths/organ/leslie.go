package organ

import "math"

// The Leslie speaker: a horn for the highs and a drum for the lows, each turning in front of a
// microphone. A rotor moving toward the microphone raises the pitch and one moving away lowers it
// (the Doppler effect: a delay that swings), and the sound is loudest when it faces the microphone
// (amplitude modulation). Horn and drum turn at their own speeds, the horn faster and quicker to
// get there, with a stop, a slow (chorale) and a fast (tremolo) speed.

// biq is a second order filter
type biq struct{ b0, b1, b2, a1, a2, z1, z2 float64 }

// butter sets a Butterworth low pass or high pass
func (b *biq) butter(fc, sr float64, high bool) {
	w := 2 * math.Pi * fc / sr
	cs, al := math.Cos(w), math.Sin(w)/math.Sqrt2
	a0 := 1 + al
	if high {
		b.b0, b.b1, b.b2 = (1+cs)/2/a0, -(1+cs)/a0, (1+cs)/2/a0
	} else {
		b.b0, b.b1, b.b2 = (1-cs)/2/a0, (1-cs)/a0, (1-cs)/2/a0
	}
	b.a1, b.a2 = -2*cs/a0, (1-al)/a0
}

func (b *biq) process(x float64) float64 {
	y := b.b0*x + b.z1
	b.z1 = b.b1*x - b.a1*y + b.z2
	b.z2 = b.b2*x - b.a2*y
	return y
}

// rotor is the horn or the drum
type rotor struct {
	slow, fast float64 // turns per second
	up, down   float64 // seconds to reach a new speed (95%): speeding up, slowing down
	doppler    float64 // seconds the sound moves to and fro (the radius over the speed of sound)
	am         float64 // 0-1, how much the level falls when it faces away
	rate       float64 // the speed now
	phase      float64 // 0-1, a turn
	line       []float64
	pos        int
}

const leslieLine = 2048

// speed returns the turns per second for a speed setting: 0 stopped, 0.5 slow, 1 fast
func (r *rotor) speed(s float64) float64 {
	s = max(0, min(s, 1))
	if s <= 0.5 {
		return r.slow * s * 2
	}
	return r.slow + (r.fast-r.slow)*(s-0.5)*2
}

// turn moves a rotor a sample on: its speed toward the target (at the pace of its weight, about
// there after up or down seconds), its phase by its speed
func (r *rotor) turn(target, sr float64) {
	t := r.down
	if target > r.rate {
		t = r.up
	}
	r.rate += (target - r.rate) * (1 - math.Exp(-3/(t*sr)))
	r.phase += r.rate / sr
	r.phase -= math.Floor(r.phase)
}

func (r *rotor) write(x float64) {
	r.line[r.pos] = x
	r.pos = (r.pos + 1) & (leslieLine - 1)
}

// hear returns what a microphone at an angle (in turns) hears of a rotor: the sound delayed as
// it comes and goes, louder when it faces the microphone
func (r *rotor) hear(angle, sr float64) float64 {
	a := 2 * math.Pi * (r.phase + angle)
	d := 3 + r.doppler*sr*(1+math.Sin(a))
	level := 1 - r.am*(0.5-0.5*math.Cos(a))
	// cubic Hermite read d samples back
	p := float64(r.pos) - d
	i := math.Floor(p)
	f := p - i
	at := func(k int) float64 { return r.line[(int(i)+k)&(leslieLine-1)] }
	y0, y1, y2, y3 := at(-1), at(0), at(1), at(2)
	c1 := 0.5 * (y2 - y0)
	c2 := y0 - 2.5*y1 + 2*y2 - 0.5*y3
	c3 := 0.5*(y3-y0) + 1.5*(y1-y2)
	return (((c3*f+c2)*f+c1)*f + y1) * level
}

// leslie is the speaker
type leslie struct {
	horn, drum rotor
	lo, lo2    biq
	hi, hi2    biq
	fc, sr     float64
	mix        float64 // the amount now, gliding to its setting
	active     bool
}

func newLeslie(sr float64, speed float64) *leslie {
	l := &leslie{sr: sr}
	// a Leslie 122: the horn faster than the drum, and the heavy drum slower to change
	l.horn = rotor{slow: 0.8, fast: 6.7, up: 1.2, down: 1.6, doppler: 0.0005, am: 0.5, line: make([]float64, leslieLine)}
	l.drum = rotor{slow: 0.7, fast: 6.0, up: 3.0, down: 4.0, doppler: 0.00012, am: 0.3, line: make([]float64, leslieLine)}
	l.drum.phase = 0.37 // the rotors don't turn in step
	l.horn.rate, l.drum.rate = l.horn.speed(speed), l.drum.speed(speed)
	l.setCrossover(800)
	return l
}

// setCrossover splits the sound for the drum and the horn with a fourth order (Linkwitz-Riley) filter
func (l *leslie) setCrossover(fc float64) {
	l.fc = fc
	for _, f := range []*biq{&l.lo, &l.lo2} {
		f.butter(fc, l.sr, false)
	}
	for _, f := range []*biq{&l.hi, &l.hi2} {
		f.butter(fc, l.sr, true)
	}
}

// turn lets the rotors spin on for a sample toward the speed setting
func (l *leslie) turn(speed float64) {
	l.horn.turn(l.horn.speed(speed), l.sr)
	l.drum.turn(l.drum.speed(speed), l.sr)
}

func (l *leslie) clear() {
	clear(l.horn.line)
	clear(l.drum.line)
	for _, f := range []*biq{&l.lo, &l.lo2, &l.hi, &l.hi2} {
		f.z1, f.z2 = 0, 0
	}
}

// process puts a sample through the speaker: the microphones are an angle (in turns) apart. It
// returns the left and the right microphone, mixed with the dry sound by mix.
func (l *leslie) process(x, amount, angle float64) (float64, float64) {
	l.mix += (amount - l.mix) * 0.002
	if l.mix < 1e-5 && amount <= 0 {
		if l.active {
			l.clear()
			l.active = false
		}
		return x, x
	}
	l.active = true
	l.drum.write(l.lo2.process(l.lo.process(x)))
	l.horn.write(l.hi2.process(l.hi.process(x)))
	left := l.horn.hear(0, l.sr) + l.drum.hear(0, l.sr)
	right := l.horn.hear(angle, l.sr) + l.drum.hear(angle, l.sr)
	m := l.mix
	return x*(1-m) + left*m, x*(1-m) + right*m
}
