// the flute plays the head, the written melody (A A B A, thirty-two bars), in the head and in the head out; in its solo (the fifth section) and the tag it sings a tune made
// for every bar (see islandVoice in score.js), swung; in the solos and the break it answers the motif with the marimba
clock("1/16")
const part = shared.register("flute", { legato: 0.95, vel: 0.55 }, day => {
  if (day.number === 2 || day.number === 6) return shared.head()
  // from the first solo on it answers the motif too, where its own solo leaves room (see shared.statements)
  if (day.number >= 3 && day.number <= 5) return shared.motifAt(day, shared.statements[day.number][0], shared.statements[day.number][1], true, bar => !shared.layers.flute(day, bar + 2) && !shared.layers.flute(day, bar + 3))
  return null
})
function section(name) { part.section(name) }
const tune = shared.islandVoice("flute", { lag: 0, lowest: 71, vel: 1, on: shared.layers.flute })
function trigger(ev) {
  part.trigger(song, note, ev)
  tune(song, note, ev)
}
