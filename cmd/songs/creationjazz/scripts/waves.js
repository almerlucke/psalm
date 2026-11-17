// the waters of the first chapter, under the intro: a recording of waves, faded in on the first bar and carried to the head, where the music takes over
clock("1/16")
let started = false
function section(name) { if (name === shared.days[0].name) started = false }
function trigger(ev) {
  if (!started && song.index === 0 && song.bar >= 0) { started = true; note(36, 1, 96) }
  mod(song.index === 0 ? shared.fade(-18, shared.at(song), 0, 9) / 2 : 0)
}
