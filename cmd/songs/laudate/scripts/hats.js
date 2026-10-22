// HATS: the high sounds of the percussion recording (shakers and hats, from second 94 on), cut at their hits, as a ride cymbal: the jazz pattern on eighth triplets,
// twelve steps to the bar: a stroke on each beat and the late one of the second and fourth (steps 0, 3, 5, 6, 9, 11, "spang-a-lang"). Each place has a piece of its own
// in a phrase, drawn by a markov chain that goes on to the next piece of the recording; the second phrase of a group says it again, the fourth changes one. The first
// four bars of the sanctuary are without it.
clock("1/8T")
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const pieces = shared.sounds.hats.filter(p => p.db > -45)
const spots = [0, 3, 5, 6, 9, 11]
const chain = genny.markov({ next: { next: 12, skip: 4, again: 3, back: 2 }, skip: { next: 10, again: 2 }, again: { next: 10, skip: 3 }, back: { next: 10, again: 2 } }, "next")
let pos = 0, mem = null, phrase = -1
const piece = () => {
  const a = chain.next()
  pos += a === "next" ? 1 : a === "skip" ? 3 : a === "back" ? -1 : 0
  pos = ((pos % pieces.length) + pieces.length) % pieces.length
  return pos
}
function trigger(ev) {
  if (!shared.known()) return
  const s = shared.s()
  if (!s.hat || (song.section === "Sanctuary" && song.bar < 4)) return
  const p = shared.phrase(), q = shared.q(), k = ev.step % 12
  if (p !== phrase) {
    phrase = p
    if (q === 0 || !mem) mem = spots.map(() => piece())
    else if (q === 3) mem = mem.map(x => R() < 0.25 ? piece() : x)
  }
  const i = spots.indexOf(k)
  if (i < 0) return
  if (k === 5 && R() > 0.55 + 0.4 * s.hat) return       // the late strokes come and go
  const d = pieces[mem[i]]
  note(d.key, Math.min(1, ((k % 3 === 0 ? 0.55 : 0.42) + 0.1 * R()) * shared.evenOut(d.db, -38)) * (0.6 + 0.4 * s.hat), Math.max(0.5, Math.min(d.sec / ev.seconds, 2)))
}
