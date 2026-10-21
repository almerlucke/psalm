// BED: the grains synth on the recording: a slow cloud of its voice holding the chord of the verse. It is the voice of the verse that is being said (the middle of the piece of the recording), and sinks into the mix: it is the quietest of the layers.
function trigger(ev) {
  if (!shared.known()) return
  const v = shared.verse(), vb = shared.verseBar(), k = shared.step(ev)
  if (k !== 0) return
  const c = shared.chordOf(v), level = shared.layer.bed[v]
  // the cloud is made of the verse that is being said: the middle of its piece of the recording
  const ps = shared.slices.chant, last = ps[ps.length - 1], p = ps[2 * v]
  set({ position: (p.at + p.len / 2) / (last.at + last.len), scan: 0.01, spray: 0.1, density: 20 + 30 * level, grainSize: 260 })
  note([c.bass(41), c.bass(41) + 7, c.bass(41) + 12 + (c.tones(53, 70)[1] - 53) % 12], 0.05 * level * shared.fade(), 63)
}
