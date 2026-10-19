// PAD: the chords of the piece, voice led (each chord moves as little as it can from the one before), held as long
// as the chord lasts, low. A drone more than a chord: its sound is a preset for each section (an airy pad, a drone in fifths,
// a pad that goes through four waves by itself, a choir far off, a warm pad), and the script
// moves it a little with every bar: the macro and the filter. The notes are low and the sound is dark, so the chains
// are read once a sample (oversample 1): nothing of the edges of a saw or a pulse is left to fold back, and eight
// voices of long releases are the cost of the piece.
const walk = genny.rwalk(0.5, 0, 1, 0.2)

const parts = {
  Drone:    { preset: "airPad", trim: -5, cut: null, thin: true },
  Pulse:    { preset: "drone5", trim: -3, cut: null },
  Glow:     { preset: "warmPad", trim: -6, cut: [800, 2400], double: true },
  Depth:    { preset: "evolve", trim: -4, cut: [500, 2200], double: true },
  Hollow:   { preset: "choirPad", trim: -8, cut: null },
  Lift:     { preset: "warmPad", trim: -5, cut: [900, 2600], double: true },
  Bloom:    { preset: "warmPad", trim: -5, cut: [1200, 3400], double: true },
  Dissolve: { preset: "airPad", trim: -2, cut: null, fade: true },
}

let part = parts.Drone, voicing = null, last = null
function section(name) {
  part = parts[name] || parts.Drone
  last = null
  set({ preset: part.preset, trim: part.trim, oversample: 1 })
}

function trigger(ev) {
  const k = ev.step % 16, bar = song.bar
  if (k !== 0) return
  const c = shared.chordFor(song.section, bar)
  set({ macro1: song.progress })
  if (part.cut) set("cutoff", part.cut[0] * Math.pow(part.cut[1] / part.cut[0], song.progress))
  // the end: the pad fades out over the last bars
  if (part.fade) set("trim", part.trim - 30 * Math.pow(Math.max(0, song.progress - 0.4) / 0.6, 1.5))
  if (c === last) return                       // the same chord goes on
  last = c
  voicing = voicing ? c.voiceLead(voicing) : c.close(43)
  // it lasts until the chord changes
  let bars = 1
  while (bars < 24 && shared.chordFor(song.section, bar + bars) === c) bars++
  const toEnd = Math.min(bars, song.bars - bar)
  // thin: only the bass and the top of the chord; double: the upper half of the piece adds the chord an octave up
  let chord = part.thin ? [voicing[0], voicing[voicing.length - 1]] : voicing
  if (part.double && song.progress > 0.3) chord = chord.concat(voicing.map(n => n + 12))
  note(chord, part.double ? 0.55 : 0.5, toEnd * 16 - 0.5)
}
