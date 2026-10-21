// COLOR: the wooden tubes, cut at their hits, and the birds. The tubes wander through their file by a markov chain: on to the next
// piece, a skip over a few, back one, the same one again; the hits are a euclidean cycle of twelve steps against the sixteen of the bar. The
// birds (in the verses of the earth and the waters, the lights, the land and the food of all flesh) sing their pieces in the order of the file.
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const tubes = shared.slices.tubes, birds = shared.slices.birds
const act = genny.markov({
  next:  { next: 12, again: 7, skip: 3, back: 3 },
  skip:  { next: 12, again: 7, back: 3 },
  again: { next: 12, skip: 7, again: 3 },
  back:  { next: 12, again: 7 },
}, "next")
let pos = 0, bird = 0, cyc = null, song_ = null, lastVerse = -1
function trigger(ev) {
  if (!shared.known()) return
  const v = shared.verse(), vb = shared.verseBar(), k = shared.step(ev), kb = k % 16
  if (v !== lastVerse) {
    lastVerse = v
    const d = shared.layer.tubes[v]
    cyc = d > 0 ? genny.euclid(Math.max(1, Math.round(d * 6)), 12, v % 5) : null
    song_ = shared.birds.includes(v) ? genny.euclid(2, 12, v % 3) : null
  }
  const fade = shared.fade()
  if (cyc && cyc.next() && vb < 3) {
    const a = act.next()
    if (a === "next") pos++
    else if (a === "skip") pos += 2 + Math.floor(R() * 4)
    else if (a === "back") pos = Math.max(0, pos - 1)
    pos %= tubes.length
    for (let guard = 0; guard < tubes.length && !shared.loud(tubes[pos]); guard++) pos = (pos + 1) % tubes.length      // not a silent piece
    const p = tubes[pos]
    note(p.key, Math.min(1, (0.4 + 0.2 * R()) * shared.evenOut(p, -36)) * fade, Math.min(p.len, 12))
  }
  if (song_ && song_.next() && vb < 3) {
    const p = birds[bird++ % birds.length]
    note(p.key, 0.5 * fade, Math.min(p.len, 16))
  }
}
