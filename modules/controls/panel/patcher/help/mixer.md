# mixer

A **mixing desk** for the patch: channels with a level, a pan and a mute, four effect buses (A to D) with sends and returns, and a main out. Cable the sound of every instrument into a channel, and the main out to the output. Choose 4, 8 or 16 channels when you add it.

## Tutorial

1. Add a mixer and a few instruments. Cable each instrument's audio out to a channel input (`1`, `2`, ...), and the mixer's `main` output to the end of the patch.
2. Set each channel's **level** (0 to 2, 1 is as it comes), **pan** (-1 left to 1 right) and **mute**. Set levels so the sum does not clip; the main **level** is the last control.

```example Three channels
[set]
level1 = 0.9
pan1 = 0
level2 = 0.6
pan2 = -0.4
level3 = 0.6
pan3 = 0.4
level = 1
```

3. Add effects with the buses. Every channel has four sends, `sendA` to `sendD`: how much of the channel (after its level) goes to the bus. The bus is an output of the mixer (`send A` ... `send D`): cable it to an effect such as a [reverb](reverb.md), and the effect's output back to the mixer's `return A`. The return sets how much of the effect comes back. One reverb shared by all channels, each sending a different amount, makes them sound in the same room.

```example A reverb on channel 1 and a little on 2
[set]
sendA1 = 0.4
sendA2 = 0.15
returnA = 1
```

4. Use a bus for anything shared: A a reverb, B a delay ([pingpong](pingpong.md)), C a [shimmer](shimmer.md), D a [granulator](granulator.md).
5. A mod cable from an [lfo](lfo.md), a [lane](lane.md) or [automation](automation.md) to a level, a pan or a send makes the mix move with the song.

## Settings

| | |
|---|---|
| level N | the level of channel N, 0 to 2 |
| pan N | -1 to 1 |
| sendA N ... sendD N | how much of channel N goes to bus A to D, 0 to 1 |
| mute N | 0 or 1 |
| returnA ... returnD | how much of the return comes back, 0 to 2 |
| level | the main level, 0 to 2 |
