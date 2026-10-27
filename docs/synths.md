# Synths

Synths (the presets of each are listed in [presets](presets.md)): [pitch bend, mod wheel and aftertouch](#pitch-bend-mod-wheel-and-aftertouch), [envelopes by message](#envelopes-by-message), [poly](#poly) (voice allocation, mono and legato), [subtractive](#subtractive), [dx](#dx) (6 operator FM, SysEx, and the 4 operator FM presets), [grains](#grains) (granular sampler), [pulsar](#pulsar), [wavetable](#wavetable), [piano](#piano), [drums](#drums), [sampler](#sampler) (SFZ), [organ](#organ), [waveguide](#waveguide) (plucked and bowed strings), [modal](#modal) (mallets and bells), [drumsynth](#drumsynth), [vocal](#vocal) (voices and choirs), [shaper](#shaper) (waves from chains of shapers, morphing), [supersaw](#supersaw).

Every synth describes its settings: `<synth>.Describe(cfg)` (`params`: each message key it takes,
its range, unit and group, where it is in the config, and its envelopes), so an editor on the panel
(`modules/controls/panel/editor`) can draw its controls, a sequencer pick what to step, a randomiser
what to move. The drumsynth's sounds and the sampler's regions are set per sound, not by message:
they describe only their level.

Presets of all synths are calibrated to the same loudness: a note at amplitude 1 plays at -12 LUFS
(or peaks at -1 dBFS for percussive sounds), see [conventions](conventions.md#levels).

## Pitch bend, mod wheel and aftertouch

Every synth but the piano (as a real one) takes the hands of a player while it sounds: the
messages `pitchBend` (-1 to 1, over `bendRange` semitones, 2 by default), and `modWheel` (0 to 1),
which does something of its own on each, and `pressure` (0 to 1): the aftertouch, pressing harder
on a held key, which adds to the wheel (the sampler leaves it to its instrument). They glide to
where they are moved in a few milliseconds, so moving them never zips or clicks, and reach every
voice of a poly synth (a message without a command). A keyboard sends them with
`keyboard.Config{PitchBend: "pitchBend", Pressure: "pressure", Controls:
map[int]string{midi.CCModWheel: "modWheel"}}`.

| Synth | The mod wheel |
|---|---|
| subtractive | opens the filter, two octaves |
| dx | a vibrato (as on a DX7) |
| wavetable | moves through the table, opens the filter |
| pulsar | lifts the formant |
| grains | sprays the grains wider (the bend reaches new grains) |
| sampler | controller 1, as the instrument says (crossfades, `gain_cc1`); the bend follows `bend_up`, `bend_down` |
| organ | the scanner vibrato, up to V3 |
| waveguide | a vibrato (a plucked string bends while it rings) |
| modal | a tremolo |
| drumsynth | longer decays (new hits); the bend moves the sounds ringing |
| vocal | opens the vowel |
| shaper | opens the filter (when it is on); a slot of the matrix can send it to the morph or a macro |
| supersaw | opens the filter, three octaves |

The code of it is `blocks/expression`, for synths of your own.

## Envelopes by message

Every synth with envelopes takes them by message the same way (`blocks/envelope`), so a panel, an
envelope widget, an automation lane or a sequence can shape them while it plays:
`<name>Attack`, `<name>Decay` and `<name>Release` in milliseconds, `<name>Sustain` the level held
while the key is down (0-1). A change takes effect from the next note.

| Synth | Envelopes |
|---|---|
| subtractive, supersaw | `amp`, `filter` |
| wavetable | `amp`, `filter`, `position` |
| pulsar | `amp`, `formant` |
| shaper | `amp`, `morphEnv` (the morph), `shapeEnv` (macro 1), `filter` |
| dx (exact operators) | `amp` (the carriers: the operators heard), `op1` to `op6` (each operator) |
| grains | `amp` |
| sampler | `amp`: over the envelopes of the regions (each part set replaces theirs) |
| vocal | `ampAttack`, `ampRelease` |
| modal | `ampDecay` (how long it rings), `ampRelease` (damped once let go) |
| waveguide | `ampAttack` (the bow), `ampDecay` (a plucked string's ring), `ampRelease` (let go) |

The piano, the organ and the drumsynth have no envelopes of this kind (the piano is its hammers
and dampers, the organ is on while a key is down, a drum is its decay); dx keeps the rates and
levels of the DX7.

```go
synth.ReceiveMessage(psalm.Message{"ampAttack": 400.0, "ampRelease": 2500.0, "filterSustain": 0.2})
```

## The voice filter

`blocks/voicefilter`

A filter after the oscillators of a voice, the same on the synths that have no filter of their own
(dx, and more as they get it): state variable (low, band, high, notch) or a ladder (24 or 12 dB),
with its own envelope, key tracking and drive. It is off until its type is set.

| Message | |
|---|---|
| `filterType` | 0 off, 1 low pass, 2 band pass, 3 high pass, 4 notch, 5 ladder 24, 6 ladder 12 |
| `cutoff`, `resonance` | Hz, 0-1 |
| `filterEnvAmount` | octaves the envelope opens the cutoff |
| `keyTrack` | 0-1, the cutoff follows the note |
| `filterDrive` | ladder input saturation |
| `filterAttack`, `filterDecay`, `filterSustain`, `filterRelease` | the envelope (ms, sustain 0-1) |

In a config it is `Config.Filter` (a `voicefilter.Config`): `cfg.Filter = voicefilter.Config{Type:
voicefilter.LowPass, Cutoff: 1200, Resonance: 0.3, EnvAmount: 2}`. The dx synth keeps it in the patch,
the way the DX7's own settings are, 0 to 99: `filterType`, `filterCutoff`, `filterResonance`,
`filterEnvAmount`, `filterKeyTrack`, `filterAttack`, `filterDecay`, `filterSustain`, `filterRelease`
(`Patch.Filter`, not part of the SysEx data), with a row in the dx editor.

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
| `{"command": "note", "id": id, "message": {...}}` | Passes `message` to the voice of the held note `id` alone (in mono, when it is the note sounding): the polyphonic aftertouch of one key |
| `{"command": "voice", ...}`, or a message without a command | Passes the whole message to all voices (change settings of every voice, `AutomateMessage` of an arrangement works this way) |

**Value inputs**: plain values on inputs above 0 are passed to all voices on the same input, so
an LFO connected to input 1 of a subtractive poly synth modulates the cutoff of every voice.

`AllNotesOff()` releases all held notes, `CallVoices(func(v))` runs a function on every voice.

**Mono**: `SetMono(true, legato)` (or the message `{"mono": 1, "legato": 1}`) plays one note at a
time on one voice. A new note takes over from the one held, and when it is let go the note still
held comes back. Legato changes the pitch of the sounding note without starting it again, so a
synth with a glide (subtractive, supersaw) slides between the notes; without legato every note
starts the voice again, from the pitch it was at. Notes of a duration (from a score) that overlap
slide as well. `SetMono(false, false)` makes it polyphonic again.

```go
lead := supersaw.NewPoly(supersaw.GlideLead(), 4).SetMono(true, true).AddTo(p)
```

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

`Unison` (1-7) plays that many detuned copies of both oscillators per voice (free running, `UnisonDetune`
cents between the lowest and the highest, the level kept in check); `Drift` (cents) lets the pitch of a
voice wander slowly, as an analog oscillator does. Messages `unison`, `unisonDetune`, `drift`; the matrix
reaches `unisonDetune` too.

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

### A second filter

`Filter2Mode` (off, low, band, high, notch) adds a second state variable filter, after the first or
beside it (`Filter2Parallel`: the two mixed): `Filter2Cutoff`, `Filter2Resonance`, `Filter2EnvAmount`
(octaves, from the filter envelope) and `Filter2KeyTrack`. A low pass and a high pass in a row make a
band; a notch beside a low pass keeps the lows and a hole above. Messages `filter2Mode`,
`filter2Cutoff`, `filter2Resonance`, `filter2EnvAmount`, `filter2KeyTrack`, `filter2Parallel`; the
matrix reaches `filter2Cutoff`.

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

### Modulation matrix

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

### Inputs and messages

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
| Filter | `filterType`, `filterMode`, `ladderMode`, `filterDrive`, `cutoff`, `resonance`, `filterEnvAmount`, `keyTrack`, `vowel`, `formantShift` |
| LFOs | `lfo1Waveform`, `lfo1Rate`, `lfo1Delay`, `lfo1KeySync`, `lfo1Pitch`, `lfo1Cutoff`, `lfo1PulseWidth`, `lfo1Amp`, and the same with `lfo2` |
| Matrix | `mod1Src`, `mod1Dest`, `mod1Amount` (to `mod8`), `ext1` to `ext4` |

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

## dx

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

### Beyond the DX7

Two things the DX7 doesn't have, kept in the patch beside the SysEx data (they are not sent as SysEx):
a **wave per operator** (`op1Wave`...`op6Wave`, 0-7: sine, half sine, abs sine, pulse sine, even sine,
even abs, square, log saw: the TX81Z's eight, `dx.WaveNames`), and the [voice filter](#the-voice-filter)
(`filterType`...`filterRelease`, 0-99). The editor has both.

### The FM presets: exact operators

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

**Modulation matrix**: `Patch.Mods` is an 8 slot [modulation matrix](#modulation-matrix) like the
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

#### Presets

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

### Patches

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

### SysEx

| Function | Meaning |
|---|---|
| `dx.LoadSysEx(path)` / `dx.ParseSysEx(data)` | Load a 32 voice bulk dump (4104 bytes) or one or more single voice dumps (163 bytes each), checksums are verified |
| `dx.EncodeBank(patches)` | Write a 32 voice bulk dump (padded with init voices) |
| `dx.EncodeSingle(patch)` | Write a single voice dump |

### Playing

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

**Messages**: `pitch`, `frequency`, `velocity`, `modScale` (brightness), and the [pitch bend, mod wheel and aftertouch](#pitch-bend-mod-wheel-and-aftertouch).
Every setting of the patch by its name, played from the next note: `algorithm` (1-32), `feedback`,
`transpose` (24 in tune), `oscSync`, `lfoWaveform`, `lfoSpeed`, `lfoDelay`, `lfoPitchDepth`,
`lfoAmpDepth`, `lfoSync`, `pitchModSens`, `pitchRate1`-`4`, `pitchLevel1`-`4`, and for each
operator `op<N>Coarse`, `Fine`, `Detune`, `Level`, `Fixed`, `VelocitySens`, `RateScaling`,
`AmpModSens`, `Rate1`-`4`, `Level1`-`4`, `BreakPoint`, `LeftDepth`, `RightDepth`, `LeftCurve`,
`RightCurve` (`"op3Coarse": 2.0`), in the DX7's own ranges (`dx.Settings` lists them,
`Patch.Set` and `Patch.Get` take them by name). A whole `patch` (a `dx.Patch`) or a `routing`.

### An algorithm of your own

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
[dxeditor](controls.md#panel) and `cmd/examples/dxedit`.

## grains

`synths/grains`

A granular sampler you play like any synth: every note starts a stream of grains, short windowed
slices of a recording, played at the pitch of the note. Moving where the grains come from, how
dense and long they are and how much they spread turns any recording into pads, drones, keys,
textures or rhythmic stutters.

```go
choir := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-angelical-choir-654.wav", 4)
pad := grains.NewPoly(grains.ChoirPad(choir, 60), 12).AddTo(p) // sample, root note
pad.ValueIn(lfo.New(true, 0.05, 0.2, 0.8, lfo.Sine).AddTo(p), 0, 1) // slowly move the position
```

`grains.New(cfg)` creates a single voice, `grains.NewPoly(cfg, n)` a `poly` synth with `n` voices.
Use mipmapped sound files, so grains that play faster don't alias.

### Settings

| Setting | Default | Meaning |
|---|---|---|
| `Sample` | | The recording (`sndfile.SoundFiler[float32]`) |
| `SoundBank` | | Optional, notes pick a sample with the `sound` key |
| `RootNote` | 60 | MIDI pitch of the recording: playing this note plays it at its own pitch |
| `Position` | 0.3 | 0-1, where in the recording the grains come from |
| `Scan` | 0 | How fast the position moves: 0 freezes a moment, 1 is the original speed, negative goes backwards. The position restarts at every note |
| `Spray` | 0.05 | Seconds, random spread of the grain positions |
| `Density` | 30 | Grains per second |
| `GrainSize` | 120 | Milliseconds |
| `PitchSpread` | 0 | Semitones, random detune per grain (chorus, smear) |
| `Octave` | 0 | 0-1, chance a grain plays an octave up (sparkle, shimmer) |
| `StereoSpread` | 0.5 | 0-1, random pan per grain |
| `Shape` | 0 | 0-1, grain window from smooth to percussive |
| `Reverse` | 0 | 0-1, chance a grain plays backwards |
| `Jitter` | 0.3 | 0-1, randomness of the time between grains |
| `VelocityDensity` | 0 | 0-1, how much the note velocity raises the density |
| `Normalize` | true | Play every recording at the same loudness (measured over the parts that sound) |
| `Level` | 1 | Output level |
| `AmpEnv` | | ADSR of the whole note (ms) |

The level stays about the same when the density, grain size or window shape changes. Notes take
an optional `velocity` (0-1, default 1), the note amplitude sets the volume.

**Value inputs**: 1 position, 2 density, 3 scan, 4 spray, 5 grain size. They only affect new
grains, so modulating them never clicks.

**Messages**: `pitch`, `frequency`, `velocity`, `sound`, `rootNote`, `position`, `scan`, `spray`,
`density`, `grainSize`, `pitchSpread`, `octave`, `stereoSpread`, `shape`, `reverse`, `jitter`,
`level`.

### Presets

Presets take the sample and its root note, every preset works with any recording:

| Name | Function | Character |
|---|---|---|
| `choirPad` | `ChoirPad` | Slow, wide pad of long grains drifting through the sample |
| `frozenDrone` | `FrozenDrone` | One frozen moment, a living drone |
| `glassKeys` | `GlassKeys` | Playable keys: dense short grains, a plucked envelope, denser when played harder |
| `texture` | `Texture` | Restless cloud: wide spray, pitch smear, reversed grains |
| `stutter` | `Stutter` | Rhythmic, percussive grains repeating small pieces |
| `shimmer` | `Shimmer` | Bright pad with grains an octave up |
| `pluck` | `Pluck` | Short plucked tone |

Many recordings only sound in part of the file (a decaying keys sample, a phrase with pauses), set
`Position` to a part that sounds.

## pulsar

`synths/pulsar`

Pulsar synthesis: every note is a train of pulses at the pitch of the note, and every pulse (a
pulsaret) is a short burst of a sine at the formant frequency, shaped by a window. No samples
needed. Three things shape the sound:

- the **formant** puts a spectral peak at a fixed frequency that stays put when the pitch changes,
  like a vowel (two formants make actual vowels);
- the **duty** sets how much of every period the pulse fills, short pulses are bright and thin,
  full pulses round;
- the **pitch**: at very low notes (a few Hz) the pulses become a rhythm, raising the pitch turns
  that rhythm into a tone.

```go
lead := pulsar.NewPoly(pulsar.VowelLeadPreset(), 2).AddTo(p)
lead.ValueIn(templ.NewDurationTrigger(0.5, 0.6, template.Template{
	"pitch":   seq(57.0, 60.0, 64.0),
	"formant": seq(700.0, 450.0, 320.0), // ah, oh, oo
}).ValueIn(timer.WithTicks(float64(psalm.PPQ())).AddTo(p)))
```

`pulsar.New(cfg)` creates a single voice, `pulsar.NewPoly(cfg, n)` a `poly` synth with `n` voices.

| Setting | Default | Meaning |
|---|---|---|
| `Formant` | 800 | Hz, the frequency inside every pulse (the spectral peak) |
| `Formant2`, `Formant2Level` | 0, 0.5 | A second formant in Hz (0 is off) and its level |
| `FormantTracking` | 0 | 0-1, how much the formants follow the pitch (relative to middle C). 0 is a fixed vowel, 1 the same timbre on every note |
| `Duty` | 0.5 | 0-1, the part of every period the pulse sounds |
| `Window` | `Hann` | Pulse envelope: `Hann` (smooth), `Gaussian` (narrow, purer formant), `Percussive` (struck attack, fast decay), `Rectangle` (hard edges, buzzy, some aliasing), `Tapered` (a rectangle with short smooth edges: buzzy without the aliasing) |
| `Masking` | 0 | 0-1, chance a pulse is skipped: crackling, stochastic textures |
| `Jitter` | 0 | 0-1, randomness of the pulse period. Small values (below 0.005) add breath to a tone, larger ones make it rough |
| `StereoSpread` | 0 | 0-1, random pan per pulse. For sparse textures: on a tone the pan changes at the pitch rate, which sounds like noise |
| `NoteSpread` | 0 | 0-1, random pan per note, to widen chords and pads |
| `FormantEnv`, `FormantEnvAmount` | | ADSR (ms) and how many octaves it raises the formants |
| `FormantLFORate`, `FormantLFODepth` | 0, 0 | Hz and octaves of a sine LFO on the formants |
| `VibratoRate`, `VibratoDepth` | 0, 0 | Hz and semitones |
| `Glide` | 0 | Milliseconds from the previous note of the voice |
| `Level` | 1 | Output level |
| `AmpEnv` | | ADSR of the note (ms) |

Every pulse is scaled by its measured energy, so the level stays about the same when the duty,
window or formants change. Pulses shorter than about one formant cycle hold too little of the
formant to be brought up to level, they get quieter instead of turning into spikes. Sparse pulse
trains (low notes, masking) have a high peak for their loudness.

**Value inputs**: 1 formant, 2 duty, 3 masking, 4 second formant. Settings are picked up by the
next pulse, so modulating them never clicks.

**Messages**: `pitch`, `frequency`, `formant`, `formant2`, `formant2Level`, `formantTracking`,
`duty`, `window` (0-4), `masking`, `jitter`, `stereoSpread`, `noteSpread`, `formantEnvAmount`, `formantLFORate`,
`formantLFODepth`, `vibratoRate`, `vibratoDepth`, `glide`, `level`.

### Presets

| Name | Function | Character |
|---|---|---|
| `vowelLead` | `VowelLeadPreset` | Singing "ah" lead with two formants, formant drift, vibrato and glide |
| `buzzBass` | `BuzzBassPreset` | Buzzy bass with tapered pulses, the formant snaps down from two octaves up like a plucked filter |
| `glass` | `GlassPreset` | Bright glassy keys: high formants, a struck attack and a long decay |
| `formantPad` | `FormantPadPreset` | Slow, wide pad sweeping through vowel colours |
| `laser` | `LaserPreset` | Falling chirp, the formant sweeps down four octaves |
| `rhythm` | `RhythmPreset` | For very low notes (send a `frequency` of a few Hz): every pulse is a tick |
| `crackle` | `CracklePreset` | For low notes (30-100 Hz): masked, jittered pulses crackling across the stereo field |
| `choir` | `ChoirPreset` | Soft "ooh" choir pad with breath and slow vibrato |
| `strings` | `StringsPreset` | Bowed string section: short bright pulses, slow bow, vibrato |
| `brass` | `BrassPreset` | Brassy lead, the formant opens up during the attack |
| `mallet` | `MalletPreset` | Struck wooden mallet with a bright strike and quick decay |
| `subBass` | `SubBassPreset` | Deep round bass, almost a sine with a touch of the octave |
| `wobbleBass` | `WobbleBassPreset` | Talking bass, a fast LFO swings the formant like a wah, tapered pulses |
| `insects` | `InsectsPreset` | For low notes (30-60 Hz): chirping insects at night |
| `raindrops` | `RaindropsPreset` | For very low notes (8-20 Hz): irregular drops at random pitches and places |
| `motor` | `MotorPreset` | For low notes (15-40 Hz): a running engine, raise the frequency to rev it up (long glide) |

### Filter, matrix, second pulsaret and bursts

Pulsar has the [voice filter](#the-voice-filter) after the pulse train (off by default) and an 8 slot
[modulation matrix](#modulation-matrix): the formant envelope as filter env, the amp envelope, the
formant LFO (needs `FormantLFORate` > 0), a second LFO at `lfo2Rate`, velocity, key, random, wheel and
Ext 1-4 on inputs 5-8, to `formant`, `formant2` (octaves), `duty`, `masking`, `jitter`, `stereoSpread`,
`pitch` (semitones), `level`, `cutoff` (octaves), `resonance`, `shapeMix` or `lfoRate`. Pulse
destinations are picked up at the start of each pulse.

The pulse content has a second pulsaret (`pulsaret2`: saw, square, triangle, built from a few
harmonics of the formant so it stays clean) crossfaded with the formant sines by `shapeMix` (0 is the
sines only). `burst` and `burstRest` play N pulses and then rest M, for stuttering rhythms (0 burst is
off); the count restarts at each note.

With the filter on, the voice stays active after the amp envelope for the filter's release time (up to
5 s) and the amp is applied before the filter, so a resonant filter rings out its tail.

## wavetable

`synths/wavetable`

A wavetable synth: the oscillator plays one frame of a table (a single cycle) and morphs through the
frames as the **position** moves from 0 to 1. Stacked in unison, with a sine sub and a state variable
filter. Tables are loaded from sound files of consecutive frames, like the 2048 sample tables in
`assets/wavetables2048`.

```go
table := wavetable.MustLoad("assets/wavetables2048/DN Harm.wav", 2048)
pad := wavetable.NewPoly(wavetable.PadPreset(table), 16).AddTo(p)
```

`wavetable.New(cfg)` creates a single voice, `wavetable.NewPoly(cfg, n)` a `poly` synth with `n`
voices.

### Tables

| Function | Meaning |
|---|---|
| `Load(path, frameSize)`, `MustLoad` | A table from a sound file (first channel) of consecutive frames |
| `NewTable(samples, frameSize)` | A table from samples |
| `FromFunc(numFrames, frameSize, f)` | A table from a function of the position and the phase. Every cycle is drawn 16 times oversampled and reduced to the harmonics that fit, so shapes with jumps (pulses, sync) come out practically alias free |
| `FromHarmonics(numFrames, frameSize, f)` | A table from the amplitudes of the harmonics (sines) of every frame, exact |
| `FromSpectrum(numFrames, frameSize, f)` | A table from the sine and cosine amplitudes of the harmonics, for control over the phases |
| `Basic()` | Sine, triangle, sawtooth and square, played when the config has no table |
| `table.Save(path)` | Writes the table as a 32 bit float WAV of consecutive frames, the format most wavetable synths load |

Every frame gets its DC removed, the table is normalized to a peak of 1. Each frame is stored as
band limited levels, each with half the harmonics of the one before. A note reads the levels whose
highest harmonic stays below the Nyquist frequency, crossfading between them as the pitch moves, so
high notes and glides never alias. Load a table once and share it between voices and synths.

### Generated tables

`synths/wavetable/gen` generates tables (2048 sample frames): every generator sweeps one synthesis
parameter over the frames, so moving the position plays the sweep.

```go
pad := wavetable.NewPoly(wavetable.PadPreset(gen.Random(64, 6, 48, 7)), 16).AddTo(p)
_ = gen.Fold(64, 6, 0.2).Save("fold.wav")
```

| Generator | The position sweeps |
|---|---|
| `Brightness(numFrames)` | From a sine to a full sawtooth, the harmonics fading in from low to high like an opening lowpass |
| `PWM(numFrames, minWidth)` | A pulse from a square to `minWidth` (for example 0.02) |
| `Sync(numFrames, maxRatio)` | A hard synced sawtooth from ratio 1 (a plain sawtooth) to `maxRatio` times the cycle |
| `FM(numFrames, ratio, maxIndex)` | Two operator FM, the index from 0 (a sine) to `maxIndex`. The ratio is a whole number: 1 brightens towards a sawtooth, higher ratios give bells and metal |
| `Fold(numFrames, maxGain, bias)` | A sine into a wavefolder, the gain from 1 to `maxGain`. A bias (-1 to 1) adds even harmonics |
| `Vowels(numFrames, pitch)` | Through the vowels a, e, i, o, u: a voice like spectrum shaped by formants placed for a note of `pitch` Hz, most natural near that pitch |
| `Drawbars(numFrames, numHarmonics)` | Organ harmonics fading in one by one, from the fundamental to `numHarmonics` |
| `Random(numFrames, keyframes, numHarmonics, seed)` | Smooth crossfades between keyframes of random spectra, every seed another table |
| `Filter(numFrames, q)` | A resonant lowpass opening over a sawtooth, the cutoff from the fundamental to harmonic 128 (`q` 0.7 smooth, 4 sharp, 10 whistling) |
| `Resonance(numFrames, maxRatio)` | The resonant waveform of phase distortion synths (Casio CZ), the resonance rising to `maxRatio` times the cycle |
| `OddEven(numFrames)` | Odd harmonics only (hollow, square like), through all (sawtooth), to the fundamental with only even harmonics |
| `Comb(numFrames, maxDelay)` | A sawtooth mixed with a copy delayed up to `maxDelay` (0-0.5) of the cycle, notches like a flanger. Half a cycle cancels the odd harmonics, an octave up |
| `Saturate(numFrames, maxDrive)` | A sine into a soft clipper, the drive from 1 to `maxDrive`, towards a square |
| `Quantize(numFrames, minLevels)` | A sine at fewer and fewer levels, from 8 bits (256) to `minLevels` |
| `Steps(numFrames, minSteps)` | A sine held in fewer and fewer steps per cycle, from 256 to `minSteps`: mirror images around multiples of the step count, the stair case sound of early wavetable synths |
| `Crossfade(numFrames, shapes...)` | Through any single cycle shapes in turn: `gen.Sine`, `gen.Triangle`, `gen.Saw`, `gen.Square` or your own functions of the phase |
| `FromSound(samples, numFrames)`, `LoadSound(path, numFrames)` | Slices of a recording taken evenly through the parts that sound (silence is skipped), each looped with a short crossfade |

A slice of a recording is not a single cycle. Its 2048 samples last one cycle at 21.5 Hz, so a note
plays the slice that many times faster: at 220 Hz the recording is about 10 times sped up and
sounds bright, buzzy or vocal. Play sliced tables low and without a lowpass, near 21.5 Hz (MIDI
note 5) a slice plays at its own speed.

The additive generators (vowels, drawbars, random) spread the phases of the harmonics (Schroeder
phases): it sounds the same, but the wave has a lower peak, so the table plays louder.

### Settings

| Setting | Default | Meaning |
|---|---|---|
| `Table` | nil | The table (`Basic()` when nil) |
| `Position` | 0 | 0-1, morphs through the frames |
| `PositionEnv`, `PositionEnvAmount` | | ADSR (ms) and how far it moves the position (-1 to 1) |
| `PositionLFORate`, `PositionLFODepth` | 0, 0 | Hz and how far a sine LFO moves the position either way |
| `Normalize` | true | Brings every frame to the same loudness (quiet frames at most 4 times), so moving the position changes the timbre, not the level |
| `Unison` | 1 | 1-7 oscillators |
| `Detune` | 20 | Cents between the lowest and highest unison oscillator |
| `Spread` | 0.7 | 0-1, stereo width of the unison oscillators |
| `SubLevel` | 0 | Sine an octave below |
| `Filter` | `FilterOff` | `FilterOff`, `LowPass`, `BandPass`, `HighPass` (state variable filter) |
| `Cutoff`, `Resonance` | 8000, 0.2 | Hz, 0-1 |
| `FilterEnv`, `FilterEnvAmount` | | ADSR (ms) and the octaves it adds to the cutoff |
| `KeyTrack` | 0.5 | 0-1, how much the cutoff follows the pitch (relative to middle C) |
| `VibratoRate`, `VibratoDepth` | 0, 0 | Hz and semitones |
| `Glide` | 0 | Milliseconds from the previous note of the voice |
| `Level` | 0.6 | Output level |
| `AmpEnv` | | ADSR of the note (ms) |

Unison keeps the level of a single oscillator. With a bright table the high frames lose the
harmonics above the Nyquist frequency on high notes, so they get a little quieter there.

**Value inputs**: 1 position, 2 cutoff, 3 resonance. The position follows changes within a few
milliseconds and the filter glides over a buffer, so modulating them (for example with an LFO)
never clicks.

**Messages**: `pitch`, `frequency`, `table` (a `*wavetable.Table`), `position`, `positionEnvAmount`,
`positionLFORate`, `positionLFODepth`, `normalize`, `unison`, `detune`, `spread`, `subLevel`,
`filter` (0-3), `cutoff`, `resonance`, `filterEnvAmount`, `keyTrack`, `vibratoRate`, `vibratoDepth`,
`glide`, `level`.

### Presets

Presets take the table, every preset works with any table. Tables differ a lot in what their frames
hold, so try moving the `Position`.

| Name | Function | Character |
|---|---|---|
| `pad` | `PadPreset` | Wide, slow pad: five detuned oscillators drifting slowly through the table |
| `lead` | `LeadPreset` | Singing lead: three oscillators, vibrato, glide, a filter that opens at every note |
| `pluck` | `PluckPreset` | Bright pluck: the position jumps up the table and falls back while the filter closes |
| `bass` | `BassPreset` | Solid bass: one oscillator, a sine sub and a snapping lowpass |
| `growl` | `GrowlPreset` | Moving bass: a fast LFO rocks the position |
| `sweep` | `SweepPreset` | Evolving texture: the position travels through the whole table over four seconds |
| `keys` | `KeysPreset` | Bell like keys: a detuned pair, struck attack, long decay |
| `anthem` | `AnthemPreset` | Huge stack of seven wide oscillators for chords |
| `strings` | `StringsPreset` | String section: five gently detuned oscillators, a slow bow, a little vibrato |
| `vox` | `VoxPreset` | Vocal like lead: a resonant bandpass picks out a formant, vibrato and glide |
| `brass` | `BrassPreset` | Brass section: the filter swells open in the attack and settles |
| `organ` | `OrganPreset` | Steady organ: one oscillator with a sub, full sustain, fast release |
| `reese` | `ReesePreset` | Reese bass: two oscillators detuned far apart, beating and phasing, with a sub |
| `acid` | `AcidPreset` | Squelchy bass line: a resonant lowpass snapping open, glide |
| `stab` | `StabPreset` | Short chord stab: four wide oscillators, a fast closing filter |
| `drone` | `DronePreset` | Breathing drone: seven close oscillators, a very slow sweep, long fades |
| `glass` | `GlassPreset` | Thin, airy bell: a highpass removes the body |
| `riser` | `RiserPreset` | Tension builder: over six seconds the position climbs and the filter opens, hold the note |

Tables that suit them in `assets/wavetables2048`: dark and harmonic `DN Harm` and `VS Morph3`
for pads and leads, `MS 20 Pulse2 PS1` for basses, `OPFoldFMFB`, `MSelfFB02` and `Add Synth3 Icarus`
from soft to harsh for sweeps and growls, `Add Bell` and `Melda CustumWave3 IcarusBell` for keys,
`KaossDrum2` (noise like, little below a few kHz) for hits.

### Oscillator B, warp and the matrix

A second oscillator, B, reads the same table at its own position, octave, semitone and detune
(`oscBPosition`, `oscBOctave`, `oscBSemi`, `oscBDetune`) and is mixed in with `oscBLevel` (0 is off).
`warpMode` and `warp` (0-1) bend the phase of oscillator A before the table is read: bend +, bend -,
sync (the frame plays 1 to 8 times per cycle), mirror, fm b (oscillator B modulates the phase) and fm
sine (`wavetable.WarpModes`). The band limited level of the table is chosen for the steepest slope of
the warp, so warps stay mostly alias-safe. A wav becomes a table with `wavetable.Load` or
`gen.LoadSound`.

`Config.Mods` is an 8 slot [modulation matrix](#modulation-matrix): sources filter env, amp env, lfo 1,
lfo 2 (plain sines, `lfo1Rate`, `lfo2Rate` in Hz, running when a slot uses them), velocity, key,
random, wheel and `Ext1`...`Ext4` (value inputs 5 to 8, `wavetable.InExt1`); destinations `position`,
`cutoff` (6 oct), `resonance`, `pitch` (24 st), `detune`, `spread`, `subLevel`, `level`, `warp`,
`oscBLevel`, `oscBPosition`, `oscBDetune` and the two LFO rates (`wavetable.Dests`).

### Oscillator B unison, additive tables

Oscillator B can be stacked like oscillator A: `oscBUnison` (1-7) oscillators spread over `oscBSpread`
cents (0 takes `detune`) and over the stereo width of `spread`, the level kept constant. In Go,
`gen.Additive(harmonics, numFrames)` builds a table from the amplitudes of its partials (index 0 is the
fundamental), and `gen.AdditiveMorph(from, to, numFrames)` crossfades between two partial lists over the
frames, so moving the position morphs one spectrum into the other. No aliasing: every harmonic is exact.

## piano

`synths/piano`

A physically modelled piano, no samples. Every key has one to three strings (one in the low bass,
two up to the tenor, three above), each a digital waveguide: a delay line closed by

- a **loss filter** that sets how long the string rings (about 20 seconds in the low bass, under 2
  at the top) and makes the highs die faster;
- a chain of **allpass filters** for the stiffness of the string: they stretch the partials sharp
  (inharmonicity), fitted per key to the stretch of a real grand (strongest at the top);
- a **tuning allpass** that puts the fundamental on pitch to a fraction of a cent.

What makes it a piano:

- **Two-stage decay.** Every string vibrates in two polarizations: one passes its energy to the
  soundboard fast (the loud prompt sound, the first second), the other keeps it (the long, quiet
  aftersound). The strings of a key are a little out of tune with each other (`Unison`) and share
  the bridge: what they do together drains into the soundboard, what they do against each other
  lasts, and the bridge passes energy between them and to the other polarization (Weinreich's
  coupled strings). So a note falls fast, then lingers and breathes as the strings trade energy.
- **A felt hammer that meets the string.** Every strike is simulated: the hammer (its mass, heavier
  in the bass) flies at the string, its felt stiffens steeply as it is compressed and pushes harder
  going in than coming out (hysteresis), the string gives way under it and the waves leaving the
  strike point come back from both ends. The force that results is the strike: a few milliseconds
  in the bass, a fraction in the treble, shorter and sharper the harder the key is played (a soft
  note is round, a loud one bright), and on some keys the hammer bounces and hits twice. The felt
  of every key is fitted once (when the piano is made) so velocity 0.8 sounds as voiced. The strike
  position leaves out the harmonics with a node where the hammer hits, and the soundboard knock is
  heard with it: a short impact, shorter in the treble (from about 80 ms in the bass to about 12 ms
  at the top), so it never outlasts the short high notes.
- **A string played hard settles.** Its own motion stretches it, so it starts a little sharp (up to
  about 5 cents in the bass at full velocity, a cent at the top) and settles within a few hundred
  milliseconds.
- **Irregular partials.** The partials of a real string don't die away evenly: the bridge takes
  more at some frequencies than at others. Every string has a slightly different pattern of
  partials that decay faster (up to 3 dB per second, with `Variation`).
- **A soundboard** every string sounds through: a board of spruce made of its modes (a few hundred:
  sparse in the bass, one every 20-odd Hz higher up, with the low Q of wood, radiating the bass
  weakly and rolling off the highs), heard from a different place on every side of the stereo image.
  The strings sound through it by convolution (`Body`), or through a measured soundboard response
  (`SoundboardIR`). Every key is voiced for the board, like a technician voicing the hammers: the
  gain that evens out where the key's partials land on the board's modes.
- **The bark of the bass.** A string played hard stretches as it moves, which rings its longitudinal
  mode and adds phantom partials at the sums of its partials: the metallic bark of loud bass notes,
  growing with the velocity (`Bark`).
- **The action** is heard: the dampers landing on the still vibrating strings, the key coming up,
  the sustain pedal lifting all dampers (a soft swish) and letting them land (a thud) (`Mechanics`).
- **Sympathetic resonance.** The strings of all 88 keys ring along with what is played when their
  dampers are up: all of them with the sustain pedal down, the keys held down (also silently), and
  the top keys, which have no dampers (`Resonance`). With the pedal down a halo blooms as the notes
  die away.
- **Dampers** stop the strings when a key is released (slower in the bass). The **sustain pedal**
  lifts them; between up and down it lifts them partly (**half pedaling**): the dampers touch the
  strings lightly, a released note rings on shorter, and the heavy bass strings push the light touch
  away more easily than the treble, so half pedaling clears the treble and lets the bass ring. The
  **sostenuto pedal** keeps the dampers of the keys held when it goes down up until it comes up:
  a bass note caught by it rings under chords played dry above it. The **soft pedal** (una corda)
  shifts the hammers so they strike one string less with a softer part of the felt: quieter and
  veiled.
- **Stretch tuning.** The partials of stiff strings are sharp, so the instrument is tuned like a
  tuner does it: octaves widened until they are clean (the 2nd partial of the lower note on the upper
  note in the treble, 4:2 and 6:3 lower down), which gives the Railsback curve: the low bass some 15
  cents flat, the top some 45 cents sharp, derived from the strings' own inharmonicity (`Stretch`).
- **No two notes alike.** Every key's hammer is a little harder or softer, hits a little elsewhere,
  every key rings a little longer or shorter and is a little louder or softer, as on any real
  instrument; every strike varies a little in force and felt, and the hammer reaches the strings of
  a key a fraction of a millisecond apart (`Variation`). The strikes draw from the random source of
  the patch, so `psalm.Seed` repeats them.
- **A listening position**: on the bench (`Player`), close microphones over the open lid
  (`Studio`), or in the audience (`Hall`): how wide the keys spread, the early reflections of the
  lid, the floor and the stage, and from the audience a little air (`Position`). A reverb adds the
  room after that.
- **Calibrated against a recorded concert grand** (single notes of the University of Iowa Steinway
  at pianissimo, mezzo forte and fortissimo): the decay of every register (a fast first second,
  roughly 6 to 18 dB per second, then a slow aftersound), the brightness of the attack at every
  dynamic (a near pure tone in the treble), the dynamic range (pianissimo to fortissimo some 25 dB
  in the middle, the level follows the speed of the hammer), the stretch tuning and the level over
  the keyboard (even up to C5, a few dB softer from C6, the top octave some 13-16 dB softer).

```go
grand := piano.NewPoly(piano.GrandPreset(), 24).AddTo(p) // a voice per sounding key

// the pedals: messages without a command reach the whole instrument
arrangement.AutomateMessage(arrange.Lane{Sections: map[string]arrange.Curve{"verse": arrange.Const(1)}}, grand, "sustain")
grand.ReceiveMessage(psalm.Message{"sustain": 0.5})   // half pedal
grand.ReceiveMessage(psalm.Message{"sostenuto": 1.0}) // catch the keys held now
grand.ReceiveMessage(psalm.Message{"soft": 1.0})
```

`piano.NewPoly(cfg, n)` creates the instrument (a `*piano.Piano`: `n` voices, the strings that ring
along and the soundboard), it takes the messages of a poly synth (triggers with a duration, or
`noteOn` and `noteOff`). `piano.New(cfg)` is a single voice, without soundboard and resonance. A
voice sounds until its strings have died away, the dampers stopped them, or as long as its key is
held; with the pedal down that can take many seconds. The voices are handed out like piano keys: a
key that still sounds is struck again on its own strings (adding to their vibration, no new voice,
nothing cut off); a new key takes a free voice or, when all sound, the quietest one (released keys
before held ones). The strings and the hammers of all keys are fitted when the piano is created
(some tens of milliseconds), so playing it does no fitting: a silent piano rests from the start, a
note costs a fraction of a millisecond. Sixteen sounding voices with the pedal down take under 10%
of a core (the strings ringing along run at half the sample rate); a piano that is silent rests. The
instrument sounds 128 samples (3 ms) late, the time the soundboard convolution takes; it reports
that latency, so a mixer lines the other strips up with it.

**Note messages**: `pitch` (or `frequency`), `velocity` (0-1, default 0.8: the tone always, the
level by `VelocitySens`), and every setting below. **Instrument messages**, the pedals: `sustain`
(0 up, 1 down, half pedaling in between: from 0.3 to 0.7 of the travel the dampers lift), `sostenuto`
and `soft` (above 0.5 is down).

| Setting | Default | Meaning |
|---|---|---|
| `Hardness` | 0.5 | 0-1, hardness of the hammer felt: soft and round to hard and bright |
| `StrikePosition` | 0.125 | Where the hammer hits, 0.05-0.3 of the string: the harmonics with a node there are weaker (1/8 is classic, nearer the end is brighter, thinner) |
| `Soundboard` | 0.6 | 0-1, the knock of the soundboard in the strike |
| `Body` | 0.6 | 0-1, how much of the sound comes through the soundboard (at 1 a little of the direct strings stays, for clarity) |
| `SoundboardIR` | nil | A measured soundboard response (one slice per side, at the sample rate) instead of the modelled board |
| `Bark` | 0.5 | 0-1, the bark of the bass strings played hard (phantom partials, longitudinal mode) |
| `Mechanics` | 0.5 | 0-1, the noise of the action: dampers, keys, the sustain pedal |
| `Stretch` | 1 | How much of a tuner's stretch tuning, 0 is equal temperament |
| `Variation` | 0.5 | 0-1, how different the keys and every strike are, and how irregularly the partials decay |
| `Treble` | 0 | dB, the top octave louder or softer than voiced |
| `Position` | `Player` | `Player`, `Studio` or `Hall`: where the piano is heard from |
| `Resonance` | 0.5 | 0-1, sympathetic resonance of the strings whose dampers are up |
| `Stiffness` | 1 | Inharmonicity, 1 is a grand, more an upright (or a toy, a struck rod), 0 perfectly harmonic |
| `Decay` | 1 | Scales the ring times |
| `Damping` | 1 | How much faster the highs die |
| `Unison` | 1 | Cents the strings of a key are apart: around 1 is in tune with a slow beat, 10+ honky tonk |
| `Strings` | 3 | Strings per key at most (1-3): fewer are thinner and cheaper |
| `DamperTime` | 0.25 | Seconds a released key rings on (longer in the bass) |
| `Width` | 0.6 | 0-1, the keys spread from left (low) to right (high), as the player hears it |
| `VelocitySens` | 0.7 | 0-1, how much the velocity changes the level (0.7 is a recorded grand: pianissimo to fortissimo some 25 dB) |
| `Level` | 1 | Output level |

**Presets** (`piano.Presets` by name):

| Preset | Character |
|---|---|
| `grand` | A concert grand as it is usually recorded, from close microphones: warm hammers, a lot of soundboard and wood, strings that beat a little |
| `concert` | The grand from the audience: the lid throwing the sound into the hall, brighter hammers to carry |
| `warm` | A grand in a jazz club: close, round hammers, a rich body, the strings ringing along freely |
| `baby` | A small grand: shorter, stiffer strings, less sustain, a boxier soundboard |
| `bright` | Hard hammers for pop and rock, clean and present |
| `upright` | Stiffer, shorter, boxier, a little out of tune |
| `vintage` | An old upright: worn hammers gone hard, strings drifting apart, a noisy action |
| `felt` | A felt piano, soft and close, for quiet music and ambient |
| `cinematic` | Soft, dark hammers from the hall, a lot of sympathetic resonance: the slow film-score piano |
| `honkyTonk` | Strings far out of tune with each other |
| `toy` | One short string per key, struck hard in the middle: a bright, hollow, bell like plink |

A struck string peaks high above its loudness, so the
presets are calibrated to a -1 dB peak at velocity 0.8 (about -14 LUFS); played harder they go
over, leave headroom or use the mixer's limiter.

**Messages**: `pitch`, `frequency`, `velocity`, `level`, the pedals `sustain`, `sostenuto`, `soft`, and the voicing: `hardness`, `strikePosition`, `velocitySens`, `variation`, `body`, `bark`, `treble`, `soundboard`, `resonance`, `stiffness`, `stretch`, `decay`, `damping`, `damperTime`, `unison`, `strings`, `mechanics`, `position`, `width`.

### The action

The piano is a physical model of a grand: every key has one to three strings (a waveguide with a loss
filter, dispersion for the inharmonicity of its register and a tuning allpass), slightly detuned and
sharing a bridge, so a note falls fast and then lingers (double decay) and beats. A felt hammer is
simulated at the velocity of the key (stiffness k * x^p), so a harder strike is shorter, peakier and
brighter and can re-contact the string in the treble. The soft pedal strikes one string less; the
dampers, the sustain pedal (half pedalling), the sostenuto pedal and the sympathetic resonance are
modelled; the bass barks when played hard (phantom partials at sums of partials); octaves are stretch
tuned (`stretch`, `stiffness`). `mechanics` is the noise of the action: the dampers landing at release
(not with the pedal down), the key coming up, the pedal, and `keyBed` (0-1), the thud of a key struck
hard landing on the keybed.

### Bridge and wound strings

The strings of all keys end on one bridge. Its admittance is a shared bank of about 60 soundboard
modes, sparse in the bass and denser above, damped like wood. Every string puts its force on the bridge
each sample, and the bridge's motion goes back into every string: a partial that meets a mode drains
fast and one between modes keeps its energy. A chord is therefore not the sum of its notes, and with
the sustain pedal down the washed tail is longer. `bridge` (0-1, default 0.6) blends from each key
draining on its own (0) to full coupling (1); the mean drain per key stays what it was. `wound` (0-1,
default 1) makes the keys below the second string break (key 44) wound strings: much less inharmonic
than plain strings, with a step at the break about a key wide, and their copper wrap damps the highs
and loads the bridge more. `precursor` (0-1, default 0.5) adds the short burst of longitudinal waves that
outruns a hard-struck bass note (velocity cubed, gone in the treble). `strikeRegister` (0-1, default 1)
moves the hammer strike point from 1/8 in the bass towards 1/12 in the treble. The mode bank is generic,
not measured soundboard data.

## drums

`synths/drums`

`drums.NewDrums(soundBank, numVoices, numChannels)` is a `poly` synth of sample `player` voices.
Each trigger plays a sound from the bank, chosen with the `"sound"` key in the note message
(`"speed"` changes its pitch). A note off fades the sound out in 8 ms (a choke): the step
sequencer (`stepseq`, see [Controls](controls.md#stepseq)) uses it for choke groups.

```go
drms := drums.NewDrums(soundBank, 16, 2).AddTo(p)
drms.ValueIn(templ.NewTrigger(0.5, template.Template{"sound": "kick", "speed": 1.0}).
	ValueIn(timer.WithTicks(float64(psalm.PPQ())).AddTo(p)))
```

## sampler

`synths/sampler`

A multi-sample instrument: recordings of an instrument spread over its keys and played at
several strengths. Each recording is a region: the keys and velocities it answers, the key it
was recorded at, how it loops and how it fades. A note plays every region that matches it, so
regions layer; alternate recordings of the same note take turns (round robin) or are picked at
random. Instruments load from SFZ, the plain text format most free sample libraries come in, or
are built in code.

```go
inst, err := sampler.LoadSFZ("Salamander/piano.sfz")       // or sampler.MustSFZ
piano := sampler.NewPoly(sampler.DefaultConfig(inst), 32).AddTo(p)

// in code: a sound from a file or from memory, the keys it plays
snd, _ := sampler.LoadFile("samples/c4.wav")
low := sampler.NewRegion(snd).Keys(0, 64, 60)               // keys 0-64, recorded at 60
inst = sampler.MustInstrument(low, high)
```

### Slices

A recording cut into pieces, each on a key of its own: a drum loop becomes a kit, a phrase a set of syllables.
`OnsetCuts(snd, rise, minGap)` finds the cuts where the level jumps (`rise` dB over the 50 ms before it, 6 to 12 is
usual; `minGap` seconds between two), `VerseCuts(snd, verses, depth)` cuts a recitation of verses each followed by a refrain in two pieces for each (the verses are found among the long pauses, about equally long); `PauseCuts(snd, depth, minPause, maxLen)` cuts at the pauses of speech (a pause is under the loud level by `depth` dB, 20 to 40, for `minPause` seconds; a piece over `maxLen` seconds is divided), `ChangeCuts(snd, sensitivity, minGap)` cuts where the spectrum changes, for what has no hits (a sung line, speech; `sensitivity` 1.1 to 2, the lower the more cuts), `EvenCuts(snd, n)` divides it in equal pieces; both return the frames of the
cuts, moved to the nearest zero crossing, the first 0 and the last the end. `Slices(snd, firstKey, cuts)` makes a
region for every piece: from `firstKey` up, each plays its piece to the end at the pitch it was recorded (`OneShot`). A piece fades in over 5 ms and runs 5 ms past its end, fading out, which is where the next piece fades in: no clicks, and
the pieces played one right after the other are the recording again, with no gaps. The regions can be changed (tune, envelope, loop) before the instrument is made.

```go
snd, _ := sampler.LoadFile("assets/sounds/various/tabla_loop_4bars_90bpm-86005.wav")
inst := sampler.MustInstrument(sampler.Slices(snd, 36, sampler.OnsetCuts(snd, 9, 0.08))...)   // 44 pieces on keys 36-79
```

`Notes(snd, firstKey, spacing, notes)` makes a region for each played note of a recording (sung notes, a phrase): `[from, to, pitch]` in seconds and a MIDI pitch
with a fraction, which tunes it. Note i is on the key `firstKey+spacing*i` and two either side, so a key a little off plays it a little higher or lower; it fades in over 40 ms, and goes on as long as the key is held (its second half goes round, with a crossfade), releasing over 0.4 s.

`go run ./cmd/examples/slicer -demo` plays the tabla loop cut at its hits, in order and then shuffled (`-file`,
`-by change` for a sung line, `-by pauses -max 4` for speech, `-by verses -verses 26` for a recitation, `-rise`, `-gap`, `-slices n`, `-first`); without `-demo` the pieces are on a MIDI keyboard.

`sampler.New(cfg)` is a single voice, `sampler.NewPoly(cfg, n)` a `poly` synth whose voices share
the round robin turns and the choke groups. An instrument doesn't change once built: one can play
in many synths. `LoadSFZWith(path, sampler.MipMaps(4))` keeps band limited copies of every sound,
for regions played far above their key.

### Regions

Start a region with `NewRegion(sound)` (all keys and velocities, root 60, in tune, full velocity
tracking, a short release):

| Field | SFZ | Meaning |
|---|---|---|
| `LoKey`, `HiKey` | `lokey`, `hikey`, `key` | The keys it plays (note names work in SFZ: `c4` is 60) |
| `LoVel`, `HiVel` | `lovel`, `hivel` | The velocities it plays, 1-127 |
| `Root` | `pitch_keycenter` | The key it was recorded at |
| `KeyTrack` | `pitch_keytrack` | Cents per key: 100 in tune, 0 one pitch on every key (drums) |
| `Transpose`, `Tune` | `transpose`, `tune` | Semitones, cents |
| `Volume`, `Amp` | `volume`, `amplitude` | dB, and a linear gain |
| `Pan` | `pan` | -1 to 1 (a stereo sound is balanced) |
| `VelTrack` | `amp_veltrack` | 0-1, how much the velocity sets the loudness (velocity squared) |
| `Offset`, `End` | `offset`, `end` | The frames it plays |
| `Loop` | `loop_mode` | `NoLoop`, `OneShot` (ignores note offs), `LoopContinuous`, `LoopSustain` (loops while held, then plays on) |
| `EndFade` | | Seconds the sound fades out in a straight line before `End`, so a piece cut from a recording does not click (`Slices` sets 5 ms) |
| `LoopStart`, `LoopEnd` | `loop_start`, `loop_end` | Frames of the loop |
| `LoopCrossfade` | `loop_crossfade` | Seconds the end of the loop fades into its start |
| `Trigger` | `trigger` | `OnAttack`; `OnRelease` plays when the key goes up; `OnFirst` only when no other key is held, `OnLegato` only when one is (slurred transitions) |
| `RtDecay` | `rt_decay` | dB per second a release region is softer the longer the key was down |
| `Group`, `OffBy`, `OffMode` | `group`, `off_by`, `off_mode` | Choke groups: a region of group `OffBy` stops this one |
| `SeqLength`, `SeqPosition` | `seq_length`, `seq_position` | Round robin |
| `LoRand`, `HiRand` | `lorand`, `hirand` | Random choice per note |
| `Env` | `ampeg_delay`, `_attack`, `_hold`, `_decay`, `_sustain`, `_release` | The amp envelope (seconds, sustain 0-1) |
| `Cutoff`, `Resonance`, `Filter`, `FilVelTrack` | `cutoff`, `resonance`, `fil_type`, `fil_veltrack` | A filter per region |
| `SwLoKey`, `SwHiKey`, `SwLast`, `SwDefault`, `SwLabel` | `sw_lokey`, `sw_hikey`, `sw_last`, `sw_default`, `sw_label` | Keyswitches: the keys `SwLoKey`-`SwHiKey` choose the articulation instead of playing; the region plays while the last one pressed is `SwLast` |
| `Fades` | `xfin_lovel` ... `xfout_hikey`, `xfin_loccN` ... `xfout_hiccN`, `xf_velcurve`, `xf_keycurve`, `xf_cccurve` | Crossfades by velocity, key or a controller (power or `Linear`); the ones by a controller follow it while the note sounds |
| `CCGain` | `gain_ccN`, `volume_onccN` | dB a controller adds at its top |
| `CCEnv` | `ampeg_attackccN` (and `delay`, `hold`, `decay`, `sustain`, `release`) | What a controller adds to the envelope when a note starts |
| `EnvDynamic` | `ampeg_dynamic` | The envelope follows the controllers while the note sounds: a stage under way goes on at the new pace |
| `AmpRandom`, `PitchRandom`, `DelayRandom` | `amp_random`, `pitch_random`, `delay_random` | Every note a little different: dB and cents spread around the level and the pitch, seconds later |
| `Delay` | `delay` | Seconds the region starts after the note |
| `VelEnv` | `ampeg_vel2delay` ... `ampeg_vel2release` | Added to the envelope at full velocity |
| `LFOs` | `pitchlfo_*`, `amplfo_*`, `lfoN_freq`, `_wave`, `_delay`, `_fade`, `_pitch`, `_volume` | Vibrato (cents) and tremolo (dB), with a delay and a fade in |
| `EQ` | `eq1_*` ... `eq3_*` (`freq`, `bw`, `gain`) | Peaking filters, bandwidth in octaves |
| `Width` | `width` | A stereo sound's width in percent: 100 as it is, 0 mono |
| `Polyphony`, `NotePolyphony` | `polyphony`, `note_polyphony` | How many notes of its group (or of one key) sound at once: past it the oldest let go, quickly |
| `BendUp`, `BendDown` | `bend_up`, `bend_down` | Cents a full pitch bend moves it (200, -200) |

Controllers reach the sampler as messages, `"cc1"` (the mod wheel) to `"cc127"`, 0-1: give a
keyboard `Controls: map[int]string{1: "cc1"}`. `Instrument.Switches()` lists the keyswitches.

The instrument has a sustain pedal of its own (`sustain_cc`, `sustain_lo`, `sustain_sw`;
`Instrument.SustainCC`, `SustainLo`, `NoSustain`): notes let go while it is down ring on, their
release regions play when it comes up. It takes `"cc64"` or `"sustain"` (0-1); only 0 is up
unless `sustain_lo` says otherwise.

The SFZ loader knows `<control>` (`default_path`, `note_offset`, `octave_offset`, `set_ccN`), `<global>`,
`<master>`, `<group>`, `<region>`, `#define` and `#include`. Opcodes it doesn't know are counted
in `Instrument.Unsupported`. Sounds load with libsndfile: wav, aiff, flac, ogg. A WAV with a loop of its own (its smpl chunk) loops there, unless the region sets `loop_mode`, `loop_start` or `loop_end` itself.

Notes take `pitch` (or `frequency`) and `velocity` (0-1). The levels are those of the recordings.

The sounds an SFZ file names are found from its folder (and its `default_path`). When they live
elsewhere, give a loader that finds them: `LoadSFZWith` hands it every path as the file has it.

```go
inst, err := sampler.LoadSFZWith("scripts/piano.sfz", func(path string) (sampler.Sound, error) {
	rel, _ := filepath.Rel("scripts", path)                // samples/C4.wav
	return sampler.LoadFile(filepath.Join("/data/piano", rel)) // /data/piano/samples/C4.wav
})
```

`cmd/examples/sfz` plays an SFZ instrument from a MIDI keyboard, with `-sounds` and `-rename` for
sounds kept elsewhere.

**Messages**: `pitch`, `frequency`, `velocity`, `level`, `sustain` (the pedal), `cc1` to `cc127` (0-1), `modWheel` (controller 1), and the [pitch bend, mod wheel and aftertouch](#pitch-bend-mod-wheel-and-aftertouch).

## streamer

`synths/streamer`

The sampler for sounds too long to keep in memory: a recording of an hour of rain, a piece of
music, a field recording. A note plays the file from the disk while it sounds. The first half
second of every file is kept in memory, so the note starts at once, and a goroutine of the
note reads the rest into a ring that the audio thread plays from. The audio thread never waits
for the disk: when the disk is too slow the sound has a gap and `Voice.Underruns` counts it.
Wav, aiff, flac and ogg play, whatever libsndfile reads, in any sample rate (it is played at
the rate of the file).

```go
inst, report, err := streamer.ParseInstrument("36 assets/sounds/various/rain-sound-188158.wav loop vol=-6")
rain := streamer.NewPoly(streamer.DefaultConfig(inst), 8).AddTo(p)
rain.NoteOn(0.8, psalm.Message{"pitch": 36.0}, psalm.CtrlTick{}, p.Config()) // or notes from a script or a keyboard
```

The sounds are a text, a line each: a key, a file, and anything of `loop` (round while the
key is down, between `loopstart=` and `loopend=` in seconds), `hold` (a note off fades the
sound out; a sound that is not held plays to its end, like a drum), `speed=2` (an octave up),
`start=` and `end=` (the part of the file to play, seconds), `release=` (seconds), `vol=` (dB),
`pan=` and `name=`. A single line plays on every key. `streamer.Open(path, key)` makes an
instrument of one file, `streamer.NewInstrument(&streamer.Entry{...})` one of entries.

In the patcher the kind `streamer` has the text in its window (`sounds`), with the report
of what every line made, and takes notes like the sampler. A loop is not crossfaded at its seam:
cut it where it goes round, or use a file that does. It plays forwards at a fixed speed
(interpolated between frames); for sounds you cut into pieces on keys, or play backwards or by
grains, use the sampler or the grains synth.

## organ

`synths/organ`

A tonewheel organ: 91 wheels spinning all the time, which the keys tap at the pitches of the nine
drawbars (16' 5 1/3' 8' 4' 2 2/3' 2' 1 3/5' 1 1/3' 1', 3 dB a step). Like the real one it is a
whole instrument, not a set of voices: keys sharing a wheel share its phase, the wheels fold back
at the ends, the percussion sounds on the first key of a phrase only (single trigger) and the
vibrato scanner moves the whole sound. Put the rotary speaker after it.

```go
org := organ.New(organ.Jazz()).AddTo(p)
leslie := rotary.New(rotary.DefaultConfig()).AddTo(p).In(org)
org.Receive(psalm.Message{"drawbars": "888000000"}, 0)
```

It takes the messages of a poly synth (triggers with a duration, or noteOn and noteOff), and the
sustain pedal.

| Setting | Meaning |
|---|---|
| `Drawbars` | `[9]float64`, 0-8; `organ.Bars("888000000")` reads the way organists write it |
| `Percussion` | `PercOff`, `PercSecond` (4'), `PercThird` (2 2/3'); with it on the 1' is silent |
| `PercFast`, `PercSoft` | A short decay, 6 dB softer |
| `Vibrato` | `VibratoOff`, `V1`-`V3`, `C1`-`C3` (chorus: mixed with the dry sound) |
| `VibratoHz` | The scanner's speed (0 is 6.9 Hz) |
| `Click` | 0-1, the key contacts |
| `Drive` | 0-1, the preamp saturating |

**Value inputs**: 1-9 the drawbars. **Messages**: `drawbars`, `drawbar1`-`drawbar9`,
`percussion`, `percFast`, `percSoft`, `vibrato`, `vibratoHz`, `click`, `drive`, `level`,
`sustain`.

| Preset | Character |
|---|---|
| `jazz` | 888000000, third percussion fast |
| `gospel` | 888800000, deep chorus |
| `full` | Every drawbar out |
| `rock` | 888000000 driven, chorus, more click |
| `ballad` | 806004000, soft, chorus |
| `flute` | 008400000, vibrato |
| `reed` | 800000888, nasal |
| `whistle` | 800000008 |
| `screamer` | Every drawbar into a preamp driven to its limit, all the click, deep chorus |
| `warble` | The scanner at 15 Hz: a fluttering, alien warble |
| `chime` | Only the high drawbars with the fast third percussion: a glassy stab |
| `growlBass` | 16' and 8' driven, all click: a growling bass organ |
| `seasick` | The scanner at 1.5 Hz, deep: everything sways out of tune and back |

### Filter, leakage, matrix

The organ has no voices, so the new parts act on the summed sound. The [filter](#the-voice-filter)
(`filterType`, `cutoff`, `resonance`, `filterEnvAmount`, `keyTrack`, `filterDrive` and the `filter`
envelope) is a single filter after the drawbars; its envelope starts with the first key of a phrase and
releases when the last key lets go, like the percussion's single trigger. `leakage` (0-1) adds the
faint crosstalk of three other tonewheels to every key, different on each key press. `clickRand` (0-1)
makes the key click vary in loudness and colour from key to key.

The 8 slot [modulation matrix](#modulation-matrix) takes the sources filter env, amp (the loudest key),
LFO 1 (the scanner's), velocity and key of the last note, a random value per phrase, the wheel and Ext
1-4 (value inputs 10-13, `organ.InExt1`) to `drive`, `click`, `vibratoDepth`, `level`, `cutoff`,
`resonance`, `percLevel`, `leakage` and `pitch`. There is no LFO 2. The rotary of a Leslie is the
`rotary` effect after the organ.

The matrix has a second LFO: `lfo2Rate` (Hz, default 1) is a plain sine, available as the source LFO 2
next to the scanner's, running only when a slot uses it.

### The tonewheel generator, scanner and Leslie

The organ models the Hammond tonewheel generator. The 91 wheels turn at the frequencies of the gear
ratios (a 20 Hz shaft, twelve gear pairs, exact octaves), so the semitones are slightly off equal
temperament, and each wheel is a sine with a little second and third harmonic (`tonewheel` 0-1; the gear
table is from memory: check it against Hammond documentation if exact tuning matters). Each key has nine
contacts, one per drawbar bus, that close a few milliseconds apart (`contactSpread`), so the attack is
percussive; `click` sets how sharply they close. `leakage` bleeds the neighbouring wheels and the
wheels an octave and a twelfth up into the bus, at -40 dB or less.

The scanner is a nine-section line swept at 6.87 Hz: V1/V2/V3 use 3/6/9 sections (about +-4, +-8 and
+-12 cents), the C modes mix the dry sound with it, and `vibratoScale` makes it deeper. A tube preamp has
two asymmetric stages (`drive`) and a tone tilt (`tone`). `leslie` (0-1) puts the sound through a
built-in rotary speaker: an 800 Hz Linkwitz-Riley split, a horn and a drum that each have Doppler and
amplitude modulation, and two microphones (`lesliePosition`). `leslieSpeed` runs from 0 (stopped)
through 0.5 (slow, 0.8 Hz horn) to 1 (fast, 6.7 Hz), and the rotors speed up and slow down at their own
pace (the horn in about 1.2 s, the drum in about 3 s). `leslie`, `leslieSpeed` and `tone` are matrix
destinations.

## waveguide

`synths/waveguide`

Strings that are plucked or bowed: a wave travelling up and down a string, fed by a pluck or kept
going by a bow, heard through the body of the instrument. The pluck is an extended Karplus-Strong
string (in tune within a few cents over the whole range); the bow is the McIntyre, Schumacher and
Woodhouse model after the STK: the bow sticks and slips, which keeps the string going as long as
it moves.

```go
guitar := waveguide.NewPoly(waveguide.Guitar(), 8).AddTo(p)
cello := waveguide.NewPoly(waveguide.Cello(), 4).AddTo(p)
```

| Setting | Model | Meaning |
|---|---|---|
| `Model` | | `Pluck` or `Bow` |
| `Decay` | pluck | Seconds the string rings at 220 Hz; lower strings ring longer |
| `Damping` | pluck | 0-1, how much faster the high partials die |
| `Brightness` | pluck | 0-1, a soft finger to a hard pick; the velocity adds |
| `PickPosition` | pluck | 0-0.5 from the bridge: thin near it, round in the middle (the partials with a node there are missing) |
| `Release` | pluck | Seconds the string dies in once let go, 0 lets it ring (a harp) |
| `Pressure` | bow | 0-1, airy to gritty |
| `BowPosition` | bow | 0.02-0.5 from the bridge |
| `BowAttack`, `BowRelease` | bow | Seconds |
| `VibratoRate`, `VibratoDepth`, `VibratoDelay` | bow | Hz, semitones, seconds |
| `Body`, `BodyMix` | | `GuitarBody`, `ViolinBody`, `CelloBody`, `HarpBody`, or `NoBody`; 0-1 |

The bow's speed is its loudness: the note amplitude sets it (faster than the model allows, a
string jumps to its octave, as a real one does). Bowed notes are in tune within a few cents from C2 to C7 (the
friction's pull is measured and corrected). **Value inputs**: 1 pressure, 2 vibrato depth.

| Preset | Character |
|---|---|
| `guitar`, `steel`, `muted` | Nylon, steel with a pick, palm muted |
| `harp` | Plucked in the middle, rings on |
| `bass` | A fingered electric bass |
| `pizzicato` | A violin plucked |
| `violin`, `cello` | Bowed, vibrato coming in |
| `slowStrings` | Swelling in and out |
| `ponticello` | Bowed near the bridge, glassy |
| `flautando` | Bowed lightly over the fingerboard |
| `glassHarp` | A string without losses: shimmers for many seconds |
| `wire` | A steel wire picked at its very end: thin, jangling, metallic |
| `eBow` | A string kept singing by a steady bow: endless sustain |
| `ghostBow` | Bowed so lightly the note floats, a slow wide vibrato |
| `scrape` | Pressed hard near the bridge: gritty, metallic, on the edge |

**Messages**: `pitch`, `frequency`, `velocity`, `level`, `decay`, `damping`, `brightness`, `pickPosition`, `release`, `bowPressure`, `bowPosition`, `vibratoRate`, `vibratoDepth`, `bodyMix`, and the [pitch bend, mod wheel and aftertouch](#pitch-bend-mod-wheel-and-aftertouch).

### Filter, matrix and sympathetic strings

`Config.Filter` is the [voice filter](#the-voice-filter) after the string and body; `Config.Mods` an 8
slot [modulation matrix](#modulation-matrix). Sources: amp env (the bow speed, or the level of a
plucked string), filter env, lfo 1 (the vibrato), lfo 2 (a plain sine, `lfo2Rate`), velocity, key,
random, wheel and ext 1-4 (value inputs 3 to 6, `waveguide.InExt1`). Destinations: `decay`, `damping`,
`brightness`, `pickPosition`, `bowPressure`, `bowPosition`, `vibratoDepth`, `bodyMix`, `pitch`,
`level`, `cutoff`. Brightness and pick position are the pluck itself: they are read at the start of
the note, so velocity, key, random, wheel and ext are the sources for them.

`sympathetic` (0-1) feeds the sound into four strings tuned an octave below, a fifth, an octave and
an octave and a fifth above the note, and adds them back: a shimmering, resonant string. It costs
nothing at 0.

### Membrane and pipe

Two more models join the pluck and the bow. `Membrane` is a struck drum skin: 14 modes of an ideal
circular membrane (1, 1.59, 2.14, 2.30, 2.65, 2.92 ... times the pitch), each a damped oscillator,
started with the loudness the place of the strike gives it (`strike`, 0 the middle to 1 the rim; the
middle leaves only the round modes). `hardness` (plus velocity) brightens the stick, `decay` is the
seconds the lowest mode rings and `damping` how much faster the higher ones die. `tension` raises the
pitch at the strike and lets it fall back (a tom's thump, a tabla's bend), `snare` adds wires buzzing
along, and `release` is a hand laid on the skin (conga). Presets: `tom`, `floorTom`, `conga`, `tabla`,
`frameDrum`, `snareDrum`.

The stick has a contact time: it presses on the skin for a moment (a fraction of a millisecond for a
hard, fast stick, several for a soft one, set by `hardness` and the velocity) instead of hitting in an
instant, so the sound begins at nothing and rises, not with a step (which was a click, and most of what
a drum sounded like). A mode that turns over in less than the contact hardly feels it, so a soft stick
leaves the highs out.

`Pipe` is a blown tube: by default a flute (an air jet driving a delay-line tube through a saturating
curve), with `reed` on a clarinet (a single reed on a closed tube, odd partials, an octave lower).
`breath` is the pressure (too little does not sound, too much overblows or chokes the reed), `noise`
the air, `embouchure` the jet's offset (even partials) or the reed's stiffness, `damping` darkens the
tube; the attack and release (`ampAttack`, `ampRelease`) shape the breath and the vibrato is the bow's.
A pitch keeper trims the tube so it plays in tune whatever the blowing: it moves the length a little
at a time (a delay line whose length jumps clicks) and waits until the tone has stopped growing. A note
speaks within a fraction of a second at any pitch (a reed's tube starts with a small puff in it, a flute's
jet has more gain while the tone is weak), and a flute that starts to sound an octave too high is eased
off the jet, as a player would (not for a breath above 1, which asks for the overblown note). Presets: `flute`, `clarinet`,
`panpipe`. Both take the voice filter, the body, and the matrix (new destinations `breath` and
`embouchure`; the membrane's `decay`, `damping` and `pitch`).

### Real strings

The waveguide string can be made more real, each part off at 0. `inharmonicity` is the stiffness B at
220 Hz: allpass sections stretch the partials to f0 * n * sqrt(1 + B * n^2) while the fundamental stays
in tune, and low strings are stiffer (nylon about 3e-5, steel 1e-4, wound bass 3e-4; the fit is
accurate to a few cents up to B = 5e-4 at 220 Hz). `polarization` adds a sideways string slightly lower
in pitch and weaker in damping, giving a prompt sound, an aftersound and beats. `bodyCoupling` lets the
bridge give way to the body's modes: notes on a body resonance die faster and the body hands energy
back. `tensionMod` makes a hard pluck start sharp and glide down (the twang), scaled by velocity.

`pluckWidth` pulls the string aside like a finger or pick, `plectrum` goes from soft finger to hard
pick, and `fretNoise` and `fretBuzz` add the finger rustle and the string hitting the fret. `harmonic`
puts a finger on a node (0.5 sounds the octave, 1/3 the twelfth), and `slide` glides a still-ringing
string to the next pitch for legato (only when the same voice gets the next note).

For the bow, `bowForce` switches to a friction curve with stick-slip and a memory of the rosin's heat,
with `bowSpeed` as its partner; the force has to fit the speed: too light is airy, too heavy scratches.
`bowNoise` adds the rosin's grain and `bowReverse` sets the seconds per stroke before the bow turns
round. `bowForce` and `bowSpeed` are matrix destinations. New presets: `uprightBass`, `harmonics`,
`rosin` (the friction-curve violin); the plucked presets are re-voiced with the new parts.

### Stiff strings, the friction bow, harmonics, fret contact

Stiff strings (`inharmonicity` up to about 2e-3 at 220 Hz: a wound bass or a piano string) use a chain of
up to 40 first-order allpass sections whose coefficient is searched against the loop's exact phase, so
the first twelve partials sit within a few cents of f0 * n * sqrt(1 + B * n^2) with the fundamental in
tune; gentle strings keep the cheap fit.

The bow has a switch, `bowModel`: 0 is the table (`pressure`), 1 the friction curve with stick-slip and a
rosin memory (`bowForce`, `bowSpeed`, `bowNoise`, `bowReverse`). The violin, cello, slowStrings,
ponticello, flautando, ghostBow and rosin presets use friction, with a weight that follows the speed,
which follows the velocity (rosin has a fixed speed and force, in the one regime of the friction curve that
stays steady over most of the range, and its bow does not turn round: `bowReverse` 0). Its pitch is held to within about 6 cents from pitch 36 to 96: when a note
starts the bow looks up a tuning for its pitch, force, position and speed, found by running the bow loop
on a background worker (the cache warms about 1-2 s after the synth is made; `waveguide.WarmBow(cfg)`
and `WaitBow()` let a test or patch wait for it). A first note at a pitch not yet in the cache plays on
an interpolated estimate (within about 20 cents) and is exact from the next. If no weight holds the
string's motion (the highest notes), the note falls back to the table bow. The cello and the others
swing in loudness between pitches (about -12 LUFS at pitch 57, louder at 69).

Slide also works in mono legato: a pitch message to a sounding string glides over `slide` seconds
without a new attack, for plucked and bowed strings. `harmonic` puts a finger on the node, a comb in the
loop that damps the other partials, also on a string that is already ringing. `fretBuzz` is a stiff
spring and damper at `fretPosition` (0.05 to 0.98 of the string, default the fret beside the finger): it
buzzes only when the action is low (`fretBuzz` high) and the pluck is hard.

## modal

`synths/modal`

Struck things: bars, tubes, bells, tines, each ringing in its modes (partials with their own
pitch ratios and decay times), struck by a mallet. The mallet is a force as long as it touches:
a soft one touches long and wakes few high modes, a hard one is bright. Where it strikes mutes the
modes with a node there.

```go
mallets := modal.NewPoly(modal.Marimba(), 8).AddTo(p)
bell := modal.New(modal.Config{Modes: []modal.Mode{{1, 1, 1}, {2.76, 0.5, 0.6}}, Decay: 3, Level: 1})
```

| Setting | Meaning |
|---|---|
| `Modes` | `[]Mode{Ratio, Amp, Decay}`: pitch over the note, loudness, decay relative to `Decay` |
| `Decay` | Seconds the lowest mode rings |
| `HighDamping` | The higher modes die sooner: decay times (f0/f)^HighDamping |
| `Hardness` | 0-1, the mallet; the velocity adds |
| `Position` | 0-1, where it is struck |
| `Release` | Seconds it dies in once let go (a damper), 0 rings on |
| `Tremolo`, `TremoloDepth` | A vibraphone's motor |
| `Beat` | Each mode splits in two this fraction apart: bells beat |
| `Noise` | 0-1, the knock of the mallet |

**Value inputs**: 1 hardness, 2 tremolo depth.

| Preset | Character |
|---|---|
| `marimba`, `xylophone`, `vibraphone`, `glockenspiel` | The bars of the orchestra |
| `tubularBells` | Chimes |
| `churchBell` | Hum, prime, tierce, quint and nominal, beating |
| `kalimba` | A thumb piano |
| `singingBowl` | Long and beating |
| `woodBlock` | A pitched knock |
| `alienBell` | Ratios no bell has, each mode beating |
| `gamelan` | A bronze metallophone shimmering in tuned pairs (ombak) |
| `crystal` | A glassy cluster of long high modes with a gentle tremolo |
| `steelPan` | Tuned harmonics, a little out of tune with themselves |
| `metalPlate` | A dozen dense modes: a long, industrial ring |
| `spaceTine` | A tine ringing for ages in a fast tremolo |

**Messages**: `pitch`, `frequency`, `velocity`, `level`, `decay`, `highDamping`, `hardness`, `position`, `release`, `tremolo`, `tremoloDepth`, `noise`, and the [pitch bend, mod wheel and aftertouch](#pitch-bend-mod-wheel-and-aftertouch).

### Excitation, inharmonicity, filter, matrix

`excitation` chooses what rings the modes: `strike` (the default mallet), `bow` (a noise-driven bow
that fades in and keeps the modes singing while the note is held, and lifts on release) or `noise
burst` (a short windowed noise burst, shorter the harder the mallet, for brushes and scrapes).
`inharmonicity` (0-1) stretches the mode ratios relative to the first mode, the higher ones more, for
stiff metal and glass; it applies at the next strike. The material presets are `wood` (dull and
short), `metal` (long, stretched) and `glass` (pure and high).

After the modes there is a [voice filter](#the-voice-filter) whose envelope starts at the strike. The
8 slot [modulation matrix](#modulation-matrix) takes filter env, amp (the decay), LFO 1 (a plain sine at
`lfoRate`, running when used), velocity, key, a random value per note, the wheel and Ext 1-4 (value
inputs 3-6, `modal.InExt1`) to `pitch`, `decay`, `hardness`, `position`, `tremoloDepth`, `beat`,
`noise`, `level`, `cutoff`, `resonance`, `inharmonicity` and `highDamping`. The destinations that shape
the strike are read when the note is struck, from the sources known then.

The matrix has LFO 2, a plain sine from the strike at `lfo2Rate` Hz (default 1), running when a slot uses
it. The pitch from the matrix is followed every 8 samples (re-tuning the modes only when pitch is a
destination), so a fast LFO on pitch is a smooth vibrato.

### Instruments: bars, bells and plates

The modal synth can ring in a physical model of its body instead of a hand-written mode list: the
`instrument` choice picks a uniform bar (1 : 2.756 : 5.404 : 8.933 ...), an undercut marimba or
vibraphone bar tuned 1 : 4 : 10, a xylophone bar (1 : 3 : 6), a glockenspiel bar, a clamped tine
(kalimba), a tubular bell (modes 2 : 3 : 4 : 5 : 6 over a strike tone that is not itself sounding), a
church bell, a handbell, a singing bowl, a free circular plate, a gong, a cymbal or a steel pan. Each
mode dies at a rate that grows with the square of its frequency, and the mallet wakes each mode by the
mode shape at the strike `position` (along the bar from its end, from the centre of a plate to its
rim, from the crown to the lip of a bell), so hitting the node of a mode silences it.

The mallet is a Hertz contact: it touches for a time that grows with its `mass` and shrinks with the
velocity, `contact` sets how strongly a harder blow brightens the sound, and `hardness` still picks the
mallet. `resonator` adds the tube under a marimba or vibraphone bar, which strengthens and lengthens
the fundamental, and the vibraphone tremolo swings the tube; the mod wheel deepens the tremolo. Bells,
bowls and plates have pairs of near-equal modes: `spread` splits them so they beat, and `drift` makes
each note a little different (the pairs turn, the split and the pitch vary); it is off by default and
draws random numbers only when on. Presets reworked onto the models: marimba, vibraphone, xylophone,
glockenspiel, tubularBells (now pitched at the strike tone), churchBell, singingBowl, kalimba,
spaceTine, metalPlate, steelPan, wood, metal; new: handbell, gong, cymbal, celesta. The handbell, gong
and cymbal ratios are approximations; the nonlinear pitch glide of real gongs is not modelled.

### Undercut bars, nonlinear gongs

Undercut bars are no longer uniform beams. At first use the marimba, vibraphone and xylophone solve
an Euler-Bernoulli beam with an arched thickness profile (40 finite elements, subspace iteration) and
tune the arch so the modes sit at 1:4:10 (marimba, vibraphone) or 1:3:6 (xylophone), within 0.2 cents.
The strike position reads the solved mode shapes. `undercut` (0.01-1, default 1) blends from a uniform
bar to the tuned arch.

Gongs and cymbals have `nonlinearity` (0-1; 0.6 and 0.8 in their presets, off elsewhere), all of it
scaling with the blow's velocity squared. For a **cymbal** (and the hi-hats and the thunder sheet) a hard
blow starts up to 5% sharp and glides down as the modes decay, each mode passes energy up to the modes
above it with a random phase (a crash blooms into a hiss), and the modes shimmer with a random detuning.
A **gong** (`gong`, `gongAgeng`, `gongSoft`, `tamTam`, `bowedGong`) is gentler, since the same would make
it noisy and steer its pitch: the push is at most 2% (about 15 to 25 cents on a hard blow), the energy
moves up at half the rate and joins what already rings, in its phase, so the high modes swell instead
of being kicked about, and the shimmer is smaller (about a cent) and drifts over a second. The church
bell uses ratios after Rossing and Perrin (hum 0.5, prime 1, tierce 1.2, quint 1.5, nominal 2, ...) with
beating pairs; the handbell, gong and cymbal ratios are approximations. New instruments and presets:
`woodBlock` (re-voiced), `templeBlock` and `bottle`, a Helmholtz cavity at the pitch plus the wall's modes.

## drumsynth

`synths/drumsynth`

Drums from oscillators and noise, the way the classic drum machines make them: a kick is a sine
falling in pitch, a snare two tuned heads and a burst of noise, a hihat six square waves at
clashing pitches through a high filter. A kit is a set of sounds by name, played like the sample
kit of `drums`: a step sequencer row plays a sound by name, `"speed"` changes its pitch, a note
off chokes it; a keyboard plays them on the General MIDI keys.

```go
kit := drumsynth.NewPoly(drumsynth.Config{Kit: drumsynth.Kit909()}, 16).AddTo(p)
seq.Row("kick", stepseq.Sound(kit, "kick"))
```

The kits `Kit808()`, `Kit909()`, `KitElectro()` (zapping kicks and toms, tight hats), `KitTrap()` (a long driven 808 boom, a big clap) and `KitIndustrial()` (everything distorted, metal for hats) have `kick`, `snare`, `hat`, `open`, `tomLow`, `tomMid`,
`tomHigh`, `clap`, `rim`, `cowbell` and `cymbal`; the hats choke each other. Every kit also has the rest of
the General MIDI percussion in its own character (driven in the industrial kit, tuned higher in
the electro kit): `crash`, `ride`, `rideBell`, `tambourine`, `shaker`, `bongoHigh`, `bongoLow`,
`congaMute`, `congaHigh`, `congaLow`, `timbaleHigh`, `timbaleLow`, `agogoHigh`, `agogoLow`,
`clave`, `woodblockHigh`, `woodblockLow`, `triangle` and `triangleMute`. `GeneralMIDI` maps the
General MIDI drum keys to them, so a drum track from a MIDI file plays every hit. A `Sound` has a
`Kind` (`Kick`, `Snare`, `Hat`, `Tom`, `Clap`, `Rim`, `Cowbell`, `Cymbal`, `Shaker`), `Pitch`, `Decay`,
`Sweep` and `SweepTime` (the pitch falling), `Tone` (the filter of its noise or metal), `Snap`
(the attack: a kick's click, a snare's wires), `Drive`, `Level`, `Pan` and `Choke` (a group).
Change one or add your own:

```go
kit := drumsynth.Kit808()
kit.Sounds["kick"] = drumsynth.Sound{Kind: drumsynth.Kick, Pitch: 45, Decay: 2, Sweep: 14, SweepTime: 0.03, Level: 0.7}
```

The sounds of the kits peak around -3 dB (kick), -5 (snare), -6 (toms, clap) and -10 (hats, rim).

**Messages**: `sound` (a name of the kit) or `pitch` (a key), `speed`, `velocity`, `level`, and the [pitch bend, mod wheel and aftertouch](#pitch-bend-mod-wheel-and-aftertouch).
Every sound of the kit by its name: `<sound><Setting>`, the settings `Pitch`, `Decay`, `Sweep`,
`SweepTime`, `Tone`, `Snap`, `Drive`, `Level` and `Pan` (`"kickDecay": 0.8`, `"hatTone": 0.9`),
taken from its next hit; the voices of a poly share the kit. `Describe` gives the kit's every sound
a group of its own: the editor of the panel is a drum kit editor.

## vocal

`synths/vocal`

Sung vowels: a band limited buzz like the vocal folds, shaped by the five formants of a bass,
tenor, alto or soprano (after the formant table of Csound), with breath, vibrato and the small
drift of a real voice. A choir is a voice sung by several singers at once, each a little out of
tune and time, spread over the stereo field.

```go
choir := vocal.NewPoly(vocal.Choir(), 8).AddTo(p)
choir.Receive(psalm.Message{"vowel": 4.0}, 0) // all sing "u", gliding there
```

| Setting | Meaning |
|---|---|
| `Range` | `Auto` (by the pitch of the note), `Bass`, `Tenor`, `Alto`, `Soprano` |
| `Vowel` | 0-4: a e i o u, fractions morph |
| `VowelRate`, `VowelSync`, `VowelDepth`, `VowelSteps` | The vowel moving by itself: Hz or beats a cycle, steps up and down; `VowelSteps` jumps to a new vowel every cycle (talking) |
| `VowelGlide` | Seconds a change of vowel takes |
| `Brightness` | 0-1, breathy to pressed; the velocity adds |
| `Breath` | 0-1, the noise of the air |
| `Attack`, `Release` | Seconds |
| `VibratoRate`, `VibratoDepth`, `VibratoDelay` | Hz, semitones, seconds |
| `Singers`, `Detune`, `Spread` | Voices per note, cents they spread over, 0-1 how wide |
| `Steady` | The pitch held perfectly still |

The first formant follows the pitch up when it is higher, as singers do. **Value inputs**: 1 vowel,
2 vibrato depth.

| Preset | Character |
|---|---|
| `solo` | One voice, natural vibrato |
| `choir`, `aah`, `ooh` | Sections on "a" and "u" |
| `breathy` | Close and airy |
| `monks` | Low men's voices on "o", no vibrato |
| `angels` | High voices, wide and airy |
| `robot` | Steady and bright |
| `talkbox` | A new vowel every sixteenth note, with the tempo: talking in rhythm |
| `wahChoir` | A choir sweeping from "a" to "u" and back |
| `throatDrone` | Sixteen low voices far apart in pitch: a huge drone |
| `whisper` | All air, no tone |
| `alien` | A soprano's throat on every note, a fast wide vibrato |
| `babble` | A crowd of twelve, each jumping between vowels |

**Messages**: `pitch`, `frequency`, `velocity`, `level`, `vowel`, `vowelGlide`, `brightness`, `breath`, `range`, `vibratoRate`, `vibratoDepth`, and the [pitch bend, mod wheel and aftertouch](#pitch-bend-mod-wheel-and-aftertouch).

### Filter, matrix and consonants

`Config.Filter` is the [voice filter](#the-voice-filter) after the formants; `Config.Mods` an 8 slot
[modulation matrix](#modulation-matrix). Destinations: `vowel`, `brightness`, `breath`,
`vibratoDepth`, `detune`, `spread`, `pitch`, `level`, `cutoff`. Sources: the envelopes, lfo 1 (the
vibrato), lfo 2 (a plain sine at 2 Hz), velocity, key, random, wheel and ext 1-4 (value inputs 3 to
6, `vocal.InExt1`).

`consonant` (none, t, k, s, sh, f) with `consonantLevel` adds a short burst of filtered noise as the
note starts: a clipped click for t and k, a swell of hiss for s, sh and f. It sits in the middle and
is not shaped by the vowel or the filter, which makes syllables of the notes. The default is none.

### Words and formant shift

The vocal synth can sing a word. Set `Word` (or send `word: "ta ke so"`) and each new note takes the
next syllable and loops at the end. A syllable is a consonant (`t k s sh f`, or none) plus a vowel
(`a e i o u`); `vocal.ParseWord` turns text into syllables. With `WordSync` (`wordSync`) beats per
syllable, a held note goes on to the next syllables on the tempo. In a Poly the voices share one place
in the word, so a note sings the next syllable whichever voice it lands on. `syllable` (alias
`wordStep`) sets the next syllable to sing. `formantShift` scales every formant: below 1 is a bigger
throat (a giant), above 1 a smaller one (a child). Presets: `chant`, `giant` and `child`.

### A glottal source, a vocal tract, twelve vowels

The voice sings with a glottal source: a Rosenberg-Klatt flow pulse whose open quotient, speed
quotient and return phase come from `brightness` (breathy, modal, pressed) or are set directly
(`openQuotient`, `speedQuotient`, `returnPhase`). `jitter` (cycle length) and `shimmer` (cycle
loudness) are off until set, and the breath noise swells with the open folds. The pulse runs through a
cascade of five formants from Peterson-Barney data for `bass`, `tenor`, `alto`, `soprano` and `child`
voices (`Range`). Sing the old scale with `vowel` (0-4: a e i o u) or pick one of twelve vowels
(i ɪ e ɛ æ a ɔ o ʊ u ʌ ɜ) with `phoneme`. `phonemeTo`, `diphthongTime` and `diphthongHold` glide to a
second vowel inside a note (`diphthong` sets a named pair). `formantTuning` lets F1 follow a high
pitch up, as a soprano's does; `training` gathers F3-F5 near 3 kHz (the singer's formant); `nasal`
opens the nose.

Words take `m n ng l r w y` as well as `t k s sh f`, and the diphthongs `ai au ei ou oi`, so
"mai la ro ne yu" speaks (the `speak` preset). Each consonant starts the formants at its locus, a nasal
also with a pole and zero pair, and they glide into the vowel over `coarticulation` seconds. The
glottal model is the Rosenberg-Klatt approximation, not a true LF fit, and bass, alto and soprano are
scaled from the male and female data. All presets are re-voiced with a little jitter and shimmer.

### The LF source, whisper and more consonants

`source` chooses the glottal model: 0 is the Rosenberg-Klatt pulse, 1 the Liljencrants-Fant model, with
one voice-quality parameter `rd` from 0.3 (pressed, bright) to 2.7 (breathy, soft); brightness sets it
unless `rd` is set. Rd gives tp, te and ta, and the growth rate is solved so the flow derivative
integrates to zero. The presets Breathy, Ooh, Monks and Angels use LF; the others keep Rosenberg-Klatt.
`whisper` (0-1) cross-fades the voice into turbulence at the glottis, shaped by the same formants, with
independent noise per ear; at 1 it has no pitch (the `Whisper` preset).

p, t and k are aspirated and keep the voice out for 50 to 65 ms; b, d and g start it within 15 ms; s, sh
and f hold it back through the noise; z, zh and v hum through it with a voice bar (new letters
p b d g z zh v). `subglottal` adds a faint windpipe pole and zero at 600 and 1500 Hz. `naturalVibrato`
makes the vibrato rate 5 to 6.5 Hz, rising with pitch and wandering slowly, with a wandering extent
and a late onset. The new features use a private noise stream.

## shaper

`synths/shaper`, the chain language in `blocks/shapechain`

A synth that builds its wave from a **chain of shapers** (genny's `float/shape`): a phase from 0 to 1
goes in, the chain bends it into a wave. Up to four chains, A to D, are written as text, and the
**morph** (0-1) goes through them: 0 is chain A, 1 the last chain, in between a blend of two
neighbours. An envelope, an LFO, the velocity or the wheel can pull the morph, so the wave itself
moves. Two lines (octave, detune, level, a morph offset of their own) read the same chains, then go
through the [voice filter](#the-voice-filter) of every synth and an amp envelope.

```go
cfg := shaper.Glass()
cfg.Chains[1] = "bipolar | mirror -0.4 0.4 | mult 2.5 | sin" // chain B
v := shaper.NewPoly(cfg, 8).AddTo(p)
v.ReceiveMessage(psalm.Message{"chainC": "par(sin, mult 3 | sin | mult 0.4)", "morph": 0.7})
```

**A chain** is stages joined by `|`: `bipolar` is a saw, `sin` a sine, `mult 3 | sin` a sine of three
cycles, `bipolar | mirror -0.5 0.5` a saw folded back, `par(sin, mult 3 | sin | mult 0.4)` two
chains side by side with their results added. The stages are genny's shapers by the same names, and
a test holds them to genny's:

| Stage | x becomes |
|---|---|
| `thru`, `add a`, `mult m`, `linear scale shift` | `x`, `x + a`, `x * m`, `x * scale + shift` |
| `mod m`, `ripple m` | the remainder of `x / m` (`m` not 0); `x` plus it |
| `bipolar`, `unipolar` | 0..1 to -1..1, and back |
| `abs`, `sin`, `tri` | `abs(x)`; `sin(2πx)`; a triangle over 0..1 |
| `mirror lo hi` | folded back into the range, as a ball bounces (0 1 to start) |
| `pulse w`, `quantize q` | 1 below `w`, else 0; the nearest multiple of `q` |
| `clip lo hi`, `softclip a` | cut off at the edges (-1 1 to start); an arctangent curve |
| `cheby 1:1 3:0.3` | Chebyshev harmonics (1 to 64) at their levels |
| `table sine 256`, `table 0 1 0.5 -1` | a wave (`sine`, `cosine`, `sawtooth`, `triangle`, `square`, and `uni-` for 0..1) read over 0..1, or your points with straight lines between them |
| `pow p`, `tanh d` | `abs(x)^p` with the sign of `x`; `tanh(d x) / tanh(d)` (these two are not genny's) |
| `par(a, b, ...)` | each chain on `x`, the results added |

**FM** is a chain too: `par(thru, mult 3 | sin | mult 0.3) | sin` is a sine at the pitch whose phase a sine at three times the pitch
moves by 0.3 of a turn (about 1.9 radians): phase modulation, the sound of a DX. The ratio is a whole number (`mult 1`, `2`, `3`,
`4`...: the wave must come round once a cycle), the index a number, or a macro the envelope pulls down over the note
(`mult m1[0,0.4]`): an electric piano, a bell, a bass.

A number in a stage can be a **macro**: `m1` to `m4`, a value from 0 to 1 that the sound moves while it
plays (`macro1`...`macro4`, the matrix, the `shapeEnv` envelope on `m1`). `m1` is 0..1, `m1[1,5]` maps
it to 1..5, and `m1[5,1]` upside down. `mult m1[1,6] | mod 1 | bipolar` is a hard synced saw whose
sync ratio the macro sweeps. Numbers are separated by spaces or commas, new lines count as spaces,
and `#` starts a comment. A chain that can't be read is an error with its column (`shapechain.Parse`);
the synth keeps the chain it had.

| Setting | |
|---|---|
| `Chains` | the text of chains A to D; the list ends at the first empty one; chain A is a sine when empty |
| `Morph`, `Macros` | 0-1, where in the chains it is; the values of `m1` to `m4` |
| `Line1`, `Line2` | `Octave`, `Detune` (cents), `Level` (line 2 is off at 0), `Morph` (-1 to 1, added to the morph) |
| `MorphEnv`, `MorphAmount` | an envelope over the note that moves the morph by up to `MorphAmount` (-1 to 1); off at 0 |
| `ShapeEnv`, `ShapeAmount` | the same on macro `m1` |
| `AmpEnv`, `Width`, `Level` | the amp envelope; line 1 left and line 2 right; output level |
| `PitchEnv`, `PitchAmount` | an envelope that bends the pitch over the note by up to `PitchAmount` semitones (-60 to 60): a kick that falls three octaves, a laser, a riser; off at 0 |
| `Glide`, `Mono`, `Legato` | milliseconds a note takes to slide from the pitch the voice was at; with `Mono` (and `Legato`) `NewPoly` plays one note at a time, so a bass or lead slides from note to note (`set({mono: 1, legato: 1})` from a script does it too) |
| `Unison`, `Spread`, `UniStereo` | copies of each line (1 to 7, taken when a note starts on a silent voice; the level stays the same), the outer ones `Spread` cents (0-100) under and over the pitch, and spread across the stereo field by `UniStereo` (0-1). Each copy costs as much as a line, so a pad of eight voices at 3 copies is six times the work of one; keep `Oversample` at 1 there |
| `Drift` | 0-1: every copy, the morph and the cutoff wander slowly and at random (a few cents, a few hundredths of the morph), as the oscillators of an analog synth do, so chords are never the same twice |
| `Sub`, `Noise`, `NoiseColor` | a sine an octave under the note (0-1); white noise (0-1) made dark (under 0) or bright (over 0) by `NoiseColor` (-1 to 1): breath in a flute, air on a pad, grit on a bass. Both go through the voice filter |
| `XMod`, `XModType`, `XModSilent` | line 1 acting on line 2 (0 is off): type 0 bends the phase of line 2 by the wave of line 1 (FM: the amount is the index, up to a whole turn, and the octave and detune of the lines are the ratio, 3.5 for a bell with `Octave: 1, Detune: 968`), type 1 multiplies the two (a ring modulator, the amount blends it in). `XModSilent` hides line 1: it only modulates. `xmod` is also a destination of the matrix (an envelope that dies with the note makes a bell, an LFO a growl) |
| `Oversample` | how many times a sample the chains are read: 0 (the start) is automatic (see below), or 1 (off), 2, 4, 8 |
| `VibratoRate/Depth/Delay`, `LFO1Rate`, `LFO1Sync`, `LFO2Rate`, `LFO2Shape` | a vibrato and the two LFOs of the matrix |
| `Filter`, `Mods`, `Ext` | the [voice filter](#the-voice-filter) (off at first) and the [matrix](#modulation-matrix): filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, Ext 1-4 to `morph`, `macro1`-`macro4`, `pitch`, `vibratoDepth`, `width`, `level`, `detune2`, `cutoff`, `resonance`, `xmod` |

**Aliasing and cost.** Folds, `mod`, `quantize` and steep curves make harmonics far above the pitch. By default
(`Oversample` 0, "auto") the synth works out what it needs: `Chain.Bandwidth()` looks at the spectrum of each chain (with
each macro it uses at 0, half and 1) and finds the highest harmonic that has a say, and from that and the pitch of the
note (and its lines) the voice reads the chain once a sample when everything stays under half the sample rate (a sine, a
few partials: the organ, the vowels and the metal presets). A chain with more than that, a fold or a sharp edge (a saw, a
pulse, steps, hard sync: harmonics without an end), is read from **tables**: one cycle of the wave of each line is worked
out (the chains at the morph and the macros it has now), cut with an FFT to the harmonics that fit under half the sample
rate at the pitch, and played by a cubic reading, which is a few multiplications. All the copies of a unison read the same
table, so a unison with a narrow chain uses one too. When the morph or a macro has moved a little (0.01) a new table is
made and the sound fades from the old one to it, so a sweep or an LFO is a series of small steps no one hears (a line makes a table at most once every half table of samples, so a fast envelope on a macro costs a couple of chain evaluations a sample and not more, and the fades between the tables keep it smooth); a chain
swapped under a note fades over the twelve milliseconds it always did; a pitch that rises until the table would fold back,
or falls far enough to make it dull, gets a new one. A table has no aliasing and keeps the harmonics of the chain to a
tenth of a decibel (a test compares them with the wave read eight times a sample), and costs a fifth of reading it
four times: `supersawLead` costs 0.3 microseconds a sample against 2.0 with 4 times oversampling. Set `Oversample` yourself to
fix it: 1 reads the chain once a sample whatever it is (cheap, and it folds), 2, 4 and 8 read it that many times and low-pass
the result (`blocks/oversample`; 8 costs twice what 4 does). A voice also stops working out a line once its amp envelope
is below -100 dB, the end of a long release. The cross modulation does not take care of what it makes above half the sample
rate: keep the index moderate for high notes.

**Presets by message.** The message `preset` makes a voice a preset by name (`set({preset: "reese"})` from a script,
`shaper.NewPoly(...).ReceiveMessage(psalm.Message{"preset": "reese"})`): every setting of it, the chains, envelopes, lines, filter,
matrix and LFOs; the keys after it in the same message then change it (`{preset: "acid", cutoff: 600}`). What is not a preset's
is left as it is: `trim` (dB, where it sits in a mix; the presets are all about the same loudness), and the mono play of
the synth (`mono`, `legato`, of the poly: `acid` and the flute need them). A sounding voice goes on: the chains are faded to, the
levels glide, a filter that comes is faded in and one that goes is left on, wide open.

A note that starts on a voice still sounding (the next note of a mono synth) carries on from it: the old note sounds until the
sample the new one starts at, and the wave, the envelope and the settings continue.

**Changes under a sounding note.** A script (or a panel) can change `morph`, the macros, the levels and the width at any
moment, as often as every step. They glide to the new value in 5 ms, so a change never makes a click, and a chain swapped for
another is faded to over 12 ms while the old one fades out (a test changes each under a note and holds the second difference of
the wave to a few thousandths). What cannot glide is the type of the filter (off, low pass, ladder): changing it under a note
steps; keep a filter on, wide open, instead of off.

A lopsided wave (a narrow pulse sits mostly on one side of zero) has its mean taken off before the envelope, worked out exactly at
the start of each note and followed slowly after; otherwise the envelope would switch the offset on and off with every note, a thump
that a small speaker plays as a tick.

**Messages**: `preset`, `trim`, `pitch`, `frequency`, `velocity`, `glide`, `pitchAmount`, `morph`, `macro1`-`macro4`, `chainA`-`chainD` (text, or a
`*shapechain.Chain` read already, which is what the patcher sends), `octave1/2`, `detune1/2`,
`level1/2`, `morph1/2`, `morphAmount`, `shapeAmount`, `oversample` (0-4: auto, off, 2x, 4x, 8x), `width`,
`unison` (1-7), `spread`, `uniStereo`, `drift`, `sub`, `noise`, `noiseColor`,
`level`, `xmod`, `xmodType`, `xmodSilent`, `vibratoDepth`, `vibratoRate`, `lfo1Rate`, `lfo2Rate`, `lfo2Shape`, the envelopes `amp`,
`morphEnv`, `shapeEnv`, `pitchEnv` and `filter`, the filter and the matrix, and the [pitch bend, mod wheel and
aftertouch](#pitch-bend-mod-wheel-and-aftertouch). **Value input** 1: the morph; 5 to 8: Ext 1-4.

Presets: 66 in nine families (see [presets](presets.md#shaper), and `go run ./cmd/examples/presets -synth shaper`
plays them all): basses (`subSine`, `kickSub`, `reese`, `acid`, `wobble`, `pulseBass`, `fmBass`, `growl`, `subFold`),
leads (`supersawLead`, `squareLead`, `syncLead`, `pluckLead`, `fmLead`, `fluteLead`, `brassLead`, `vowel`, `reed`), pads
(`warmPad`, `stringPad`, `choirPad`, `airPad`, `darkPad`, `evolve`, `glassPad`, `foldPad`, `morphPad`, `swarm`, `organ`), keys and
plucks (`pluck`, `harp`, `kalimba`, `marimba`, `ePiano`, `clav`, `vibes`, `stab`, `musicBox`, `pwm`), bells (`bellMorph`,
`glassBell`, `churchBell`, `gong`, `metal`, `glass`), drones (`drone5`, `subDrone`, `crush`) and effects (`laser`, `riser`,
`impact`, `zap`) cross modulation (`xmodBell`, `xmodGrowl`, `ringMetal`) and beyond, what only this synth does (`rainbow`, `crystalSea`, `fourBeasts`, `trumpetVoice`, `sevenLamps`, `crownsCast`, `thunder`, `lightning`, `lionBass`, `throneDrone`, `manyEyes`). A test plays every one and holds it to the same measures: no clipping, as loud as the others, nothing
shrill, silent after the key is let go, and not like another.

In the patcher the synth is a module of its own (Synths, "shaper"): its four chains are boxes in the
window (Ctrl+Enter takes one, an error shows in red and the sound goes on; choosing a factory
preset puts that preset's chains in the boxes). The same text is a shaper in a script:
`genny.shapers.chain("mult 3 | sin")` (see [scripting](scripting.md#shapers)).

## supersaw

`synths/supersaw`

The supersaw of the Roland JP-8000: seven detuned sawtooths, the sound of trance pads and leads.
The detune and the mix of the center saw against the others follow the curves Adam Szabo measured
on the original; the saws are band limited, start at random phases, spread over the stereo field
and go through a filter with its envelope.

```go
pad := supersaw.NewPoly(supersaw.TrancePad(), 8).AddTo(p)
```

| Setting | Meaning |
|---|---|
| `Saws` | 0 is 7 |
| `Detune`, `Mix` | 0-1 each, as on the JP-8000 |
| `Stereo` | 0-1, the saws left and right |
| `Sub` | 0-1, a square an octave down |
| `HighPass` | Takes out what lies below the note, as the original does |
| `Cutoff`, `Resonance`, `FilterEnvAmount`, `KeyTrack`, `Velocity` | The filter (Hz, 0-1, octaves, 0-1, octaves) |
| `FilterEnv`, `AmpEnv` | ADSR (ms) |
| `Glide` | Seconds |
| `LFORate`, `LFOSync`, `LFOCutoff`, `LFODetune` | An LFO (Hz, or beats a cycle) on the cutoff (octaves) and the detune |

**Value inputs**: 1 cutoff, 2 detune. Presets: `hypersaw` (sixteen saws, a wall of sound), `wobbleBass`
(the filter in eighth notes), `breathing` (the detune swelling), `swarm` (a four second rise),
`glideLead` (sliding: play it mono legato), `trancePad`, `lead`, `pluck`, `hoover`, `stab`,
`unisonBass`.

**Messages**: `pitch`, `frequency`, `velocity`, `level`, `detune`, `mix`, `stereo`, `saws`, `sub`, `cutoff`, `resonance`, `filterEnvAmount`, and the [pitch bend, mod wheel and aftertouch](#pitch-bend-mod-wheel-and-aftertouch).

### Matrix, ensemble, high pass

An 8 slot [modulation matrix](#modulation-matrix): sources filter env, amp env, lfo 1 (`lfoRate`,
`lfoCutoff`, `lfoDetune`), lfo 2 (a second sine or triangle, `lfo2Rate`, `lfo2Shape`, running when a
slot uses it), velocity, key, random, wheel and Ext 1-4 (value inputs 5-8); destinations `detune`,
`mix`, `stereo`, `sub`, `cutoff` (octaves), `resonance`, `pitch` (semitones), `level`, `lfoRate` and
`ensemble`.

`ensemble` (0-1) adds a built-in string-machine chorus: three modulated delays of about 7 ms per
channel, swept slowly at `ensembleRate` Hz (default 0.6). It makes a centered voice wide and moves it;
at 0 it is off. `hpCutoff` (20-2000 Hz, 20 is off) is an extra high pass in series before the filter;
the `HighPass` setting that follows the note stays as it was.
