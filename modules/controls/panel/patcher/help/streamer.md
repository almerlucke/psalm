# streamer

The sampler for **sounds too long to keep in memory**: an hour of rain, a piece of music, a field recording. A note plays the file straight from the disk while it sounds; only the first half second of each file is in memory, so the note starts at once. It takes `notes`.

The module has no settings: its sounds are the text **sounds** (in the window of the module), a line for each, and the status below it reports what every line made. A text that cannot be read (a missing file) leaves the sounds as they were. If the disk is too slow the sound has a gap.

## A line of the text

```
36 file.wav loop vol=-6
```

A key, a file (a path as seen from where psalm runs) and any of these:

| Word | Meaning |
|---|---|
| `loop` | goes round while the key is down (between `loopstart=` and `loopend=`, in seconds). The loop is not crossfaded: use a file that loops well, or cut it where it goes round. |
| `hold` | a note off fades the sound out. Without `hold` a sound plays to its end, like a drum. |
| `speed=2` | plays faster: 2 is an octave up, 0.5 an octave down. It plays forwards only. |
| `start=` `end=` | the part of the file to play, in seconds. |
| `release=0.5` | how many seconds the fade after a note off takes. |
| `vol=-6` `pan=-1..1` `name=rain` | level in dB, place, a name. |

A `#` starts a comment. A single line plays on every key.

## Tutorial

1. Add a streamer and a keyboard (or a [voice](voice.md)). Cable the `notes` output to the streamer.
2. Type the first example below into the text and play any key: rain goes on as long as the key is held, because of `loop` and `hold`.
3. Give the voice a long `length`, or hold the key, and put a [lane](lane.md) or an [automation](automation.md) on something else in the patch: the rain is a bed under it.
4. Try two lines: different recordings on different keys, so one key is the rain and the next the thunder (the second example).

For example, rain as a bed:

```
36 assets/sounds/various/rain-sound-188158.wav loop hold vol=-6 name=rain
```

For example, rain and thunder, two keys:

```
36 assets/sounds/various/rain-sound-188158.wav loop hold vol=-8 name=rain
37 assets/sounds/various/rain-and-distant-thunder-60230.wav loop hold vol=-8 name=storm
```

## Good to know

- Use the streamer for long files and the [sampler](sampler.md) for sounds you cut into pieces, play backwards, or tune over the keys.
- For a recording as a cloud of tiny grains, see [grains](grains.md).
