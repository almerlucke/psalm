// LEAD: a saw lead on the tune of the flute, the same notes, so that the two are one big instrument. It comes in
// in the second half of the Second and is there all the Third; the echo of the mixer (a dotted eighth) makes a second line.
function trigger(ev) {
  const lv = shared.level("lead", song.section, song.bar)
  if (!lv) return
  for (const n of shared.notesAt(song.section, song.bar, ev.step % 6)) {
    note(n.pitch, Math.min(1, lv * (0.5 + 0.25 * (n.vel - 76) / 16)), Math.max(0.5, n.len * 0.9), n.delay)
  }
}
