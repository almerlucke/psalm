// "be fruitful, and multiply" (Genesis 1:28): the children laugh once, through the last four bars of the sixth day
clock("1/16")
let started = false
function section(name) { if (name === shared.days[5].name) started = false }
function trigger(ev) {
  if (!started && song.index === 5 && song.bar >= 8) { started = true; note(36, 1, 4) }
  mod(song.index === 5 ? shared.fade(-3, shared.at(song), 8, 12) / 2 : 0)
}
