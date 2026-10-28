// plucked strings, a small thread among the stars: in the Sand five hits in sixteen, turned three steps every bar, walking the notes of the chord around the middle of the range;
// quieter in the Light. A thread that comes back to its place only every sixteen bars
clock("1/16")
const rand = shared.rng(shared.seed || 1, "pizzicato-walk")
const part = shared.part("pizzicato", { vel: 0.5, legato: 1 }, sec => {
  const out = []
  let pos = 4
  if (sec.name !== "Sand" && sec.name !== "Light") return out
  for (let b = 0; b < sec.bars; b++) {
    if (sec.name === "Light" && b < 4) continue
    const tones = shared.chordFor(sec.name, b).tones(60, 84)
    genny.euclid(5, 16, b * 3).take(16).forEach((hit, i) => {
      if (!hit) return
      pos = Math.max(0, Math.min(tones.length - 1, pos + Math.round((rand() - 0.5) * 3)))
      out.push({ at: b * shared.beats + i / 4, p: tones[pos], beats: 0.5, vel: (sec.name === "Light" ? 0.22 : 0.32) + 0.14 * rand() })
    })
  }
  return out
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
