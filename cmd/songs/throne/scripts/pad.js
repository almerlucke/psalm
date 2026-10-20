// PAD: the chords of the piece, voice led (each chord moves as little as it can from the one before), held as long as the
// chord lasts. Three sounds of the shaper that move by themselves: the rainbow (a pad that travels through four colours in
// one held chord), the drone of the partials 1, 3, 7 and 12 whose levels breathe on waves of twelve and seven seconds, and the sea
// of glass, a pad that never comes back to the same place. The script moves a macro and the filter with every bar.
const parts = {
  Door:      { preset: "throneDrone", trim: -5, thin: true, fadeIn: 4 },
  Throne:    { preset: "rainbow", trim: -10 },
  Lamps:     { preset: "rainbow", trim: -13 },
  Glass:     { preset: "crystalSea", trim: -7 },
  Creatures: { preset: "rainbow", trim: -15 },
  Elders:    { preset: "rainbow", trim: -14 },
  Holy:      { preset: "rainbow", trim: -13, silent: true },   // no pad in the holy: the choir carries it
  Worthy:    { preset: "crystalSea", trim: -8, fade: true },
}
let part = parts.Door, voicing = null, last = null
function section(name) {
  part = parts[name] || parts.Door
  last = null
  // two copies of each voice in the unison (the preset has three): the pad is low in the mix, and it is the cost of the piece
  set({ preset: part.preset, trim: part.fadeIn ? part.trim - 38 : part.trim, unison: name === "Glass" ? 3 : 2 })
}

function trigger(ev) {
  const k = ev.step % 16, bar = song.bar
  if (k !== 0 || part.silent) return
  const c = shared.chordFor(song.section, bar)
  // the end: the pad fades out over the last bars
  // the door opens out of the quiet the worthy ended in: the drone fades in over its first bars
  if (part.fadeIn) set("trim", part.trim - 38 * Math.pow(Math.max(0, 1 - bar / part.fadeIn), 1.5))
  if (part.fade) set("trim", part.trim - 40 * Math.pow(Math.max(0, song.progress - 0.3) / 0.7, 1.4))
  if (c === last) return                       // the same chord goes on
  last = c
  voicing = voicing ? c.voiceLead(voicing) : c.close(50)
  // it lasts until the chord changes
  let bars = 1
  while (bars < 24 && shared.chordFor(song.section, bar + bars) === c) bars++
  const toEnd = Math.min(bars, song.bars - bar)
  // thin: only the bass and the top of the chord; double: its two top notes an octave up as well in the second half
  let chord = part.thin ? [voicing[0], voicing[voicing.length - 1]] : voicing
  if (part.double && song.progress > 0.3) chord = chord.concat(voicing.slice(-2).map(n => n + 12))
  note(chord, part.double ? 0.5 : 0.55, toEnd * 16 - 0.5)
}
