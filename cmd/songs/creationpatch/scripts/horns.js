// light breaks, the sun rises (the mornings of the first and fourth days); the beasts (the evening of the sixth)
clock("1/16")
const part = shared.register("horns", { legato: 0.95 }, day => {
  return shared.perBar(day, (b, br) => {
    if ((shared.in(day, 1, 4) && shared.morning(b)) || (day.number === 6 && !shared.morning(b))) {
      return [shared.chord(shared.pitches(br.mid, -12), shared.beats)]
    }
    return null
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
