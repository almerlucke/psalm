# grains

A **granular sampler** you play like any synth. Every note starts a stream of grains: very short windowed slices of a recording, played at the pitch of the note. Moving where the grains come from, how dense and long they are and how much they spread turns any recording into pads, drones, keys, textures or a rhythmic stutter. It takes `notes`.

## What is on the module

- **source** (the first group) says what the grains play: a **recording**, a single cycle **wave**, or **live** (the sound a [grain buffer](grainbuffer.md) hears). A new module plays a choir (when the folder has it), so the first note already sounds.
- The box **recordings** is a list of sounds to play with the source **recording**: a line each, a name, a file and the MIDI pitch the recording is at (60 without): `choir assets/sounds/chants/Johannes.Hymnus.wav 62`. The first line is what plays; a [script](script.md) can pick another with `set({sound: "name"})`. A line may also be a sound that is made: `tone osc sine saw` (single cycle waves) or `me live` (a grain buffer): a script picks those by name too. A text that cannot be read (a missing file) is reported and leaves the sound as it was.
- The box **grain buffer to listen to** is the name of the grain buffer that the source **live** plays (`live` unless you named it otherwise).
- The groups: **Source** (where the grains are taken from), **Grains** (how often, how long, what shape), **Pitch** (a scale, chords, scatter), **Output** (stereo, level, and the envelope of a note). Rest the pointer on any control to read what it does.
- The **Note volume** envelope is the volume of a whole note: how fast it comes in when a key is pressed (A), how far it falls while the key is held (D, to the level S), and how long it fades after the key is let go (R). It is not the shape of a grain: that is **window** and **shape**.
- A script can do more than the controls: it gives every grain its own value (see below).

The presets are ways of using the grains (`choirPad`, `frozenDrone`, `glassKeys`, `pluck`, `shimmer`, `stutter`, `texture`) and work with any sound.

## Tutorial

1. Add a grains, put this line in its text **sounds** (a choir that ships with psalm, as seen from the folder psalm runs in): `choir assets/sounds/chants/mixkit-angelical-choir-654.wav 60`. Add a keyboard (or a [voice](voice.md)) and cable its `notes` output to the grains.
2. Load the first example (it sets the settings; the text above is typed) and play a chord: each note is the choir at that pitch, in a slow drifting cloud.
3. Try the other presets of the module. `frozenDrone` holds one moment of the recording, `glassKeys` plays short and plucked.
4. Move **position** (where in the recording the grains come from) while a note is held. If the recording only sounds in part of the file, put the position there.
5. **scan** moves the position by itself: 0 freezes a moment, 1 is the speed of the recording, a negative number goes backwards. A [lane](lane.md) on `position` moves the cloud through the song.

```example A choir
[set]
position = 0.4
scan = 0.05
density = 40
grainSize = 250
```

```example A frozen moment
[set]
scan = 0
position = 0.5
spray = 0.3
density = 60
grainSize = 300
```

```example Shimmer
[set]
octave = 0.4
pitchSpread = 0.2
stereoSpread = 1
```

```example Stutter
[set]
scan = 0
density = 12
grainSize = 60
shape = 0.8
spray = 0.02
jitter = 0.1
```

## Settings

