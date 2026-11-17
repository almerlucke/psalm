// the claves keep the bell pattern of the bembe: seven hits in twelve sixteenths (the seven days; euclid(7, 12)), turned a little every day; the intro and the tag
// play the three of the Trinity, one a beat. In the break and the head out castanets answer on the offbeats
clock("1/16")
const k = shared.kits.claves
const play = shared.player("claves")
const CYCLES = [[3, 0], [7, 0], [7, 5], [5, 0], [7, 0], [7, 5], [3, 0]] // [hits, turned by] a section
let pattern = null, last = -1, n = 0
function section(name) {
  const day = shared.day(name)
  const [hits, rot] = CYCLES[day.number - 1]
  const g = genny.euclid(hits, 12, rot)
  pattern = Array.from({ length: 12 }, () => g.next())
}
function trigger(ev) {
  if (!pattern) return
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const at = song.bar * 12 + s
  if (!day || at === last || !shared.layers.claves(day, song.bar)) return
  last = at
  if (pattern[s]) play(note, ev, s, k.clave[n++ % k.clave.length], s % 4 === 0 ? 0.5 : 0.38, 1, true)
  else if ((day.number === 4 || day.number === 6) && s % 4 === 2 && s % 8 !== 2) play(note, ev, s, k.castanet[n++ % k.castanet.length], 0.3, 1, true)
}
