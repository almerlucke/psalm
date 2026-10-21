// HARP: arpeggios of the chord of the verse on a euclidean cycle of five in twelve steps, a shape for each round (up by steps, by thirds, an
// arch, a wave: the tones of the chord, a base that wanders), played softly. It rests in the verses of Egypt and sounds in the others, the
// more in the verses of remembering.
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const shapes = {
  run:    (k, h) => k,
  thirds: (k, h) => 2 * k,
  arch:   (k, h) => 2 * Math.min(k, h - 1 - k),
  wave:   (k, h) => [0, 2, 1, 3, 2, 4, 3, 5][k % 8],
}
const names = Object.keys(shapes)
let cyc = null, lastVerse = -1, k5 = 0, shape = "run", base = 0
function trigger(ev) {
  if (!shared.known()) return
  const v = shared.verse(), vb = shared.verseBar(), k = shared.step(ev)
  if (v !== lastVerse) {
    lastVerse = v
    const d = shared.layer.harp[v]
    cyc = d > 0 ? genny.euclid(5, 12, v % 4) : null
  }
  if (!cyc || !cyc.next() || vb === 3) return
  const tones = shared.chordOf(v).tones(55, 91)
  const n = tones.length, h = 5
  if (k5 % h === 0) {
    shape = names[Math.floor(R() * names.length)]
    base = Math.max(0, Math.min(Math.max(0, n - 9), base + Math.floor(R() * 3) - 1 + (R() < 0.3 ? 2 : 0) * (R() < 0.5 ? -1 : 1)))
  }
  const j = k5++ % h
  const i = Math.max(0, Math.min(n - 1, base + shapes[shape](j, h)))
  note(tones[i], (0.28 + 0.1 * (j === 0)) * shared.layer.harp[v] * shared.fade(), 12)
}
