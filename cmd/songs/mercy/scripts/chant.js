// CHANT: the hymn (Johannes), cut at its syllables and told over the verses (shared.calls): each verse the next ones, in the order they were
// sung, at the distances they were sung at. The refrain bar of a verse is the choir's; the call is over before it.
function trigger(ev) {
  if (!shared.known()) return
  const v = shared.verse(), k = shared.step(ev)
  for (const n of shared.calls[v]) {
    if (n.at + 2 === k) note(n.key, Math.min(1, 0.75 * n.gain) * shared.fade(), n.len)
  }
}
