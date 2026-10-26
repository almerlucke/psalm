// a marimba plays a little tune on the chord of the bar, a kalimba answers it half a bar later an octave up (a canon in the manner of the plants of the third day):
// a markov chain walks the tones of the chord (mostly a step up or down, now and then a leap), on a euclidean seven in twelve turned a little every day; the tune
// goes on from bar to bar, onto the tones of the next chord. The kalimba reads what the marimba played from shared.canon
clock("1/16")
const R = genny.rand(0, 1)
const play = shared.player("marimba")
const ROT = [0, 0, 0, 1, 3, 5, 2]
const TONES = [0, 3, 7, 9, 12, 15, 19, 21, 24] // above the root; the third is major or minor below
let pattern = null, walk = null, pos = 3, last = -1
shared.canon = shared.canon || {}
function section(name) {
  const day = shared.day(name)
  const g = genny.euclid(7, 12, ROT[day.number - 1])
  pattern = Array.from({ length: 12 }, () => g.next())
  walk = genny.markov({ "-2": { "-1": 3, "1": 3, "-2": 1 }, "-1": { "-1": 3, "1": 2, "2": 1, "-2": 1 }, "1": { "1": 3, "-1": 2, "-2": 1, "2": 1 }, "2": { "1": 3, "-1": 3, "2": 1 } }, "1")
  pos = 3
}
function trigger(ev) {
  if (!shared.jazz || !pattern) return
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const k = song.bar * 12 + s
  if (!day || k === last || !pattern[s] || !shared.layers.marimba(day, song.bar)) return
  last = k
  const br = shared.bar(day, song.bar)
  const q = shared.quality(br)
  if (!q) return
  pos = Math.max(0, Math.min(TONES.length - 1, pos + Number(walk.next())))
  const third = q.third
  const tone = [0, third, 7, 9, 12, 12 + third, 19, 21, 24][pos]
  const p = br.root + 12 + tone
  const v = s % 4 === 0 ? 0.5 : s % 2 === 0 ? 0.4 : 0.34
  shared.canon[song.index + ":" + k] = { p, v }
  play(note, ev, s, p, v, 2.5)
}
