// FX: not in the file either: the joins. A low hit where a section begins (the shaper's impact), and before the Theme, the Canon and
// the Glory a riser in the last bars (the pitch, the brightness and the fold all going up in the length of one note).
const hits = { Gathering: { trim: -9, vel: 0.7 }, Theme: { trim: -7, vel: 0.95 }, Canon: { trim: -8, vel: 0.85 }, Glory: { trim: -6, vel: 1 }, Throne: { trim: -8, vel: 0.9 } }
// the bar of the section (from it to the end) a riser starts at, in the section before
const risers = { Gathering: 5, Theme: 5, Canon: 5 }
function trigger(ev) {
  if (ev.step % 12 !== 0) return
  const bar = song.bar
  const h = hits[song.section]
  if (h && bar === 0) {
    set({ preset: "impact", trim: h.trim, mono: 0, legato: 0 })
    note(43, h.vel, 24)
  }
  const r = risers[song.section]
  if (r !== undefined && r < song.bars && bar === r) {
    set({ preset: "riser", trim: -12, filterEnvAmount: 1.5, mono: 0, legato: 0 })
    note(55, 0.8, (song.bars - r) * 12 - 1)
  }
}
