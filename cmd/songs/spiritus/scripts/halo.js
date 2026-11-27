// HALO: tongues of fire. Grains of a sine, short and high, in the scale of the key, a few every second, each at its own
// height and in its own place of the stereo field: the flames that rested on each one of them. They come with the Fire,
// and burn in the Glory. The heights are the walk of a generator, held to the pentatonic scale by the synth.
const walk = genny.rwalk(7, 0, 16, 3)
const plan = {
  Wind:       { level: [0, 0.21], density: 3, on: bar => bar >= 8 },
  Fire:       { level: [0.28, 0.63], density: 6 },
  Indwelling: { level: [0.21, 0.35], density: 3 },
  Glory:      { level: [0.49, 0.7], density: 8 },
  Amen:       { level: [0.21, 0], density: 3, on: bar => bar < 6 },
}
let cfg = null
function section(name) { cfg = plan[name] }
function trigger(ev) {
  if (!cfg || ev.step % 16 !== 0) return
  if (cfg.on && !cfg.on(song.bar)) return
  const lerp = (a, b) => a + (b - a) * song.progress
  const root = song.section === "Glory" ? 64 : 62
  set({
    level: lerp(cfg.level[0], cfg.level[1]), density: cfg.density, grainSize: 260, jitter: 0.8, window: 1, wave: 0, scan: 0,
    filter: 1, cutoff: 6000, resonance: 0.1, scale: "pentatonic", scaleRoot: root, stereoSpread: 1, spray: 0, pitchSeq: walk.take(32),
  })
  if (song.bar % 4 === 0) note(root + 12, 0.4, Math.min(4, song.bars - song.bar) * 16 - 0.5)
}
