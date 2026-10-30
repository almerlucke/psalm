# lane

A **curve per section** of a song: a value that moves through a section as it plays. Cable its `mod` output to any setting (a filter cutoff, a volume, a chance) and the setting follows the form of the song. For curves you draw over many bars, see [automation](automation.md).

## Tutorial

1. Add a [song](song.md) and a lane. The curves are a line each: a section, a start and an end value from 0 to 1:

```example A build and a drop
[curves]
Intro: 0 → 0.3
Verse: 0.3
Chorus: 0.3 → 1 smooth
Outro: 1 → 0.1 exp
```

The shapes are `lin` (straight), `exp` and `smooth`. A line without a section is for the others.

2. Cable `mod` to a setting. In the module's inspector below the patch, choose the range it moves over.
3. `value` gives the number as it is, for things that are not settings: cable it to a [notes](notes.md) velocity or a [sequence](sequence.md).
4. Several lanes make several things move at once: one for the filter, one for the amount of reverb, one for how many hits the drums play.
