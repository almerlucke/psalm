// the harp is the warp: eighths that go up and down the chord in a zigzag, over and under (0 2 1 3 2 4 3 5), up in one bar and down in the next, every bar on the chord
// of the bar. It is what stays while the voices of the weft pass through: sparse in the Thread, a bass note only; full in the Knit; wide in the Wonder; half the notes in the Shadow
clock("1/16")
const ZIG = [0, 2, 1, 3, 2, 4, 3, 5]
const WIDE = [0, 3, 1, 4, 2, 5, 3, 6]
const ACCENT = [1, 0.6, 0.75, 0.6, 0.85, 0.6, 0.75, 0.6]
const how = {
  Thread: b => (b < 2 ? "bass" : b < 4 ? "half" : "full"),
  Knit: () => "full",
  Wonder: () => "full",
  Shadow: b => (b < 4 ? "bass" : "half"),
  Light: () => "full",
  Sand: () => "half",
  Awake: b => (b < 6 ? "half" : b < 9 ? "bass" : null),
}
const part = shared.part("harp", { vel: 0.62, legato: 2.4 }, sec => {
  const out = []
  for (let b = 0; b < sec.bars; b++) {
    const mode = how[sec.name](b)
    if (!mode) continue
    const wide = sec.name === "Wonder" || sec.name === "Light"
    const pattern = wide ? WIDE : ZIG
    const pool = shared.chordFor(sec.name, b).tones(50, wide ? 79 : 71)
    if (pool.length <= Math.max(...pattern)) continue
    const order = b % 2 ? [...pattern].reverse() : pattern
    const idx = mode === "bass" ? [pattern[0]] : mode === "half" ? order.filter((_, i) => i % 2 === 0) : order
    const gap = mode === "full" ? 0.5 : mode === "half" ? 1 : 0
    idx.forEach((j, i) => {
      const accent = mode === "full" ? ACCENT[i] : mode === "half" ? [1, 0.7, 0.85, 0.7][i] : 1
      out.push({ at: b * shared.beats + i * gap, p: pool[j], beats: 1, vel: 0.62 * accent })
    })
  }
  return out
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
