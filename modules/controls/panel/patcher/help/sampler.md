# sampler

A **multi-sample instrument** made from sound files: a recording is cut into pieces that each play on a key of their own, or an SFZ instrument (a piano, a string section) is played over its keys. It takes `notes`. A drum loop becomes a kit, a sung line a set of syllables, a chant something that sings the chords you write.

What it plays is the text **sounds** (in the window of the module): a line for each sound. The module has no settings of its own: all it says is in the text, and the status below the text reports what every line made, or what is wrong with it. A text that cannot be read (a missing file, a mode it does not know) leaves the sounds as they were.

## A line of the text

```
36 file.wav hits 9 0.08
```

The first number is the key it starts on, then the file (a path as seen from where psalm runs), then how the file is cut:

| How | Meaning |
|---|---|
| `hits rise gap` | a piece at every hit, where the level jumps by `rise` dB (6 to 12 is usual); `gap` is the least seconds between two. The pieces lie on the keys from the first one up. |
| `change sensitivity gap` | a piece where the sound changes (a sung line, speech, what has no hits); `sensitivity` 1.1 to 2, the lower the more pieces. |
| `even n` | `n` pieces of equal length. |
| `whole hiKey root` | the whole file on the keys up to `hiKey`, recorded at the key `root`: played higher or lower it is tuned. |
| `chops rise gap from to longest` | pitched chops: every piece answers every key and is tuned to the key played; a note plays one piece, chosen at random. Play chords and a voice sings them. |
| `0 file.sfz sfz` | an SFZ instrument. |

Add `vol=-6` (dB), `pan=0.3` and `name=word` to a line. A `#` starts a comment.

## Tutorial

1. Add a sampler and a [voice](voice.md). Cable the voice's `notes` output to the sampler.
2. Type the first example below into the text of the sampler (paths are as seen from the folder psalm runs in): a tabla loop cut at its hits, a piece on every key from 36 up. The status says how many pieces it made.
3. In the voice, put a pattern and set `pitch` to 36: you hear the first piece. Give the voice pitches `36 37 38 39 40 41` (see [voice](voice.md)) and the loop is played piece by piece, in order. Shuffle the order (`order` of the voice) and you have a new rhythm out of the old one.
4. Try the second: a sung line cut where the sound changes. Every syllable is on a key. A [melodist](melodist.md) or a [part](part.md) walking over those keys makes a new tune out of the old words.

For example, a drum loop as a kit:

```
36 assets/sounds/various/tabla_loop_4bars_90bpm-86005.wav hits 9 0.08
```

For example, a sung line in pieces:

```
48 assets/sounds/chants/Johannes.Hymnus.wav change 1.5 0.2
```

For example, a shofar over the keys:

```
0 assets/sounds/various/shofar.wav whole 127 60 vol=-3
```

5. Several lines make several sounds, as long as their keys do not mean the same: a kit of three loops is three lines starting on keys far apart.

## Good to know

- Cut by `hits` for rhythm, by `change` for voices, by `even` when you want to be sure of the number of pieces.
- Pieces fade in and out over a few milliseconds, so the pieces played right after each other are the recording again, without gaps and clicks.
- The velocity of a note is how loud the piece is.
- For a recording too long to keep in memory, use the [streamer](streamer.md); to play a recording as a cloud of tiny grains, use [grains](grains.md).
