// KALIMBA: thin threads of single notes, of lengths that do not meet (3 hits in 11 eighths, 3 in 13, 2 in 9), each wandering a step or two through the notes of the
// scale, so they cross and come apart. One thread in the rain, high like the drops; all three while the bells ring; two for the animals; one in the storm. Now and then
// a note answers itself a step lower.
clock("1/8")
const rand = shared.rng(11)
const threads = [
  { len: 11, hits: 3, rot: 0, lo: 69, hi: 86 },
  { len: 13, hits: 3, rot: 5, lo: 62, hi: 81 },
  { len: 9, hits: 2, rot: 3, lo: 64, hi: 79 },
].map(t => ({ ...t, cycle: genny.euclid(t.hits, t.len, t.rot).take(t.len), at: 3 }))
let last = -1
function trigger() {
  const k = shared.eighth(song)
  if (k === last) return
  last = k
  const t = k / 2
  const lv = shared.lv("kalimba", t) * shared.hush(t)
  const count = t < 72 ? 1 : t < 190 ? 3 : t < 290 ? 2 : 1
  threads.forEach((th, i) => {
    const hit = th.cycle[k % th.len] // the cycles go round whether or not they sound
    if (!hit || i >= count || lv <= 0.02 || (t < 72 && rand() < 0.4)) return
    const tones = shared.tones(th.lo, th.hi, t)
    th.at = Math.max(0, Math.min(tones.length - 1, th.at + Math.round((rand() - 0.5) * 4)))
    const v = (0.22 + 0.12 * rand()) * lv
    note(shared.pitch(tones[th.at], t), v, 3)
    if (rand() < 0.2 && th.at > 0) note(shared.pitch(tones[th.at - 1], t), v * 0.6, 3, 3)
  })
}
