# grains

A **granular sampler** you play like any synth. Every note starts a stream of grains: very short windowed slices of a recording, played at the pitch of the note. Moving where the grains come from, how dense and long they are and how much they spread turns any recording into pads, drones, keys, textures or a rhythmic stutter. It takes `notes`.

The recording is the text **sounds** in the window of the module, a line each: a name, a file and the MIDI pitch the recording is at (60 without). The first is what it plays. The presets are ways of using the grains (`choirPad`, `frozenDrone`, `glassKeys`, `pluck`, `shimmer`, `stutter`, `texture`) and work with any recording. A text that cannot be read (a missing file) is reported and leaves the sound as it was.

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
| `shape` | The window of a grain, from smooth (0) to percussive (1). |
| `jitter` | How uneven the time between grains is. |
| `reverse` | The chance that a grain plays backwards. |
| `pitchSpread` | A random detune of each grain, in semitones: a chorus, a smear. |
| `octave` | The chance that a grain plays an octave up: sparkle. |
| `stereoSpread` | A random place in the stereo field for each grain. |
| `level` | The output level. |

The level stays about the same when the density, size or shape change. The velocity of a note is its volume. The presets can also be changed in the editor, with the envelope of the whole note.

Other modules that play recordings: the [sampler](sampler.md) and the [streamer](streamer.md).
