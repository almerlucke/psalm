// ELDERS: four and twenty elders in white raiment, a choir that comes in with the beasts (the man, the third) and stays
// to the end. They sing the chord of the bar in every octave, as many voices as the section has bars of progress: a
// chord of two bars is held, and the voices come in one by one over its first bar, as the elders take their seats. At the end
// of the Elders they cast their crowns: the choir stops for a bar (the fx script throws the gold down) and comes back.
const parts = {
  Creatures: { preset: "choirPad", trim: -13, on: bar => bar >= 12, voices: bar => 3, vel: 0.4 },
  Elders:    { preset: "choirPad", trim: -13, on: bar => bar >= 2 && bar < 20, voices: bar => 2, vel: 0.45 },
  Holy:      { preset: "choirPad", trim: -12, on: bar => true, voices: bar => 2, vel: 0.45 },
  Worthy:    { preset: "choirPad", trim: -10, on: bar => bar < 9, voices: bar => 3, vel: 0.45 },
}
let part = null
function section(name) {
  part = parts[name]
  if (part) set({ preset: part.preset, trim: part.trim, unison: 2 })
}

function trigger(ev) {
  const k = ev.step % 16, bar = song.bar
  if (!part || !part.on(bar)) return
  const c = shared.chordFor(song.section, bar)
  // the steps since the chord began and the bars it has left: the voices take their seats one every 3 steps (three)
  let first = bar, last = bar
  while (first > 0 && shared.chordFor(song.section, first - 1) === c) first--
  while (last + 1 < song.bars && shared.chordFor(song.section, last + 1) === c) last++
  const into = (bar - first) * 16 + k
  const tones = c.tones(47, 79)
  const n = Math.min(part.voices(bar), tones.length)
  const slot = into / 3
  if (into % 3 !== 0 || slot >= n) return
  // from the middle outwards: the elders nearest the throne first
  const mid = Math.floor(tones.length / 2)
  const order = tones.map((t, i) => [Math.abs(i - mid) + (i < mid ? 0.5 : 0), t]).sort((a, b) => a[0] - b[0]).map(x => x[1])
  note(order[slot], part.vel, (last - bar + 1) * 16 - k - 0.5)
}
