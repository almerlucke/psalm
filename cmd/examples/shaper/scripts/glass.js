// GLASS: a few low bells, far apart: two or three notes of the chord a bar on a rhythm that turns a step every bar.
// The bells are presets of the shaper synth (a church bell, a struck glass), and what the mixer does with them
// (the echo, a dotted eighth, and above all the granulator and the shimmer) is most of it. They come in during
// the drone, go on through the piece and the last of them ring out the end.
const parts = {
  Drone:    { preset: "churchBell", trim: -6, hits: 2, lo: 57, hi: 76, on: bar => bar >= 2, vel: 0.5 },
  Pulse:    { preset: "churchBell", trim: -6, hits: 2, lo: 57, hi: 76, on: bar => bar >= 8, vel: 0.45 },
  Glow:     { preset: "glassBell", trim: -7, hits: 3, lo: 62, hi: 84, on: bar => bar % 2 === 0, vel: 0.5 },
  Depth:    { preset: "churchBell", trim: -6, hits: 3, lo: 57, hi: 78, on: bar => bar % 4 === 3 || bar >= 16, vel: 0.5 },
  Hollow:   { preset: "glassBell", trim: -7, hits: 3, lo: 57, hi: 81, on: bar => true, vel: 0.5 },
  Lift:     { preset: "glassBell", trim: -7, hits: 4, lo: 62, hi: 84, on: bar => true, vel: 0.5 },
  Bloom:    { preset: "glassBell", trim: -7, hits: 5, lo: 62, hi: 86, on: bar => true, vel: 0.5 },
  Dissolve: { preset: "churchBell", trim: -6, hits: 3, lo: 57, hi: 79, on: bar => bar < 14, vel: 0.5 },
}
let part = null, hits = null, order = null
function section(name) {
  part = parts[name]
  if (part) set({ preset: part.preset, trim: part.trim, filterType: 5, cutoff: 7000, resonance: 0.1, keyTrack: 0.3, filterEnvAmount: 0 })
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
