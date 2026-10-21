// PERC: the congas, the darbuka, the shaker and the shofar, whole hits on the keys of a sampler. A cycle of twelve steps against the sixteen of
// the bar (a euclidean rhythm, the hits growing with the verse) walks over the congas by a markov chain; the refrain bar is plain, a low conga
// on the beat and two muted ones between; the kings march (a low conga on every beat), the shaker rides the lights; the darbuka is the
// smiting of Egypt, the shofar sounds at the calls (shared.shofar).
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const keys = { open: 36, open2: 37, low: 38, high: 40, muted: 42, mutedlow: 44 }
const walk = genny.markov({
  open:     { low: 12, high: 7, muted: 7, mutedlow: 3 },
  open2:    { low: 12, muted: 7, high: 3 },
  low:      { high: 12, muted: 7, open: 7, open2: 3, mutedlow: 3 },
  high:     { low: 12, muted: 7, open2: 7, open: 3 },
  muted:    { open: 12, low: 7, high: 7, open2: 3 },
  mutedlow: { muted: 12, open: 7, low: 7, high: 3 },
}, "low")
const kings = [16, 17, 18, 19]          // the verses of the great kings
const lights = [6, 7, 8]
let cyc = null, shaker = null, lastVerse = -1
function trigger(ev) {
  if (!shared.known()) return
  const v = shared.verse(), vb = shared.verseBar(), k = shared.step(ev), kb = k % 16
  if (v !== lastVerse) {
    lastVerse = v
    const d = shared.layer.perc[v]
    cyc = d > 0 ? genny.euclid(Math.max(2, Math.round(2 + d * 5)), 12, v % 4) : null
    shaker = lights.includes(v) ? genny.euclid(7, 16, 0) : null
  }
  const fade = shared.fade()
  // the shofar
  if (shared.shofar[v] === vb && kb === 0) note(120, 0.8, 40)
  // the darbuka: Egypt smitten, Pharaoh overthrown
  if ((v === 9 || v === 14) && k === 0) note(keys.low + 8, 0.9, 6)
  if (shaker && shaker.next() && vb < 3) note(48, 0.35, 2)
  if (!cyc) return
  if (vb === 3) {            // the refrain: plain
    if (kb === 0 || kb === 8) note(keys.low, 0.7 * fade, 4)
    if (kb === 6 || kb === 14) note(keys.muted, 0.4 * fade, 2)
    return
  }
  if (kings.includes(v) && kb % 4 === 0) { note(keys.low, 0.6, 4); return }
  if (!cyc.next()) return
  note(keys[walk.next()], (kb % 4 === 0 ? 0.7 : 0.5) + 0.2 * R(), 3)
}
