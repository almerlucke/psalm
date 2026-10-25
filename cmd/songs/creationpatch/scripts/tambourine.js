// the tambourine: the sun on the second and third beat, the creatures dancing, the beasts, then a soft jingle of praise
clock("1/16")
const k = shared.kits.tambourine
const part = shared.register("tambourine", { legato: 1, raw: true }, day => {
  const n = day.number
  return shared.groove(day, (b, br) => {
    const kinds = { h: k.hit, j: k.jingle }
    if (n === 4 && b >= 6) return { pattern: "....h...h...", kinds, vel: 0.4 }
    if (n === 5 && b >= 6) return { pattern: "....h...h.jh", kinds, vel: 0.42 }
    if (n === 6 && b >= 6) return { pattern: "H..j.hH..j.h", kinds, vel: 0.42 }
    if (n === 7 && b >= 4 && b < 16) return { pattern: "....j...j...", kinds, vel: 0.3 }
    return null
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
