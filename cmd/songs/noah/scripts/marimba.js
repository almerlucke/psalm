// MARIMBA: a soft mallet, low and slow. While the bells ring a note every four or five bars (2 hits in 17 eighths) in the middle of the scale; for the animals
// pairs of notes, a second apart, like two creatures walking (3 hits in 12 eighths).
clock("1/8")
const rand = shared.rng(23)
const slow = genny.euclid(2, 17, 0).take(17)
const walk = genny.euclid(3, 12, 1).take(12)
let last = -1, at = 4
function trigger() {
  const k = shared.eighth(song)
  if (k === last) return
  last = k
  const t = k / 2
  const lv = shared.lv("marimba", t) * shared.hush(t)
  if (lv <= 0.02) return
  const animals = t >= 190
  if (!(animals ? walk[k % 12] : slow[k % 17])) return
  const tones = shared.tones(52, 70, t)
  at = Math.max(0, Math.min(tones.length - 1, at + Math.round((rand() - 0.5) * 4)))
  const v = (0.3 + 0.12 * rand()) * lv
  note(shared.pitch(tones[at], t), v, 4)
  if (animals && rand() < 0.7) {
    const b = Math.min(tones.length - 1, at + 1)
    note(shared.pitch(tones[b], t), v * 0.7, 4, 1.5)
  }
}
