// "and the Spirit of God moved upon the face of the waters": a breathing choir in the darkness, the heaven between the waters on the second morning, the
// morning stars on the fourth, and the seventh day
clock("1/16")
const part = shared.register("spirit", { legato: 1 }, day => {
  return shared.perBar(day, (b, br) => {
    switch (day.number) {
      case 1:
        if (shared.morning(b)) return [shared.chord(br.mid, shared.beats)]
        return [shared.chord([51, 58], shared.beats, { vel: 0.6, amp: shared.DARK })]
      case 2: case 4:
        return shared.morning(b) ? [shared.chord(br.mid, shared.beats)] : null
      case 7: return [shared.chord(br.mid, shared.beats)]
    }
    return null
  })
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note) }
