// the tabla keeps the bembe, the bell pattern of seven hits in twelve (the seven days, spread evenly over the twelve sixteenths of a bar of three beats;
// euclid(7, 12)), the pieces of the tabla loop cut at its hits played in the order of the loop, the hits on the beat louder, the others swung. The Doxology plays the
// three of the Trinity, one a beat
clock("1/16")
const play = shared.player("tabla")
const CYCLES = [[0, 0], [5, 0], [7, 0], [7, 0], [7, 5], [7, 0], [3, 0]] // [hits, turned by] a day
let pattern = null, last = -1
function section(name) {
  const day = shared.day(name)
  const [hits, rot] = CYCLES[day.number - 1]
  const g = genny.euclid(hits, 12, rot)
  pattern = Array.from({ length: 12 }, () => g.next())
}
function trigger(ev) {
  if (!shared.jazz || !pattern) return
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const at = song.bar * 12 + s
  if (!day || at === last || !pattern[s] || !shared.layers.tabla(day, song.bar)) return
  last = at
  const pieces = shared.pieces.tabla
  shared.piece(play, note, ev, s, pieces[at % pieces.length], s % 4 === 0 ? 0.46 : 0.3, -26)
}
