# melodist

A melody that is made **while the song plays**. Where a note editor makes a melody once, the melodist makes a pass of it (a section of the song) at a time, over the chords of the [harmony](harmony.md), in a scale, in phrases that repeat and vary. The next pass is made ahead, so it is there at the bar line. Cable its `notes` output to an instrument.

## Tutorial

1. Add a [song](song.md) with chords, a melodist and an instrument; cable the melodist to the instrument. Start the transport. You hear a tune over the chords.
2. Pick a feel. The **presets** in the window set the settings to a kind of melody. Try these values:

```example A music box tune
[set]
form = 4        # call: a question and its answer
contour = 3     # fall
rhythm = 1      # even
phrase = 2      # 8 beats
grid = 1.5      # the shortest note, in beats
density = 0.1
rest = 0.4
steps = 0.92    # mostly steps, few leaps
chord = 0.85    # the beats keep to the chord
low = 70
high = 88
mutate = 0.12
dynamics = 0.8
velocity = 0.42
```

3. The settings, from a slider to its opposite:

| | |
|---|---|
| density, rest | how many notes, how often it is silent |
| steps | steps against leaps (1: only steps) |
| chord | how much the notes on the beat keep to the chord |
| grid | the shortest note in beats; 1 or 2 makes a slow tune |
| low, high | the range of the melody (MIDI notes) |
| velocity, dynamics | how loud, and how much the notes differ |

4. Shape it: **form** (`AA'`, `ABAB`, `AABA`, `call`, or a grammar), **contour** (`arch`, `rise`, `fall`, `wave`), **rhythm** and the length of a **phrase**.
5. Decide what stays. **mutate** is how much of a section changes the next time through: 0 the same melody every time, about 0.2 to 0.5 the same melody changed, 1 a new one. So the second chorus is the first chorus, varied. The **seed** makes the same melody every time.

```example A glittering high layer
[set]
form = 4
contour = 3
rhythm = 3      # triplet
phrase = 1
grid = 0.5
density = 0.2
rest = 0.55
steps = 0.8
chord = 0.7
low = 84
high = 103
mutate = 0.6
velocity = 0.35
```

6. Choose where it plays: the chips in the window take a section in or out, so a melodist can answer in the Chorus and be silent in the Verse.
7. Several melodists make layers: a tune, a lower answer, a high glitter. Give each its own seed and range, and each its own sections.

## Buttons of the window

- **Draw again** forgets what the sections had: the next pass is a new melody.
- **Freeze** keeps what each section has.
- **Keep in the roll** writes what plays into a note editor, to edit by hand.
- **Second voice** adds a counter-line to the same instrument.

For a figure you write yourself, over the chords, use a [part](part.md).
