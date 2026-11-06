// DRUMS: an electro kit (the drumsynth) under the jig, in two layers; the kick is another script (kick.js), so that it can drive the ducking. The jig layer:
// a clap on the fourth, a ghost rim on the third of every second bar, hats on every eighth with the accents on the first and the
// fourth, and an open one on the sixth of every second bar; every eighth bar ends in a roll of the snare.
// The modern layer, in the second half of the First and in the Second and the Third (more of it in the Third): sixteenth hats between
// the eighths (a half step later, with the velocity going up and down), a snare under the clap, a ghost snare just before the
// first beat of the next half bar, rolls of the hat that are
// three notes in a step (trap), and a shaker. Which of them play, and how hard, is the layers of the score (snare, hats).
function trigger(ev) {
  const k = ev.step % 6, bar = song.bar, s = song.section
  const snare = shared.level("snare", s, bar), hats = shared.level("hats", s, bar)
  const modern = (s === "First" && bar >= 16) || s === "Second" || s === "Third"
  const full = s === "Third"
  if (snare) {
    if (k === 3) {
      note(39, 0.8 * snare, 1)
      if (modern) note(38, 0.75 * snare, 1)                                    // a snare under the clap
    }
    if (k === 2 && bar % 2 === 0) note(37, 0.3 * snare, 1)
    if (modern && k === 2) note(38, 0.2 * snare, 0.5, 0.5)                     // ghost snares before the back beats
    if (full && k === 5) note(38, 0.18 * snare, 0.5, 0.5)
    if (bar % 8 === 7 && k >= 3) {
      note(38, (0.4 + 0.18 * (k - 3)) * snare, 0.5)
      if (modern) note(38, (0.35 + 0.18 * (k - 3)) * snare, 0.4, 0.5)          // the roll in sixteenths
    }
  }
  if (hats) {
    note(42, (k === 0 || k === 3 ? 0.7 : 0.45) * hats, 0.5)
    if (k === 5 && bar % 2 === 1) note(46, 0.4 * hats, 1)
    if (modern) {
      // the sixteenth between this eighth and the next, soft and a little different every time
      const v = [0.25, 0.33, 0.27, 0.38, 0.26, 0.42][(k + bar) % 6]
      note(42, v * hats, 0.3, 0.5)
      // rolls: three hats in a step, rising (trap), at the end of every fourth bar, and of every second one in the Third
      const roll = k === 4 ? (full ? bar % 2 === 1 : bar % 4 === 3) : k === 5 && full && bar % 8 === 7
      if (roll) [0, 1, 2].forEach(i => note(42, (0.25 + 0.12 * i) * hats, 0.3, i / 3))
      // a shaker on the sixteenths of the Third
      if (full && k % 2 === 1) note(70, 0.22 * hats, 0.3, 0.5)
    }
  }
}
