// SUNG: the long notes I sang (patch.go), one in most verses, after the voice has said the verse and before the refrain: a note of the recording, held for two bars (the second half of the note goes round), that is
// a tone of the chord of the verse, or a step or two off one (the sampler sings a key off a little higher or lower). The notes go from near to near:
// one of the three nearest to the note before, and not the same piece twice in a row.
let lastVerse = -1, prev = 48, lastPiece = -1, next = null
const rest = [12, 25]                                  // the parting of the sea (and the last verse, which has its own)
// the last verse: the voices sing together, in the chord of F minor (C, A flat, F and the F below), after the verse is said, where in the other verses a single
// note is sung; in the coda they answer the choir an octave lower (F, C, F)
const heaven = [24, 29, 34, 49], farewell = [49, 44, 34]       // the keys of the pieces at C4, A flat 3, F3, F2 and C3

const rnd = genny.rand(0, 1)

function choose(v) {
  const tones = shared.chordOf(v).tones(36, 72), found = []
  shared.sung.forEach((n, i) => {
    for (let d = -2; d <= 2; d++) {
      const t = n.pitch + d
      if (!tones.includes(t)) continue
      const score = 2 * Math.abs(d) + Math.abs(t - prev) / 3 + (i === lastPiece ? 6 : 0) + (t === prev ? 5 : 0) + (t > 58 ? 1 : 0)
      found.push({ score, i, d, t })
    }
  })
  found.sort((x, y) => x.score - y.score)
  // one of the three that fit best, mostly the best
  return found.length ? found[Math.min(found.length - 1, Math.floor(rnd.next() * rnd.next() * 3))] : null
}

function trigger(ev) {
  if (shared.coda()) {
    if (song.bar === 4 && ev.step % 16 === 0) farewell.forEach(key => note(key, 0.5, 34))
    return
  }
  if (!shared.known()) return
  const v = shared.verse(), k = shared.step(ev)
  if (v !== lastVerse) {
    lastVerse = v
    next = rest.includes(v) ? null : choose(v)
    if (next) { prev = next.t; lastPiece = next.i }
  }
  if (v === 25 && k === 20) heaven.forEach(key => note(key, 0.5, 26))
  if (next && k === 20) {
    const n = shared.sung[next.i]
    note(n.key + next.d, Math.min(1, 0.7 * shared.evenOut(n, -16)) * shared.fade(), 26)
  }
}
