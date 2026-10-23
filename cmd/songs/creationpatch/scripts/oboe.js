// the oboe follows the flute two bars later, in the canon of the plants
clock("1/16")
const part = shared.register("oboe", { legato: 0.95 }, day => {
  return day.number === 3 ? shared.join(shared.rest(8), shared.seedA(), shared.seedB()) : null
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note) }
