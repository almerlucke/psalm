// THEME: the tune of the piece, sixteenths from the Theme to the end, on the subtractive synth: a triangle with a sine an octave
// under it, a little wide, a filter that opens softly on every note.
let ready = false
function trigger(ev) {
  if (!ready) {
    set({ preset: "pad", osc1Waveform: 2, osc1Level: 1, osc2Level: 0, subLevel: 0, noiseLevel: 0, ringModLevel: 0, fmAmount: 0, sync: 0, drive: 0, subLevel: 0.25, unison: 3, unisonDetune: 9, drift: 5, cutoff: 3200, resonance: 0.1, filterEnvAmount: 0.8, keyTrack: 0.5, ampAttack: 12, ampDecay: 250, ampSustain: 0.7, ampRelease: 350, level: 0.45 })
    ready = true
  }
  for (const n of shared.notesNow("theme", ev)) note(n.pitch, 0.5 + 0.5 * n.vel, n.len)
}
