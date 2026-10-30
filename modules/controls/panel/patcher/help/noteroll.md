# noteroll

The **note editor**: a piano roll in the module's side panel (double-tap the module) that plays with the tempo of the patch. A loop of notes, edited by hand or made by the melody tools, sent to the synths it is cabled to. It is kept with the patch.

## Tutorial

1. Add a noteroll and a synth; cable its `notes` output to the synth. Double-tap the module to open the roll. It plays when the patch plays.
2. Draw notes. Choose the **draw** tool (D) and click on a key at a time; **select** (V) moves and resizes them (drag the right edge to resize, Alt-drag to copy); **erase** (E) takes them away. Clicking a key plays it, so you hear what you write.
3. Shape the loop: its **length** in beats is the loop; Ctrl+D repeats the selection; Ctrl+C / X / V copy, cut and paste; Ctrl+Z undoes. The velocity lane under the notes is bars to drag.
4. Tidy: **quantize** and **legato** buttons work on the notes selected, or on all.
5. With a [song](song.md) in the patch, the roll has a **clip for every section**: pick the section above the roll and draw its own notes. A section without a clip plays the loop. A clip can be copied to the other sections.
6. With a [harmony](harmony.md), the chords of the section are drawn over the clip and the tones of the chord are lit on the keys. Notes out of the chord have a dashed outline; **keep to the scale** snaps what you draw.
7. Let the machine start you off. The **Melody** panel makes a melody for the region chosen in the ruler (else the notes selected, else the whole clip): choose a **form**, **contour**, **rhythm** and phrase length, how busy it is, how many rests, steps against leaps, how much it keeps to the chord, a scale and a range, then **Generate**. **Variations** makes six to hear and choose from; **Every section** makes a clip for every section over its own chords.
8. Add more: **Counter-line** puts a second voice under the notes (thirds and sixths, never parallel fifths); **Groove** puts the feel of a groove on the sixteenths; **Invert**, **Reverse**, **Humanize** and **Thin out** work on the selection.

The noteroll has no text or settings to load: all of it is drawn in the window. For a melody that is made again while the song plays, use a [melodist](melodist.md) (its **Keep in the roll** button writes what it plays into a roll, to edit).
