// the plants in canon on the morning of the third day, the moon on the evening of the fourth, the birds in the morning of the fifth
clock("1/16")
const part = shared.register("flute", { legato: 0.95 }, day => {
  switch (day.number) {
    case 3: return shared.join(shared.rest(6), shared.seedA(), shared.seedB(), shared.seedC())
    case 4: return shared.join(shared.moon(), shared.rest(6))
    case 5: return shared.perBar(day, (b, br) => shared.morning(b) ? shared.birdsong(br, b % 2 === 1) : null)
  }
  return null
})
function section(name) { part.section(name) }
const island = shared.islandVoice("flute", { lag: 0, lowest: 71, vel: 1, on: shared.layers.flute })
function trigger(ev) {
  part.trigger(song, note, ev)
  island(song, note, ev)
}
