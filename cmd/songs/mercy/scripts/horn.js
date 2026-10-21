// HORN: the call, a fifth up and a fourth up (F, C, F: "come up hither"), in the verses of the shofar, the horn of the orchestra (or the shaper's
// trumpet voice when it is not there) soft, louder, loudest.
function trigger(ev) {
  if (!shared.known()) return
  const v = shared.verse(), k = shared.step(ev)
  if (!shared.horn.includes(v)) return
  const call = { 0: [53, 0.22], 16: [60, 0.32], 32: [65, 0.42] }[k]
  if (call) note(call[0], call[1] * shared.fade(), 14)
}
