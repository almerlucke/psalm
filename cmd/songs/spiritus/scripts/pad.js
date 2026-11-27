// PAD: a synth that nobody hears. It plays the chords of the piece, voice led, long and soft, and its sound goes to a
// grain buffer (named pad) and nowhere else: the grains of the CLOUD play what it plays, a moment later and
// in pieces. The piece is only granular; this is what the cloud is made from.
let last = null, voicing = null
function section(name) { last = null; set({ preset: name === "Waiting" ? "airPad" : "warmPad", trim: -6 }) }
function trigger(ev) {
  if (ev.step % 16 !== 0) return
  const c = shared.chordFor(song.section, song.bar)
  if (c === last) return
  last = c
  voicing = voicing ? c.voiceLead(voicing) : c.close(50)
  const bars = shared.lasts(song.section, song.bar, song.bars)
  note(voicing, 0.5, bars * 16 - 0.5)
}
