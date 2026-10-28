// the cello is the third voice, an octave below, and the ground: the root of the chord held through a bar when it does not sing. In the Thread it holds the D under the flute; in
// the Knit it sings the tune from bar 8; in the Shadow it has the tune of the shadow alone
clock("1/16")
const part = shared.part("cello", { vel: 0.55, legato: 0.98 }, sec => {
  const roots = (from, to, vel) => {
    const out = []
    for (let b = from; b < to;) {
      const c = shared.chordFor(sec.name, b)
      let n = 1
      while (b + n < to && shared.chordFor(sec.name, b + n) === c) n++
      out.push({ at: b * shared.beats, p: c.bass(36), beats: n * shared.beats, vel })
      b += n
    }
    return out
  }
  switch (sec.name) {
    case "Thread": return [{ at: 4 * shared.beats, p: 38, beats: 4 * shared.beats, vel: 0.4 }]
    case "Knit": return roots(4, 8, 0.4).concat(shared.phrases(shared.alternate(8, 16, "A"), -12, { vel: 0.6 }))
    case "Wonder": return roots(0, 16, 0.5)
    case "Shadow": return shared.phrases([[0, "S"], [4, "S"], [8, "S"]], -12, { vel: 0.6 })
    case "Light": return roots(0, 8, 0.45).concat(shared.phrases(shared.alternate(8, 12, "A"), -12, { vel: 0.6 }))
    case "Sand": return roots(0, 16, 0.36)
    case "Awake": return [{ at: 0, p: 38, beats: 8 * shared.beats, vel: 0.36 }, { at: 8 * shared.beats, p: 38, beats: 4 * shared.beats, vel: 0.45, legato: 1.2 }]
  }
  return []
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
