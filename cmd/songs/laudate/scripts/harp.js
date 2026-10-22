// HARP: arpeggios of the chord on sixteenth triplets, twenty-four steps to the bar, against the sixteen of the piano's rhythm: a euclidean cycle of
// 5 to 11 hits in 24, turned every phrase. A markov chain walks over the tones of the chord (mostly up, as a run, sometimes a step down or a skip) and the cycle says when.
// It plays in the phrases the score gives it, mostly where the piano rests.
clock("1/16T")
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const chain = genny.markov({
  up:    { up: 8, down: 2, skip: 1, again: 1 },
  down:  { down: 8, up: 3, skip: 1, again: 1 },
  skip:  { down: 3, up: 4, again: 1 },
  again: { up: 4, down: 2, skip: 1 },
}, "up")
let grid = null, i = 4
function trigger(ev) {
  if (!shared.known()) return
  const s = shared.s()
  if (!s.harp || !s.hp[shared.q()]) return
  const k = ev.step % 48                       // two bars: the cycle of the phrase
  if (k === 0 || !grid) { grid = genny.euclid(Math.round(4 + 7 * s.harp), 24, (shared.phrase() * 5) % 24); i = 2 }
  if (!grid.next()) return
  const tones = shared.chordAt(shared.bar()).tones(43, 79)
  const m = chain.next()
  i += { up: 1, down: -1, skip: R() < 0.5 ? 2 : -2, again: 0 }[m]
  if (i < 0 || i >= tones.length) i = Math.max(0, Math.min(tones.length - 1, i < 0 ? 1 : tones.length - 2))
  note(tones[i], 0.3 + 0.15 * R(), 6)
}
