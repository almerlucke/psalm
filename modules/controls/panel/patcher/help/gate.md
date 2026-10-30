# gate

A part of the song that plays in **some sections only**. Sound, notes and triggers pass through the gate in those sections; in the others they are stopped. It is how a patch makes an arrangement: the drums come in at the Chorus, the strings leave in the Outro. A note that was let go always goes through, so nothing hangs.

## Tutorial

1. Add a [song](song.md) and a gate. Put the gate between a source and what it plays: audio (`in` to `out`), notes (a [part](part.md) or [voice](voice.md) to the gate and the gate to an instrument), or triggers.
2. Say which sections it plays in:

```example Only in the loud parts
[name]
drums
[sections]
Verse, Chorus
```

`not Intro` plays everywhere but the Intro; `all` plays always.

3. Its **name** is a tile in the song's window: a tap switches the part off and on, and it lights when it plays. So you can mute a part while the song goes.
4. Parts that decide themselves where they play ([melodist](melodist.md)'s section chips, a [voice](voice.md) with per-section patterns) do not need a gate.
