// the low hand drums of the percussion recording, cut at their hits, a markov chain walking the pieces (mostly on to the next, now and then two on, one back or the same
// again) on a euclidean cycle of twelve: three, one a beat, for the beasts' tread on the evening of the sixth day and in the praise, five for the creatures, seven
// for the dance of mankind
clock("1/16")
const play = shared.player("hands")
const R = genny.rand(0, 1)
let pattern = null, walk = null, last = -1
function section(name) {
  const day = shared.day(name)
  walk = shared.walker(shared.pieces.hands.length, 0)
  const hits = day.number === 6 ? 7 : day.number === 5 ? 5 : 3
  const g = genny.euclid(hits, 12, 0)
  pattern = Array.from({ length: 12 }, () => g.next())
}
function trigger(ev) {
  if (!pattern) return
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const at = song.bar * 12 + s
  if (!day || at === last || !pattern[s] || !shared.layers.hands(day, song.bar)) return
  last = at
  const evening = day.number === 6 && !shared.morning(song.bar)
  const hits = evening ? (s % 4 === 0 ? [0.6] : []) : [0.42 + 0.08 * R.next()] // the beasts tread on the beat; otherwise the pattern says
  hits.forEach(v => shared.piece(play, note, ev, s, shared.pieces.hands[walk()], v * (s % 4 === 0 ? 1.15 : 0.85), -22))
}
