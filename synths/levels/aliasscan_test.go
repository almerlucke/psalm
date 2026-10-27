package levels

import (
	"fmt"
	"math"
	"math/cmplx"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/shaper"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
)

func fftScan(x []complex128) []complex128 {
	n := len(x)
	if n == 1 {
		return x
	}
	even, odd := make([]complex128, n/2), make([]complex128, n/2)
	for i := range n / 2 {
		even[i], odd[i] = x[2*i], x[2*i+1]
	}
	even, odd = fftScan(even), fftScan(odd)
	out := make([]complex128, n)
	for k := range n / 2 {
		t := cmplx.Rect(1, -2*math.Pi*float64(k)/float64(n)) * odd[k]
		out[k], out[k+n/2] = even[k]+t, even[k]-t
	}
	return out
}

// renderAt plays a note at a sample rate (the buffer in step with it, so the note starts at the same time) for a second
func renderAt(mk func() played, pitch, sr float64, buf int) []float64 {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: sr, BufferSize: buf})
	defer psalm.PopConfig()
	fresh()
	v := mk()
	v.Note(0.8, 0.8, psalm.Message{"pitch": pitch, "velocity": 0.8}, psalm.CtrlTick{}, v.Config())
	var sig []float64
	for b := 0; len(sig) < int(sr); b++ {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			sig = append(sig, float64(s))
		}
	}
	return sig[:int(sr)]
}

// decimate4 lowpasses a 176.4 kHz signal at 20 kHz (windowed sinc, no delay) and keeps every fourth sample
func decimate(x []float64, by int) []float64 {
	const half = 128
	h := make([]float64, 2*half+1)
	sum := 0.0
	for i := range h {
		k := float64(i - half)
		c := 2 * 20000.0 / 176400 * 4 / float64(by)
		sinc := c
		if k != 0 {
			sinc = math.Sin(math.Pi*c*k) / (math.Pi * k)
		}
		w := 0.35875 - 0.48829*math.Cos(2*math.Pi*float64(i)/float64(2*half)) + 0.14128*math.Cos(4*math.Pi*float64(i)/float64(2*half)) - 0.01168*math.Cos(6*math.Pi*float64(i)/float64(2*half))
		h[i] = sinc * w
		sum += h[i]
	}
	var out []float64
	for n := half; n < len(x)-half; n += by {
		a := 0.0
		for i, c := range h {
			a += c * x[n+i-half]
		}
		out = append(out, a/sum)
	}
	return out
}

// spectrum is the power of the Blackman-Harris windowed 32768 samples from 0.1 s on
func spectrum(x []float64) []float64 {
	const n = 1 << 15
	c := make([]complex128, n)
	for i := range n {
		a := 2 * math.Pi * float64(i) / n
		c[i] = complex(x[len(x)/10+i]*(0.35875-0.48829*math.Cos(a)+0.14128*math.Cos(2*a)-0.01168*math.Cos(3*a)), 0)
	}
	f := fftScan(c)
	p := make([]float64, n/2)
	for k := range p {
		p[k] = real(f[k] * cmplx.Conj(f[k]))
	}
	return p
}

// foldBack is the energy at a sample rate that the sound at 176.4 kHz does not have (the power of each bin that is over twice the most of the bins around it in the other,
// as dB of the power of the sound): a delay or a change of phase leaves the magnitudes as they are, what is added at other frequencies is the fold-back
func foldBack(mk func() played, pitch float64, by int) float64 {
	sr := 176400 / float64(by)
	a := spectrum(renderAt(mk, pitch, sr, 8192/by))
	b := spectrum(decimate(renderAt(mk, pitch, 176400, 8192), by))
	var excess, total float64
	top := int(20000 / sr * float64(1<<15))
	for k := 8; k < top-8; k++ {
		near := 0.0
		for d := -6; d <= 6; d++ {
			near = math.Max(near, b[k+d])
		}
		total += b[k]
		if a[k] > 2*near {
			excess += a[k] - 2*near
		}
	}
	return 10 * math.Log10((excess+1e-20)/(total+1e-20))
}

// TestAliasScan lists what the fold-back adds to every pitched preset of the synths, at C5 and C6: ALIAS=1 go test -v -run TestAliasScan (ONLY=subtractive for the
// one synth, OVER=n the oversampling of its ladder). Each is rendered at 44.1 kHz and at 176.4 kHz, brought down; a delay or a phase does not count, the energy at
// frequencies the better sound does not have does. Noise is not the same twice, so the presets with noise show some in any case
func TestAliasScan(t *testing.T) {
	if os.Getenv("ALIAS") == "" {
		t.Skip()
	}
	type row struct {
		name      string
		a, a2, a3 float64
	}
	var rows []row
	add := func(name string, mk func() played) {
		rows = append(rows, row{name, foldBack(mk, 72, 4), foldBack(mk, 72, 2), foldBack(mk, 84, 4)})
	}
	over, _ := strconv.Atoi(os.Getenv("OVER"))
	only := func(synth string) bool { return os.Getenv("ONLY") == "" || strings.Contains(os.Getenv("ONLY"), synth) }
	if only("subtractive") {
		for n, f := range subtractive.Presets {
			add("subtractive/"+n, func() played {
				cfg := f()
				cfg.FilterOversample = over
				return subtractive.New(cfg)
			})
		}
	}
	if only("shaper") {
		for n, f := range shaper.Presets {
			add("shaper/"+n, func() played { return shaper.New(f()) })
		}
	}
	if only("wavetable") {
		table := gen.Brightness(32)
		for n, f := range wavetable.Presets {
			add("wavetable/"+n, func() played {
				cfg := f(table)
				cfg.Oversample = over
				return wavetable.New(cfg)
			})
		}
	}
	if only("organ") {
		for n, f := range organ.Presets {
			add("organ/"+n, func() played { return organ.New(f()) })
		}
	}
	if only("pulsar") {
		for n, f := range pulsar.Presets {
			add("pulsar/"+n, func() played { return pulsar.New(f()) })
		}
	}
	sort.Slice(rows, func(i, j int) bool { return math.Max(rows[i].a, rows[i].a3) > math.Max(rows[j].a, rows[j].a3) })
	for _, r := range rows {
		fmt.Printf("ALIAS %-34s C5 at 44.1k %6.1f dB  at 88.2k %6.1f dB  C6 at 44.1k %6.1f dB\n", r.name, r.a, r.a2, r.a3)
	}
}
