// the land walks: the dry land on the evening of the third day, the beasts on the evening of the sixth
clock("1/16")
const part = shared.register("pizzicato", { legato: 0.5 }, day => {
  return shared.perBar(day, (b, br) => {
    if (!shared.in(day, 3, 6) || shared.morning(b)) return null
    const r = br.root + 12
    return shared.notes(1, r, r + 7, r + 12)
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
