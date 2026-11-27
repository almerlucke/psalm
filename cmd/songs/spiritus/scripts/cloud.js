// CLOUD: grains of the pad (the sound of a synth that is not heard), long ones: the harmony of the piece as a slow
// breath. It plays the pad at the pitch it was played (the note 60) and an octave above it, the grains look back from a second to
// a few seconds, and what they take is blurred. In the Indwelling the grains follow the beat (a grain every beat and a half) like a pulse that
// is felt more than heard.
const plan = {
  Waiting:    { level: [0, 0.5], pos: [0.12, 0.25], size: 700, density: 14, spray: 0.4, octave: 0, sync: 0, vel: 0.5 },
  Wind:       { level: [0.5, 0.9], pos: [0.2, 0.35], size: 600, density: 18, spray: 0.5, octave: 0.03, sync: 0, vel: 0.55 },
  Fire:       { level: [0.9, 1], pos: [0.1, 0.4], size: 450, density: 22, spray: 0.5, octave: 0.05, sync: 0, vel: 0.6 },
  Indwelling: { level: [0.8, 0.9], pos: [0.08, 0.2], size: 380, density: 20, spray: 0.3, octave: 0.03, sync: 1.5, vel: 0.55 },
  Glory:      { level: [0.9, 1], pos: [0.1, 0.45], size: 520, density: 24, spray: 0.6, octave: 0.05, sync: 0, vel: 0.65 },
  Amen:       { level: [0.8, 0.25], pos: [0.2, 0.3], size: 800, density: 14, spray: 0.4, octave: 0.03, sync: 0, vel: 0.5 },
}
let cfg = null
function section(name) { cfg = plan[name]; }
function trigger(ev) {
  if (!cfg || ev.step % 16 !== 0) return
  const lerp = (a, b) => a + (b - a) * song.progress
  set({
    level: lerp(cfg.level[0], cfg.level[1]), position: lerp(cfg.pos[0], cfg.pos[1]), grainSize: cfg.size, density: cfg.density,
    spray: cfg.spray, octave: cfg.octave, sync: cfg.sync, scan: 0, stereoSpread: 1,
    // a filter that breathes (an lfo on the cutoff) and a look-back that drifts slowly (an lfo on the position)
    filter: 1, cutoff: 4500, resonance: 0.1, mod1Source: 1, mod1Dest: 5, mod1Amount: 0.3, mod1Rate: 0.05,
    mod2Source: 1, mod2Dest: 0, mod2Amount: 0.08, mod2Rate: 0.03,
  })
  // one long note for the section, and a new one every eight bars so the voices do not stay the same
  if (song.bar % 8 === 0) note([60, 67], cfg.vel, Math.min(8, song.bars - song.bar) * 16 - 0.5)
}
