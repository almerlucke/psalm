// CHIMES: bells made of grains of a sine. Every grain is a ping: a sharp attack (a window that is all attack and decay, `shape` 1) and a long
// ring (700 ms), and each is three sines at once (a stack: the note, an octave and a twelfth above it), at a height that a slow walk picks in
// the pentatonic scale of the key. A few a second, at random times and in random places of the stereo field, like light that
// is thrown about. They begin rarely in the Waiting (one now and then), and in the Glory there are many: the Spirit that is
// everywhere. In the glory (E major) the scale lifts a whole step with the rest.
const walk = genny.rwalk(6, 0, 12, 3)
const plan = {
  Waiting:    { level: [0, 0.4], density: 0.5, on: bar => bar >= 4 },
  Wind:       { level: [0.3, 0.7], density: 1.5 },
  Fire:       { level: [0.7, 1], density: 4 },
  Indwelling: { level: [0.5, 0.7], density: 2.5 },
  Glory:      { level: [0.9, 1], density: 8 },
  Amen:       { level: [0.6, 0.15], density: 2, on: bar => bar < 8 },
}
let cfg = null
function section(name) { cfg = plan[name] }
function trigger(ev) {
  if (!cfg || ev.step % 16 !== 0) return
  if (cfg.on && !cfg.on(song.bar)) return
  const lerp = (a, b) => a + (b - a) * song.progress
  const root = song.section === "Glory" ? 64 : 62
  set({
    level: lerp(cfg.level[0], cfg.level[1]), density: cfg.density, grainSize: 700, jitter: 0.9, window: 1, shape: 1, wave: 0, scan: 0,
    scale: "pentatonic", scaleRoot: root, stack: [0, 12, 19], stereoSpread: 1, spray: 0, pitchSeq: walk.take(32),
    filter: 1, cutoff: 9000, resonance: 0.1,
  })
  if (song.bar % 4 === 0) note(root + 12, 0.6, Math.min(4, song.bars - song.bar) * 16 - 0.5)
}
