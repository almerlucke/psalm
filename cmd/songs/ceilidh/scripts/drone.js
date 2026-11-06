// DRONE: a hurdy-gurdy on a G and a D, a fifth, whatever the chords do, said again every two bars so that it follows the fade of the outro.
function trigger(ev) {
  if (ev.step % 6 !== 0 || song.bar % 2 !== 0) return
  const lv = shared.level("drone", song.section, song.bar)
  if (!lv) return
  note([43, 50, 55], 0.6 * lv, 12)
}
