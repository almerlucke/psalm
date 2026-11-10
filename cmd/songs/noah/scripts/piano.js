// PIANO: the melody that goes through the whole piece, gently: short phrases of four to seven notes, a note every second and a half, with long rests between them. The
// phrases are made once, at the start, from four small shapes (degrees of the scale, going up and coming back) that are told in turn, each a little higher or lower than
// the one before, in the notes of the scale of the time they are played in (so the same shape is in D with the bells, without a third with the animals, in C in the storm).
// They keep out of the thunder. While the voice sings (336 to 372 s) the piano goes under it: lower than the voice (C3 to G4, the voice is around C4 to C5), softer, with longer rests; the last phrase comes after it and ends on the C of the voice.
clock("1/8")
const rand = shared.rng(61)
const shapes = [[0, 2, 1, 3, 2, 1], [2, 4, 3, 2, 0], [1, 2, 4, 3, 2, 1, 0], [3, 2, 3, 1, 0]]
const events = new Map() // eighth -> notes [{ p, v, len }]
let n = 0, base = 6, t = 14
const put = (k, e) => events.set(k, [...(events.get(k) || []), e])
while (t < 366) {
  // a phrase must not run into the thunder: wait for the end of it
  const shape = shapes[n++ % shapes.length]
  const long = shape.length * 1.5 + 2
  if (t + long > 371) break
  let clear = true // the whole phrase, not only its two ends, must be out of the thunder
  for (let x = t; x <= t + long; x += 0.5) if (shared.hush(x) < 1) clear = false
  if (!clear) { t += 3; continue }
  base = Math.max(4, Math.min(9, base + Math.round((rand() - 0.5) * 3)))
  const peak = Math.max(...shape)
  shape.forEach((d, i) => {
    const at = t + i * 1.5 + (i === shape.length - 1 ? 0.5 : 0)
    const voice = at >= 336 // under the voice: low and soft
    const tones = shared.tones(voice ? 48 : 57, voice ? 67 : 86, at)
    const p = tones[Math.min(tones.length - 1, voice ? 1 + d + (n % 2) : base + d)]
    const v = Math.min(1, (0.26 + 0.1 * (d / peak) + 0.05 * rand()) * 1.35) * (t < 40 ? 0.7 : voice ? 0.6 : 1)
    put(Math.round(at * 2), { p, v, len: i === shape.length - 1 ? 10 : 5 })
  })
  // more air at the start and in the storm, less in the middle
  t += long + (t < 70 ? 14 : t >= 336 ? 8 + 4 * rand() : t > 290 ? 12 : 6 + 8 * rand())
}
// the last phrase, after the voice: down the scale of C to the C itself, and a low C under it
;[[374, 79], [375.5, 77], [377, 74], [379, 72]].forEach(([at, p], i, a) => put(Math.round(at * 2), { p, v: 0.36 - 0.03 * i, len: i === a.length - 1 ? 14 : 5 }))
put(Math.round(379 * 2), { p: 60, v: 0.26, len: 14 })
let last = -1
function trigger() {
  const k = shared.eighth(song)
  if (k === last) return
  last = k
  const tt = k / 2
  ;(events.get(k) || []).forEach(e => note(shared.pitch(e.p, tt), e.v, e.len))
}
