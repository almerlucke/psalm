// PAD: the organ (the diapason, the foundation of a church organ): the chord of the bar, held for two bars, four tones of it close together. It never stops.
clock("1/4")
function trigger(ev) {
  if (!shared.known()) return
  const s = shared.s(), bar = shared.bar()
  if (ev.step % 4 !== 0 || bar % 2 !== 0) return
  // four neighbouring tones of the chord, the group whose middle lies nearest to the same place (58) whatever the chord, so the voicing never jumps up or down
  const tones = shared.chordAt(bar).tones(46, 70)
  let at = 0
  for (let i = 0; i + 4 <= tones.length; i++) if (Math.abs(tones[i + 1] / 2 + tones[i + 2] / 2 - 58) < Math.abs(tones[at + 1] / 2 + tones[at + 2] / 2 - 58)) at = i
  note(tones.slice(at, at + 4), 0.45 * s.pad, 8.5)
}
