// the oboe follows the flute two bars later, in the canon of the plants
clock("1/16")
const part = shared.register("oboe", { legato: 0.95 }, day => {
  return day.number === 3 ? shared.join(shared.rest(8), shared.seedA(), shared.seedB()) : null
})
function section(name) { part.section(name) }
const island = shared.islandVoice("oboe", { lag: 2, lowest: 71, vel: 0.85, on: shared.layers.oboe })
function trigger(ev) {
  part.trigger(song, note, ev)
  island(song, note, ev)
}
