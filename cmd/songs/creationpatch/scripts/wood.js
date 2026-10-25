// wooden tubes, cut at their hits, three euclidean cycles that run against the bar and one another: five in twelve, four in nine and three in seven (the five, four and
// three whose sum is the twelve of the bar), so the bar, the nine and the seven only meet again after a long while. Each has its own place in the file; a markov chain
// goes on to the next piece, skips a few, goes back one or plays the same again. The second joins with the creatures, the third on the seventh day
clock("1/16")
const play = shared.player("wood")
const R = genny.rand(0, 1)
let voices = null, last = -1
function section(name) {
  const n = shared.pieces.wood.length
  voices = [
    { cycle: genny.euclid(5, 12, 0), walk: shared.walker(n, 3), vel: 0.4, day: 3 },
    { cycle: genny.euclid(4, 9, 2), walk: shared.walker(n, Math.floor(n / 3)), vel: 0.34, day: 5 },
    { cycle: genny.euclid(3, 7, 1), walk: shared.walker(n, Math.floor(2 * n / 3)), vel: 0.3, day: 7 },
  ]
}
function trigger(ev) {
  if (!voices) return
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const at = song.bar * 12 + s
  if (!day || at === last) return
  last = at
  const on = shared.layers.wood(day, song.bar)
  voices.forEach(v => {
    const hit = v.cycle.next() // the cycles go round whether or not the tubes sound, so they meet where they meet
    if (!on || !hit || day.number < v.day) return
    shared.piece(play, note, ev, s, shared.pieces.wood[v.walk()], v.vel + 0.08 * R.next(), -30)
  })
}
