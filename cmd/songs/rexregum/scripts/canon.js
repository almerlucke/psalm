// CANON: the theme again, nineteen seconds after it and a fifth and an octave above, a sine on the subtractive synth with a second sine
// an octave up and a little apart; the mixer sends it to the echo.
let ready = false
function trigger(ev) {
  if (!ready) {
    set({ preset: "pad", osc1Waveform: 0, osc1Level: 1, osc2Level: 0, subLevel: 0, noiseLevel: 0, ringModLevel: 0, fmAmount: 0, sync: 0, drive: 0, osc2Waveform: 0, osc2Level: 0.35, osc2Octave: 1, osc2Detune: 0.08, unison: 2, unisonDetune: 10, cutoff: 6000, resonance: 0, filterEnvAmount: 0.3, ampAttack: 8, ampDecay: 300, ampSustain: 0.5, ampRelease: 500, level: 0.4 })
    ready = true
  }
  for (const n of shared.notesNow("canon", ev)) note(n.pitch, 0.45 + 0.5 * n.vel, n.len)
}
