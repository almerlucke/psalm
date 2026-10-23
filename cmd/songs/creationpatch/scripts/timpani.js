// "Let there be light"; the dry land appears and the beasts tread; the Amen
clock("1/16")
const part = shared.register("timpani", { legato: 1 }, day => {
  return shared.perBar(day, (b, br) => {
    if (day.number === 1 && b === shared.EVENING) return [shared.chord([39], 3, { vel: 0.9 })]
    if (shared.in(day, 3, 6) && !shared.morning(b)) {
      let r = br.root
      while (r < 36) r += 12
      return [shared.chord([r], 2, { vel: 0.6 }), shared.chord([r], 1, { vel: 0.4 })]
    }
    if (day.number === 7 && b === shared.FINAL) return [shared.chord([39], 3, { vel: 0.6 })]
    return null
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
