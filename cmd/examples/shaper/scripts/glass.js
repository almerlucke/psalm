// GLASS: a few low bells, far apart: two or three notes of the chord a bar on a rhythm that turns a step every bar.
// The bells are presets of the shaper synth (a church bell, a struck glass, a vibraphone), and what the echo
// does with them (a dotted eighth, from the second send) and the slow auto-pan is most of it. They come in during
// the drone, go on through the piece and the last of them ring out the end.
const parts = {
  Drone:    { preset: "churchBell", trim: -6, hits: 2, lo: 50, hi: 69, on: bar => bar >= 4, vel: 0.5 },
  Pulse:    { preset: "churchBell", trim: -6, hits: 2, lo: 50, hi: 69, on: bar => bar >= 8, vel: 0.45 },
  Depth:    { preset: "churchBell", trim: -6, hits: 2, lo: 50, hi: 69, on: bar => bar % 4 === 3, vel: 0.5 },
  Hollow:   { preset: "glassBell", trim: -7, hits: 3, lo: 50, hi: 71, on: bar => true, vel: 0.5 },
  Return:   { preset: "vibes", trim: -9, hits: 3, lo: 50, hi: 71, on: bar => bar % 2 === 1, vel: 0.45 },
  Dissolve: { preset: "churchBell", trim: -6, hits: 3, lo: 50, hi: 69, on: bar => bar < 10, vel: 0.45 },
}
let part = null, hits = null, order = null
function section(name) {
  part = parts[name]
  if (part) set({ preset: part.preset, trim: part.trim, filterType: 5, cutoff: 3200, resonance: 0.1, keyTrack: 0.3, filterEnvAmount: 0 })
}

function trigger(ev) {
  if (!part) return
  const k = ev.step % 16, bar = song.bar
  if (!part.on(bar)) return
  if (k === 0) {
    const tones = shared.chordFor(song.section, bar).tones(part.lo, part.hi)
    order = genny.arpeggio(tones, "random", "exclusive")
    hits = genny.euclid(part.hits, 16, bar)
  }
  if (!hits.next()) return
  note(order.next(), part.vel * (k % 4 === 0 ? 1 : 0.7), 12)
}
