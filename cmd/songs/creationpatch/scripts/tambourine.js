// the tambourine loop of two bars, cut at its hits and played in its own order, but only where a euclidean cycle lets the pieces through: five in twelve with the sun
// (the fourth day), seven in twelve with the creatures and mankind, three in the Doxology (the Trinity); the cycle is turned a little every day. The accents are on the beat
clock("1/16")
const play = shared.player("tambourine")
const HITS = [0, 0, 5, 7, 7, 7, 5]
let pattern = null, last = -1
function section(name) {
  const day = shared.day(name)
  const g = genny.euclid(HITS[day.number - 1] || 1, 12, (day.number * 5) % 12)
  pattern = Array.from({ length: 12 }, () => g.next())
}
function trigger(ev) {
  if (!pattern) return
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const at = song.bar * 12 + s
  if (!day || at === last || !pattern[s] || !shared.layers.tambourine(day, song.bar)) return
  last = at
  const pieces = shared.pieces.tambourine
  shared.piece(play, note, ev, s, pieces[at % pieces.length], (s % 4 === 0 ? 0.5 : s % 2 === 0 ? 0.38 : 0.3), -30)
}
