// LOW: the three lower voices of the choir (a chord for seven bars, then for three and a half), on the shaper's crystalSea: a pad
// that does not repeat, two slow waves (twelve seconds and seven) move its morph and a new value for every note moves a macro.
// Tuned: a slow swell, seven voices to a note spread wide.
let ready = false
function trigger(ev) {
  if (!ready) {
    tune({ preset: "crystalSea", ampAttack: 2800, ampDecay: 400, ampSustain: 1, ampRelease: 3600, unison: 5, spread: 22, uniStereo: 0.9, drift: 0.4, trim: -4, mono: 0 })
    ready = true
  }
  for (const n of shared.notesNow("low", ev)) note(n.pitch, 0.5 + 0.4 * n.vel, n.len)
}
