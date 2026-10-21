// CHANT: the psalm in the voice that spoke it (goedertierenheid), the verse of the song at the start of the verse and its refrain in the refrain bar
// (shared.calls): the pieces of the recording, in the order they were said.
function trigger(ev) {
  if (!shared.known()) return
  const v = shared.verse(), k = shared.step(ev)
  for (const n of shared.calls[v]) {
    if (n.at === k) note(n.key, Math.min(1, 0.75 * n.gain) * shared.voiceFade(), n.len)
  }
}
