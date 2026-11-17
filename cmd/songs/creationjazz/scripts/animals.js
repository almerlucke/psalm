// the beasts of the earth in the break of the percussion: a recording of animals going round, faded in from the third bar
clock("1/16")
let started = false
function section(name) { if (name === shared.days[3].name) started = false }
function trigger(ev) {
  if (!started && song.index === 3 && song.bar >= 0) { started = true; note(36, 1, 192) }
  mod(song.index === 3 ? shared.fade(-14, shared.at(song), 2, 17) / 2 : 0)
}
