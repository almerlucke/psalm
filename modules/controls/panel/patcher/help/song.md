# song

The **form** of a piece: its sections (Intro, Verse, Chorus...), how many bars each lasts, the chords of each, and the chord model that can draw more. One song a patch (the last made). Everything else that follows the form looks at it: [voices](voice.md), [parts](part.md), [melodists](melodist.md), [lanes](lane.md), [gates](gate.md) and [automation](automation.md).

## Tutorial

1. Add a song. The **sections** text is a name and a number of bars each, separated by commas or lines. The song plays them in order and loops:

```example A small song
[sections]
Intro 4, Verse 8, Chorus 8, Outro 4
```

2. Write the **chords**. A line `Section: chords` each, a chord a bar; `*2` lasts two bars; the lines without a section are for the others. They loop over the bars of the section:

```example Chords per section
[sections]
Intro 4, Verse 8, Chorus 8, Outro 4
[chords]
Intro: Am7
Verse: Am7 Dm7 G7 Cmaj7
Chorus: F G Am Am
Outro: Am7
[set]
key = 9     # A
scale = 11  # minor
```

`key` and `scale` say what roman numerals mean (`i VI V7`) and where the chord model draws.

3. Make things follow it. A part with chord notes (`@1 @3 @5`), a melodist and a roll all play over the chords of the moment. A [voice](voice.md) can write a different pattern per section (`Chorus: x x x x`).
4. Use its tiles. Tapping a section plays it next; its dot skips it; repeat plays the one playing again. The [gates](gate.md) of the patch have tiles too: a tap switches a part on and off.
5. Draw chords instead of typing them. The song has the **chord model** inside it (see [chordmodel](chordmodel.md)): draw a progression for a section, lock the chords you like, and switch on **new chords every section** so verse 2 does not have the chords of verse 1.

## Time signatures

`3/4` among the sections makes the bar three beats (four unless it says):

```example A waltz
[sections]
3/4, Theme 8, Trio 8
```

## Outputs

| | |
|---|---|
| bar | a trigger every bar |
| section | a trigger at every section |
| number | the number of the section (a value) |
| progress | how far through the section (a mod from 0 to 1) |
| hear chords | the notes of the chords, to hear them on an instrument |

The sections grid (the Sections button of the patch) shows the whole song as a grid with a pattern for every part in every section.
