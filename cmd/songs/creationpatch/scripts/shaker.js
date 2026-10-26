// eggs and maracas under the days, happy and tropical: the waters hush, the plants grow in eighths, then the shaker dances on the offbeat (an accent on the "and" of every beat,
// the maraca on the beat) through the sun, the creatures and mankind, and keeps going in the Doxology
clock("1/16")
const k = shared.kits.shaker
const part = shared.register("shaker", { legato: 1, raw: true }, day => {
  const n = day.number
  return shared.groove(day, (b, br) => {
    const kinds = { e: k.egg, m: k.maraca }
    if (n === 2 && b >= 4) return { pattern: b >= 8 ? "m.e.m.e.m.e." : "m...m...m...", kinds, vel: 0.3 }
    if (n === 3 && b >= 2) return { pattern: "m.E.m.E.m.E.", kinds, vel: 0.32 }
    if (n === 4) return { pattern: "meEemeEemeEe", kinds, vel: 0.34 }
    if (n === 5) return { pattern: "meEemeEemeEe", kinds, vel: 0.36 }
    if (n === 6) return { pattern: b >= 6 ? "EeEeEeEeEeEe" : "m.E.m.E.m.E.", kinds, vel: 0.36 }
    if (n === 7 && b >= 2 && b < 16) return { pattern: "m.E.m.E.m.E.", kinds, vel: 0.3 }
    return null
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
