// the maracas on the beat and the eggs on the offbeats, from the fifth bar: the waters hush, then they dance; the tambourine chicks on the second and
// third beats in the break and the head out. A cymbal swells where the light comes, where the creatures and where mankind begin, and when the last chord comes home
clock("1/16")
const k = shared.kits.shakers
const play = shared.player("shakers")
let last = -1, n = 0
function trigger(ev) {
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const at = song.bar * 12 + s
  if (!day || at === last) return
  last = at
  const d = day.number
  if (s === 0 && (((d === 1 && song.bar === shared.EVENING) || ((d === 4 || d === 6) && song.bar === 0)) || (d === 7 && song.bar === shared.FINAL))) {
    play(note, ev, s, k.swell[0], d === 7 ? 0.4 : 0.25, 12, true)
  }
  if (!shared.layers.shakers(day, song.bar)) return
  if (s % 4 === 0) play(note, ev, s, k.maraca[n++ % k.maraca.length], s === 0 ? 0.36 : 0.28, 1, true)
  else if (s % 4 === 2 && d >= 2) play(note, ev, s, k.egg[n++ % k.egg.length], 0.3, 1, true)
  if (d >= 4 && d <= 6 && (s === 4 || s === 8)) play(note, ev, s, k.tambourine[n++ % k.tambourine.length], 0.34, 1, true)
}
