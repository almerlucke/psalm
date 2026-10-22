// HORN: the call, a fifth up and a fourth up (F, C, F: "come up hither"), in the verses of the shofar, the horn of the orchestra (or the shaper's
// trumpet voice when it is not there) soft, louder, loudest.
// in the verses of Egypt between the calls the horn goes on from its first note, the F: a few notes of the minor scale, each on the nearest tone of the chord
const lines = { 10: [0, 2, 4, 2, 1, 0], 11: [0, 1, 2, 4, 2, 0], 13: [0, 2, 3, 4, 3, 0] }          // degrees from the F
const low = [53, 55, 56, 58, 60, 61, 63, 65]
const near = (v, p) => shared.chordOf(v).tones(48, 70).reduce((a, b) => Math.abs(b - p) < Math.abs(a - p) ? b : a)

function trigger(ev) {
  if (!shared.known()) return
  const v = shared.verse(), k = shared.step(ev)
  if (lines[v]) {
    if (k % 8 === 0 && k < 48) note(near(v, low[lines[v][k / 8]]), 0.38 * shared.fade(), 12)
    return
  }
  if (!shared.horn.includes(v)) return
  const call = { 0: [53, 0.22], 16: [60, 0.32], 32: [65, 0.42] }[k]
  if (call) note(call[0], call[1] * shared.fade(), 14)
}
