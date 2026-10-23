// the waters flow on the morning of the second day
clock("1/16")
const part = shared.register("harp", { legato: 1.6 }, day => {
  if (day.number !== 2) return null
  return shared.perBar(day, (b, br) => shared.morning(b) ? shared.arp([br.root + 12, ...br.mid]) : null)
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
