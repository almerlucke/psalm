// the vibraphone comps on the chords: sevenths and ninths (the third, fifth, seventh and ninth of the chord, or the seventh, ninth, third and fifth: a markov chain
// stays in a voicing mostly and changes now and then), on the hits of a euclidean rhythm of five in twelve sixteenths, turned a little further every day, with
// a hit that falls on the first beat now and then left out (the bass has it)
clock("1/16")
const R = genny.rand(0, 1)
const play = shared.player("vibes")
const ROT = [0, 0, 0, 2, 5, 3, 7]
let pattern = null, voicing = null, last = -1
function section(name) {
  const day = shared.day(name)
  const g = genny.euclid(5, 12, ROT[day.number - 1])
  pattern = Array.from({ length: 12 }, () => g.next())
  voicing = genny.markov({ A: { A: 3, B: 1 }, B: { A: 1, B: 3 } }, "A")
}
function trigger(ev) {
  if (!shared.jazz || !pattern) return
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const k = song.bar * 12 + s
  if (!day || k === last || !pattern[s] || !shared.layers.vibes(day, song.bar)) return
  last = k
  if (s === 0 && R.next() < 0.6) return
  const br = shared.bar(day, song.bar)
  const q = shared.quality(br)
  if (!q) return
  const type = s === 0 || s === 4 || s === 8 ? voicing.next() : "A"
  play(note, ev, s, shared.voice(br, q, type, 58), s % 4 === 0 ? 0.42 : 0.34, 2.5)
}
