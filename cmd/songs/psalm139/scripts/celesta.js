// the celesta is the light: stars high above the chord, a few in a bar in the Wonder (marvellous are thy works); in the Sand a cycle of seven hits in sixteen that walks up and down
// the high notes and is turned a step every bar, so the stars are never in the same place twice; one bell at the very end
clock("1/16")
const rand = shared.rng(shared.seed || 1, "celesta-stars")
const part = shared.part("celesta", { vel: 0.5, legato: 4 }, sec => {
  const out = []
  const q = shared.beats
  let pos = 3
  for (let b = 0; b < sec.bars; b++) {
    const c = shared.chordFor(sec.name, b)
    const high = c.tones(79, 98)
    const at = b * q
    switch (sec.name) {
      case "Knit":
        if (b >= 12) out.push({ at: at + 2, p: high[(b * 2) % high.length], beats: 2, vel: 0.26 })
        break
      case "Wonder":
        if (b % 2 === 0) { out.push({ at: at + 1.5, p: high[1 % high.length], beats: 2, vel: 0.5 }); out.push({ at: at + 2.25, p: high[3 % high.length], beats: 2, vel: 0.36 }) }
        else out.push({ at: at + 3.5, p: high[2 % high.length], beats: 1, vel: 0.4 })
        break
      case "Light":
        if (b % 2 === 1) out.push({ at: at + 2.5, p: high[2 % high.length], beats: 2, vel: 0.34 })
        break
      case "Sand": {
        const tones = c.tones(74, 100)
        genny.euclid(7, 16, b).take(16).forEach((hit, i) => {
          if (!hit) return
          pos = clamp(pos + Math.round((rand() - 0.5) * 4), 0, tones.length - 1)
          out.push({ at: at + i / 4, p: tones[pos], beats: 1, vel: 0.2 + 0.2 * rand() })
        })
        break
      }
      case "Awake":
        if (b === 11) out.push({ at: at + 0.5, p: 86, beats: 4, vel: 0.3, legato: 3 })
        break
    }
  }
  return out
})
function clamp(x, lo, hi) { return Math.max(lo, Math.min(hi, x)) }
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
