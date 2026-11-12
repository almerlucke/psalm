// TIMPANI: a hundred D and A from the Gathering to the end, one to a beat. A soft sine thump on the subtractive synth: a short fall
// in pitch at the start of every note and a long round tail.
let ready = false
function trigger(ev) {
  if (!ready) {
    set({ preset: "pad", osc1Waveform: 0, osc1Level: 1, osc2Level: 0, subLevel: 0, noiseLevel: 0, ringModLevel: 0, fmAmount: 0, sync: 0, drive: 0, unison: 1, glide: 0, filterEnvToPitch: 7, cutoff: 900, resonance: 0, filterEnvAmount: 0, ampAttack: 3, ampDecay: 700, ampSustain: 0, ampRelease: 500, level: 0.4 })
    ready = true
  }
  for (const n of shared.notesNow("timpani", ev)) note(n.pitch, 0.6 + 0.4 * n.vel, n.len)
}
