// MANDOLIN: the chords of the bar, chopped: a strong down stroke on the first and the fourth eighth, a light up stroke
// on the third and the sixth (the top of the chord only). The strokes are strummed: the notes are a few
// milliseconds apart.
function trigger(ev) {
  const lv = shared.level("mandolin", song.section, song.bar)
  if (!lv) return
  const k = ev.step % 6
  if (k === 1 || k === 4) return
  const tones = shared.chordAt(song.section, song.bar, k).close(55)
  const up = k === 2 || k === 5
  const v = (up ? 0.3 : k === 0 ? 0.8 : 0.65) * lv
  const play = up ? tones.slice(-2).reverse() : tones
  play.forEach((p, i) => note(p, v, up ? 0.8 : 1.6, i * 0.05))
}
