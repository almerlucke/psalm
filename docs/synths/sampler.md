# sampler

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

## Slices

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

`Chops(snd, cuts, maxSec)` makes pitched chops of the pieces: every piece answers every key and is tuned to the key it is played on, its root the pitch `DetectPitch(snd, from, to)` finds in it (by autocorrelation: a sung syllable is on the chord tone it is played at; a consonant or a breath with no pitch plays as it is on every key), and a note plays one piece, chosen at random (`LoRand`/`HiRand` divide 0 to 1 between them). In a text of sounds: `36 voice.wav chops 9 0.06 0 0 0.4` (rise, gap, from, to, the longest piece). Play it with chord notes (a part of `@1 @3 @5`) and the voice sings the chords.

`Notes(snd, firstKey, spacing, notes)` makes a region for each played note of a recording (sung notes, a phrase): `[from, to, pitch]` in seconds and a MIDI pitch
with a fraction, which tunes it. Note i is on the key `firstKey+spacing*i` and two either side, so a key a little off plays it a little higher or lower; it fades in over 40 ms, and goes on as long as the key is held (its second half goes round, with a crossfade), releasing over 0.4 s.

`go run ./cmd/examples/slicer -demo` plays the tabla loop cut at its hits, in order and then shuffled (`-file`,
`-by change` for a sung line, `-by pauses -max 4` for speech, `-by verses -verses 26` for a recitation, `-rise`, `-gap`, `-slices n`, `-first`); without `-demo` the pieces are on a MIDI keyboard.

`sampler.New(cfg)` is a single voice, `sampler.NewPoly(cfg, n)` a `poly` synth whose voices share
the round robin turns and the choke groups. An instrument doesn't change once built: one can play
in many synths. `LoadSFZWith(path, sampler.MipMaps(4))` keeps band limited copies of every sound,
for regions played far above their key.

## Regions

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

**Messages**: `pitch`, `frequency`, `velocity`, `level`, `sustain` (the pedal), `cc1` to `cc127` (0-1), `modWheel` (controller 1), and the [pitch bend, mod wheel and aftertouch](../../synths.md#pitch-bend-mod-wheel-and-aftertouch).
