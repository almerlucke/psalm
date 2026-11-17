// the upright bass walks: the root on the first beat, a tone of the chord the markov chain picks on the second (the simpler the ratio of the tone to the root, the
// likelier: 1:1, 2:1, 3:2, 5:4 and 6:5, 9:5, 15:8), and on the third beat, going to the root of the next bar, either a step from it (a chromatic approach, below or
// above) or another tone of the chord. Three notes to a walk, the walk of the Trinity, but the second and third are syncopated, on the "and" of the second and of the
// third beat (the third anticipates the next bar), as a bass plays in the islands
clock("1/16")
const R = genny.rand(0, 1)
const play = shared.player("upright bass")
const chains = {}
let last = -1
function trigger(ev) {
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const k = song.bar * 12 + s
  if (!day || k === last || !(s === 0 || s === 6 || s === 10) || !shared.layers.bass(day, song.bar)) return
  last = k
  const br = shared.bar(day, song.bar)
  const q = shared.quality(br)
  const beat = s === 0 ? 0 : s === 6 ? 1 : 2
  const root = br.root
  if (!q) { if (beat === 0) play(note, ev, s, root, 0.6, 7); return } // the bare fifths: the root, held
  const id = (q.minor ? "m" : q.dominant ? "d" : "M")
  const chain = chains[id] || (chains[id] = shared.walkChain(q))
  if (beat === 0) return play(note, ev, s, root, 0.72, 3.2)
  const tone = Number(chain.next())
  if (beat === 1) return play(note, ev, s, root + tone, 0.55, 3.2)
  const next = song.bar + 1 < day.bars ? shared.bar(day, song.bar + 1).root : br.root
  const target = next - root
  const approach = target !== 0 && R.next() < 0.6
  const step = R.next() < 0.5 ? -1 : 1
  play(note, ev, s, approach ? next + step : root + Number(chain.next()), 0.6, 3.4)
}
