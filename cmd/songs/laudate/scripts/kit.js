// KIT: the loud deep drums of the percussion recording (seconds 42 to 94), cut at their hits, on eighth triplets (twelve steps to the bar), a few hits in a bar: the ghost notes and the accents under the other parts.
// Like the lead it goes in phrases of two bars,
// four to a group, as the score says (kp: in which of the four it plays): the first phrase makes a rhythm, the second says it again, the third is another, the
// fourth is the first again with some changed. A euclidean cycle says where something may happen (the second bar of a phrase is the cycle turned five steps), a
// markov chain walks the pieces of the recording: mostly on to the next, now and then two on, one back, or to a quiet piece far off.
clock("1/8T")
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const pieces = shared.sounds.kit.filter(p => p.db > -34)
const chain = genny.markov({
  next: { next: 14, skip: 4, back: 2, jump: 1, again: 3 },
  skip: { next: 12, skip: 2, back: 2, again: 2 },
  back: { next: 12, again: 3, skip: 2 },
  jump: { next: 12, again: 2 },
  again: { next: 12, skip: 3, back: 2 },
}, "next")
let pos = 0
const piece = () => {
  const a = chain.next()
  if (a === "next") pos++
  else if (a === "skip") pos += 2
  else if (a === "back") pos = Math.max(0, pos - 1)
  else if (a === "jump") pos = Math.floor(R() * pieces.length)
  pos %= pieces.length
  return pos
}
let memA = null, memB = null, phrase = -1, cur = null
function make(s, turn) {
  const out = []
  for (let b = 0; b < 2; b++) {
    const grid = genny.euclid(Math.round(2 + 4 * s.kit), 12, (turn + 5 * b) % 12)
    for (let i = 0; i < 12; i++) out.push(grid.next() ? piece() : -1)
  }
  return out
}
function trigger(ev) {
  if (!shared.known()) return
  const s = shared.s()
  if (!s.kit) return
  const p = shared.phrase(), q = shared.q(), k = ev.step % 24
  if (p !== phrase) {
    phrase = p
    if (q === 0 || !memA) { memA = make(s, (p * 5) % 12); cur = memA }
    else if (q === 1) cur = memA
    else if (q === 2) { memB = make(s, (p * 5 + 3) % 12); cur = memB }
    else cur = memA.map(x => (x >= 0 && R() < 0.25) ? piece() : x)
  }
  if (!s.kp[q] || !cur || cur[k] < 0) return
  const d = pieces[cur[k]]
  note(d.key, Math.min(1, ((k % 3 === 0 ? 0.62 : 0.48) + 0.1 * R()) * shared.evenOut(d.db, -22)) * (0.6 + 0.4 * s.kit), Math.max(1, Math.min(d.sec / ev.seconds, 4)))
}
