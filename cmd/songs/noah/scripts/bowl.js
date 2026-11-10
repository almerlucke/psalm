// BOWL: a high bowl, the sound the piece is meant to have more of. Struck to mark a change of the recording: the first bell (72 s), the animals (188), the quiet before the storm (296), and once at the end
// on the C of the voice (372 s), when everything else has left.
clock("1/8")
const rand = shared.rng(83)
const cues = [
  { t: 72, p: 74, v: 0.34 },
  { t: 188, p: 69, v: 0.32 },
  { t: 296, p: 67, v: 0.26 },
  { t: 372, p: 72, v: 0.3 },
]
// and between them a bowl every twenty seconds or so on a note of the scale of the time, higher or lower, not in the thunder
for (let t = 28; t < 320; t += 16 + Math.round(rand() * 10)) {
  if (shared.hush(t) < 1 || cues.some(c => Math.abs(c.t - t) < 10)) continue
  const tones = shared.tones(64, 86, t)
  cues.push({ t, p: tones[Math.floor(rand() * tones.length)], v: 0.2 + 0.1 * rand() })
}
let last = -1
function trigger() {
  const k = shared.eighth(song)
  if (k === last) return
  last = k
  cues.filter(c => Math.round(c.t * 2) === k).forEach(c => note(shared.pitch(c.p, c.t), c.v, 12))
}
