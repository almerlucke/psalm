// PAD: the drone under the piece, the root of the verse's chord: the partials 1, 3, 7 and 12 whose levels breathe against one another (the
// shaper's throneDrone). It changes with the verse and is never off, which is the mercy that endures; it follows the fade at the ends.
let last = -1
function trigger(ev) {
  if (!shared.known()) return
  const v = shared.verse(), k = shared.step(ev)
  if (k === 0 && v !== last) {
    last = v
    note(shared.chordOf(v).bass(29), 0.6, 63.5)
  }
  if (k % 16 === 0) set("trim", -6 - (shared.verseBar() === 3 ? 4 : 0) + 20 * Math.log10(shared.fade()))   // 4 dB less under the refrain, which is spoken
}
