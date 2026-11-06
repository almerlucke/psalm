// WHISTLE: the same notes, brought into the range of a tin whistle (D5 and up): alone in the Hearth, the Bridge and
// the Outro, and an octave over the flute in the long sections.
const high = p => { while (p < 74) p += 12; while (p > 93) p -= 12; return p }
function trigger(ev) {
  const lv = shared.level("whistle", song.section, song.bar)
  if (!lv) return
  for (const n of shared.notesAt(song.section, song.bar, ev.step % 6)) {
    note(high(n.pitch), Math.min(1, lv * (0.5 + 0.3 * (n.vel - 76) / 16)), Math.max(0.5, n.len * 0.9), n.delay)
  }
}
