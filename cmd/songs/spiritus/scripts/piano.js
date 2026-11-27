// PIANO: the one instrument that is not granular, a modelled grand that plays along through the whole piece: gentle, glad, and with a
// lilt. It plays like a player, not like a pattern: every bar a new phrase is made (the same bar is never the same twice), from
//  - a GROOVE in the left hand: the root on the one, the fifth and the octave placed off the beat (the "and" of two, the "and" of three),
//    and a soft stab of the guide tones (the third, the seventh, the ninth) where the beat leans, a jazz comp;
//  - a RIGHT HAND LINE: a rhythm picked from a few cells (even eighths, a 3-3-2 syncopation, anticipations that land before the beat) and notes
//    that walk the scale of the key in steps, turn and rise and fall over four bars, and land on a tone of the chord on the strong beats,
//    often with a grace note a half step under it. The line swells and ebbs with the bars of a phrase and with the section;
//  - the PEDAL, which lifts at each change of chord.
// In the Waiting it only touches a few keys, in the Fire it dances, in the Indwelling it is tender, in the Glory it plays in octaves and leans
// on the beat, and in the Amen it fades into a last chord. The random numbers are seeded by the bar, so a render is always the same.
const C = 60 - pitch("C4")
// a small seeded random generator (mulberry32)
function rng(seed) {
  let a = seed >>> 0
  return () => { a = (a + 0x6D2B79F5) >>> 0; let t = a; t = Math.imul(t ^ (t >>> 15), t | 1); t ^= t + Math.imul(t ^ (t >>> 7), t | 61); return ((t ^ (t >>> 14)) >>> 0) / 4294967296 }
}
// the rhythm cells: the sixteenths of a bar on which a note of the line starts
const cells = {
  even:   [0, 2, 4, 6, 8, 10, 12, 14],
  clave:  [0, 3, 6, 8, 11, 14],
  lilt:   [0, 4, 6, 10, 12],
  ahead:  [2, 4, 7, 10, 12, 15],
  broken: [0, 2, 3, 6, 8, 10, 11, 14],
  sparse: [0, 6, 8, 12],
  one:    [0, 8],
}
const plan = {
  Waiting:    { cells: ["one", "sparse"], vel: 0.46, on: bar => bar >= 1, grid: null, range: [64, 81], grace: 0, stab: [] },
  Wind:       { cells: ["sparse", "lilt"], vel: 0.46, on: bar => bar >= 1, grid: { bass: [0, 10], stab: [] }, range: [62, 83], grace: 0.1 },
  Fire:       { cells: ["lilt", "clave", "even", "ahead"], vel: 0.54, grid: { bass: [0, 6, 10], stab: [4, 14] }, range: [62, 86], grace: 0.3 },
  Indwelling: { cells: ["lilt", "sparse", "clave"], vel: 0.52, grid: { bass: [0, 10], stab: [6] }, range: [62, 83], grace: 0.2 },
  Glory:      { cells: ["even", "clave", "broken", "ahead"], vel: 0.68, grid: { bass: [0, 6, 8, 10], stab: [4, 12, 14] }, range: [64, 86], grace: 0.35, octave: true },
  Amen:       { cells: ["sparse", "one"], vel: 0.42, on: bar => bar < 10, grid: { bass: [0], stab: [] }, range: [62, 81], grace: 0, fade: true },
}
const arc = [0.85, 1, 1.12, 0.92] // the swell of the four bars of a phrase
const gain = { Waiting: 22, Wind: 22, Fire: 20, Indwelling: 29, Glory: 13, Amen: 24 }
let cfg = null, last = null, cur = 72, bar = null
function section(name) { cfg = plan[name]; last = null; cur = 72; bar = null; set({ level: gain[name] || 16 }) }

// the scale of the key (D major, E major in the Glory) and the tones of it in a range
const inKey = p => [0, 2, 4, 5, 7, 9, 11].includes((((p - (song.section === "Glory" ? 4 : 2)) % 12) + 12) % 12)
const scaleNear = (p, dir, n) => { // n scale steps from p in the direction dir
  let q = p
  while (n > 0) { q += dir; if (inKey(q)) n-- }
  return q
}

