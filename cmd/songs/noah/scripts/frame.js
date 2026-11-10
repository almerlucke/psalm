// FRAME DRUM: a small, high skin, played with the fingers. In the rain two soft taps every eight bars, like something far off; for the animals a shuffle of 5 hits in 12 eighths.
clock("1/8")
const rand = shared.rng(53)
const shuffle = genny.euclid(5, 12, 0).take(12)
let last = -1
function trigger() {
  const k = shared.eighth(song)
  if (k === last) return
  last = k
  const t = k / 2
  const lv = shared.lv("frame", t) * shared.hush(t)
  if (lv <= 0.02) return
  if (t < 190) {
    if (k % 32 === 8) { note(64, 0.3 * lv, 1); note(64, 0.17 * lv, 1, 1.5) }
    return
  }
  if (shuffle[k % 12]) note(64 + (rand() < 0.3 ? 5 : 0), (0.15 + 0.12 * rand()) * lv, 1)
}