| Setting | What it does |
|---|---|
| `source` | What the grains play: `recording`, `wave` or `live`. |
| `wave` | The wave of the source `wave`: sine, triangle, saw or square; with `morph` the position picks the wave (0 the first, 1 the last). |
| `rootNote` | The MIDI pitch of the recording: playing this note plays it at its own pitch. |
| `position` | Where in the recording the grains come from (0 to 1). |
| `scan` | How fast the position moves: 0 freezes, 1 is the original speed, negative goes backwards. It starts over at every note. |
| `spray` | A random spread of the places the grains come from, in seconds. |
| `density` | How many grains per second. Few is a stutter, many is a smooth cloud. |
| `grainSize` | How long a grain is, in milliseconds. Short is rough and percussive, long is soft. |
| `sync` | Beats between the grains, following the tempo (0 is off: `density` is used). 0.25 is a grain every sixteenth. Jitter still loosens it. |
| `window` | The shape of a grain: `hann` (smooth), `gauss` (a bell), `tukey` (a flat middle: grains you hear as notes), `triangle`. |
| `shape` | The window of a grain, from the window above (0) to percussive (1). |
| `jitter` | How uneven the time between grains is. |
| `reverse` | The chance that a grain plays backwards. |
| `scale` | Holds the pitch of every grain to a scale (`off`, `major`, `minor`, `pentatonic`, `blues`...), so the cloud is always in key. |
| `scale root` | The note the scale starts on. |
| `stack` | Every grain is a chord (`octave`, `fifth`, `major`, `minor`, `major 7`...): a grain for each of its notes. |
| `pitchSpread` | A random detune of each grain, in semitones: a chorus, a smear. |
| `octave` | The chance that a grain plays an octave up: sparkle. |
| `stereoSpread` | A random place in the stereo field for each grain. |
| `level` | The output level. |
| `filter`, `cutoff`, `resonance`, `keyTrack` | A filter on every grain (off, lowpass, highpass, bandpass), where it cuts, how much it rings, and how much the cutoff follows the pitch of the grain. |
| `mod1Source` ... `mod4Rate` | The **modulation** slots: a source (lfo, random, rise, velocity, envelope) times an amount moves the position, pitch, size, pan, level, cutoff or density of every grain. |
| `feedback`, `feedbackTime`, `feedbackPitch` | The chance that a grain plays what the voice played a moment ago, how far back, and how many semitones higher (12: a swell that climbs). |
| `stretch`, `overlap` | **Time stretch**: the recording moves this many times slower at the same pitch (4 is four times as long). `overlap` is how many grains sound at once, the density follows the size. |
| `slices`, `slice` | **Slice markers** in a recording (the onsets, or 4 to 32 equal parts) and which one is played; the position is then a place inside the slice. |

The level stays about the same when the density, size or shape change. The velocity of a note is its volume. The presets can also be changed in the editor, with the envelope of the whole note.

## Other sounds than a recording

- **source: wave** plays a single cycle wave. Pick it with **wave** (sine, triangle, saw, square), or choose `morph` and let **position** pick (0 the first, 1 the last). A grain is a short burst of the wave at the pitch of the note: grains of a wavetable, a smooth cloud or a rough buzz by **density**, **size** and **window**. The waves are made of their harmonics, fewer for the high notes, so they do not alias.
- **source: live** plays what a [grain buffer](grainbuffer.md) hears (the one named in the box *grain buffer to listen to*): **position** is how far back the grains look, 0 is just now.
- A script picks any of the lines of the box *recordings* by name, also the made ones (`tone osc sine saw`, `me live`), and so can switch the sound while it plays.

```example A time stretch
[set]
stretch = 6
overlap = 6
grainSize = 250
spray = 0.02
```

```example A beat from slices
[set]
slices = 1
position = 0.2
grainSize = 80
density = 8
jitter = 0
```

```example Bright and dark
[set]
filter = 1
cutoff = 1200
resonance = 0.5
mod1Source = 2
mod1Dest = 5
mod1Amount = 0.7
```

## Controlling each grain from a script

The settings are one value for all grains. A [script](script.md) can give the grains lists of numbers that they use one after the other, a value for each grain that starts, with `set`:

| Key | The list is |
|---|---|
| `positionSeq` | where each grain looks (0 to 1) |
| `intervalSeq` | seconds until the next grain: a rhythm (instead of `density`) |
| `pitchSeq` | semitones added to the note for each grain |
| `sizeSeq` | the length of each grain (ms) |
| `panSeq` | the place in the stereo field (0 to 1) |
| `ampSeq` | the level of each grain; 0 is a rest |
| `cutoffSeq` | the cutoff of the filter of each grain (Hz) |
| `sliceSeq` | the slice each grain plays (0 is the first), see `slices` |
| `scale` | a list of semitones above the root (`[0, 2, 4, 7, 9]`) or a name (`"major"`, `"minor"`, `"pentatonic"`, `"blues"`, `"wholeTone"`...): the pitch of every grain is held to it; `"off"` takes it away |
| `scaleRoot` | the pitch of the root of that scale (any octave) |
| `stack` | semitones above the pitch: every grain is a chord (`[0, 4, 7]` makes three grains for each) |

An empty list takes a sequence away again. The lists are cycled and start over at every note. The values of a generator are a list with `take`:

```js
const walk = genny.rwalk(0.5, 0, 1, 0.2)
const rhythm = genny.euclid(5, 8)

function trigger(ev) {
  if (ev.step % 16 === 0) {
    set({
      positionSeq: walk.take(16),
      intervalSeq: [0.125, 0.125, 0.25, 0.125, 0.25],
      scale: "minor", scaleRoot: 57,
      stack: [0, 7],
    })
    note(48, 0.8, 8)
  }
}
```

Other modules that play recordings: the [sampler](sampler.md) and the [streamer](streamer.md).
