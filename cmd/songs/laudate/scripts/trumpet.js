// TRUMPET: "the sound of the trumpet", a reed stop of the organ (nasalReed). A short call where the dance and the cymbals begin, a second in the middle of the
// dance, and two in the breath: the root and the fifth of the chord together on the first beat of the bar, no longer than a beat, low (around the middle C and
// below) and quiet, so it fills the chord and does not stand out. With a clock of quarters.
clock("1/4")
const calls = { Dance: [0, 8], Cymbals: [0], Breath: [0, 6] }
function trigger(ev) {
  if (!shared.known() || ev.step % 4 !== 0) return
  if (!(calls[song.section] || []).includes(song.bar)) return
  const tones = shared.chordAt(shared.bar()).tones(48, 67)
  note([tones[0], shared.nearest(tones, tones[0] + 7)], 0.36, 2)
}
