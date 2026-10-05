# Synths

## poly

`modules/poly`

Polyphonic voice allocation for any voice type. `poly.New(numChannels, voices)` mixes the voices
to one output and assigns notes to free voices. When all voices are busy it steals the oldest
one, with a short fade out so stealing doesn't click.

A voice implements `poly.Voice`:

```go
type Voice interface {
	psalm.Modular
	NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, config psalm.Config)
	NoteOff()
	Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, config psalm.Config)
	Clear()
	IsActive() bool
}
```

**Messages** (usually built with `templ.NewTrigger` / `templ.NewDurationTrigger`, see [controls](controls.md#templ)):

| Message | Effect |
|---|---|
| `{"command": "trigger", "tick", "amplitude", "message", "duration"}` | Plays a note of `duration` seconds on a free voice, the voice receives `message` (pitch, settings, ...) |
| `{"command": "trigger", "tick", "amplitude", "message", "noteOn": id}` | Starts a held note, remembered under `id` |
| `{"command": "trigger", "tick", "noteOff": id}` | Releases the held note `id` |
| `{"command": "voice", ...}` | Passes the whole message to all voices (change settings of every voice) |

**Value inputs**: plain values on inputs above 0 are passed to all voices on the same input, so
an LFO connected to input 1 of a subtractive poly synth modulates the cutoff of every voice.

`AllNotesOff()` releases all held notes, `CallVoices(func(v))` runs a function on every voice.

## subtractive

`synths/subtractive`

A classic subtractive synth voice: two band limited oscillators, a sub oscillator, noise, ring
modulation, sync and FM, through a choice of filters (state variable, Moog style ladder or
formant), with filter and amp envelopes, two LFOs and glide.

```go
synth := subtractive.NewPoly(subtractive.LeadPreset(), 8).AddTo(p) // config, number of voices

cfg := subtractive.Presets["acidBass"]() // presets by name
cfg.Glide = 60
bass := subtractive.NewPoly(cfg, 1).AddTo(p) // one voice: every note glides from the last
```

`subtractive.New(cfg)` creates a single voice, `subtractive.NewPoly(cfg, n)` a `poly` synth with
`n` voices. `subtractive.DefaultConfig()` is a two saw voice with a 12 dB lowpass.

Signal flow: `osc 1 + osc 2 + sub + ring mod + noise -> drive -> filter -> amp (envelope, tremolo)`

### Oscillators

| Setting | Default | Meaning |
|---|---|---|
| `Osc1Waveform`, `Osc2Waveform` | Sawtooth | Any `blosc.Waveform` (see [blocks](blocks.md#blosc)) |
| `Osc1Level`, `Osc2Level` | 0.5, 0.5 | Mix levels |
| `Osc2Detune` | 0.07 | Osc 2 detune in semitones (fractions for cents, 7 for a fifth, 12 for an octave) |
| `Osc2Octave` | 0 | Osc 2 octave offset |
| `PulseWidth` | 0.5 | Pulse width (0-1) for the pulse width aware waveforms (Rectangle, ...) |
| `SubLevel` | 0 | Square sub oscillator one octave below osc 1 |
| `NoiseLevel` | 0 | Noise level |
| `NoiseType` | `WhiteNoise` | `WhiteNoise` or `PinkNoise` |
| `RingModLevel` | 0 | Level of osc 1 x osc 2 (metallic, bell like tones) |
| `Sync` | false | Hard sync osc 2 to osc 1 |
| `FMAmount` | 0 | Osc 1 frequency modulates osc 2 |
| `Drive` | 1 | Saturation before the filter (1 is clean) |

### Pitch

| Setting | Default | Meaning |
|---|---|---|
| `Glide` | 0 | Portamento in milliseconds from the previous note of the same voice (use 1 voice for mono slides) |
| `FilterEnvToPitch` | 0 | Semitones of pitch sweep from the filter envelope (zaps, kicks, falling sounds) |

### Filter

| Setting | Default | Meaning |
|---|---|---|
| `FilterType` | `FilterSVF` | `FilterSVF` (12 dB state variable), `FilterLadder` (Moog style 24/12 dB ladder) or `FilterFormant` (vowel filter) |
| `FilterMode` | `LowPass` | SVF output: `LowPass`, `BandPass`, `HighPass`, `Notch`, `Peak`, `AllPass` |
| `LadderMode` | `ladder.LowPass24` | Ladder output: `LowPass24`, `LowPass12`, `BandPass24`, `BandPass12`, `HighPass24`, `HighPass12` |
| `FilterDrive` | 1 | Ladder input saturation |
| `Cutoff` | 800 | Cutoff in Hz |
| `Resonance` | 0.3 | 0-1 (the ladder self-oscillates near 1), for the formant filter the sharpness of the formants |
| `FilterEnvAmount` | 3 | Octaves the filter envelope adds to the cutoff (vowel steps for the formant filter, can be negative) |
| `KeyTrack` | 0.5 | 0-1, how much the cutoff follows the pitch (relative to middle C) |
| `Vowel` | 0 | Formant filter vowel 0-4 (a, e, i, o, u, fractions morph) |
| `FormantShift` | 1 | Scales the formant frequencies (above 1 a smaller voice, below 1 a bigger one) |

With the formant filter, all cutoff modulation (filter envelope, LFO cutoff amounts) moves the
vowel instead of a cutoff, in vowel steps.

### Envelopes

`FilterEnv` and `AmpEnv` are ADSR settings (`adsr.NewSetting(attackLevel, attackMs, sustainLevel,
decayMs, sustainMs, releaseMs)` from `github.com/almerlucke/genny/float/envelopes/adsr`). Held
notes stay at the sustain level until note off, timed notes release after their duration.

### LFOs

`LFO1` and `LFO2` are `subtractive.LFOConfig`, per voice:

| Field | Meaning |
|---|---|
| `Waveform` | `LFOSine`, `LFOTriangle`, `LFOSaw`, `LFORamp`, `LFOSquare`, `LFOSampleHold` |
| `Rate` | Hz |
| `Delay` | Fade in time in milliseconds after note on (delayed vibrato) |
| `KeySync` | Restart the LFO on every note |
| `Pitch` | Semitones of vibrato |
| `Cutoff` | Octaves of cutoff modulation (vowel steps for the formant filter) |
| `PulseWidth` | Pulse width modulation |
| `Amp` | Tremolo depth 0-1 |

An LFO without any destination amount, or with rate 0, is off.

### Inputs and messages

**Value inputs** (connect LFO modules or generators to the poly synth): 1 cutoff, 2 resonance,
3 pulse width, 4 vowel.

**Messages** (in the note message of a trigger, or with `"command": "voice"`):

| Group | Keys |
|---|---|
| Pitch | `pitch` (MIDI), `frequency` (Hz), `glide`, `filterEnvToPitch` |
| Oscillators | `osc1Waveform`, `osc2Waveform`, `osc1Level`, `osc2Level`, `osc2Detune`, `osc2Octave`, `pulseWidth`, `subLevel`, `noiseLevel`, `noiseType`, `ringModLevel`, `sync`, `fmAmount`, `drive` |
| Filter | `filterType`, `filterMode`, `ladderMode`, `filterDrive`, `cutoff`, `resonance`, `filterEnvAmount`, `keyTrack`, `vowel`, `formantShift` |
| LFOs | `lfo1Waveform`, `lfo1Rate`, `lfo1Delay`, `lfo1KeySync`, `lfo1Pitch`, `lfo1Cutoff`, `lfo1PulseWidth`, `lfo1Amp`, and the same with `lfo2` |

Settings sent with a note change that voice and stay until changed again, so a sequence can step
through settings note by note (see `cmd/examples/subtractive2`).

### Presets

All presets are functions returning a `Config`, also available by name in `subtractive.Presets`.

| Name | Function | Sound |
|---|---|---|
| `bass` | `BassPreset` | Punchy saw bass with a square an octave below, 24 dB ladder |
| `acidBass` | `AcidBassPreset` | Squelchy resonant saw through a driven ladder |
| `subBass` | `SubBassPreset` | Clean sine sub bass |
| `reeseBass` | `ReeseBassPreset` | Heavily detuned saws with a slow cutoff LFO |
| `fmBass` | `FMBassPreset` | Growly FM bass |
| `squareBass` | `SquareBassPreset` | Rubbery pulse bass with short glide |
| `ladderBass` | `LadderBassPreset` | Fat, driven 24 dB ladder bass |
| `wobble` | `WobblePreset` | Resonant bass for an external cutoff LFO |
| `lfoWobble` | `LFOWobblePreset` | Wobble bass driven by its own LFO |
| `lead` | `LeadPreset` | Bright detuned saw lead |
| `ladderLead` | `LadderLeadPreset` | Warm driven ladder lead with glide and vibrato |
| `syncLead` | `SyncLeadPreset` | Hard sync lead |
| `chipLead` | `ChipLeadPreset` | 8-bit square and pulse lead |
| `chipArp` | `ChipArpPreset` | Thin pulse with a very short envelope for arpeggios |
| `pluck` | `PluckPreset` | Short saw pluck |
| `pad` | `PadPreset` | Wide, slow detuned saw pad |
| `hollowPad` | `HollowPadPreset` | Darker pulse and triangle pad |
| `pwmPad` | `PWMPadPreset` | Pulse width modulated pad with vibrato |
| `choirPad` | `ChoirPadPreset` | "ooh/aah" pad through a morphing formant filter |
| `notchPad` | `NotchPadPreset` | Detuned saws through a swept notch, phaser like |
| `strings` | `StringsPreset` | String ensemble style |
| `brass` | `BrassPreset` | Saw brass with a filter swell, 12 dB ladder |
| `organ` | `OrganPreset` | Square and octave triangle organ |
| `ePiano` | `EPianoPreset` | FM electric piano with tremolo |
| `glassKeys` | `GlassKeysPreset` | Glassy FM bell keys |
| `vibes` | `VibesPreset` | Mallet tone with tremolo |
| `ringBell` | `RingBellPreset` | Ring modulated bell |
| `flute` | `FlutePreset` | Breathy triangle with delayed vibrato |
| `clarinet` | `ClarinetPreset` | Woody square |
| `theremin` | `ThereminPreset` | Sine with long glide and wide vibrato (use 1 voice) |
| `talkingSynth` | `TalkingSynthPreset` | Talk box like formant sweep |
| `synthKick` | `SynthKickPreset` | Analog style kick (play around MIDI 36) |
| `synthSnare` | `SynthSnarePreset` | Tone and noise snare (around MIDI 50) |
| `synthHat` | `SynthHatPreset` | Highpassed noise hat |
| `synthTom` | `SynthTomPreset` | Pitched tom, the pitch sets the size |
| `zap` | `ZapPreset` | Laser zap from a fast pitch drop |
| `riser` | `RiserPreset` | Noise and saw sweep that rises over a few seconds |
| `wind` | `WindPreset` | Noise through a resonant bandpass |

## drums

`synths/drums`

`drums.NewDrums(soundBank, numVoices, numChannels)` is a `poly` synth of sample `player` voices.
Each trigger plays a sound from the bank, chosen with the `"sound"` key in the note message
(`"speed"` changes its pitch):

```go
drms := drums.NewDrums(soundBank, 16, 2).AddTo(p)
drms.ValueIn(templ.NewTrigger(0.5, template.Template{"sound": "kick", "speed": 1.0}).
	ValueIn(timer.WithTicks(float64(psalm.PPQ())).AddTo(p)))
```
