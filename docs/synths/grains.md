# grains

`synths/grains`

A granular synth you play like any synth: every note starts a stream of grains, short windowed
slices of a **source**, played at the pitch of the note. The source is a recording, single cycle
waves, or the sound that is coming in now. Moving where the grains come from, how
dense and long they are and how much they spread turns any recording into pads, drones, keys,
textures or rhythmic stutters.

```go
choir := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-angelical-choir-654.wav", 4)
pad := grains.NewPoly(grains.ChoirPad(choir, 60), 12).AddTo(p) // sample, root note
pad.ValueIn(lfo.New(true, 0.05, 0.2, 0.8, lfo.Sine).AddTo(p), 0, 1) // slowly move the position
```

`grains.New(cfg)` creates a single voice, `grains.NewPoly(cfg, n)` a `poly` synth with `n` voices.
Use mipmapped sound files, so grains that play faster don't alias.

## Settings

| Setting | Default | Meaning |
|---|---|---|
| `Sample` | | The recording (`sndfile.SoundFiler[float32]`) |
| `Source` | | Any other [source](#sources), read in place of `Sample` |
| `SoundBank` | | Optional, notes pick a sample with the `sound` key |
| `RootNote` | 60 | MIDI pitch of the recording: playing this note plays it at its own pitch |
| `Position` | 0.3 | 0-1, where in the recording the grains come from |
| `Scan` | 0 | How fast the position moves: 0 freezes a moment, 1 is the original speed, negative goes backwards. The position restarts at every note |
| `Spray` | 0.05 | Seconds, random spread of the grain positions |
| `Density` | 30 | Grains per second |
| `Sync` | 0 | Beats between the grains, following the tempo (0: `Density`); 0.25 is a grain every sixteenth |
| `GrainSize` | 120 | Milliseconds |
| `PitchSpread` | 0 | Semitones, random detune per grain (chorus, smear) |
| `Octave` | 0 | 0-1, chance a grain plays an octave up (sparkle, shimmer) |
| `StereoSpread` | 0.5 | 0-1, random pan per grain |
| `Window` | `WindowHann` | The window a grain starts from: `WindowHann`, `WindowGauss` (a bell), `WindowTukey` (a flat middle) or `WindowTriangle` |
| `Shape` | 0 | 0-1, from that window to a percussive one |
| `Reverse` | 0 | 0-1, chance a grain plays backwards |
| `Jitter` | 0.3 | 0-1, randomness of the time between grains |
| `VelocityDensity` | 0 | 0-1, how much the note velocity raises the density |
| `Scale` | | Hold the pitch of every grain to a scale: the semitones above its root (`grains.ScaleNamed("minor")`) |
| `ScaleRoot` | 0 | The root of the scale, as a MIDI pitch (any octave) |
| `Stack` | | Every grain is a chord: one grain for each of these semitones above the pitch (`0, 4, 7`) |
| `Streams` | | A value for each grain from a list or a generator, see [streams](#streams) |
| `Normalize` | true | Play every recording at the same loudness (measured over the parts that sound) |
| `Level` | 1 | Output level |
| `AmpEnv` | | ADSR of the whole note (ms) |

The level stays about the same when the density, grain size or window shape changes. Notes take
an optional `velocity` (0-1, default 1), the note amplitude sets the volume.

**Value inputs**: 1 position, 2 density, 3 scan, 4 spray, 5 grain size. They only affect new
grains, so modulating them never clicks.

**Messages**: `pitch`, `frequency`, `velocity`, `sound`, `rootNote`, `position`, `scan`, `spray`,
`density`, `sync`, `grainSize`, `window`, `scale`, `scaleRoot`, `stack`, `pitchSpread`, `octave`, `stereoSpread`, `shape`, `reverse`, `jitter`,
`level`; the lists `positionSeq`, `intervalSeq`, `pitchSeq`, `sizeSeq`, `panSeq` and `ampSeq` (see [streams](#streams)); `source`
and `sources` (sources made by a patch).

## Sources

What the grains read is a `grains.Source`. A grain asks the source where it starts (`Locate`) and reads its samples a block at a
time (`Read`), so a source costs one call per grain and buffer. Set `cfg.Source`, or send `source` in a message:

| Source | Made by | What it is |
|---|---|---|
| recording | `grains.NewSampleSource(file)` (what `Sample` makes) | a sound file with its mipmaps; the position is where in the file |
| oscillator | `grains.NewOscillator("sine", "saw")` | single cycle waves (`sine`, `triangle`, `saw`, `square`); the position picks the wave. A grain is a burst of a wave at the pitch of the note (the root note 60 plays 261.6 Hz); the waves are made of their harmonics, fewer in the levels that are read fast, so high notes do not alias |
| live | `rec := grains.NewRecorder(8)`; `rec.Source()` | the last seconds of the sound that comes in. The recorder is a module with a stereo input that passes the sound on; the position is how far back the grains look (0 is now, 1 the oldest second). A grain that reads faster than the sound comes in is kept further behind, so it never reads what is not there yet. `rec.Freeze(true)` holds a moment |

```go
rec := grains.NewRecorder(8)
// the sound to listen to goes to the input of rec like to any effect, its output on to the mixer
cfg := grains.DefaultConfig()
cfg.Source = rec.Source()
cfg.Position = 0.2 // listen to what was played a second and a half ago
synth := grains.NewPoly(cfg, 8).AddTo(p)
```

A source of your own implements the interface (`Rate`, `Span`, `Depth`, `Loudness`, `Locate`, `Read`, `Ready`).

## Streams

The settings are one value for all grains. A **stream** gives the value for each grain that starts, in the order they start:

```go
cfg.Streams = grains.Streams{
	Position: func() grains.Stream { return grains.Gen(rwalk.New(0.5, 0, 1, 0.1, rwalk.Bounce)) }, // a walk over the source
	Interval: func() grains.Stream { return grains.Seq(0.125, 0.125, 0.25) },                       // a rhythm
	Pitch:    func() grains.Stream { return grains.Seq(0, 7, 12, 7) },                              // semitones above the note
}
```

| Stream | The value is | Replaces |
|---|---|---|
| `Interval` | seconds until the next grain | `Density`, `Jitter` and `Sync` |
| `Position` | 0 to 1, where the grain looks (the scan moves it on) | `Position` |
| `Pitch` | semitones added to the pitch of the note | |
| `Size` | milliseconds | `GrainSize` |
| `Pan` | 0 to 1 | `StereoSpread` |
| `Amp` | the level of the grain; 0 is a rest, no grain | |

`grains.Seq(values...)` cycles a list, `grains.Gen(g)` takes any generator of genny (`genny.Generator[float64]`: a walk (`rwalk`) or a ramp for the
`Position`, a distribution (`gaussian`, `exponential`) for the `Interval`, a map of a sequence of scale degrees for the `Pitch`); a generator that ends goes on with
its last value. The functions make a stream for each voice, so the voices of a poly synth do not share a place in a list. Streams start
over at every note. From a script of the patcher the same lists are sent with `set({positionSeq: [...]})`; see the
[grains help](../../modules/controls/panel/patcher/help/grains.md).

## Presets

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
