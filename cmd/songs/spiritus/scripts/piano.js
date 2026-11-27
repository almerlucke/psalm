// PIANO: the one instrument that is not granular, a modelled piano (the soft "satie" voicing) that plays along through the whole piece, and
// like the piano of the Noah piece it plays little: short phrases of four to seven notes, one note every beat and a half, with long rests between
// them, so the granular parts have the room. A phrase is one of four small shapes (degrees of the pentatonic scale of the key, going up and
// coming back), each told a little higher or lower than the last. The rests are longest in the Waiting and shortest in the Glory, where the phrases
// are a little fuller and sit an octave up with a low note under them. The pedal lifts where the chord changes. In the Amen a last phrase, then
// one soft chord that rings. The random numbers are seeded, so a render is always the same.
const rand = (() => { let a = 61; return () => { a = (a + 0x6D2B79F5) >>> 0; let t = a; t = Math.imul(t ^ (t >>> 15), t | 1); t ^= t + Math.imul(t ^ (t >>> 7), t | 61); return ((t ^ (t >>> 14)) >>> 0) / 4294967296 } })()
const shapes = [[0, 2, 1, 3, 2, 1], [2, 4, 3, 2, 0], [1, 2, 4, 3, 2, 1, 0], [3, 2, 3, 1, 0]]
const plan = {
  Waiting:    { vel: 0.48,  rest: [48, 32], from: 16, base: 5, gain: 13 },
  Wind:       { vel: 0.5, rest: [40, 32], from: 0,  base: 5, gain: 13 },
  Fire:       { vel: 0.55, rest: [32, 24], from: 0,  base: 6, gain: 13 },
  Indwelling: { vel: 0.55, rest: [32, 32], from: 0,  base: 5, gain: 13, low: true },
  Glory:      { vel: 0.6, rest: [24, 24], from: 0,  base: 7, gain: 11, low: true },
  Amen:       { vel: 0.45,  rest: [999, 0], from: 0,  base: 5, gain: 12, once: true },
}
let cfg = null, last = null, tick = 0, next = 0, n = 0
const queue = new Map() // tick -> notes [{ p, v, len }]
const put = (k, e) => queue.set(k, [...(queue.get(k) || []), e])
function section(name) { cfg = plan[name]; last = null; next = tick + (cfg ? cfg.from : 0); set({ level: cfg ? cfg.gain : 14 }) }
// the pentatonic scale of the key (D, and E in the Glory)
const keyTones = (lo, hi) => { const r = song.section === "Glory" ? 4 : 2, pcs = [0, 2, 4, 7, 9].map(x => (x + r) % 12), out = []; for (let p = lo; p <= hi; p++) if (pcs.includes(p % 12)) out.push(p); return out }
function phrase(c) {
  const shape = shapes[n++ % shapes.length]
  const peak = Math.max(...shape)
  const tones = keyTones(57, 86)
  const base = Math.max(3, Math.min(9, cfg.base + Math.round((rand() - 0.5) * 3)))
  shape.forEach((d, i) => {
    const at = tick + i * 6 + (i === shape.length - 1 ? 2 : 0)
    const p = tones[Math.min(tones.length - 1, base + d)]
    put(at, { p, v: cfg.vel * (0.9 + 0.3 * (d / peak)) * (0.95 + 0.1 * rand()), len: i === shape.length - 1 ? 40 : 14 })
  })
  if (cfg.low) put(tick, { p: c.bass(38), v: cfg.vel * 0.7, len: 40 })
  next = tick + shape.length * 6 + cfg.rest[0] + Math.floor(rand() * cfg.rest[1])
}
function trigger(ev) {
  tick++
  if (!cfg) return
  const step = ev.step % 16
  const c = shared.chordFor(song.section, song.bar)
  // the last chord of the piece: soft, wide, ringing
  if (song.section === "Amen" && song.bar === 10 && step === 0) {
    const b = c.bass(38)
    note([b, b + 7, ...c.close(55)], 0.28, 4 * 16)
    return
  }
  // the pedal: up where the chord changes, down a sixteenth later
  if (step === 0 && c !== last) { set({ sustain: 0 }); last = c }
  if (step === 1 && (song.bar === 0 || shared.chordFor(song.section, song.bar - 1) !== c)) set({ sustain: 1 })
  if (step === 0 && tick >= next && !(cfg.once && n > 0 && song.bar > 2) && !(song.section === "Amen" && song.bar >= 6)) phrase(c)
  ;(queue.get(tick) || []).forEach(e => note(e.p, Math.min(e.v, 0.8), e.len))
  queue.delete(tick)
}
