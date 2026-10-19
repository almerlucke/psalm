// ARP: a few notes of the chord, low and soft, on a euclidean rhythm that is turned a step every bar, so the
// figure keeps changing without changing the notes. It comes in late (the depth of the piece) and goes round
// in a cycle of five against the sixteen steps when the piece returns. Its sound is a preset for each section
// (a marimba, a kalimba, a harp); on the mixer the echo (a dotted eighth) makes two lines of it and the granulator
// turns some of it into a cloud.
const parts = {
  Drone:  { preset: "kalimba", trim: -6, mode: "alternate", hits: 3, lo: 57, hi: 79, from: 0.15, to: 0.35, bars: b => b < 6 ? 0 : 2 },
  Pulse:  { preset: "kalimba", trim: -3, mode: "up", hits: 5, lo: 57, hi: 81, from: 0.25, to: 0.5, bars: b => b < 8 ? 0 : 1 },
  Glow:   { preset: "kalimba", trim: -3, mode: "alternate", hits: 7, lo: 55, hi: 84, from: 0.4, to: 0.7, bars: b => b < 2 ? 2 : 1 },
  Depth:  { preset: "marimba", trim: -2, mode: "alternate", hits: 5, lo: 50, hi: 76, from: 0.45, to: 0.8, bars: b => b < 4 ? 0 : 1 },
  Hollow: { preset: "harp", trim: -7, mode: "up", hits: 3, lo: 62, hi: 84, from: 0.3, to: 0.2, bars: b => b < 4 ? 0 : 2 },
  Lift:   { preset: "harp", trim: -6, mode: "cycle5", hits: 7, lo: 55, hi: 84, from: 0.45, to: 0.7, bars: b => 1 },
  Bloom:  { preset: "marimba", trim: -2, mode: "cycle5", hits: 9, lo: 55, hi: 86, from: 0.5, to: 0.85, bars: b => 1 },
  Dissolve: { preset: "harp", trim: -7, mode: "up", hits: 3, lo: 57, hi: 84, from: 0.5, to: 0.1, bars: b => b < 8 ? 1 : 2 },
}
let part = null, notes = null, hitsNow = null
function section(name) {
  part = parts[name]
  // these presets have no filter: a low pass of the script's takes the harshest sparkle off the top
  if (part) set({ preset: part.preset, trim: part.trim, filterType: 5, cutoff: 6000, resonance: 0.1, keyTrack: 0.3, filterEnvAmount: 0 })
}

function trigger(ev) {
  if (!part) return
  const k = ev.step % 16, bar = song.bar
  if (k === 0) {
    const tones = shared.chordFor(song.section, bar).tones(part.lo, part.hi)
    notes = part.mode === "cycle5"
      ? genny.sequence([0, 2, 4, 2, 1].map(i => tones[i % tones.length]))
      : genny.arpeggio(tones, part.mode, "exclusive")
    hitsNow = genny.euclid(part.hits, 16, bar)
  }
  const hit = hitsNow.next()
  const dens = part.bars(bar)
  if (!dens || !hit || (dens === 2 && k % 8 !== 0 && k % 8 !== 3)) return
  // louder through the section
  const p = part.from + (part.to - part.from) * song.progress
  note(notes.next(), (k % 4 === 0 ? 0.85 : 0.55) * (0.55 + 0.45 * p), 1.4)
}
