// claves and castanets: the first shoots of the plants, the stars ticking, the birds chirping, the beasts, the Doxology's quiet steps
clock("1/16")
const k = shared.kits.wood
const part = shared.register("wood", { legato: 1, raw: true }, day => {
  const n = day.number
  return shared.groove(day, (b, br) => {
    const kinds = { c: k.clave, t: k.castanet }
    if (n === 3) return b >= 6 ? { pattern: "c.....c..t..", kinds, vel: 0.4 } : b >= 3 && b % 2 === 1 ? { pattern: "c...........", kinds, vel: 0.35 } : null
    if (n === 4 && b >= 6 && b % 2 === 0) return { pattern: "..c...c...c.", kinds, vel: 0.4 }
    if (n === 5 && b >= 6) return { pattern: b % 2 ? "..t.tt..t..." : "t.tt....t.t.", kinds, vel: 0.4 }
    if (n === 6 && b >= 6) return { pattern: "t..t..t.tt..", kinds, vel: 0.4 }
    if (n === 7 && b >= 6 && b < 16 && b % 2 === 0) return { pattern: "c.....t.....", kinds, vel: 0.3 }
    return null
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
