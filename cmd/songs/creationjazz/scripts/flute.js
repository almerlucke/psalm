// the flute plays the head, the written melody (A A B A, thirty-two bars), in the head and in the head out; in its solo (the fifth section) and the tag it sings a tune made
// for every bar (see islandVoice in score.js), swung
clock("1/16")
const part = shared.register("flute", { legato: 0.95, vel: 0.55 }, day => (day.number === 2 || day.number === 6 ? shared.head() : null))
function section(name) { part.section(name) }
const tune = shared.islandVoice("flute", { lag: 0, lowest: 71, vel: 1, on: shared.layers.flute })
function trigger(ev) {
  part.trigger(song, note, ev)
  tune(song, note, ev)
}
