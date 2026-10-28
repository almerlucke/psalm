// the piano is the floor: the root of the chord, then its fifth, low and soft, a note that rings and is let go (the satie preset). In the Wonder and the Light it moves in quarters
// through the chord, a figure that goes round; at the end it lays the D and the A and leaves them
clock("1/16")
const part = shared.part("piano", { vel: 0.5, legato: 3.2 }, sec => {
  const out = []
  const q = shared.beats
  for (let b = 0; b < sec.bars; b++) {
    const c = shared.chordFor(sec.name, b)
    const root = c.bass(38)
    const at = b * q
    switch (sec.name) {
      case "Thread":
        if (b >= 2) out.push({ at, p: root, beats: 2, vel: 0.34 })
        if (b >= 4) out.push({ at: at + 2, p: root + 7, beats: 2, vel: 0.26 })
        break
      case "Knit":
        out.push({ at, p: root, beats: 2, vel: 0.4 })
        out.push({ at: at + 2, p: root + 7, beats: 2, vel: 0.3 })
        if (b >= 8) out.push({ at: at + 3, p: c.close(52)[1], beats: 1, vel: 0.24 })
        break
      case "Wonder":
      case "Light": {
        const t = c.close(43)
        const run = [t[0], t[1], t[2], t[1]]
        run.forEach((p, i) => out.push({ at: at + i, p, beats: 1, vel: i === 0 ? 0.42 : 0.3 }))
        break
      }
      case "Shadow":
        if (b % 2 === 0) out.push({ at, p: root, beats: 4, vel: 0.32 })
        break
      case "Awake":
        if (b < 8) { if (b % 2 === 0) out.push({ at, p: root, beats: 4, vel: 0.3 * (1 - b / 12) }) }
        else if (b === 8) { out.push({ at, p: 38, beats: 4, vel: 0.34 }); out.push({ at: at + 1, p: 45, beats: 3, vel: 0.24 }) }
        break
    }
  }
  return out
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
