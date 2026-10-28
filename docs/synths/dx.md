# dx

`synths/dx`

A 6 operator FM engine compatible with DX7 style voice data: the 32 classic algorithms and the
full set of voice parameters, so you can load SysEx banks (`.syx` files) and play them.

```go
patches, err := dx.LoadSysEx("my_bank.syx") // a 32 voice bulk dump, or single voice dumps
if err != nil {
	panic(err)
}

synth := dx.NewPoly(patches[0], 12, psalm.Stereo).AddTo(p)
dx.SetPatch(synth, patches[5]) // switch all voices to another patch
```

No factory sounds are included: load banks you own or that are shared for free use. Four
original patches are built in (`dx.Presets`: `glassKeys`, `woodBass`, `softBrass`, `bellPad`),
and `dx.InitVoice()` is the classic single sine starting point.

## Beyond the DX7

Two things the DX7 doesn't have, kept in the patch beside the SysEx data (they are not sent as SysEx):
a **wave per operator** (`op1Wave`...`op6Wave`, 0-7: sine, half sine, abs sine, pulse sine, even sine,
even abs, square, log saw: the TX81Z's eight, `dx.WaveNames`), and the [voice filter](../../synths.md#the-voice-filter)
(`filterType`...`filterRelease`, 0-99). The editor has both.

## The FM presets: exact operators

dx took over the 4 operator FM synth (`synths/fm`, gone now). Its presets are dx patches with
**exact operators** (`Operator.Free`, switched on by `FreeOn`, the setting `op1Exact` in the editor): a
sine operator with a frequency ratio of any size, cents of detune, a level and an ADSR envelope in
milliseconds, instead of the 0-99 values of the DX7, and the same sound as before. Their routing is
a `Routing` of your own (the four operators, feedback on the fourth), and the 0-99 values beside it
are the nearest to the exact ones, for the editor and the SysEx. Not part of the SysEx data.

```go
epiano := dx.NewPoly(dx.EPianoPreset(), 16, psalm.Stereo).AddTo(p)

patch := dx.Presets["bell"]()
patch.ModDepth = 0.7                                  // a bit darker
patch.Ops[1].Free.Ratio = 5                           // the ratio of operator 2
patch.Ops[0].Free.Env = *adsr.NewSetting(1.0, 4.0, 0.0, 6000.0, 1.0, 3000.0)
bells := dx.NewPoly(patch, 8, psalm.Stereo).AddTo(p)
```

Per operator (`patch.Ops[0].Free` is operator 1; a patch from the presets has the first four):

| Field | Meaning |
|---|---|
| `Ratio` | Frequency ratio to the note (1 unison, 2 an octave up, 3.5 inharmonic) |
| `Fixed` | Fixed frequency in Hz instead of a ratio (when > 0), for pitch independent tones |
| `Detune` | Cents |
| `Level` | Carrier: output level. Modulator: modulation index (1 is 2*pi radians) |
| `VelocitySens` | 0-1, how much the note velocity scales the level |
| `Env` | ADSR (`adsr.NewSetting(...)`, durations in ms) |

For the patch: `FreeFeedback` (0-1, the fourth operator modulates itself: towards a saw, then
noise), `ModDepth` (scales the exact modulators: overall brightness, 0 is 1), `Glide` (portamento in
ms from the previous note of the voice), `VibratoRate`, `VibratoDepth`, `VibratoDelay` (Hz,
semitones, ms), `Level` (the output level). The note amplitude sets the volume; `velocity` (0-1) in
the note message drives the velocity sensitivity.

**Messages** (from the next note on): `op1Ratio`, `op1FixedHz`, `op1Cents`, `op1Gain` (and the same
for `op2`...`op6`), the envelopes `op1Attack`, `op1Decay`, `op1Sustain`, `op1Release` (ms) and `amp...`
(`ampRelease`: the envelope of the carriers), `modDepth`, `freeFeedback`, `glide`, `vibratoRate`,
`vibratoDepth`, `vibratoDelay`, and `fmAlgorithm` (0-7, one of the routings below). Value input 1
scales the modulators (an LFO for timbre sweeps).

The routings of the four operators (`a>b`: a modulates b):

| `fmAlgorithm` | Routing | Carriers | Typical use |
|---|---|---|---|
| 0 | 4>3>2>1 | 1 | Basses, leads, complex evolving tones |
| 1 | (4+3)>2>1 | 1 | Brass, metallic sounds |
| 2 | (4 + 3>2)>1 | 1 | Plucks |
| 3 | (4>3 + 2)>1 | 1 | Variations of the above |
| 4 | 4>3, 2>1 | 1, 3 | Two layered tones: electric piano, bells, mallets |
| 5 | 4>(3, 2, 1) | 1, 2, 3 | Pads and ensembles |
| 6 | 4>3, 2, 1 | 1, 2, 3 | One modulated tone plus two sines |
| 7 | 4, 3, 2, 1 | all | Additive: organs |

**Modulation matrix**: `Patch.Mods` is an 8 slot [modulation matrix](../subtractive.md#modulation-matrix) like the
subtractive synth's, with the filter and carrier envelopes, the LFO, a second LFO (`LFO2Rate`),
velocity, key, random, wheel and `Ext1`...`Ext4` (value inputs 5 to 8, `dx.InExt1`) as sources, and
`modDepth`, `freeFeedback` (the feedback), `op1Gain`...`op6Gain`, `pitch`, `level`, `cutoff` and
`resonance` (of the filter) and `lfo2Rate` as destinations (`dx.Dests`).

How FM works here: every operator is a sine oscillator with its own envelope. A **carrier** is
heard, a **modulator** bends the phase of the operator it is connected to, which adds partials
(sidebands) around it at the carrier frequency plus and minus multiples of the modulator frequency.
The modulator level sets how many sidebands appear (the modulation index), so a modulator envelope
that decays makes the sound go from bright to mellow, like a struck string. Whole number ratios give
harmonic sounds, other ratios inharmonic (bell, metal) ones; velocity makes modulators louder.

### Presets

| Name | Function | Sound |
|---|---|---|
| `ePiano` | `EPianoPreset` | Tine electric piano, brighter when played harder |
| `bell` | `BellPreset` | Clear bell with inharmonic partials |
| `bass` | `BassPreset` | Punchy bass with a snappy attack |
| `brass` | `BrassPreset` | Brass section with a swelling attack and vibrato |
| `organ` | `OrganPreset` | Additive organ (sub, unison, octave, octave and fifth) |
| `mallet` | `MalletPreset` | Woody marimba like mallet |
| `pad` | `PadPreset` | Warm, slowly evolving pad |
| `pluck` | `PluckPreset` | Bright plucked string |
| `metallic` | `MetallicPreset` | Gong / metal plate |
| `lead` | `LeadPreset` | Bright lead with feedback edge, glide and vibrato |

More presets cover the instrument families of General MIDI:

| Family | Presets |
|---|---|
| Pianos and keys | `grandPiano`, `honkyTonk` (detuned second string), `harpsichord`, `clavinet` |
| Chromatic percussion | `celesta`, `glockenspiel`, `musicBox`, `xylophone`, `tubularBells` |
| Organs | `percussiveOrgan` (with a decaying third harmonic click), `churchOrgan`, `accordion` |
| Guitars | `nylonGuitar`, `steelGuitar`, `overdriveGuitar` |
| Basses | `uprightBass`, `slapBass`, `fretlessBass` (with glide) |
| Strings | `strings`, `pizzicato`, `harp` |
| Brass | `trumpet`, `frenchHorn` |
| Reeds and pipes | `saxophone`, `clarinet` and `oboe`, `flute` and `panFlute` (breath from a feedback noise operator) |
| Synth leads and pads | `squareLead`, `sawLead`, `warmPad`, `sweepPad` (brightens over a few seconds) |
| Ethnic | `sitar`, `kalimba`, `steelDrum`, `koto` |
| Percussion | `timpani`, `woodblock`, `taiko` |

The function names follow the preset names (`GrandPianoPreset`, `TubularBellsPreset`, ...). The classic
FM building blocks, handy when designing your own: a modulator at ratio 1 gives all harmonics
(strings, brass, saw like), at ratio 2 only odd harmonics (`clarinet`, `squareLead`); a high ratio
modulator with a very fast decay gives the attack of a hammer, pick or mallet (`grandPiano`,
`xylophone`); inharmonic ratios (1.41, 2.76, 3.5, ...) give bells, bars and drums; feedback gives
buzz (`sitar`, `overdriveGuitar`), a saw (`sawLead`), or at its maximum noise (breath in `flute`,
the skin hit of `taiko`).

## Patches

A `dx.Patch` has six `Operator`s (`Ops[0]` is operator 1) and the voice settings, all in the
classic 0-99 style ranges:

| Operator setting | Meaning |
|---|---|
| `EGRates`, `EGLevels` | 4 rates and 4 levels: from level 4 the envelope moves to level 1, 2 and 3, holds level 3 while the note is held and returns to level 4 on release |
| `OutputLevel` | 0-99, about 0.75 dB per step |
| `FixedFreq`, `Coarse`, `Fine` | Ratio mode: coarse ratio (0 is 0.5) times 1 + fine/100. Fixed mode: 1, 10, 100 or 1000 Hz (coarse) times 10^(fine/100) |
| `Detune` | 0-14, 7 is centered |
| `BreakPoint`, `LeftDepth`, `RightDepth`, `LeftCurve`, `RightCurve` | Keyboard level scaling around the break point (39 is C3), curves -LIN, -EXP, +EXP, +LIN |
| `RateScaling` | 0-7, higher notes get faster envelopes |
| `VelocitySens` | 0-7, softer notes get a lower level |
| `AmpModSens` | 0-3, sensitivity to LFO amplitude modulation |

| Voice setting | Meaning |
|---|---|
| `Algorithm` | 0-31 (algorithm 1-32), see `dx.Algorithms` |
| `Feedback` | 0-7 |
| `PitchEGRates`, `PitchEGLevels` | Pitch envelope, level 50 is no shift |
| `OscKeySync` | Restart the operators at every note |
| `LFOSpeed`, `LFODelay`, `LFOWaveform` | LFO (triangle, saw down, saw up, square, sine, sample & hold) |
| `LFOPitchDepth`, `PitchModSens` | Vibrato depth |
| `LFOAmpDepth` | Tremolo depth (with the operators' `AmpModSens`) |
| `LFOKeySync` | Restart the LFO at every note |
| `Transpose` | 0-48, 24 is no transposition |
| `Name` | Up to 10 characters |

## SysEx

| Function | Meaning |
|---|---|
| `dx.LoadSysEx(path)` / `dx.ParseSysEx(data)` | Load a 32 voice bulk dump (4104 bytes) or one or more single voice dumps (163 bytes each), checksums are verified |
| `dx.EncodeBank(patches)` | Write a 32 voice bulk dump (padded with init voices) |
| `dx.EncodeSingle(patch)` | Write a single voice dump |

## Playing

Notes take an optional `velocity` (0-1, default 1) in their message for the velocity
sensitivity, the note amplitude sets the volume. Value input 1 (or message `modScale`) scales all
modulators, an overall brightness control that glides smoothly.

The engine maps the 0-99 parameters to sound with approximations of the original behavior
(level and rate curves, LFO speeds, feedback, pitch envelope range). Patches sound recognizably
like themselves, but not bit identical to the hardware. The approximations are collected in
`synths/dx/eg.go` and easy to tune. Like the original, rising envelope stages are faster than
falling ones at the same rate, and the output has a 10 Hz DC blocker (FM puts a sideband at 0 Hz
when operator frequencies coincide, the hardware removes it in its output stage).

`TestSysExBanks` plays every voice of every bank in `assets/dx7` (if present) and checks for NaN,
clipping, DC offset and voices that never finish.

**Messages**: `pitch`, `frequency`, `velocity`, `modScale` (brightness), and the [pitch bend, mod wheel and aftertouch](../../synths.md#pitch-bend-mod-wheel-and-aftertouch).
Every setting of the patch by its name, played from the next note: `algorithm` (1-32), `feedback`,
`transpose` (24 in tune), `oversample` (0-2, see below), `oscSync`, `lfoWaveform`, `lfoSpeed`, `lfoDelay`, `lfoPitchDepth`,
`lfoAmpDepth`, `lfoSync`, `pitchModSens`, `pitchRate1`-`4`, `pitchLevel1`-`4`, and for each
operator `op<N>Coarse`, `Fine`, `Detune`, `Level`, `Fixed`, `VelocitySens`, `RateScaling`,
`AmpModSens`, `Rate1`-`4`, `Level1`-`4`, `BreakPoint`, `LeftDepth`, `RightDepth`, `LeftCurve`,
`RightCurve` (`"op3Coarse": 2.0`), in the DX7's own ranges (`dx.Settings` lists them,
`Patch.Set` and `Patch.Get` take them by name). A whole `patch` (a `dx.Patch`) or a `routing`.

**Oversampling** (`oversample`, `Patch.Oversample`, off by default in a patch; 22 hard presets have it on: four times a sample for `sawLead`, `lead`, `sitar`, `overdriveGuitar`, `metallic`, `acidBass`, `tubularBells`, `pulseClav`, `logSawBrass`, `hollowReed`, `saxophone`, `woodClarinet`, `pwmLead`, `syncLead` and `reedOrgan`, twice for seven more, see `synths/dx/presets_oversample.go`): 1 plays the operators twice a sample and 2 four times, and a lowpass (`blocks/oversample`) brings them down, so the sidebands that hard FM (a high index, feedback, a high note) puts above the Nyquist frequency do not fold back as inharmonic alias. The envelopes, the LFO, the filter and the DC blocker run once a sample; the sound is delayed by about 0.4 ms. Most presets have little to gain (the median of the harmonic ones improves by about 1 dB), the hard ones a lot: `syncLead` at C5 has its alias at -20 dB, at -27 dB with 2 and at -33 dB with 4, for 2 and 4 times the work of the operators. `TestOversamplingKeepsTheSoundAndLowersTheAlias` measures it.

## An algorithm of your own

`Patch.Routing` is an algorithm of your own instead of one of the 32: which operator modulates which
(a modulator has a higher number than the operator it modulates: the voice runs them from 6 down to
1), the carriers (heard; a carrier can modulate too), and a feedback loop from an operator into
itself or a higher one. `Algorithm.Check` says what is wrong with one; one that can't play falls
back to the patch's algorithm. A routing isn't part of the SysEx data.

```go
p := dx.Presets["bellPad"]()
p.Routing = &dx.Algorithm{
	Carriers:     []int{0, 1},                       // operators 1 and 2 heard
	Mods:         [][2]int{{1, 0}, {5, 1}, {4, 1}},   // 2 into 1, 6 and 5 into 2
	FeedbackFrom: 5, FeedbackTo: 5,                 // 6 into itself
}
synth := dx.NewPoly(p, 12, 2)
```

The panel has an editor of all of it, the algorithm drawn as a diagram: see
[dxeditor](../controls.md#panel) and `cmd/examples/dxedit`.
