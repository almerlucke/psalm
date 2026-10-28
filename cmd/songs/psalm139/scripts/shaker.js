// eggs: a thread of three against four. Every third sixteenth, counted through the whole piece and not from the bar, is an accent, so the accent walks across the beats of the bar
// and falls back on the beat only every three bars; the sixteenths between are ghosts, some of them left out. It comes with the knitting, and rests in the Thread, the Shadow and the end
clock("1/16")
const play = shared.player("shaker")
const keys = shared.kits.shaker.egg
const rand = shared.rng(shared.seed || 1, "shaker-ghosts")
const on = { Knit: b => b >= 4, Wonder: () => true, Light: () => true, Sand: () => true }
let current = null, last = -1
function section(name) { current = on[name] ? shared.sec(name) : null; last = -1 }
function trigger(ev) {
  if (!current) return
  const k = shared.sixteenth(song)
  if (k === last) return
  last = k
  if (!on[current.name](song.bar)) return
  const accent = k % 3 === 0
  if (!accent && rand() > 0.5) return
  play(note, keys[(k * 7) % keys.length], (accent ? 0.34 : 0.14) * current.dyn * (k % 4 === 0 ? 1.25 : 1), 1, ev, true)
}
