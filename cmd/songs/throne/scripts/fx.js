// FX: the joins and the gold. A low hit where a section begins (the impact: a fold falling from high to low over a sine
// thump), a church bell struck three times as the holy begins, a gong at the throne, a riser in the bars before a change, and in the
// Elders the crowns: bar 22, the elders cast their crowns before the throne, a clatter of gold (the crownsCast preset) that the
// choir leaves a bar for.
const hits = {
  Throne:    [{ at: 0, preset: "gong", pitch: 38, trim: -12, vel: 0.7, len: 60 }],
  Lamps:     [{ at: 0, preset: "gong", pitch: 45, trim: -13, vel: 0.5, len: 60 }],
  Glass:     [{ at: 0, preset: "gong", pitch: 50, trim: -10, vel: 0.6, len: 60 }],
  Creatures: [{ at: 0, preset: "gong", pitch: 57, trim: -10, vel: 0.7, len: 60 }, { at: 12, preset: "gong", pitch: 62, trim: -11, vel: 0.5, len: 60 }],
  Elders:    [{ at: 0, preset: "gong", pitch: 50, trim: -13, vel: 0.6, len: 60 }, { at: 22, preset: "crownsCast", pitch: 74, trim: -19, vel: 0.5, len: 12 },
              { at: 22, preset: "crownsCast", pitch: 81, trim: -21, vel: 0.4, len: 12, step: 5 }],
  Holy:      [{ at: 0, preset: "glassBell", pitch: 74, trim: -11, vel: 0.6, len: 40 }, { at: 0, preset: "glassBell", pitch: 81, trim: -11, vel: 0.55, len: 40, step: 4 },
              { at: 0, preset: "glassBell", pitch: 86, trim: -11, vel: 0.5, len: 40, step: 8 }],
  Worthy:    [{ at: 0, preset: "gong", pitch: 38, trim: -13, vel: 0.6, len: 60 }],
}
// risers: in which section, from which bar to the end
// no risers: nothing in this piece builds up tension; it is meant to be left on, for hours
const risers = {}

// a preset is loaded when it changes, not again for each hit of a bell, which would reload it under the one still ringing
let current = null
function load(preset, trim) {
  if (current !== preset) set({ preset, trim, mono: 0, legato: 0 })
  else set({ trim })
  current = preset
}

function trigger(ev) {
  const bar = song.bar, k = ev.step % 16
  for (const h of (hits[song.section] || [])) {
    if (h.at === bar && k === (h.step || 0)) {
      load(h.preset, h.trim)
      note(h.pitch, h.vel, h.len)
    }
  }
  const r = risers[song.section]
  if (r !== undefined && bar === r && k === 0) {
    const bars = song.bars - r
    set({ preset: "riser", trim: -12, filterEnvAmount: 1.5, mono: 0, legato: 0 })
    current = "riser"
    note(shared.fit(50, shared.chordFor(song.section, bar)), 0.8, bars * 16 - 1)
  }
}
