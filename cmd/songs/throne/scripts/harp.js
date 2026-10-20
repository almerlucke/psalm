// HARP: the harps of heaven (Revelation 5:8, every one of the elders had harps), a flowing arpeggio of the chord, on a euclidean rhythm of the twelve-step cycle (five, seven of twelve) against the sixteen steps of the bar, so the water
// comes round again every three bars. It comes in under the throne, with the man and the eagle, and is the water under the elders
// and the holy. Every round of the rhythm is a phrase with its own shape (shared.shapes: an arch, a wave, a low note and a climbing line, broken
// wide...), chosen by a markov chain, on a base that wanders along the chord, the first note leaning and the odd one left out. Soft: the harp preset of the shaper, on the echo and the shimmer.
const parts = {
  Throne:    { trim: -9, rhythm: [5, 12], lo: 55, hi: 91, vel: 0.3, on: bar => bar >= 8 },
  Creatures: { trim: -10, rhythm: [3, 12], lo: 60, hi: 93, vel: 0.28, on: bar => bar >= 12 },
  Elders:    { trim: -9, rhythm: [5, 12], lo: 55, hi: 93, vel: 0.32, on: bar => bar >= 2 && bar < 18 },
  Holy:      { trim: -8, rhythm: [5, 12], lo: 55, hi: 96, vel: 0.34, on: bar => true },
  Worthy:    { trim: -9, rhythm: [3, 12], lo: 57, hi: 93, vel: 0.26, on: bar => bar < 12 },
}
const shapes = genny.markov(shared.shapeTable, "arch")
const rnd = genny.rand(0, 1)
let part = null, hits = null, k = 0, shape = "arch", base = 0, h = 5
function section(name) {
  part = parts[name]
  hits = part ? genny.euclid(part.rhythm[0], part.rhythm[1], 0) : null
  k = 0
  h = part ? part.rhythm[0] : 5
  if (part) set({ preset: "harp", trim: part.trim })
}

function trigger(ev) {
  if (!part) return
  const hit = hits.next()
  const bar = song.bar
  if (!hit || !part.on(bar)) return
  const tones = shared.chordFor(song.section, bar).tones(part.lo, part.hi)
  const n = tones.length
  if (n === 0) return
  // a phrase is one round of the euclidean rhythm: it takes the next shape, and its base wanders along the chord
  if (k % h === 0) {
    shape = shapes.next()
    const room = Math.max(0, n - 9)
    base = Math.max(0, Math.min(room, base + (rnd.next() < 0.5 ? -1 : 1) * (rnd.next() < 0.4 ? 3 : 1)))
  }
  const j = k++ % h
  if (j > 0 && rnd.next() < 0.1) return                  // a breath: the odd note left out
  const i = Math.max(0, Math.min(n - 1, base + shared.shapes[shape](j, h)))
  // the first note of a phrase leans, the rest follow
  note(tones[i], part.vel * shared.swell(song.section, song.progress) * (j === 0 ? 1.2 : 0.85 + 0.15 * j / h), 12)
}
