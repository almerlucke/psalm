// PEDAL: the low voice of the file, a note of twelve seconds and more on the shaper's throneDrone: four partials (the first, the
// third, the seventh and the twelfth) whose levels breathe against each other on two slow waves, so the colour of the held note
// changes through it. Tuned: it comes in over two seconds and goes in three, with a sine an octave under it.
let ready = false
function trigger(ev) {
  if (!ready) {
    set({ preset: "throneDrone", ampAttack: 2200, ampDecay: 400, ampSustain: 1, ampRelease: 3200, sub: 0.3, trim: -3, mono: 0 })
    ready = true
  }
  for (const n of shared.notesNow("pedal", ev)) note(n.pitch, 0.7, n.len)
}
