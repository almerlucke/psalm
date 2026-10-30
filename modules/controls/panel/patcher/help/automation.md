# automation

**Lanes of points** over the bars of a song, to draw: a cutoff that opens over 12 bars, a volume that dips. Every lane has a mod output (0 to 1) to cable to any setting. Where a [lane](lane.md) is a curve per section, automation is a curve over the whole song.

## Tutorial

1. Add a [song](song.md) and an automation module. The preset says how many lanes (and outputs) it has.
2. Type the lanes, a line each: a name, then points `bar value`, with a shape after a point (`lin`, `smooth`, `step` or `exp`):

```example A cutoff that opens and closes
[lanes]
Cutoff: 0 0.2, 6 0.8 smooth, 12 0.5 step
```

3. Or press the **Lanes** button of the patch and draw them over the bars of the song: click to add a point, drag to move it.
4. Or **record** a knob: arm a lane (the ● of the lane), play the song and move a knob. What you did is the lane.
5. Cable each lane's output (`lane 1`, `lane 2`...) to the setting it should move. Without a song, **bars** says how long the round is.
