// a cymbal swells where a day turns to its morning (the light, the sun, man), and once when the last chord comes home
clock("1/16")
const k = shared.kits.cymbal
const part = shared.register("cymbal", { legato: 1, raw: true }, day => {
  const n = day.number
  return shared.perBar(day, (b, br) => {
    const at = (n === 1 || n === 4 || n === 6) && b === shared.EVENING, home = n === 7 && b === shared.FINAL
    if (!at && !home) return null
    return [shared.N([k.swell[0]], home ? 12 : 3, { vel: home ? 0.4 : n === 1 ? 0.4 : 0.25, hold: 3 })]
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
