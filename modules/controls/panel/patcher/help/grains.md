# grains

A **granular sampler** you play like any synth. Every note starts a stream of grains: very short windowed slices of a recording, played at the pitch of the note. Moving where the grains come from, how dense and long they are and how much they spread turns any recording into pads, drones, keys, textures or a rhythmic stutter. It takes `notes`.

The recording is the text **sounds** in the window of the module, a line each: a name, a file and the MIDI pitch the recording is at (60 without). The first is what it plays. A line can also be a sound that is made: `tone osc sine saw` (single cycle waves, see below) or `me live` (what a [grain buffer](grainbuffer.md) hears, as it is played). The presets are ways of using the grains (`choirPad`, `frozenDrone`, `glassKeys`, `pluck`, `shimmer`, `stutter`, `texture`) and work with any recording. A text that cannot be read (a missing file) is reported and leaves the sound as it was.

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
| `pitchSpread` | A random detune of each grain, in semitones: a chorus, a smear. |
| `octave` | The chance that a grain plays an octave up: sparkle. |
| `stereoSpread` | A random place in the stereo field for each grain. |
| `level` | The output level. |

The level stays about the same when the density, size or shape change. The velocity of a note is its volume. The presets can also be changed in the editor, with the envelope of the whole note.

## Other sounds than a recording

- `tone osc sine saw` in the sounds: single cycle waves (`sine`, `triangle`, `saw`, `square`) in place of a recording. **position** picks the wave (0 the first, 1 the last), a grain is a short burst of it at the pitch of the note: grains of a wavetable, with a smooth cloud or a rough buzz by **density** and **grainSize**. The waves are made of their harmonics, fewer for the high notes, so they do not alias.
- `me live` plays what a [grain buffer](grainbuffer.md) hears: **position** is how far back the grains look.

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
