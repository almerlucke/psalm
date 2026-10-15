package dx

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
)

func energyOf(v *Voice, blocks int) float64 {
	v.NoteOn(0.8, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
	var e float64
	for range blocks {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			e += float64(s) * float64(s)
		}
	}
	return e
}

// a value cabled in moves any setting a slot joins it to, for the DX operators and the exact ones
func TestMatrixExtMovesLevel(t *testing.T) {
	for _, name := range []string{"bellPad", "ePiano"} { // DX operators; exact operators
		p := Presets[name]()
		p.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "level", Amount: -1}
		loud, quiet := New(p, 2), New(p, 2)
		quiet.Receive(1.0, InExt1)
		if el, eq := energyOf(loud, 40), energyOf(quiet, 40); el < 0.1 || eq > el*0.02 || quiet.patch.Ext[0] != 1 {
			t.Errorf("%s: energy %g with the value at 0, %g at 1", name, el, eq)
		}
	}
}

// an LFO through the matrix moves the gain of a modulator, so the sound changes
func TestMatrixLFOToGain(t *testing.T) {
	plain := energyOf(New(Presets["ePiano"](), 2), 40)
	p := Presets["ePiano"]()
	p.LFO2Rate = 6
	p.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "op2Gain", Amount: 1}
	if e := energyOf(New(p, 2), 40); e == plain || math.IsNaN(e) {
		t.Errorf("energy %g, as without (%g)", e, plain)
	}
}

func TestMatrixEverything(t *testing.T) {
	for _, name := range []string{"bellPad", "lead"} {
		for src := modmatrix.FilterEnv; src < modmatrix.NumSources; src++ {
			for _, d := range Dests {
				p := Presets[name]()
				p.LFO2Rate = 3
				p.Filter, p.FilterSet = DefaultFilter(), true
				p.Filter.Type = int(voicefilter.Ladder24)
				p.Mods[0] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: 1}
				p.Mods[1] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: -1}
				v := New(p, 2)
				v.NoteOn(1, psalm.Message{"pitch": 36.0}, psalm.CtrlTick{}, v.Config())
				for range 20 {
					v.Synthesize()
					for _, s := range v.Output(0).Channels[0] {
						if math.IsNaN(float64(s)) || math.IsInf(float64(s), 0) || math.Abs(float64(s)) > 20 {
							t.Fatalf("%s %s to %s: sample %v", name, modmatrix.SourceNames[src], d.Key, s)
						}
					}
				}
			}
		}
	}
}

func TestMatrixMessages(t *testing.T) {
	v := New(Presets["bellPad"](), 2)
	v.ReceiveMessage(psalm.Message{"mod2Src": int(modmatrix.Wheel), "mod2Dest": 1 + dFeedback, "mod2Amount": 0.5, "lfo2Rate": 4.0})
	if s := v.patch.Mods[1]; s.Src != modmatrix.Wheel || s.Dest != "freeFeedback" || s.Amount != 0.5 || v.patch.LFO2Rate != 4 {
		t.Errorf("slot %+v, lfo 2 %g", s, v.patch.LFO2Rate)
	}
}
