// Eve sings the theme of man; on the seventh day the soprano leads the Doxology
clock("1/16")
const part = shared.register("soprano", { legato: 0.95 }, day => {
  switch (day.number) {
    case 6: return shared.join(shared.rest(6), shared.theme())
    case 7: return shared.join(shared.doxology(), shared.rest(5))
  }
  return null
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note) }
