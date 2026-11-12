// CELESTA: eight soft bells at the start, a falling figure over the G minor chord. Not a celesta: the shaper's glassBell, phase
// modulation at three times the pitch that falls to a pure tone, a struck glass, tuned to ring for three seconds.
let ready = false
function trigger(ev) {
  if (!ready) {
    tune({ preset: "glassBell", ampRelease: 3000, trim: -2, mono: 0 })
    ready = true
  }
  for (const n of shared.notesNow("celesta", ev)) note(n.pitch, 0.35 + 0.6 * n.vel, n.len)
}
