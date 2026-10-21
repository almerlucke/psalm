// BED: the grains synth on the hymn: a slow cloud of its syllables holding the chord of the verse. The point in the hymn it plays from moves
// on with the verses (the 26 verses go through the hymn once), and the cloud is thicker where the verse has more.
function trigger(ev) {
  if (!shared.known()) return
  const v = shared.verse(), vb = shared.verseBar(), k = shared.step(ev)
  if (k !== 0) return
  const c = shared.chordOf(v), level = shared.layer.bed[v]
  set({ position: 0.04 + 0.9 * v / 26, scan: 0.03, spray: 0.12, density: 20 + 30 * level, grainSize: 260 })
  note([c.bass(41), c.bass(41) + 7, c.bass(41) + 12 + (c.tones(53, 70)[1] - 53) % 12], 0.12 * level * shared.fade(), 63)
}
