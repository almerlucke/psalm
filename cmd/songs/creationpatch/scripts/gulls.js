// the gulls come in over the sea in the morning of the fifth day
clock("1/16")
let started = false
function section(name) { if (name === shared.days[4].name) started = false }
function trigger(ev) {
  if (!started && song.index === 4 && song.bar >= 5) { started = true; note(36, 1, 7 * 3 * 4) }
  mod(song.index === 4 ? shared.fade(-16, shared.at(song), shared.EVENING, 12) / 2 : 0)
}
