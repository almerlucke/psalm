// HIGH: the three higher voices of the choir, on the wavetable synth's evolvingPad. Tuned: the position of the table sways on a
// slow wave, five detuned copies a note, and the filter opens section by section from a dark, closed sound in the Dawn to an
// open one in the Throne, so the piece grows brighter as it climbs.
const cutoff = [1700, 2100, 2700, 3500, 4600, 6200, 8200]
let ready = false
function section(name, i) {
  if (ready) tune({ cutoff: cutoff[i] })
}
function trigger(ev) {
  mod(BASE * shared.fadeIn(ev, 5) / 2) // in over the first five bars: the level of the strip
  if (!ready) {
    tune({ preset: "evolvingPad", ampAttack: 3500, ampDecay: 400, ampSustain: 1, ampRelease: 4500, position: 0.35,
      positionLFORate: 0.08, positionLFODepth: 0.35, unison: 5, detune: 16, spread: 0.9, resonance: 0.15, filterEnvAmount: 0.4,
      cutoff: cutoff[Math.max(0, song.index)], level: 0.16 })
    ready = true
  }
  for (const n of shared.notesNow("high", ev)) note(n.pitch, 0.5 + 0.4 * n.vel, n.len)
}
