// the Word is spoken twice into the darkness of the intro: John 1:1 (the voice, which the vocoder says through the pad of open fifths)
clock("1/16")
let said = 0
function section(name) { if (name === shared.days[0].name) said = 0 }
function trigger(ev) {
  if (song.index !== 0) return
  if ((said === 0 && song.bar >= 1) || (said === 1 && song.bar >= 3)) { said++; note(36, 1, 4) }
}
