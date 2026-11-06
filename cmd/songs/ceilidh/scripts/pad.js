// PAD: the chord of the bar, voice led (each chord moves as little as it can from the one before), held for the bar.
let voicing = null
function trigger(ev) {
  if (ev.step % 6 !== 0) return
  const lv = shared.level("pad", song.section, song.bar)
  if (!lv) { voicing = null; return }
  const c = shared.chordAt(song.section, song.bar, 0)
  voicing = voicing ? c.voiceLead(voicing) : c.close(50)
  note(voicing, 0.5 * lv, 5.8)
}
