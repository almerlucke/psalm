// ARP: a few notes of the chord, low and soft, on a euclidean rhythm that is turned a step every bar, so the
// figure keeps changing without changing the notes. It comes in late (the depth of the piece) and goes round
// in a cycle of five against the sixteen steps when the piece returns. Its sound is a preset for each section
// (a marimba, a kalimba, a harp); the echo (a dotted eighth, on the second send of the mixer) makes two lines
// of it and a mild crusher roughens it.
const parts = {
  Depth:  { preset: "marimba", trim: -6, mode: "alternate", hits: 5, lo: 50, hi: 69, from: 0.2, to: 0.55, bars: b => b < 4 ? 0 : 1 },
  Return: { preset: "kalimba", trim: -5, mode: "cycle5", hits: 7, lo: 50, hi: 69, from: 0.35, to: 0.65, bars: b => b < 2 ? 0 : 1 },
  Dissolve: { preset: "harp", trim: -7, mode: "up", hits: 3, lo: 50, hi: 66, from: 0.4, to: 0.05, bars: b => b < 3 ? 1 : 0 },
}
let part = null, notes = null, hitsNow = null
function section(name) {
  part = parts[name]
  // these presets have no filter: a low pass of the script's takes the sparkle off the top
  if (part) set({ preset: part.preset, trim: part.trim, filterType: 5, cutoff: 2400, resonance: 0.1, keyTrack: 0.3, filterEnvAmount: 0 })
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
  if (!part.bars(bar) || !hit) return
  // louder through the section
  const p = part.from + (part.to - part.from) * song.progress
  note(notes.next(), (k % 4 === 0 ? 0.85 : 0.55) * (0.55 + 0.45 * p), 1.4)
}
