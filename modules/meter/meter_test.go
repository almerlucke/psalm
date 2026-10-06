package meter

import (
	"math"
	"testing"
)

const sr = 44100.0

// feed adds seconds of a sine of an amplitude (on the left, or both sides)
func feed(l *Loudness, freq, amp, seconds float64, both bool) {
	for i := range int(seconds * sr) {
		x := amp * math.Sin(2*math.Pi*freq*float64(i)/sr)
		r := 0.0
		if both {
			r = x
		}
		l.Add(x, r)
	}
}

// The reference of BS.1770: a 997 Hz sine at full scale on one channel is -3.01 LUFS
func TestReferenceLevel(t *testing.T) {
	l := NewLoudness(sr)
	feed(l, 997, 1, 5, false)
	if got := l.Integrated(); math.Abs(got+3.01) > 0.1 {
		t.Fatalf("expected -3.01 LUFS, got %.2f", got)
	}
	if got := l.Momentary(); math.Abs(got+3.01) > 0.1 {
		t.Fatalf("momentary %.2f", got)
	}
}

// The same at 48 kHz, the filters are computed per sample rate
func TestOtherSampleRate(t *testing.T) {
	l := NewLoudness(48000)
	for i := range 48000 * 5 {
		l.Add(math.Sin(2*math.Pi*997*float64(i)/48000), 0)
	}
	if got := l.Integrated(); math.Abs(got+3.01) > 0.1 {
		t.Fatalf("expected -3.01 LUFS at 48 kHz, got %.2f", got)
	}
}

func TestStereoAndLevels(t *testing.T) {
	l := NewLoudness(sr)
	feed(l, 997, 0.1, 5, true) // -20 dB on both sides: -20 + 3 (two channels) - 3.01
	if got := l.Integrated(); math.Abs(got-(-20+3.01-3.01)) > 0.15 {
		t.Fatalf("expected -20 LUFS, got %.2f", got)
	}
}

// The K weighting: the low end counts less, the highs a little more
func TestWeighting(t *testing.T) {
	low, high := NewLoudness(sr), NewLoudness(sr)
	feed(low, 30, 0.5, 4, false)
	feed(high, 5000, 0.5, 4, false)
	ref := NewLoudness(sr)
	feed(ref, 997, 0.5, 4, false)
	if d := low.Integrated() - ref.Integrated(); d > -1 {
		t.Fatalf("30 Hz should measure quieter than 1 kHz, difference %.1f LU", d)
	}
	if d := high.Integrated() - ref.Integrated(); d < 2 || d > 5 {
		t.Fatalf("5 kHz should measure about 3.5 LU louder than 1 kHz, difference %.1f LU", d)
	}
}

// Silence and quiet passages are gated out of the integrated loudness
func TestGating(t *testing.T) {
	l := NewLoudness(sr)
	feed(l, 997, 1, 4, false)
	feed(l, 997, 0, 10, false)     // silence
	feed(l, 997, 0.001, 10, false) // -60 dB, far below the rest
	if got := l.Integrated(); math.Abs(got+3.01) > 0.2 {
		t.Fatalf("silence and quiet passages should be gated, got %.2f LUFS", got)
	}
	if got := l.ShortTerm(); got > -55 {
		t.Fatalf("short-term follows the quiet end, got %.2f", got)
	}
}

func TestMeterModule(t *testing.T) {
	m := New()
	bs := m.Config().BufferSize
	for b := range int(3 * sr / float64(bs)) {
		for i := range bs {
			x := float32(0.5 * math.Sin(2*math.Pi*997*float64(b*bs+i)/sr))
			m.Input(0).Channels[0][i] = x
			m.Input(0).Channels[1][i] = x
		}
		m.Synthesize()
		if m.Output(0).Channels[0][7] != m.Input(0).Channels[0][7] {
			t.Fatal("the meter should pass its input")
		}
	}
	if math.Abs(m.Peak()-ToDB(0.5)) > 0.01 || math.Abs(m.RMS()-ToDB(0.5/math.Sqrt2)) > 0.05 {
		t.Fatalf("peak %.2f rms %.2f", m.Peak(), m.RMS())
	}
	m.Reset()
	if m.Peak() != Silence || m.Loudness().Integrated() != Silence {
		t.Fatal("reset")
	}
}
