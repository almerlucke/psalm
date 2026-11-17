// the darbuka is the heart of it: its doum on the first beat of every bar and, from the head, a tak on the third beat and a doum that anticipates the next bar
// on the "and" of the third (the tumbao of the bass, a beat early); the pa answers on the second beat in the break and the head out. The first bar of the piece is its
// doum alone
clock("1/16")
const R = genny.rand(0, 1)
const k = shared.kits.darbuka
const play = shared.player("darbuka")
let last = -1, n = 0
function trigger(ev) {
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const at = song.bar * 12 + s
  if (!day || at === last || !shared.layers.darbuka(day, song.bar)) return
  last = at
  const d = day.number
  const hit = keys => keys[n++ % keys.length]
  if (s === 0) play(note, ev, s, hit(k.doum), d === 1 ? 0.4 : 0.5, 3, true)
  if (d >= 2 && s === 8) play(note, ev, s, hit(k.tak), 0.3, 1, true)
  if (d >= 3 && s === 10 && R.next() < 0.7) play(note, ev, s, hit(k.doum), 0.3, 2, true)
  if (d >= 4 && d <= 6 && s === 4 && R.next() < 0.6) play(note, ev, s, hit(k.pa), 0.3, 1, true)
}
