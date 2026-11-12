// LAMENT: twelve slow notes that fall, one voice, on the subtractive synth: a triangle, a slide of ninety milliseconds between the
// notes, a vibrato that comes in after most of a second and a filter that is half closed. Soft.
let ready = false
function trigger(ev) {
  if (!ready) {
    tune({ preset: "pad", osc1Waveform: 2, osc1Level: 1, osc2Level: 0, subLevel: 0, noiseLevel: 0, ringModLevel: 0, fmAmount: 0, sync: 0, drive: 0, unison: 2, unisonDetune: 8, glide: 90, filterType: 0, cutoff: 1800, resonance: 0.1, filterEnvAmount: 0.6, keyTrack: 0.3, lfo1Waveform: 0, lfo1Rate: 5, lfo1Pitch: 0.14, lfo1Delay: 700, lfo1KeySync: 1, ampAttack: 140, ampDecay: 300, ampSustain: 0.8, ampRelease: 1400, level: 0.5 })
    ready = true
  }
  for (const n of shared.notesNow("lament", ev)) note(n.pitch, 0.5 + 0.45 * n.vel, n.len)
}
