// PEDAL: the low voice of the file, a note of twelve seconds and more on the shaper's throneDrone: four partials (the first, the
// third, the seventh and the twelfth) whose levels breathe against each other on two slow waves, so the colour of the held note
// changes through it. Tuned: it comes in over two seconds and goes in three, with a sine an octave under it.
function trigger(ev) {
  mod(BASE * shared.fadeIn(ev, 5) / 2) // in over the first five bars: the level of the strip
}
