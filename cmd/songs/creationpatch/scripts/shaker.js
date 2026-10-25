// eggs and maracas under the days: the waters hush, the plants grow in eighths, the lights tick, the creatures shuffle, a soft breath in the Doxology
clock("1/16")
const k = shared.kits.shaker
const part = shared.register("shaker", { legato: 1, raw: true }, day => {
  const n = day.number
  return shared.groove(day, (b, br) => {
    const kinds = { e: k.egg, m: k.maraca }
    if (n === 2 && b >= 8) return { pattern: "m...m...m...", kinds, vel: 0.3 }
    if (n === 3 && b >= 6) return { pattern: "e.e.e.e.e.e.", kinds, vel: 0.3 }
    if (n === 4 && b >= 6) return { pattern: "E.e.e.e.e.e.", kinds, vel: 0.32 }
    if (n === 5 && b >= 3) return { pattern: b >= 6 ? "E.ee.eE.ee.e" : "e.....e.....", kinds, vel: 0.34 }
    if (n === 6) return { pattern: b >= 6 ? "EeeeEeeeEeee" : "e.e.e.e.e.e.", kinds, vel: 0.3 }
    if (n === 7 && b >= 2 && b < 16) return { pattern: "m.....m.....", kinds, vel: 0.28 }
    return null
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
