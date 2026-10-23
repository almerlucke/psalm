// the great creatures of the sea, low in the deep (the evening of the fifth day)
clock("1/16")
const part = shared.register("the deep", { legato: 1 }, day => {
  if (day.number !== 5) return null
  return shared.perBar(day, (b, br) => shared.morning(b) ? null : [shared.chord([br.root], 3, { hold: 3.2 })])
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note) }
