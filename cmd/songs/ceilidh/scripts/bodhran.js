// BODHRAN: the pulse of a jig, a hit on the first eighth (the low one, the middle of the skin), a light one on the
// third, a medium one on the fourth and a light one on the sixth; every eighth bar ends in a fill that climbs.
function trigger(ev) {
  const lv = shared.level("bodhran", song.section, song.bar)
  if (!lv) return
  const k = ev.step % 6
  const fill = song.bar % 8 === 7 && song.section !== "Outro"
  if (k === 0) note(48, 0.9 * lv, 1)
  else if (k === 2) note(55, 0.35 * lv, 0.5)
  else if (k === 3) note(52, (fill ? 0.55 : 0.65) * lv, 1)
  else if (k === 5) note(57, 0.35 * lv, 0.5)
  else if (fill && k === 4) note(55, 0.5 * lv, 0.5)
}
