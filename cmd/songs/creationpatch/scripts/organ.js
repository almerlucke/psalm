// the organ plays the Doxology in four parts and holds the Amen
clock("1/16")
const part = shared.register("organ", { legato: 1 }, day => {
  if (day.number !== 7) return null
  return shared.perBar(day, (b, br) => shared.ringing(b, [br.root, ...br.mid]))
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note) }
