// SHAKE: the cymbal and the long shake of a tambourine (it fades away in five seconds). The cymbal is on key 84, the shake on the keys from 60, the higher a key the
// higher and shorter it is: "loud cymbals" and "high sounding cymbals". The cymbal where a group of four phrases begins in the cymbals, the shake high in the
// answer, a soft shake into the dance, the breath and the last chord.
clock("1/4")
const swells = { Dance: { 12: [60, 0.6] }, Cymbals: { 0: [84, 0.8], 4: [72, 0.5], 8: [84, 0.7] }, Breath: { 0: [72, 0.45] }, Hallelujah: { 0: [66, 0.5] } }
function trigger(ev) {
  if (!shared.known() || ev.step % 4 !== 0) return
  const s = (swells[song.section] || {})[song.bar]
  if (s) note(s[0], s[1], 18)
}
