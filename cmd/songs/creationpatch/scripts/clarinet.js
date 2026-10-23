// the clarinet two bars after the oboe, an octave down
clock("1/16")
const part = shared.register("clarinet", { legato: 0.95 }, day => {
  return day.number === 3 ? shared.join(shared.rest(10), shared.transpose(shared.seedA(), -12)) : null
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
