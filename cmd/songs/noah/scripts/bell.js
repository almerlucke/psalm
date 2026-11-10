// BELL: a handbell, soft, in two things. A fifth (D and A; in the storm and the voice C and G) struck together now and then, a bell for every note the others stand on, one every
// twelve bars or so; and a thin thread of single high notes (2 hits in 11 eighths) that answers the bells of the recording from far off, wandering through the scale. They
// keep still for the thunder and fade out with the voice.
clock("1/8")
const rand = shared.rng(71)
const thread = genny.euclid(2, 11, 4).take(11)
let last = -1, at = 6
function trigger() {
  const k = shared.eighth(song)
  if (k === last) return
  last = k
  const t = k / 2
  const lv = shared.lv("bell", t) * shared.hush(t)
  if (lv <= 0.02) return
  if (k % 48 === 4) {
    const low = t >= 324 ? [60, 67] : [62, 69]
    note(shared.pitch(low[0], t), 0.4 * lv, 10)
    note(shared.pitch(low[1], t), 0.32 * lv, 10, 1)
  }
  if (t >= 76 && t < 300 && thread[k % 11]) {
    const tones = shared.tones(72, 93, t)
    at = Math.max(0, Math.min(tones.length - 1, at + Math.round((rand() - 0.5) * 4)))
    note(shared.pitch(tones[at], t), (0.26 + 0.12 * rand()) * lv, 8)
  }
}
