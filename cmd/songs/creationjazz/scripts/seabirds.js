// the sea and the birds under the steel pan's solo (the waters gathered, the land appearing): a recording going round, faded in over the first bar
clock("1/16")
let started = false
function section(name) { if (name === shared.days[2].name) started = false }
function trigger(ev) {
  if (!started && song.index === 2 && song.bar >= 0) { started = true; note(36, 1, 384) }
  mod(song.index === 2 ? shared.fade(-16, shared.at(song), 0, 33) / 2 : 0)
}
