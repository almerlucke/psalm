// KICK: the kick drum, alone on a drumsynth of its own, because it is also the sidechain: its sound goes to the
// mixer and to the compressors of the bass, the pad, the arp and the mandolin, which it pulls down at every hit (the pump).
// On the first and the fourth eighth; in the second half of the First and in the Second and the Third one more on the
// sixth of every second bar, and a kick that skips (off the beat, every fourth bar).
function trigger(ev) {
  const k = ev.step % 6, bar = song.bar, s = song.section
  const kick = shared.level("kick", s, bar)
  if (!kick) return
  const modern = (s === "First" && bar >= 16) || s === "Second" || s === "Third"
  const full = s === "Third"
  if (k === 0) note(36, 0.95 * kick, 1)
  if (k === 3) note(36, 0.8 * kick, 1)
  if (modern && k === 5 && bar % 2 === 1) note(36, 0.6 * kick, 1)
  if (modern && k === 2 && bar % 4 === 3) note(36, 0.55 * kick, 1, 0.5)       // skips: a kick off the beat
  if (full && k === 1 && bar % 4 === 1) note(36, 0.5 * kick, 1, 0.5)
}
