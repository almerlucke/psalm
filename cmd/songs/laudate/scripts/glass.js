// GLASS: bells of glass far above, on a clock of quarters. They answer: in the phrases where the lead rests (the score says which), and in the quiet parts, on the
// first and the third beat of the second bar of the phrase, a bell: a tone of the chord of the bar, by a markov chain over the chord (stay, a tone up, a tone down,
// a leap), in the register of the glass recordings. Slow, so the rest of the piece can be heard through it.
clock("1/4")
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const chain = genny.markov({
  stay: { up: 3, down: 3, leap: 1, stay: 1 },
  up:   { down: 4, up: 2, stay: 1, leap: 1 },
  down: { up: 4, down: 2, stay: 1, leap: 1 },
  leap: { down: 4, stay: 2, up: 1 },
}, "stay")
let i = 4
function trigger(ev) {
  if (!shared.known()) return
  const s = shared.s(), bar = shared.bar(), beat = ev.step % 4
  if (!s.gp[shared.q()] || bar % 2 !== 1 || (beat !== 0 && beat !== 2) || R() > 0.6 + 0.4 * s.glass) return
  const tones = shared.chordAt(bar).tones(72, 89)
  const m = chain.next()
  i = Math.max(0, Math.min(tones.length - 1, i + { stay: 0, up: 1, down: -1, leap: 3 }[m] * (m === "leap" && R() < 0.5 ? -1 : 1)))
  note(tones[i], 0.35 + 0.2 * R(), 4)
}
