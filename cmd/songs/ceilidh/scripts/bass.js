// BASS: the root of the chord low, a fifth on the fourth eighth, "oom-pah" in 6/8. A pattern for each section: the
// semitones above the bass of the chord (null: a rest). The octave is a short pop that jumps out of the line; the Third has an eighth
// more. The bass of a chord is the lowest chord tone at or above C2.
const patterns = {
  First:  [0, null, 12, 7, null, null],
  Break:  [0, null, 0, null, 0, null],
  Second: [0, null, 12, 7, null, 12],
  Third:  [0, 12, 0, 7, 12, 7],
}
function trigger(ev) {
  const lv = shared.level("bass", song.section, song.bar)
  const pat = patterns[song.section]
  if (!lv || !pat) return
  const k = ev.step % 6
  const semis = pat[k]
  if (semis === null) return
  const root = shared.chordAt(song.section, song.bar, k).bass(36)
  // the root rings, the octave pops (short, a little later than the grid) so the line bounces
  note(root + semis, (k === 0 ? 1 : k === 3 ? 0.85 : 0.75) * lv, semis === 12 ? 0.5 : 1.4, semis === 12 ? 0.1 : 0)
}
