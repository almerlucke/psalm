// LEAD: not a tune: a motif of four notes, with a long silence round each, said again and again for four bars,
// low and dull. It is written in words (a note a step, `-` holds it, `.` rests) and the script plays it, each note
// fitted to the chord of the bar (shared.fit). The same figure is a different thing in each section, a different
// preset: a flute that slides from note to note in the depth, a talking vowel in the hollow, brass in the return.
const motif = shared.line(`
  D4 - - - - - - - . . . . F4 - - -
  - - - - - - - - . . . . . . . .
  A3 - - - - - - - . . . . C4 - - -
  - - - - D4 - - - - - - - . . . .
`)
const slow = shared.line(`
  D4 - - - - - - - - - - - - - - -
  - - - - - - - - - - - - - - - -
  F4 - - - - - - - - - - - - - - -
  - - - - - - - - - - - - - - - -
  A3 - - - - - - - - - - - - - - -
  - - - - - - - - - - - - - - - -
`)

const parts = {
  Depth:  { line: motif, set: { preset: "fluteLead", mono: 1, legato: 1, trim: -1 } },
  Hollow: { line: slow, set: { preset: "vowel", mono: 0, legato: 0, trim: -9 } },
  Return: { line: motif, set: { preset: "brassLead", mono: 0, legato: 0, trim: -8, filterType: 5, cutoff: 2200, resonance: 0.1, keyTrack: 0.4, filterEnvAmount: 0.5 } },
}
let part = null
function section(name) {
  part = parts[name]
  if (part) set(part.set)
}

function trigger(ev) {
  if (!part) return
  const i = song.bar * 16 + ev.step % 16
  const n = part.line[i % part.line.length]
  if (!n) return
  // the motif was written for D minor: over another chord a note that does not belong moves to one that does
  note(shared.fit(n.pitch, shared.chordFor(song.section, song.bar)), 0.6 + 0.25 * song.progress, Math.max(0.9, n.len - 0.3))
}
