// CHOIR: the refrain, "for his mercy endureth for ever": in the last bar of each verse the choir holds the chord of the verse, in three voices
// of the chord, from the second verse on; in the last verse it holds it through all four bars, and it is the quiet the piece ends in.
function trigger(ev) {
  if (!shared.known()) return
  const v = shared.verse(), vb = shared.verseBar(), k = shared.step(ev)
  const last = v === 25
  if (last ? k !== 0 : !(vb === 3 && (ev.step % 16) === 0 && v > 0)) return
  const tones = shared.chordOf(v).tones(53, 77)
  const voices = [tones[0], tones[Math.min(2, tones.length - 1)], tones[Math.min(4, tones.length - 1)]]
  note(voices, 0.5 * shared.fade(), last ? 63.5 : 15.5)
}
