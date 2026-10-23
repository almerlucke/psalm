// Adam answers Eve with the figure of the cross; on the seventh day the cross leads through the Amen into the rest
clock("1/16")
const part = shared.register("horn", { legato: 0.95 }, day => {
  switch (day.number) {
    case 6: return shared.join(shared.rest(6), shared.answer())
    case 7: return shared.join(shared.rest(shared.FINAL - 1), shared.cross(), shared.rest(3))
  }
  return null
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note) }
