// WOOD: the ark creaking. A few knocks before the rain, far apart; then two knocks that come round every 15 eighths; for the animals two small threads (3 in 7 and 2 in 5)
// on two pitches, a step apart, like feet on boards. Very soft; it keeps still during the thunder.
clock("1/8")
const rand = shared.rng(41)
const early = [18.5, 23, 24, 29.5, 34]
const slow = genny.euclid(2, 15, 2).take(15)
const feet = [
  { cycle: genny.euclid(3, 7, 0).take(7), len: 7, p: 74 },
  { cycle: genny.euclid(2, 5, 1).take(5), len: 5, p: 79 },
]
let last = -1
function trigger() {
  const k = shared.eighth(song)
  if (k === last) return
  last = k
  const t = k / 2
  const lv = shared.lv("wood", t) * shared.hush(t)
  if (lv <= 0.02) return
  if (t < 44) {
    if (early.includes(t)) note(70 + Math.floor(rand() * 4) * 2, 0.3, 1)
    return
  }
  if (t < 190) {
    if (slow[k % 15]) note(72 + Math.floor(rand() * 3) * 2, (0.2 + 0.1 * rand()) * lv, 1)
    return
  }
  feet.forEach(f => {
    if (f.cycle[k % f.len]) note(f.p, (0.16 + 0.1 * rand()) * lv, 1)
  })
}
