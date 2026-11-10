// CHIME: a soft bell, where the piece first had wood. Before the rain a few single notes, far apart, low in the scale (the ark creaking, heard as bells); then a note now and then
// (2 hits in 15 eighths); for the animals two small threads (3 in 7 and 2 in 5), one low and one higher, each a step or two at a time, like feet on boards. Very soft; it keeps still
// during the thunder.
clock("1/8")
const rand = shared.rng(41)
const early = [18.5, 23, 24, 29.5, 34]
const slow = genny.euclid(2, 15, 2).take(15)
const feet = [
  { cycle: genny.euclid(3, 7, 0).take(7), len: 7, lo: 62, hi: 72, at: 3 },
  { cycle: genny.euclid(2, 5, 1).take(5), len: 5, lo: 72, hi: 84, at: 3 },
]
let last = -1
function trigger() {
  const k = shared.eighth(song)
  if (k === last) return
  last = k
  const t = k / 2
  const lv = shared.lv("chime", t) * shared.hush(t)
  if (lv <= 0.02) return
  if (t < 44) {
    if (early.includes(t)) {
      const tones = shared.tones(62, 74, t)
      note(tones[Math.floor(rand() * tones.length)], 0.22, 8)
    }
    return
  }
  if (t < 190) {
    if (slow[k % 15]) {
      const tones = shared.tones(64, 78, t)
      note(shared.pitch(tones[Math.floor(rand() * tones.length)], t), (0.14 + 0.08 * rand()) * lv, 8)
    }
    return
  }
  feet.forEach(f => {
    if (!f.cycle[k % f.len]) return
    const tones = shared.tones(f.lo, f.hi, t)
    f.at = Math.max(0, Math.min(tones.length - 1, f.at + Math.round((rand() - 0.5) * 3)))
    note(shared.pitch(tones[f.at], t), (0.13 + 0.08 * rand()) * lv, 6)
  })
}
