// LEAD: not a tune: a motif of four notes, with a long silence round each, said again and again for four bars,
// slow. It is written in words (a note a step, `-` holds it, `.` rests) and the script plays it, each note
// fitted to the chord of the bar (shared.fit). The same figure is a different thing in each section, a different
// preset: a flute that slides from note to note in the pulse and the depth (an octave higher in the second half), a talking
// vowel in the hollow, a real melody in the lift, the flute high and alone in the dissolve.
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

// a melody for the lift, over four chords of a bar each
const tune = shared.line(`
  F4 - - - A4 - C5 - - - - - D5 - - -
  C5 - - - - - A4 - - - F4 - - - . .
  D4 - - - G4 - - - Bb4 - - - A4 - G4 -
  F4 - - - - - - - D4 - - - - - . .
`)

// and two more: the glow is gentle and climbs, the bloom is the high one
const glow = shared.line(`
  A4 - - - - - D5 - - - C5 - A4 - - -
  Bb4 - - - D5 - - - F5 - - - - - . .
  C5 - - - A4 - F4 - - - A4 - C5 - - -
  D5 - - - - - - - A4 - - - - - . .
`)
const bloom = shared.line(`
  D5 - - - - - C5 - Bb4 - - - D5 - - -
  Eb5 - - - - - D5 - Bb4 - - - G4 - - -
  F5 - - - - - D5 - C5 - - - A4 - - -
  Bb4 - - - C5 - D5 - - - - - . . . .
`)

// up: octaves added in some bars (the lead climbs in the second half of the depth)
const parts = {
  Pulse:  { line: motif, on: bar => bar >= 12, set: { preset: "fluteLead", mono: 1, legato: 1, trim: -4 } },
  Glow:   { line: glow, up: bar => 0, set: { preset: "fluteLead", mono: 1, legato: 1, trim: -2 } },
  Depth:  { line: motif, up: bar => bar >= 12 ? 12 : 0, set: { preset: "fluteLead", mono: 1, legato: 1, trim: -1 } },
  Hollow: { line: slow, set: { preset: "vowel", mono: 0, legato: 0, trim: -9 } },
  Lift:   { line: tune, up: bar => 0, set: { preset: "fluteLead", mono: 1, legato: 1, trim: -1 } },
  Bloom:  { line: bloom, up: bar => 0, set: { preset: "fluteLead", mono: 1, legato: 1, trim: -1 } },
  Dissolve: { line: slow, on: bar => bar < 12, up: bar => 12, set: { preset: "fluteLead", mono: 1, legato: 1, trim: -4 } },
}
let part = null
function section(name) {
  part = parts[name]
  if (part) set(part.set)
}

function trigger(ev) {
  if (!part || (part.on && !part.on(song.bar))) return
  const i = song.bar * 16 + ev.step % 16
  const n = part.line[i % part.line.length]
  if (!n) return
  // the motif was written for D minor: over another chord a note that does not belong moves to one that does
  note(shared.fit(n.pitch + (part.up ? part.up(song.bar) : 0), shared.chordFor(song.section, song.bar)), 0.6 + 0.25 * song.progress, Math.max(0.9, n.len - 0.3))
}
