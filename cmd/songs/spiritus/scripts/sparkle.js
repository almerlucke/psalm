// SPARKLE: grains of the bells (the sound of the second synth): the strikes of the figure, cut small. The grains look back
// at the last seconds of the bells; with a short position they are the strike that just rang (a clear note, a bell played with a slow
// tremble), with a long one a shower of old ones. In the Fire the grains follow the beat (a grain every half a beat): the
// bells twinkle in time, softly (a lowpass takes the edge off every grain; an lfo on the pan moves the grains).
const plan = {
  Wind:       { level: [0, 0.6], pos: [0.03, 0.1], size: 200, density: 10, spray: 0.05, octave: 0.02, sync: 0 },
  Fire:       { level: [0.7, 1], pos: [0.03, 0.3], size: 120, density: 20, spray: 0.15, octave: 0.05, sync: 0.5 },
  Indwelling: { level: [0.6, 0.8], pos: [0.03, 0.15], size: 220, density: 12, spray: 0.08, octave: 0.03, sync: 0 },
  Glory:      { level: [0.9, 1], pos: [0.03, 0.4], size: 130, density: 22, spray: 0.2, octave: 0.05, sync: 0.5 },
  Amen:       { level: [0.7, 0.2], pos: [0.04, 0.12], size: 250, density: 8, spray: 0.06, octave: 0.02, sync: 0 },
}
let cfg = null
function section(name) { cfg = plan[name] }
function trigger(ev) {
  if (!cfg || ev.step % 16 !== 0) return
  const lerp = (a, b) => a + (b - a) * song.progress
  set({
    level: lerp(cfg.level[0], cfg.level[1]), position: lerp(cfg.pos[0], cfg.pos[1]), grainSize: cfg.size, density: cfg.density,
    spray: cfg.spray, octave: cfg.octave, sync: cfg.sync, scan: 0, stereoSpread: 1, pitchSpread: 0.05,
    filter: 1, cutoff: 4500, resonance: 0.1, keyTrack: 0.5, mod1Source: 2, mod1Dest: 5, mod1Amount: 0.2,
  })
  if (song.bar % 8 === 0) note([60, 72], 0.5, Math.min(8, song.bars - song.bar) * 16 - 0.5)
}
