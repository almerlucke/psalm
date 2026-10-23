// the beasts of the earth, on the sixth day: a recording of animals going round, over the evening
clock("1/16")
let started = false
function section(name) { if (name === shared.days[5].name) started = false }
function trigger(ev) {
  if (!started && song.index === 5) { started = true; note(36, 1, 7 * 3 * 4) }
  mod(song.index === 5 ? shared.fade(-10, shared.at(song), 0, 6.6) / 2 : 0)
}
