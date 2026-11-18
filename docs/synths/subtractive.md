# subtractive

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

## Oscillators

| Setting | Default | Meaning |
|---|---|---|
| `Osc1Waveform`, `Osc2Waveform` | Sawtooth | Any `blosc.Waveform` (see [blocks](../blocks.md#blosc)) |
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

`Unison` (1-7) plays that many detuned copies of both oscillators per voice (free running, `UnisonDetune`
cents between the lowest and the highest, the level kept in check); `Drift` (cents) lets the pitch of a
voice wander slowly, as an analog oscillator does. Messages `unison`, `unisonDetune`, `drift`; the matrix
reaches `unisonDetune` too.

## Pitch

| Setting | Default | Meaning |
|---|---|---|
| `Glide` | 0 | Portamento in milliseconds from the previous note of the same voice (use 1 voice for mono slides) |
| `GlideLegato` | false | Glide only when the voice is still sounding as it gets the note (a voice stolen from a chord with more notes than voices slides to the new pitch), a free voice jumps straight to it |
| `FilterEnvToPitch` | 0 | Semitones of pitch sweep from the filter envelope (zaps, kicks, falling sounds) |

## Filter

| Setting | Default | Meaning |
|---|---|---|
| `FilterType` | `FilterSVF` | `FilterSVF` (12 dB state variable), `FilterLadder` (Moog style 24/12 dB ladder) or `FilterFormant` (vowel filter) |
| `FilterMode` | `LowPass` | SVF output: `LowPass`, `BandPass`, `HighPass`, `Notch`, `Peak`, `AllPass` |
| `LadderMode` | `ladder.LowPass24` | Ladder output: `LowPass24`, `LowPass12`, `BandPass24`, `BandPass12`, `HighPass24`, `HighPass12` |
| `FilterDrive` | 1 | Ladder input saturation |
| `FilterOversample` | 1 | The ladder and the saturation of the oscillators (`Drive`) run once (0), twice (1) or four times (2) a sample, so the harmonics of their saturation do not fold back: the ladder presets have 6 to 20 dB less alias at twice a sample, and a hard drive (`rockOrgan`, `acidBass`, `acidSquare`, `growlBass`, set to four times) 20 to 60 dB less |
| `Cutoff` | 800 | Cutoff in Hz |
| `Resonance` | 0.3 | 0-1 (the ladder self-oscillates near 1), for the formant filter the sharpness of the formants |
| `FilterEnvAmount` | 3 | Octaves the filter envelope adds to the cutoff (vowel steps for the formant filter, can be negative) |
| `KeyTrack` | 0.5 | 0-1, how much the cutoff follows the pitch (relative to middle C) |
| `Vowel` | 0 | Formant filter vowel 0-4 (a, e, i, o, u, fractions morph) |
| `FormantShift` | 1 | Scales the formant frequencies (above 1 a smaller voice, below 1 a bigger one) |

With the formant filter, all cutoff modulation (filter envelope, LFO cutoff amounts) moves the
vowel instead of a cutoff, in vowel steps.

## A second filter

`Filter2Mode` (off, low, band, high, notch) adds a second state variable filter, after the first or
beside it (`Filter2Parallel`: the two mixed): `Filter2Cutoff`, `Filter2Resonance`, `Filter2EnvAmount`
(octaves, from the filter envelope) and `Filter2KeyTrack`. A low pass and a high pass in a row make a
band; a notch beside a low pass keeps the lows and a hole above. Messages `filter2Mode`,
`filter2Cutoff`, `filter2Resonance`, `filter2EnvAmount`, `filter2KeyTrack`, `filter2Parallel`; the
matrix reaches `filter2Cutoff`.

## Envelopes

`FilterEnv` and `AmpEnv` are ADSR settings (`adsr.NewSetting(attackLevel, attackMs, sustainLevel,
decayMs, sustainMs, releaseMs)` from `github.com/almerlucke/genny/float/envelopes/adsr`). Held
notes stay at the sustain level until note off, timed notes release after their duration.

## LFOs

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

## Modulation matrix

`Config.Mods` is a matrix of 8 slots (`blocks/modmatrix`): each takes a source to a destination over
an amount (-1 to 1, of the span of the destination). It does what the fixed LFO amounts can't:
per note, from any source, to nearly any setting.

```go
cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "cutoff", Amount: 0.5}   // wheel opens the filter 3 octaves
cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "pulseWidth", Amount: 1}
cfg.Mods[2] = modmatrix.Slot{Src: modmatrix.FilterEnv, Dest: "lfo1Rate", Amount: 0.3}
```

| Sources | |
|---|---|
| `FilterEnv`, `AmpEnv` | the envelopes of the voice, 0-1 |
| `LFO1`, `LFO2` | the LFOs, -1 to 1 (they run when a slot uses them, rate from `LFO1.Rate`) |
| `Velocity` | the note's velocity, 0-1 |
| `Key` | the note, -1 to 1 over five octaves either side of middle C |
| `Random` | a new value -1 to 1 for every note |
| `Wheel` | the mod wheel and the aftertouch, 0-1 |
| `Ext1`...`Ext4` | values 0-1 cabled in (inputs 5 to 8, below) |

The destinations are `subtractive.Dests` (a full source with a full amount moves it by its span):
`pitch` (24 st), `cutoff` (6 oct), `resonance`, `pulseWidth`, `osc1Level`, `osc2Level`, `subLevel`,
`noiseLevel`, `osc2Detune` (12 st), `fmAmount` (5), `drive` (3 oct), `vowel` (4), `level`, `lfo1Rate`
and `lfo2Rate` (3 oct). The slots are settings like any: `mod1Src`, `mod1Dest` (0 none, then the
destinations in order) and `mod1Amount` to `mod8...`, saved in a preset and edited in the editor
(group Modulation).

## Inputs and messages

**Value inputs** (connect LFO modules or generators to the poly synth): 1 cutoff, 2 resonance,
3 pulse width, 4 vowel (they set the setting itself); 5 to 8 (`subtractive.InExt1`...) the sources
Ext 1 to 4 of the matrix (0-1): they move whatever a slot joins them to, over the setting, so an LFO
module on input 5 with a slot `Ext1` to `cutoff` is a vibrato of the filter that keeps the knob.
`ext1`...`ext4` are the same as settings (a mod cable of the patcher can move them).

**Messages** (in the note message of a trigger, or with `"command": "voice"`):

| Group | Keys |
|---|---|
| Pitch | `pitch` (MIDI), `frequency` (Hz), `glide`, `filterEnvToPitch` |
| Oscillators | `osc1Waveform`, `osc2Waveform`, `osc1Level`, `osc2Level`, `osc2Detune`, `osc2Octave`, `pulseWidth`, `subLevel`, `noiseLevel`, `noiseType`, `ringModLevel`, `sync`, `fmAmount`, `drive` |
| Filter | `filterType`, `filterMode`, `ladderMode`, `filterDrive`, `filterOversample`, `cutoff`, `resonance`, `filterEnvAmount`, `keyTrack`, `vowel`, `formantShift` |
| LFOs | `lfo1Waveform`, `lfo1Rate`, `lfo1Delay`, `lfo1KeySync`, `lfo1Pitch`, `lfo1Cutoff`, `lfo1PulseWidth`, `lfo1Amp`, and the same with `lfo2` |
| Matrix | `mod1Src`, `mod1Dest`, `mod1Amount` (to `mod8`), `ext1` to `ext4` |

Settings sent with a note change that voice and stay until changed again, so a sequence can step
through settings note by note (see `cmd/examples/subtractive2`).

## Presets

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
