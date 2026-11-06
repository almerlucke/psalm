// FX: the joins. A low hit where a section begins (the impact preset), one more every sixteen bars of the Third, a riser
// in the last bars before a change (the pitch, the brightness and the fold all going up in the length of one note).
const hits = {
  First:  [{ at: 0, pitch: 43, trim: -8, vel: 0.9, len: 12 }],
  Break:  [{ at: 0, pitch: 43, trim: -8, vel: 0.8, len: 12 }],
  Second: [{ at: 0, pitch: 43, trim: -7, vel: 1, len: 12 }],
  Third:  [{ at: 0, pitch: 43, trim: -6, vel: 1, len: 12 }, { at: 16, pitch: 43, trim: -8, vel: 0.9, len: 12 }, { at: 32, pitch: 43, trim: -8, vel: 0.9, len: 12 }],
  Outro:  [{ at: 0, pitch: 43, trim: -9, vel: 0.7, len: 18 }],
}
// from which bar to the end of the section
const risers = { First: 28, Break: 4, Second: 28, Bridge: 0 }

function trigger(ev) {
  if (ev.step % 6 !== 0) return
  const bar = song.bar
  const h = (hits[song.section] || []).find(h => h.at === bar)
  if (h) {
    set({ preset: "impact", trim: h.trim, mono: 0, legato: 0 })
    note(h.pitch, h.vel, h.len)
  }
  const r = risers[song.section]
  if (r !== undefined && bar === r) {
    set({ preset: "riser", trim: -12, filterEnvAmount: 1.5, mono: 0, legato: 0 })
    note(55, 0.8, (song.bars - r) * 6 - 1)
  }
}
