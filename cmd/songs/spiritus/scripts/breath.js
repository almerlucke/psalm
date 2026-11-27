// BREATH: time stretch. A cymbal (4.8 seconds, a few kHz of air and no pitch at all) played backwards and stretched eight times: the
// grains walk back through its decay at an eighth of the speed, so the tail of a hit becomes 39 seconds of a rising wind that
// swells towards the strike and fades (the stretch keeps the pitch, `overlap` keeps six grains sounding whatever the grain size, so it is
// smooth). It is the sound from heaven of the Wind, comes back as the breath under the Fire and the Glory, and is a lowpass away from
// being felt more than heard.
const plan = {
  Wind:  { level: [0.4, 1], cut: 2600, vel: 0.6, bars: 8 },
  Fire:  { level: [0.7, 0.5], cut: 2200, vel: 0.5, bars: 8, on: bar => bar >= 4 },
  Glory: { level: [0.6, 0.9], cut: 2800, vel: 0.5, bars: 8 },
}
let cfg = null
function section(name) { cfg = plan[name] }
function trigger(ev) {
  if (!cfg || ev.step % 16 !== 0) return
  const bar = song.bar
  if (cfg.on && !cfg.on(bar)) return
  const lerp = (a, b) => a + (b - a) * song.progress
  set({ level: lerp(cfg.level[0], cfg.level[1]) })
  if (bar % cfg.bars !== (cfg.on ? 4 : 0)) return
  set({
    sound: "cymbal", stretch: -8, overlap: 6, position: 0.98, spray: 0.02, grainSize: 350, jitter: 0.2, window: 0, shape: 0,
    filter: 1, cutoff: cfg.cut, resonance: 0.1, stereoSpread: 1, scan: 0,
    mod1Source: 1, mod1Dest: 5, mod1Amount: 0.2, mod1Rate: 0.1,
  })
  note(48, cfg.vel, Math.min(cfg.bars, song.bars - bar) * 16 - 0.5)
}
