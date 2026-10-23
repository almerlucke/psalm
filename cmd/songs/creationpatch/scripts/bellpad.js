// a pad of bells under the stars
clock("1/16")
const part = shared.register("bell pad", { legato: 1 }, day => {
  return shared.perBar(day, (b, br) => {
    if ((day.number === 1 && shared.morning(b)) || (day.number === 4 && !shared.morning(b))) return [shared.chord(br.mid, shared.beats)]
    return null
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note) }
