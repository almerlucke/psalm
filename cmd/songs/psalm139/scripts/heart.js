// the heart: a lub and a dub on every beat, the D at the bottom of the piece and the A above it, the lub a little stronger. It is there where there is a mother's womb to hear
// it in: under the first threads, the knitting, the shadow and at the end, when the sleeper wakes and is still with Him
clock("1/16")
const play = shared.player("heart")
const level = {
  // the loudness of the beat in a bar of the section (null: no heart)
  Thread: b => (b < 2 ? null : 0.1 + 0.2 * (b - 2) / 5),
  Knit: b => (b < 12 ? 0.3 - 0.1 * b / 12 : 0.2 * (15 - b) / 3),
  Shadow: () => 0.14,
  Sand: b => (b < 12 ? null : 0.18 * (b - 11) / 4),
  Awake: b => (b < 8 ? 0.26 : 0.26 * (11 - b) / 4),
}
let current = null, last = -1
function section(name) { current = level[name] ? shared.sec(name) : null; last = -1; if (current) current.level = level[name] }
function trigger(ev) {
  if (!current) return
  const k = shared.sixteenth(song)
  if (k === last || k % 4 !== 0) return
  last = k
  const v = current.level(song.bar)
  if (!v || v <= 0) return
  play(note, 38, Math.min(1, v * 1.1 * current.dyn / 0.5), 3, ev)
  note(shared.tune(45), Math.min(1, v * 0.65 * current.dyn / 0.5), 3, 1.5) // the dub, a moment after the lub
}
