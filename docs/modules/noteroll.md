# noteroll

`modules/controls/panel/noteroll`, a kind of the [patcher](../controls.md#panel). In: none. Out: `notes`.

The note editor: a piano roll in the module's side panel (double-tap the module) that plays with the tempo of the patch and sends its notes to the synths it is cabled to. Kept with the patch.

## Editing

| | |
|---|---|
| Tools | select (V), draw (D), erase (E) |
| Select | click, shift-click, or a rectangle |
| Move, resize | drag the notes (several at once); the right edge resizes; Alt-drag copies |
| Copy, cut, paste | Ctrl+C / X / V; paste goes at the cursor |
| Duplicate | Ctrl+D repeats the selection, or the region chosen in the ruler (repeating) |
| Undo, redo | Ctrl+Z, Ctrl+Shift+Z |
| Velocity lane | drag the bars under the notes |
| Quantize, legato | buttons above the roll; on the notes selected, or all |
| Zoom, scroll | the wheel scrolls the keys, Shift+wheel the time, Ctrl+wheel zooms around the pointer |

A key you click, and a note you put or move to another key, plays, so you hear what you write.

## Clips per section

With a [song](../scripting.md#song-lane-gate-and-part) in the patch, the roll has a **clip for every section**: choose the section above the roll. A section with no clip of its own plays the loop. A clip can be **copied to other sections** (all of them, or one) and taken away again. The length of a clip is the length of its section.

## Chords

With a [harmony](harmony.md) in the patch, the chords of the section are drawn over the clip, and the tones of the chord are lit on the keys. Notes out of the chord have a dashed outline.

## Melody

The Melody panel (above the roll) makes melodies with `tools/melody`, for the **region** chosen in the ruler, else the **notes selected**, else the **whole clip**.

| Control | Meaning |
|---|---|
| Form | `free`; `AA'` (a period); `ABAB`; `AABA`; question and answer. A phrase is made once and repeated, varied or fitted to the chord |
| Contour | `free`, `arch`, `rise`, `fall`, `wave` over a phrase |
| Rhythm | `mixed`, `even`, `syncopated`, `dotted`, `triplet` |
| Phrase | 2, 4, 8 or 16 beats |
| Busy | how many notes |
| Rests | how often it rests |
| Smooth | steps against leaps (a Markov chain on the step sizes, with the direction kept and a leap filled back) |
| Chord tones | how much the notes on the strong beats and the long notes keep to the chord of the [harmony](harmony.md) |
| Scale | the scale of the harmony (its key and scale setting), or a key and mode of its own |
| From, to | the lowest and highest key |
| Keep to the scale | the notes you put and move snap to the scale |

**Generate** puts a melody in (undo takes it away); **Variations** makes six to hear (Hear) and choose from (Use); **Every section** makes a clip for every section of the song (asking first when there are clips), each over the chords of its own section and from the same seed, so they are made the same way (`GenerateSections` in Go; replaces the clips there are). The melody ends on a cadence (the root when closed). The seed changes at every press, so the same settings give a new melody each time.

For the notes selected (or the region, or all): **Invert** (in the scale, around the first note), **Reverse**, **Step up / down** (a step through the scale), **Humanize** (a little off the grid and the loudness) and **Thin out** (some of the short notes off the beat).

## Messages and Go

`Roll` (`noteroll.New`): `Notes()`, `Length()`, `Clip(section)`, `Clips()`, `SetClip`, `CopyClip`, `DropClip`; `Link` gives it the song position, the sections, the chords (`Link.Chords`) and the scale (`Link.Scale`); in a patcher this is wired by `rollLink`. The page sends messages `gen` (a melody request: from, to, count, seed, parameters) and `preview` (notes to hear); `Generate` and `Preview` do the same in Go.

Tests: `TestPageNoteRoll`, `TestPageNoteRollEditing`, `TestPageNoteRollClips`, `TestPageNoteRollMelody` in `patcher/page_test.go`; `noteroll_test.go`; `tools/melody/melody_test.go`.
