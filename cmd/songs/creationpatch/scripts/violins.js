// the violins: a chord in the morning of the first, fourth and sixth days; the waters above on the second day, very high; the last chord ringing on the seventh
clock("1/16")
const part = shared.register("violins", { legato: 1 }, day => {
  return shared.perBar(day, (b, br) => {
    switch (day.number) {
      case 1: case 4: case 6:
        if (shared.morning(b)) return [shared.chord(br.high, shared.beats)]
        return null
      case 2: return [shared.chord(shared.pitches(br.high, 12), shared.beats)]
      case 7: return shared.ringing(b, br.high)
    }
    return null
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
