// the creatures of the sea and the birds, on the fifth day: a recording of the sea with birds, going round, faded in and out over the day
clock("1/16")
let started = false
function section(name) { if (name === shared.days[4].name) started = false }
function trigger(ev) {
  if (!started && song.index === 4) { started = true; note(36, 1, 12 * 3 * 4) }
  mod(song.index === 4 ? shared.fade(-8, shared.at(song), 0, 12) / 2 : 0)
}
