// the voices hum the chords (a female choir, the sound of a mother): they come in at the middle of the knitting, sing the hymn's harmony in the Wonder, hum low in the shadow, and at the
// end hold the last chord, D, with the cello and the alto flute
clock("1/16")
const part = shared.part("choir", { vel: 0.4, legato: 0.99 }, sec => {
  switch (sec.name) {
    case "Knit": return shared.holdChords(sec.name, 57, 8, 16, { vel: 0.3 })
    case "Wonder": return shared.holdChords(sec.name, 57, 0, 16, { vel: 0.4 })
    case "Shadow": return shared.holdChords(sec.name, 55, 4, 12, { vel: 0.3 })
    case "Light": return shared.holdChords(sec.name, 57, 0, 12, { vel: 0.36 })
    case "Awake": return shared.holdChords(sec.name, 57, 8, 12, { vel: 0.4 })
  }
  return []
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
