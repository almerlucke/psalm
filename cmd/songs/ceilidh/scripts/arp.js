// ARP: the chord of the eighth as a figure on a pluck, high, six notes a bar. The notes it takes of the chord are
// the same pattern every bar, so the figure turns with the chords; the Break leaves rests in it.
const order = [0, 2, 1, 3, 2, 1]
function trigger(ev) {
  const lv = shared.level("arp", song.section, song.bar)
  if (!lv) return
  const k = ev.step % 6
  if (song.section === "Break" && (k === 1 || k === 4)) return
  const tones = shared.chordAt(song.section, song.bar, k).tones(62, 86)
  note(tones[order[k] % tones.length], (k === 0 || k === 3 ? 0.65 : 0.45) * lv, 1)
}
