// the bongos hammer a martillo from the solo of the steel pan on, the high drum and the low (a low, b high): the little drums of the island, answering the congas
clock("1/16")
const k = shared.kits.bongos
const play = shared.player("bongos")
const P = [null, null, "a.b.a.b.a.b.", "a.bba.b.abb.", "a.b.a.b.a.b.", "a.bba.b.abb.", "a.b.a.b.a.b."]
let last = -1, n = 0
function trigger(ev) {
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const at = song.bar * 12 + s
  if (!day || at === last || !P[day.number - 1]) return
  last = at
  const c = P[day.number - 1][s]
  if (c === "." || !shared.layers.bongos(day, song.bar)) return
  const keys = c === "a" ? k.low : k.high
  play(note, ev, s, keys[n++ % keys.length], s % 4 === 0 ? 0.42 : 0.32, 2, true)
}
