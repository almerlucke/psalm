// the stars of the first morning and of the fourth night
clock("1/16")
const part = shared.register("celesta", { legato: 0.9 }, day => {
  return shared.perBar(day, (b, br) => {
    if ((day.number === 1 && shared.morning(b)) || (day.number === 4 && !shared.morning(b))) return shared.stars(br, b)
    return null
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note) }
