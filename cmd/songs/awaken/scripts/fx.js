// FX: the joins. A low hit where a section begins (the impact preset: a fold falling from high to low over a sine
// thump), a riser in the four bars before a change (the pitch, the brightness and the fold all going up in
// the length of one note), a gong where the piece lifts and where it comes apart. Its synth is on a heavy send of the reverb.
const hits = {
  Pulse:    [{ at: 0, preset: "impact", pitch: 38, trim: -9, vel: 0.8, len: 30 }],
  Glow:     [{ at: 0, preset: "gong", pitch: 50, trim: -10, vel: 0.7, len: 60 }],
  Depth:    [{ at: 0, preset: "impact", pitch: 38, trim: -7, vel: 1, len: 30 }, { at: 12, preset: "impact", pitch: 38, trim: -8, vel: 0.9, len: 30 }],
  Lift:     [{ at: 0, preset: "gong", pitch: 50, trim: -9, vel: 0.8, len: 60 }],
  Bloom:    [{ at: 0, preset: "impact", pitch: 43, trim: -7, vel: 1, len: 30 }, { at: 10, preset: "gong", pitch: 50, trim: -9, vel: 0.7, len: 60 }],
  Dissolve: [{ at: 0, preset: "gong", pitch: 38, trim: -9, vel: 0.8, len: 60 }],
}
// risers: in which section, from which bar to the end
const risers = { Pulse: 16, Glow: 12, Depth: 20, Hollow: 8, Lift: 16, Bloom: 16 }

function trigger(ev) {
  if (ev.step % 16 !== 0) return
  const bar = song.bar
  const h = (hits[song.section] || []).find(h => h.at === bar)
  if (h) {
    set({ preset: h.preset, trim: h.trim, mono: 0, legato: 0 })
    note(h.pitch, h.vel, h.len)
  }
  const r = risers[song.section]
  if (r !== undefined && bar === r) {
    const bars = song.bars - r
    set({ preset: "riser", trim: -12, filterEnvAmount: 1.5, mono: 0, legato: 0 })
    note(shared.fit(50, shared.chordFor(song.section, bar)), 0.8, bars * 16 - 1)
  }
}
