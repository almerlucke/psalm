// DRONE: grains of a sine, very dense, with a bell of a window: a warm hum under the piece, the home note of each chord
// and its fifth, a bit above the bass so it is warm, not heavy. The ground that the others stand on. It is a granular wave, a sine, made of grains of one cycle.
const level = { Waiting: [0.8, 0.8], Wind: [0.8, 0.9], Fire: [0.9, 0.9], Indwelling: [0.9, 0.9], Glory: [0.9, 0.9], Amen: [0.9, 0.3] }
let last = null
function section(name) { last = null }
function trigger(ev) {
  if (ev.step % 16 !== 0) return
  const lv = level[song.section]
  if (!lv) return
  set({ level: (lv[0] + (lv[1] - lv[0]) * song.progress) * 0.7, filter: 1, cutoff: 1200, resonance: 0.05 })
  const c = shared.chordFor(song.section, song.bar)
  if (c === last) return
  last = c
  const bass = c.bass(45)
  const bars = shared.lasts(song.section, song.bar, song.bars)
  note([bass, bass + 7], 0.6, bars * 16 - 0.5)
}
