// the alto flute is the dark voice: in the Shadow it sings the tune of the shadow four bars behind the cello; in the Light the second half an octave below; and at the end it is
// the third of the last chord
clock("1/16")
const part = shared.part("alto flute", { vel: 0.55, legato: 0.97 }, sec => {
  switch (sec.name) {
    case "Shadow": return shared.phrases([[4, "S"], [8, "S"]], 0, { vel: 0.5 })
    case "Light": return shared.phrases([[4, "B"], [8, "A"]], -12, { vel: 0.55 })
    case "Awake": return [{ at: 8 * shared.beats, p: 66, beats: 4 * shared.beats, vel: 0.4, legato: 1.1 }]
  }
  return []
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
