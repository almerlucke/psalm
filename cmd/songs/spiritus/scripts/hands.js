// HANDS: more percussion, played like the pulse (a hit of a recording for every grain, found by its onset, in a rhythm of levels) but
// from other recordings, a new one for every part of the song: wooden tubes in the Wind (a few soft knocks), a tambourine in the Fire
// (a light shake on the off beats), a tabla in the Indwelling (the Spirit speaking quietly in us), conga drums in the Glory (the hands
// that clap for joy), and the wooden tubes again, sparse, in the Amen. They sit above the heartbeat of the pulse and never on top of it.
const walk = genny.rwalk(30, 0, 63, 9)
const plan = {
  Wind:       { sound: "tubes", level: [1.2, 2.2], sync: 2,   amps: [0.8, 0, 0, 0.5, 0, 0, 0.6, 0], on: bar => bar >= 4, cut: 6000, note: 60 },
  Fire:       { sound: "tamb",  level: [1.5, 2.5], sync: 0.5, amps: [0, 0.6, 0, 0.3, 0, 0.7, 0, 0.3], cut: 7000, note: 64 },
  Indwelling: { sound: "tabla", level: [0.9, 1.3], sync: 0.5, amps: [0.8, 0, 0.4, 0, 0.6, 0.3, 0, 0], cut: 5500, note: 62 },
  Glory:      { sound: "conga", level: [0.7, 1],   sync: 0.5, amps: [0.7, 0.3, 0.5, 0, 0.6, 0.3, 0.5, 0.4], cut: 6000, note: 62 },
  Amen:       { sound: "tubes", level: [1.2, 0.3], sync: 2,   amps: [0.7, 0, 0, 0, 0, 0.4, 0, 0], on: bar => bar < 6, cut: 5000, note: 60 },
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
    sound: cfg.sound, slices: 1, position: 0, spray: 0, scan: 0, sync: cfg.sync, jitter: 0.04, grainSize: 260, window: 0, shape: 0.75,
    filter: 1, cutoff: cfg.cut, resonance: 0.1, keyTrack: 0.2, pitchSpread: 0.03, stereoSpread: 0.8,
    sliceSeq: walk.take(16).map(Math.round), ampSeq: cfg.amps,
  })
  note(cfg.note, 0.6, Math.min(4, song.bars - song.bar) * 16 - 0.5)
}
