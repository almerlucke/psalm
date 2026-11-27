// PULSE: slices. The recording is 2 minutes of hand percussion, unpitched, at about 66 beats a minute; the grains synth has marked
// where each hit begins (slices: onsets), and a slice is a hit. A grain starts at the beginning of a slice and plays that one hit: a
// list of slices (`sliceSeq`, a slow random walk through the hits, so neighbours sound alike) and a list of levels (`ampSeq`, with rests)
// make a rhythm out of them, on the beat of the song (sync: beats between the grains). It is the heartbeat of the song, low and soft: a
// lowpass takes only the harshest edge off, and the hits are played higher than recorded (not deep: low, they are only thumps, high they are
// heard as the percussion they are). It comes with the Indwelling
// (the Spirit in us), goes on through the Glory, and is a single slow beat in the first bars of the Amen.
const walk = genny.rwalk(30, 0, 63, 7)
const plan = {
  Indwelling: { level: [0.5, 1], sync: 1, amps: [0.9, 0, 0.5, 0], on: bar => bar >= 4, cut: 5000 },
  Glory:      { level: [0.8, 1], sync: 0.5, amps: [0.9, 0, 0.4, 0, 0.6, 0, 0.4, 0.3], cut: 6500 },
  Amen:       { level: [0.7, 0.2], sync: 1, amps: [0.8, 0, 0, 0], on: bar => bar < 6, cut: 4500 },
}
let cfg = null, count = 0
function section(name) { cfg = plan[name]; count = 0 }
function trigger(ev) {
  if (!cfg || ev.step % 16 !== 0) return
  if (cfg.on && !cfg.on(song.bar)) return
  const lerp = (a, b) => a + (b - a) * song.progress
  set({ level: lerp(cfg.level[0], cfg.level[1]) })
  if (song.bar % 4 !== 0 && count > 0) return
  count++
  set({
    sound: "perc", slices: 1, position: 0, spray: 0, scan: 0, sync: cfg.sync, jitter: 0.04, grainSize: 220, window: 0, shape: 0.8,
    filter: 1, cutoff: cfg.cut, resonance: 0.1, keyTrack: 0.2, pitchSpread: 0.05, stereoSpread: 0.6,
    sliceSeq: walk.take(16).map(Math.round), ampSeq: cfg.amps,
  })
  note(66, 0.6, Math.min(4, song.bars - song.bar) * 16 - 0.5)
}
