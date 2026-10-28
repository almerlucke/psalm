# pulsar

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

## Presets

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

## Filter, matrix, second pulsaret and bursts

Pulsar has the [voice filter](../../synths.md#the-voice-filter) after the pulse train (off by default) and an 8 slot
[modulation matrix](../subtractive.md#modulation-matrix): the formant envelope as filter env, the amp envelope, the
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
