// CHIMES: the notes that ring over the Gathering and on, on the subtractive synth's ePiano: a soft electric piano, left to ring.
let ready = false
function trigger(ev) {
  if (!ready) {
    set({ preset: "ePiano", ampRelease: 2500, level: 0.4 })
    ready = true
  }
  for (const n of shared.notesNow("chimes", ev)) note(n.pitch, 0.3 + 0.6 * n.vel, n.len)
}
