# arp

An **arpeggiator**: it takes the notes you hold (from a keyboard or a note editor) and plays them one at a time, on the beat of a clock. One held chord becomes a figure.

## Tutorial

1. Add a keyboard (or a [noteroll](noteroll.md)), an arp, a [clock](clock.md) and a synth. Cables: the keyboard's `notes` to the arp's `notes` input, the clock's `trigger` to the arp's `clock` input, the arp's `notes` output to the synth.
2. Hold a chord. On every trigger of the clock the arp plays the next held note. The clock's pattern and division make the rhythm: `x x . x` leaves a gap, `1/8` is slower than `1/16`.
3. Choose the **mode**: `up`, `down`, `up & down`, `down & up`, `as played` (in the order you pressed them), `random`, or `chord` (all the notes at once on every trigger).

```example A rising figure two octaves high
[set]
mode = 0       # up
octaves = 2
gate = 0.7     # how much of a step a note lasts
```

4. **octaves** climbs the figure through that many octaves; **gate** is how much of a step each note lasts (short is staccato); **transpose** moves it in semitones.
5. **hold** keeps the notes after you let go, until you play new ones, so your hands are free to change other things:

```example Hands free
[set]
mode = 2       # up & down
octaves = 1
gate = 0.9
hold = 1
```

6. Put a [clock](clock.md) with `E(5,16)` or lanes on its trigger and the same chord plays a different rhythm each time. A [lane](lane.md) cabled to `transpose` makes the figure move through the song.

For arpeggios that follow the chords of a song without anyone holding keys, write them in a [part](part.md) (`@1 @3 @5 @3`).
