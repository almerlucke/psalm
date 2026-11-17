// the steel pan solos over the changes of the third section (a tune made for every bar, a euclidean seven in twelve, a walk on the tones of the chord), follows the
// flute two bars after it in the second half of its solo, and doubles the head an octave above in the head out
clock("1/16")
const part = shared.register("steel pan", { legato: 0.9, vel: 0.5 }, day => (day.number === 6 ? shared.transpose(shared.head(), 12) : null))
function section(name) { part.section(name) }
const solo = shared.islandVoice("steel pan", { lag: 0, lowest: 66, vel: 1, on: (d, b) => d.number === 3 })
const follow = shared.islandVoice("steel pan", { lag: 4, lowest: 62, vel: 0.9, on: (d, b) => d.number === 5 && b >= 8 })
function trigger(ev) {
  part.trigger(song, note, ev)
  solo(song, note, ev)
  follow(song, note, ev)
}
