// the violas hold the chords under the voices, voice led, softly: they come in with the third voice of the knitting, fill the Wonder, lie low under the shadow
clock("1/16")
const part = shared.part("strings", { vel: 0.4, legato: 0.99 }, sec => {
  switch (sec.name) {
    case "Knit": return shared.holdChords(sec.name, 50, 8, 16, { vel: 0.34 })
    case "Wonder": return shared.holdChords(sec.name, 50, 0, 16, { vel: 0.42 })
    case "Shadow": return shared.holdChords(sec.name, 47, 4, 12, { vel: 0.34 })
    case "Light": return shared.holdChords(sec.name, 50, 0, 12, { vel: 0.4 })
    case "Sand": return shared.holdChords(sec.name, 50, 0, 16, { vel: 0.24 })
    case "Awake": return shared.holdChords(sec.name, 50, 8, 12, { vel: 0.36 })
  }
  return []
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
