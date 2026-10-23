// the Word sounds through open fifths, held through the first evening (the pad the voice of John 1:1 is spoken through, in the vocoder)
clock("1/16")
const part = shared.register("the word", { legato: 1 }, day => {
  return day.number === 1 ? shared.join(shared.rest(2), [shared.chord([39, 46, 51, 58], 12, { amp: shared.DARK })], shared.rest(6)) : null
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
