// FLUTE: the tunes as the files have them, one voice, on an electronic flute (the shaper synth's fluteLead, one note at
// a time, sliding a little from note to note). A note is as loud as the file says (the files have three loudnesses,
// the accents of the jig) and as long as it is written. The sixteenths of the first jig are played a half step late (the
// delay). It plays in the First, the Second and the Third, and softly in the Outro.
function section() { set({ preset: "fluteLead", mono: 1, legato: 1, trim: -2 }) }

function trigger(ev) {
  const lv = shared.level("flute", song.section, song.bar)
  if (!lv) return
  for (const n of shared.notesAt(song.section, song.bar, ev.step % 6)) {
    note(n.pitch, Math.min(1, lv * (0.55 + 0.3 * (n.vel - 76) / 16)), Math.max(0.5, n.len * 0.95), n.delay)
  }
}
