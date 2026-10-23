// the celli: the waters below on the second day, very low; the fifths above the root on the land, the creatures and mankind; the seventh day's chord ringing
clock("1/16")
const part = shared.register("celli", { legato: 1 }, day => {
  return shared.perBar(day, (b, br) => {
    switch (day.number) {
      case 2: return [shared.chord([br.root, br.root + 7], shared.beats)]
      case 3: case 5: case 6: return [shared.chord([br.root + 12, br.root + 19], shared.beats)]
      case 7: return shared.ringing(b, [br.root + 12, br.root + 19])
    }
    return null
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
