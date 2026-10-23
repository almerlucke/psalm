// the bells ring once, as the Amen comes home
clock("1/16")
const part = shared.register("bells", { legato: 1 }, day => {
  return day.number === 7 ? shared.join(shared.rest(shared.FINAL), [shared.chord([63, 70, 75], 12, { hold: 12, vel: 0.5 })]) : null
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
