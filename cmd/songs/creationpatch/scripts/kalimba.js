// the kalimba answers the marimba half a bar later, an octave up and softer (see marimba.js)
clock("1/16")
const play = shared.player("kalimba")
const LATE = 6
let last = -1
function trigger(ev) {
  if (!shared.jazz || !shared.canon) return
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const k = song.bar * 12 + s
  if (!day || k === last) return
  last = k
  const e = shared.canon[song.index + ":" + (k - LATE)]
  if (!e || !shared.layers.kalimba(day, song.bar)) return
  play(note, ev, s, e.p + 12, e.v * 0.8, 3)
}
