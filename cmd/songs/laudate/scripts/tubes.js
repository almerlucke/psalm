// TUBES: wooden tubes, cut at their hits (a piece on each key), on eighth triplets, twelve steps to the bar. Three euclidean cycles that run against the
// bar and one another: 5 in 12, 4 in 9 and 3 in 7, so the bar, the nine and the seven only meet again after a long while. Each has its own place in the
// file: the markov chain goes on to the next piece, skips a few, goes back one or plays the same again.
clock("1/8T")
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const tubes = shared.sounds.tubes.filter(t => t.db > -45)
const chain = () => genny.markov({
  next:  { next: 16, again: 3, skip: 1, back: 1 },
  skip:  { next: 12, again: 3, back: 1 },
  again: { next: 12, skip: 2, again: 1 },
  back:  { next: 12, again: 3 },
}, "next")
const voices = [
  { cycle: genny.euclid(5, 12, 0), act: chain(), pos: 3,  vel: 0.5 },
  { cycle: genny.euclid(4, 9, 2),  act: chain(), pos: 20, vel: 0.42 },
  { cycle: genny.euclid(3, 7, 1),  act: chain(), pos: 40, vel: 0.36 },
]
function trigger(ev) {
  if (!shared.known()) return
  const t = shared.s().tubes
  if (shared.bar() % 4 === 3 && ev.step % 12 >= 6) return        // a breath: the second half of the last bar of a group
  voices.forEach((v, n) => {
    if (!v.cycle.next() || t < [0.01, 0.6, 0.95][n]) return   // the second cycle joins at half, the third at the most
    const a = v.act.next()
    if (a === "next") v.pos++
    else if (a === "skip") v.pos += 2 + Math.floor(R() * 4)
    else if (a === "back") v.pos = Math.max(0, v.pos - 1)
    v.pos %= tubes.length
    const p = tubes[v.pos]
    note(p.key, Math.min(1, v.vel * shared.evenOut(p.db, -36)) * (0.7 + 0.3 * t), Math.max(1, Math.min(p.sec / ev.seconds, 6)))
  })
}
