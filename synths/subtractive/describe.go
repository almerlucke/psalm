package subtractive

import (
	"fmt"

	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/params"
)

// Waveforms are the names of the oscillator waveforms (blosc), by number
var Waveforms = []string{"sine", "cosine", "triangle", "square", "rectangle", "saw", "ramp", "mod triangle",
	"mod square", "half sine", "full sine", "tri pulse", "trapezoid", "trapezoid var"}

// LFOWaveforms are the names of the LFO waveforms, by number
var LFOWaveforms = []string{"sine", "triangle", "saw", "ramp", "square", "sample & hold"}

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params)
func Describe(cfg Config) params.Set {
	s := params.Set{Name: "subtractive", Params: []params.Param{
		params.C("Oscillators", "osc1Waveform", Waveforms, int(cfg.Osc1Waveform), params.Label("osc 1")),
		params.K("Oscillators", "osc1Level", 0, 1, cfg.Osc1Level, params.Label("level 1")),
		params.C("Oscillators", "osc2Waveform", Waveforms, int(cfg.Osc2Waveform), params.Label("osc 2")),
		params.K("Oscillators", "osc2Level", 0, 1, cfg.Osc2Level, params.Label("level 2")),
		params.K("Oscillators", "osc2Octave", -3, 3, float64(cfg.Osc2Octave), params.Step(1), params.Label("octave 2")),
		params.K("Oscillators", "osc2Detune", -24, 24, cfg.Osc2Detune, params.Unit("st"), params.Label("detune 2")),
		params.K("Oscillators", "pulseWidth", 0.05, 0.95, cfg.PulseWidth, params.Label("pulse width")),
		params.K("Oscillators", "subLevel", 0, 1, cfg.SubLevel, params.Label("sub")),
		params.K("Oscillators", "noiseLevel", 0, 1, cfg.NoiseLevel, params.Label("noise")),
		params.C("Oscillators", "noiseType", []string{"white", "pink"}, int(cfg.NoiseType), params.Label("noise type")),
		params.K("Oscillators", "ringModLevel", 0, 1, cfg.RingModLevel, params.Label("ring mod")),
		params.S("Oscillators", "sync", cfg.Sync, params.Label("sync")),
		params.K("Oscillators", "fmAmount", 0, 5, cfg.FMAmount, params.Label("fm")),
		params.K("Oscillators", "drive", 0.5, 8, max(0.5, cfg.Drive), params.Exp()),
		params.K("Oscillators", "unison", 1, 7, float64(max(1, cfg.Unison)), params.Step(1)),
		params.K("Oscillators", "unisonDetune", 0, 100, cfg.UnisonDetune, params.Unit("ct"), params.Label("unison detune")),
		params.K("Oscillators", "drift", 0, 50, cfg.Drift, params.Unit("ct")),

		params.C("Filter", "filterType", []string{"state variable", "ladder", "formant"}, int(cfg.FilterType), params.Label("type")),
		params.C("Filter", "filterMode", []string{"low pass", "band pass", "high pass", "notch", "peak", "all pass"}, int(cfg.FilterMode), params.Label("mode")),
		params.C("Filter", "ladderMode", []string{"low 24", "low 12", "band 24", "band 12", "high 24", "high 12"}, int(cfg.LadderMode), params.Label("ladder")),
		params.K("Filter", "cutoff", 20, 18000, max(20, cfg.Cutoff), params.Exp(), params.Unit("Hz")),
		params.K("Filter", "resonance", 0, 1, cfg.Resonance),
		params.K("Filter", "filterEnvAmount", -6, 6, cfg.FilterEnvAmount, params.Unit("oct"), params.Label("env amount")),
		params.K("Filter", "keyTrack", 0, 1, cfg.KeyTrack, params.Label("key track")),
		params.K("Filter", "filterDrive", 1, 8, max(1, cfg.FilterDrive), params.Exp(), params.Label("drive")),
		params.C("Filter", "filterOversample", []string{"once", "2x", "4x"}, max(0, min(cfg.FilterOversample, 2)), params.Label("ladder oversample")),
		params.K("Filter", "vowel", 0, 4, cfg.Vowel),
		params.K("Filter", "formantShift", 0.5, 2, orOne(cfg.FormantShift), params.Exp(), params.Label("formant shift")),

		params.C("Filter 2", "filter2Mode", []string{"off", "low pass", "band pass", "high pass", "notch"}, int(cfg.Filter2Mode), params.Label("mode")),
		params.K("Filter 2", "filter2Cutoff", 20, 18000, max(20, cfg.Filter2Cutoff), params.Exp(), params.Unit("Hz"), params.Label("cutoff")),
		params.K("Filter 2", "filter2Resonance", 0, 1, cfg.Filter2Resonance, params.Label("resonance")),
		params.K("Filter 2", "filter2EnvAmount", -6, 6, cfg.Filter2EnvAmount, params.Unit("oct"), params.Label("env amount")),
		params.K("Filter 2", "filter2KeyTrack", 0, 1, cfg.Filter2KeyTrack, params.Label("key track")),
		params.S("Filter 2", "filter2Parallel", cfg.Filter2Parallel, params.Label("parallel")),

		params.K("Pitch", "glide", 0, 2000, cfg.Glide, params.Unit("ms")),
		params.C("Pitch", "glideMode", []string{"exponential", "constant time", "constant rate"}, int(cfg.GlideMode), params.Label("glide shape")),
		params.K("Pitch", "filterEnvToPitch", -48, 48, cfg.FilterEnvToPitch, params.Unit("st"), params.Label("env to pitch")),
		params.K("Pitch", "level", 0, 4, orOne(cfg.Level)),
	}}
	for i, l := range []LFOConfig{cfg.LFO1, cfg.LFO2} {
		g, k := fmt.Sprintf("LFO %d", i+1), fmt.Sprintf("lfo%d", i+1)
		s.Params = append(s.Params,
			params.C(g, k+"Waveform", LFOWaveforms, int(l.Waveform), params.Label("wave")),
			params.K(g, k+"Rate", 0.01, 40, max(0.01, l.Rate), params.Exp(), params.Unit("Hz"), params.Label("rate")),
			params.K(g, k+"Delay", 0, 3000, l.Delay, params.Unit("ms"), params.Label("delay")),
			params.S(g, k+"KeySync", l.KeySync, params.Label("key sync")),
			params.K(g, k+"Pitch", -12, 12, l.Pitch, params.Unit("st"), params.Label("pitch")),
			params.K(g, k+"Cutoff", -4, 4, l.Cutoff, params.Unit("oct"), params.Label("cutoff")),
			params.K(g, k+"PulseWidth", 0, 0.5, l.PulseWidth, params.Label("pulse width")),
			params.K(g, k+"Amp", 0, 1, l.Amp, params.Label("tremolo")),
		)
	}
	m := modmatrix.New(Dests)
	for i, sl := range cfg.Mods {
		m.Set(i, sl)
	}
	for i, e := range cfg.Ext {
		m.SetExt(i, e)
	}
	s.Params = append(s.Params, m.Describe("Modulation")...)
	s.Envelopes = []params.Envelope{
		params.E("Envelopes", "amp", "amp", cfg.AmpEnv.AttackDuration, cfg.AmpEnv.DecayDuration, cfg.AmpEnv.DecayLevel, cfg.AmpEnv.ReleaseDuration),
		params.E("Envelopes", "filter", "filter", cfg.FilterEnv.AttackDuration, cfg.FilterEnv.DecayDuration, cfg.FilterEnv.DecayLevel, cfg.FilterEnv.ReleaseDuration),
	}
	return s
}

func orOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}
