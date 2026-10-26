// the clarinet two bars after the oboe, an octave down
clock("1/16")
const part = shared.register("clarinet", { legato: 0.95 }, day => {
  return day.number === 3 ? shared.join(shared.rest(10), shared.transpose(shared.seedA(), -12)) : null
})
function section(name) { part.section(name) }
const island = shared.islandVoice("clarinet", { lag: 4, lowest: 59, vel: 0.8, on: shared.layers.clarinet })
function trigger(ev) {
  part.trigger(song, note, ev)
  island(song, note, ev)
}