// make the notes of one bar: [step] -> {p, v, len}
function makeBar(c, k) {
  const r = rng(song.bar * 7919 + k * 104729 + (song.section.length * 31))
  const events = {}
  const lo = cfg.range[0], hi = cfg.range[1]
  const cell = cells[cfg.cells[Math.floor(r() * cfg.cells.length)]]
  const phrase = song.bar % 4
  const rising = phrase < 2
  let dir = rising ? 1 : -1
  const tones = c.tones(lo, hi)
  const vscale = arc[phrase] * (cfg.fade ? 1 - song.progress * 0.5 : 0.92 + 0.16 * song.progress)
  cell.forEach((step, i) => {
    const strong = step % 8 === 0
    const gap = (cell[i + 1] === undefined ? 16 : cell[i + 1]) - step
    let p
    if (strong) {
      // land on a tone of the chord, the nearest to where the line is
      p = tones.length ? tones.reduce((b, t) => Math.abs(t - cur) < Math.abs(b - cur) ? t : b) : cur
    } else {
      const n = r() < 0.7 ? 1 : 2
      if (r() < 0.18) dir = -dir // the line turns now and then
      p = scaleNear(cur, dir, n)
      if (p > hi) { dir = -1; p = scaleNear(cur, dir, n) }
      if (p < lo) { dir = 1; p = scaleNear(cur, dir, n) }
    }
    cur = p
    // the notes on the beat lean in, the ones that anticipate (the odd steps) lean a little less
    const accent = strong ? 1.1 : (step % 2 === 1 ? 0.88 : 0.96)
    events[step] = { p, v: cfg.vel * vscale * accent * (0.94 + 0.1 * r()), len: Math.min(gap * 1.6, 8) }
    // a grace note a half step under a strong note, in the sixteenth before it
    if (strong && step > 0 && r() < cfg.grace && events[step - 1] === undefined) events[step - 1] = { p: p - 1, v: cfg.vel * vscale * 0.45, len: 1 }
  })
  return events
}

let events = {}
function trigger(ev) {
  if (!cfg) return
  const step = ev.step % 16
  const c = shared.chordFor(song.section, song.bar)
  // the last chord of the piece: a wide soft chord that rings
  if (song.section === "Amen" && song.bar === 10 && step === 0) {
    const b = c.bass(38)
    note([b, b + 7, ...c.close(55)], 0.34, 4 * 16)
    return
  }
  if (cfg.on && !cfg.on(song.bar)) return
  const bars = shared.lasts(song.section, song.bar, song.bars)
  // the pedal: up where the chord changes, down again a sixteenth later
  if (step === 0 && c !== last) { set({ sustain: 0 }); last = c }
  if (step === 1 && (song.bar === 0 || shared.chordFor(song.section, song.bar - 1) !== c)) set({ sustain: 1 })
  if (step === 0) events = makeBar(c, bars)
  const g = cfg.grid
  const w = arc[song.bar % 4]
  // the left hand
  if (g && g.bass.includes(step)) {
    const b = c.bass(40)
    const n = step === 0 ? [b, b + 7] : (step === 8 || step === 10) && g.bass.length > 3 ? [b + 12] : [b + 7]
    note(n, cfg.vel * 0.7 * (step === 0 ? 1 : 0.8) * w, 12)
  }
  // the stabs: guide tones of the chord, short, on the places where the beat leans
  if (g && g.stab.includes(step)) {
    const t = c.tones(52, 67)
    const pcs = t.map(n => (((n - c.root) % 12) + 12) % 12)
    const pick = [[3, 4], [10, 11], [2]].map(set => t.find((n, i) => set.includes(pcs[i]))).filter(n => n !== undefined)
    const shell = pick.length >= 2 ? pick : shared.voices(c, 55)
    note(shell, cfg.vel * 0.5 * w, 3)
  }
  // the line
  const e = events[step]
  if (e) note(cfg.octave && step % 8 === 0 ? [e.p, e.p - 12] : e.p, Math.min(e.v, 0.9), e.len)
}
