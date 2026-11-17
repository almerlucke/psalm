// the piano comps on the chords of the bars, a jazz waltz: the second and third beats (boom, chick, chick) in voicings of the third, fifth, seventh and ninth, or the
// seventh, ninth, third and fifth; a markov chain stays in a voicing mostly and changes now and then. The "and" of the second beat and of the third (the offbeats,
// swung) come in on a euclidean five in twelve, turned a little every day, and the third beat sometimes anticipates the next bar: the Charleston of the waltz
clock("1/16")
const R = genny.rand(0, 1)
const play = shared.player("piano")
const ROT = [0, 0, 0, 2, 5, 3, 7]
let pattern = null, voicing = null, last = -1
function section(name) {
  const day = shared.day(name)
  const g = genny.euclid(5, 12, ROT[day.number - 1])
  pattern = Array.from({ length: 12 }, (_, i) => (i === 4 || i === 8 ? 1 : (i === 6 || i === 10) && g.next() && R.next() < 0.7))
  voicing = genny.markov({ A: { A: 4, B: 2 }, B: { A: 2, B: 3 } }, "A")
}
function trigger(ev) {
  if (!pattern) return
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const k = song.bar * 12 + s
  if (!day || k === last || !pattern[s] || !shared.layers.comp(day, song.bar)) return
  last = k
  let br = shared.bar(day, song.bar)
  if (s === 10 && song.bar + 1 < day.bars) br = shared.bar(day, song.bar + 1) // the anticipation: the chord of the next bar, a beat early
  const q = shared.quality(br)
  if (!q) return
  play(note, ev, s, shared.voice(br, q, voicing.next(), 52), s === 4 ? 0.46 : s === 8 ? 0.4 : 0.34, 2.5)
}
