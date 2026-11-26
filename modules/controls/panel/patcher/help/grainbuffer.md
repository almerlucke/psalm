# grainbuffer

A **grain buffer** keeps the last seconds (eight) of the sound that comes in, so the [grains](grains.md) of a granular synth can play it: a synth that granulates what you sing, play or what another synth plays, as it happens. The sound goes on to the output unchanged, so the buffer can sit in a chain.

## Tutorial

1. Cable a sound (a [voice](voice.md), an [input](input.md), a synth) to the `in` of a grain buffer. The buffer listens; what is played has no other route to the grains.
2. Add a [grains](grains.md) and put this line in its text **sounds**: `me live`. The line is a name for the sound (`me`), the word `live` and the name of a grain buffer (`live` when it is not given another: the **name** text of the buffer).
3. Play a note on the grains. Each grain is a short piece of what the buffer heard. **position** says how far back the grains look: 0 is just now, 1 is the oldest second there is. A position of 0.3 lets the grains play what was played a bit over two seconds ago, in a cloud.
4. Keep **spray** wide for a cloud, narrow for an echo of a phrase. A high note of the grains reads faster than the sound comes in; the grain then stays further behind so it never runs into what is being written.
5. **freeze** stops the buffer where it is: the grains play the moment it holds, however long. Turn it off to listen again.

```example Hold a moment
[set]
freeze = 1
```

## Settings

| Setting | What it does |
|---|---|
| freeze | stops listening: the grains play what the buffer holds now |
| name (text) | the name of the buffer, in a line of the sounds of a grains: `me live name` |

Two buffers need two names. The line can also give the pitch of the sound, as with a file: `me live name 57` says the sound is at pitch 57, so a note of 57 plays it as it was.

## See also

[grains](grains.md), [granulator](granulator.md), [input](input.md), [recorder](recorder.md).
