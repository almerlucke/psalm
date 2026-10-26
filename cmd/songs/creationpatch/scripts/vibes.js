// the vibraphone comps on the chords, a little tropical: sevenths and ninths (the third, fifth, seventh and ninth of the chord, or the seventh, ninth, third and
// fifth) and the sixth chord of the islands (the third, fifth, sixth and ninth), a markov chain stays in a voicing mostly and changes now and then. It strums the
// offbeats, the "and" of every beat (the second, sixth and tenth sixteenth of the bar), and adds the hits of a euclidean rhythm of five in twelve sixteenths, turned a
// little further every day, now and then; the first beat is left to the bass
clock("1/16")
const R = genny.rand(0, 1)
const play = shared.player("vibes")
const ROT = [0, 0, 0, 2, 5, 3, 7]
let pattern = null, voicing = null, last = -1
function section(name) {
  const day = shared.day(name)
  const g = genny.euclid(5, 12, ROT[day.number - 1])
  pattern = Array.from({ length: 12 }, (_, i) => (i === 2 || i === 6 || i === 10 ? 1 : g.next() && R.next() < 0.5))
  voicing = genny.markov({ A: { A: 4, B: 1, C: 2 }, B: { A: 1, B: 3, C: 1 }, C: { A: 2, B: 1, C: 4 } }, "C")
}
function trigger(ev) {
  if (!shared.jazz || !pattern) return
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const k = song.bar * 12 + s
  if (!day || k === last || !pattern[s] || !shared.layers.vibes(day, song.bar)) return
  last = k
  if (s === 0) return
  const br = shared.bar(day, song.bar)
  const q = shared.quality(br)
  if (!q) return
  const type = s === 2 || s === 6 || s === 10 ? voicing.next() : "A"
  play(note, ev, s, shared.voice(br, q, type, 58), s % 4 === 0 ? 0.42 : 0.34, 2.5)
}
