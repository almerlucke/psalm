// low hand drums, cut from a recording at their hits, five hits in twelve sixteenths turned by two (so they fall on the "and", like a conga), walked by a markov chain. They come with the
// second half of the knitting, play through the Wonder and the Light, and again in the second half of the Sand
clock("1/16")
const play = shared.player("hands")
const R = genny.rand(0, 1)
const on = { Knit: b => b >= 8, Wonder: () => true, Light: () => true, Sand: b => b >= 8 }
let cycle = null, walk = null, current = null, last = -1
function section(name) {
  current = on[name] ? shared.sec(name) : null
  last = -1
  cycle = genny.euclid(5, 12, 2).take(12)
  walk = shared.walker(shared.pieces.hands.length, 0)
}
function trigger(ev) {
  if (!current) return
  const k = shared.sixteenth(song)
  if (k === last) return
  last = k
  if (!cycle[k % 12] || !on[current.name](song.bar)) return
  shared.piece(play, note, ev, shared.pieces.hands[walk()], (0.42 + 0.08 * R.next()) * current.dyn / 0.7 * (k % 4 === 0 ? 1.15 : 0.85), -22)
}
