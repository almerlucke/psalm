// the choir: the light, the morning stars, man, the Doxology
clock("1/16")
const part = shared.register("choir", { legato: 1 }, day => {
  return shared.perBar(day, (b, br) => {
    if ((day.number === 1 || day.number === 4 || day.number === 6) && shared.morning(b)) return [shared.chord(br.mid, shared.beats)]
    if (day.number === 7) return shared.ringing(b, br.mid)
    return null
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note) }
