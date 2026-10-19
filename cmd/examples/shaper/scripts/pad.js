// PAD: the chords of the piece, voice led (each chord moves as little as it can from the one before), held as long
// as the chord lasts, low and dull. A drone more than a chord: its sound is a preset for each section (a dark pad,
// a drone in fifths, a pad that goes through four waves by itself, a choir far off, strings), and the script
// moves it a little with every bar: the macro and the filter. The notes are low and the sound is dark, so the chains
// are read once a sample (oversample 1): nothing of the edges of a saw or a pulse is left to fold back, and eight
// voices of long releases are the cost of the piece.
const walk = genny.rwalk(0.5, 0, 1, 0.2)

const parts = {
  Drone:    { preset: "darkPad", trim: 2, cut: null },
  Pulse:    { preset: "drone5", trim: -2, cut: null },
  Depth:    { preset: "evolve", trim: -4, cut: [500, 1400] },
  Hollow:   { preset: "choirPad", trim: -2, cut: null },
  Return:   { preset: "stringPad", trim: -5, cut: [450, 1500] },
  Dissolve: { preset: "darkPad", trim: 2, cut: null, fade: true },
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
  note(voicing, 0.5, toEnd * 16 - 0.5)
}
