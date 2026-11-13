// PULSE: not in the file, made from it: a sub bass that plays the note the pedal holds, an octave up where it would be too low to
// hear on a small speaker, in eighths with an accent on every beat. It comes in with the Gathering, and every section after it is
// louder, so under the theme and the canon the piece has a pulse. The subtractive synth's pulseBass, tuned dark and short.
const level = [0, 0, 0.5, 0.7, 0.85, 1, 1]
function trigger(ev) {
  const lv = level[Math.max(0, song.index)] || 0
  if (!lv) return
  const s = shared.stepNow(ev)
  if (s < 0 || s % 2) return
  let p = shared.heldAt("pedal", s)
  if (p < 0) return
  while (p < 36) p += 12
  note(p, (s % 4 === 0 ? 0.8 : 0.5) * lv, 1.6)
}
