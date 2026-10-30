# drumsynth

**Drums made from oscillators and noise**, the way the classic drum machines make them: a kick is a sine that falls in pitch, a snare two tuned heads and a burst of noise, a hihat six square waves at clashing pitches through a high filter. There are no sound files: every sound of the kit has knobs of its own. The presets are the kits `808`, `909`, `electro` (zapping kicks and toms, tight hats), `industrial` (everything distorted, metal for hats) and `trap` (a long driven 808 boom, a big clap).

The drumsynth takes `notes` and plays a sound for each key, on the General MIDI drum keys, so a [voice](voice.md), a [sequence](sequence.md) or a keyboard can play it.

## The keys

| Key | Sound | Key | Sound |
|---|---|---|---|
| 35, 36 | kick | 42, 44 | hat (closed) |
| 37 | rim | 46 | open hat |
| 38, 40 | snare | 41, 43 | tomLow |
| 39 | clap | 45, 47 | tomMid |
| 56 | cowbell | 48, 50 | tomHigh |
| 49, 52, 55, 57 | crash | 51, 59 | ride |

The rest of the General MIDI percussion (tambourine, shaker, bongos, congas, timbales, agogo, clave, woodblocks, triangle) is on keys 53 to 81, in the character of the kit. The closed and open hats choke each other, as on the real machines.

## Tutorial

1. Add a [voice](voice.md), a drumsynth and a mixer (or the master). Cable the voice's `notes` output to the drumsynth.
2. In the voice, type the pattern `x . . . x . . .` with `pitch = 36` (the kick) and `division` at 1/16. You hear a kick.
3. Add a second voice for the snare: pattern `. . . . x . . .`, `pitch = 38`, cabled to the same drumsynth. A third on `42` with `x x x x x x x x` gives the hat. Three small voices are a beat. (The [voice](voice.md) page has the details of patterns and grooves.)
4. Open the drumsynth. Every sound of the kit has a group of settings. Try the kick:

```example A long, boomy kick
[set]
kickPitch = 42
kickDecay = 1.8
kickSweep = 18
kickDrive = 0.3
```

5. Change the preset to hear another character of the same sounds, then change what you do not like.

## What each setting does

Every sound has the same nine settings, in the group of its name (Kick, Snare, Clap, Hat...). A change is heard from the sound's next hit.

| Setting | What it does |
|---|---|
| `pitch` | The pitch of the sound in Hz: the body of a kick or a tom, the tuning of the heads of a snare, the metal of a hat. |
| `decay` | How long it rings, in seconds. A short hat is closed and tight, a long one washes. |
| `sweep` | How far the pitch falls at the start, in semitones: more is a deeper "boom" or a "zap". |
| `sweepTime` | How fast it falls. A short time is a click, a long one a slide. |
| `tone` | The filter of the noise or the metal: low is dark, high is bright. |
| `snap` | The attack: the click of a kick, the wires of a snare. |
| `drive` | Distortion: the sound gets louder, harder and grittier. |
| `level` | How loud the sound is in the kit. |
| `pan` | Where it sits, left to right. |

Apart from these there is `level` for the whole output.

```example Tighter snare and hats
[set]
snareDecay = 0.2
snareTone = 0.7
snareSnap = 0.8
hatDecay = 0.04
hatTone = 0.9
openDecay = 0.35
```

```example A dirty kit
[set]
kickDrive = 0.7
snareDrive = 0.5
clapDrive = 0.5
kickSweep = 24
snareSnap = 0.9
```

## Playing it

- The velocity of the note is how hard the sound is hit.
- The pitch bend moves the sounds that ring; the mod wheel makes new hits ring longer.

See also [voice](voice.md), [arp](arp.md) and [script](script.md).
