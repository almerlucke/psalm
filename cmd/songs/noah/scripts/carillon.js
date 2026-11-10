// CARILLON: a soft set of small bells, where the piece first had a plucked string. While the bells ring, an open fifth (D and A) strummed softly every sixteen bars; for the animals an ostinato that does not repeat with the
// bar: 4 hits in 7 eighths, walking up and down the scale (so the pattern comes back to the same place of the bar every seven bars).
clock("1/8")
const rand = shared.rng(31)
const cycle = genny.euclid(4, 7, 0).take(7)
const order = [0, 2, 1, 3, 2, 4, 3, 1]
let last = -1, n = 0
function trigger() {
  const k = shared.eighth(song)
  if (k === last) return
  last = k
  const t = k / 2
  const lv = shared.lv("carillon", t) * shared.hush(t)
  if (lv <= 0.02) return
  if (t < 190) {
    if (k % 64 === 0) { note(62, 0.3 * lv, 8); note(69, 0.26 * lv, 8, 0.5) }
    return
  }
  if (!cycle[k % 7]) return
  const tones = shared.tones(62, 79, t)
  const p = tones[Math.min(tones.length - 1, order[n++ % order.length])]
  note(shared.pitch(p, t), (0.22 + 0.08 * rand()) * lv, 3)
}
