// the waters: a recording of waves, faded in on the first evening (-14 dB), carried through the second day and the evening of the third (-8 dB). The note holds
// the recording going round for all of it; the mod output is the level of its strip (0 to 1 over the range 0 to 2 of the level, a gain of 1 is 0.5)
clock("1/16")
const DAYS = 30 * 3 * 4 // steps: the first two days and the first six bars of the third
let started = false
function section(name) { if (name === shared.days[0].name) started = false }
function trigger(ev) {
  if (!started && song.section === shared.days[0].name && song.bar === 0) { started = true; note(36, 1, DAYS) }
  const b = shared.at(song)
  let gain = 0
  switch (song.index) {
    case 0: gain = shared.fade(-14, b, 0, shared.EVENING); break
    case 1: gain = shared.fade(-8, b, 0, 13); break
    case 2: gain = shared.fade(-8, b, -1, shared.EVENING); break
  }
  mod(gain / 2)
}
