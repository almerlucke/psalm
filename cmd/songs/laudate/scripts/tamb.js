// TAMB: a tambourine loop of two bars at 90 beats (a stroke in every sixteenth), cut at its hits, 29 pieces on the keys from 36. The pieces are played in order, so the
// loop is what it was, but only where a euclidean cycle of nine to thirteen in sixteen lets them through,
// in the answer the pieces go the other way round. The accents are on the beat.
clock("1/16")
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const pieces = shared.sounds.tamb
let grid = null, hits = -1, phrase = -1
function trigger(ev) {
  if (!shared.known()) return
  const s = shared.s()
  if (!s.tamb) return
  const p = shared.phrase(), q = shared.q(), i = ev.step % 16, bar = song.bar
  const h = Math.round(7 + 6 * s.tamb)
  if (h !== hits || p !== phrase) { hits = h; phrase = p; grid = genny.euclid(h, 16, (3 * q) % 16) }
  const on = grid.next()
  if (!s.tp[q] || !on) return
  const at = (bar * 16 + i) % pieces.length
  const d = pieces[q === 2 ? pieces.length - 1 - at : at]
  const v = (i % 4 === 0 ? 0.55 : i % 2 === 0 ? 0.42 : 0.34) + 0.08 * R()
  note(d.key, Math.min(1, v * shared.evenOut(d.db, -35)) * (0.6 + 0.4 * s.tamb), 1)
}
