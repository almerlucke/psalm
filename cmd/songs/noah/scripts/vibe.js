// VIBE: a soft bell bar, where the piece first had a frame drum. In the rain two soft notes every eight bars, like something far off; for the animals a shuffle of 5 hits in 12 eighths,
// a step up or down at a time through the middle of the scale.
clock("1/8")
const rand = shared.rng(53)
const shuffle = genny.euclid(5, 12, 0).take(12)
let last = -1, at = 4
function trigger() {
  const k = shared.eighth(song)
  if (k === last) return
  last = k
  const t = k / 2
  const lv = shared.lv("vibe", t) * shared.hush(t)
  if (lv <= 0.02) return
  const tones = shared.tones(60, 76, t)
  if (t < 190) {
    if (k % 32 === 8) {
      note(shared.pitch(tones[2], t), 0.24 * lv, 6)
      note(shared.pitch(tones[1], t), 0.15 * lv, 6, 1.5)
    }
    return
  }
  if (!shuffle[k % 12]) return
  at = Math.max(0, Math.min(tones.length - 1, at + Math.round((rand() - 0.5) * 3)))
  note(shared.pitch(tones[at], t), (0.13 + 0.08 * rand()) * lv, 6)
}
