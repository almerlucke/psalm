// the hand drums: the creatures' bongos, the beasts treading on the low drums, the dance of man, the clapping of hands in the Doxology
clock("1/16")
const k = shared.kits.hands
const part = shared.register("hands", { legato: 1, raw: true }, day => {
  const n = day.number
  return shared.groove(day, (b, br) => {
    const kinds = { b: k.bongo, c: k.conga, l: k.low, j: k.cajon, p: k.clap }
    if (n === 5 && b >= 6) return { pattern: b % 2 ? "b..b.b..b.bb" : "b.b..b.b..b.", kinds, vel: 0.4 }
    if (n === 6 && b < 6 && b >= 2) return { pattern: "L.....l.....", kinds, vel: 0.5 }
    if (n === 6 && b >= 6) return { pattern: b % 4 === 3 ? "L..c.c.L.cbb" : "L..c.c.l.cb.", kinds, vel: 0.46 }
    if (n === 7 && b >= 4 && b < 16) return { pattern: "....p...p...", kinds, vel: 0.34 }
    return null
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
