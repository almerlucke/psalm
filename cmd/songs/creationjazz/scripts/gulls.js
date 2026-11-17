// the gulls over the sea of the heavens, under the solo of the flute
clock("1/16")
let started = false
function section(name) { if (name === shared.days[4].name) started = false }
function trigger(ev) {
  if (!started && song.index === 4 && song.bar >= 0) { started = true; note(36, 1, 288) }
  mod(song.index === 4 ? shared.fade(-18, shared.at(song), 0, 25) / 2 : 0)
}
