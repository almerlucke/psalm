# wavetable

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

## Tables

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

## Generated tables

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

## Settings

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

## Presets

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

## Oscillator B, warp and the matrix

A second oscillator, B, reads the same table at its own position, octave, semitone and detune
(`oscBPosition`, `oscBOctave`, `oscBSemi`, `oscBDetune`) and is mixed in with `oscBLevel` (0 is off).
`warpMode` and `warp` (0-1) bend the phase of oscillator A before the table is read: bend +, bend -,
sync (the frame plays 1 to 8 times per cycle), mirror, fm b (oscillator B modulates the phase) and fm
sine (`wavetable.WarpModes`). The band limited level of the table is chosen for the steepest slope of
the warp, which keeps most of the alias out, but a warp makes corners (mirror, bend) and steps (sync) the table cannot band limit, and the FM of the warps puts sidebands past the
Nyquist frequency. `oversample` (`Config.Oversample`, 0 once, 1 twice, 2 four times a sample, the oscillators only, through `blocks/oversample`) brings those down: at C6 the sync
has 7 dB less alias at twice a sample and 13 dB at four times, the mirror 6 and 12 dB, the FM tines 30 to 60 dB; the presets with a warp have it on (1 for the sync, mirror and bend,
2 for the FM, except the noises: `snareHit`, `hatMetal`, `windFx`). A wav becomes a table with `wavetable.Load` or
`gen.LoadSound`.

`Config.Mods` is an 8 slot [modulation matrix](../subtractive.md#modulation-matrix): sources filter env, amp env, lfo 1,
lfo 2 (plain sines, `lfo1Rate`, `lfo2Rate` in Hz, running when a slot uses them), velocity, key,
random, wheel and `Ext1`...`Ext4` (value inputs 5 to 8, `wavetable.InExt1`); destinations `position`,
`cutoff` (6 oct), `resonance`, `pitch` (24 st), `detune`, `spread`, `subLevel`, `level`, `warp`,
`oscBLevel`, `oscBPosition`, `oscBDetune` and the two LFO rates (`wavetable.Dests`).

## Oscillator B unison, additive tables

Oscillator B can be stacked like oscillator A: `oscBUnison` (1-7) oscillators spread over `oscBSpread`
cents (0 takes `detune`) and over the stereo width of `spread`, the level kept constant. In Go,
`gen.Additive(harmonics, numFrames)` builds a table from the amplitudes of its partials (index 0 is the
fundamental), and `gen.AdditiveMorph(from, to, numFrames)` crossfades between two partial lists over the
frames, so moving the position morphs one spectrum into the other. No aliasing: every harmonic is exact.
