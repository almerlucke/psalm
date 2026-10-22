// PERC: the hand percussion (66-bpm-uniq-percussions), cut at its hits, 62 pieces on the keys from 36, in swung sixteenths. Like the lead it goes in phrases of two bars,
// four to a group, as the score says (pp: in which of the four it plays): the first phrase makes a rhythm, the second says it again, the third is another, the
// fourth is the first again with some changed. A euclidean cycle says where something may happen (the second bar of a phrase is the cycle turned five steps), a
// markov chain walks the pieces of the recording: mostly on to the next, now and then two on, one back, or to a quiet piece far off.
clock("1/16", 0.58)
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const pieces = shared.sounds.perc.filter(p => p.db > -36)
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
    const grid = genny.euclid(Math.round(3 + 8 * s.perc), 16, (turn + 5 * b) % 16)
    for (let i = 0; i < 16; i++) out.push(grid.next() ? piece() : -1)
  }
  return out
}
function trigger(ev) {
  if (!shared.known()) return
  const s = shared.s()
  if (!s.perc) return
  const p = shared.phrase(), q = shared.q(), k = ev.step % 32
  if (p !== phrase) {
    phrase = p
    if (q === 0 || !memA) { memA = make(s, (p * 5) % 16); cur = memA }
    else if (q === 1) cur = memA
    else if (q === 2) { memB = make(s, (p * 5 + 3) % 16); cur = memB }
    else cur = memA.map(x => (x >= 0 && R() < 0.25) ? piece() : x)
  }
  if (!s.pp[q] || !cur || cur[k] < 0) return
  const d = pieces[cur[k]]
  note(d.key, Math.min(1, ((k % 4 === 0 ? 0.62 : 0.5) + 0.1 * R()) * shared.evenOut(d.db, -27)) * (0.6 + 0.4 * s.perc), Math.max(1, Math.min(d.sec / ev.seconds, 4)))
}
