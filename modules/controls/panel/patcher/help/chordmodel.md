# chordmodel

A model of how **chord progressions** go: a matrix of how likely each chord is to follow another, a progression drawn from it, and a button that writes it into the [harmony](harmony.md). Use it when you want chords that sound like a style but are not the same every time. The [song](song.md) has one inside it.

## Tutorial

1. Add a chord model and a harmony (or use a song). Cable the model's `notes` output to an instrument if you want to hear a progression on its own.
2. The window shows the progression as cards, a card a chord with its symbol large and its numeral small. **New** draws a whole progression; **Hear all** plays it; **Bars** is its length.
3. Choose a style with a preset (pop, minor, jazz, rock). The matrix behind it is how likely each chord (`I ii iii IV V vi vii°`) is to follow each other chord. Drag a cell to change it, right-click for never.
4. **Lock** the chords you like (🔒), then **Redraw the unlocked**: only the others are drawn again, each fitting the chords on both sides.
5. Press **Write** to put the progression in the harmony, as chords or as numerals. The parts and melodists follow at once.
6. Teach it your own style: type progressions in numerals and it learns what follows what.
7. Switch on **Live** and the chords change while the song plays: every time the song leaves a section, the chords that are not locked are drawn again, so the second verse is not the first. A chord that is locked stays.

The chord model has no text to load, because it is all drawn in its window; a patch keeps its matrix with it.

## With a song

In the sections grid, **Draw chords** puts a progression from the model in a section; a switch makes it draw new chords every section.
