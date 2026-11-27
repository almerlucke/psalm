// CHOIR: the chants. Two choirs play this script (ROLE is "low" or "high", set above it): a granular synth that
// plays a recording of a chant, grain by grain. The grains are not taken from anywhere: they come from the places where the
// singers rest on their home note (shared.chants says where, and how far off each one is), so every grain is
// put right to the note that is played, the chords are in tune and the chant is never a clash.
// A chant for every part of the song, in the order of the story: the Ad te levavi (to you I lift up my soul) for the stillness,
// the Factus est repente (suddenly there came a sound from heaven, the Holy Spirit on Pentecost) for the wind and the fire, the
// Epistle of Easter (Christ is risen, and the Spirit of him who raised him lives in you) for the indwelling, the Crux
// fidelis (the likeness of Christ) for the glory, and the Ad te levavi again at the end.
// level: from one end of the section to the other; vel: how loud the notes are; size, spray: the grains.
const high = ROLE === "high"
const plan = {
  Waiting:    { low: { sound: "adte", level: [0.35, 0.6], vel: 0.5, density: 7, size: 520, spray: 0.03, on: bar => bar >= 1 } },
  Wind:       { low: { sound: "factus", level: [0.32, 0.63], vel: 0.5, density: 10, size: 420, spray: 0.04 },
                high: { sound: "factus", level: [0, 0.42], vel: 0.5, density: 8, size: 380, spray: 0.03, on: bar => bar >= 5 } },
  Fire:       { low: { sound: "epistle", level: [0.56, 0.7], vel: 0.5, density: 12, size: 360, spray: 0.03 },
                high: { sound: "factus", level: [0.42, 0.63], vel: 0.55, density: 11, size: 300, spray: 0.03, octave: 0.02, echo: 0.25 } },
  Indwelling: { low: { sound: "crux", level: [0.63, 0.7], vel: 0.5, density: 12, size: 380, spray: 0.03, both: true },
                high: { sound: "epistle", level: [0.35, 0.56], vel: 0.5, density: 9, size: 400, spray: 0.03, on: bar => bar >= 4 } },
  Glory:      { low: { sound: "crux", level: [0.7, 0.7], vel: 0.55, density: 14, size: 340, spray: 0.03, both: true },
                high: { sound: "adte", level: [0.56, 0.7], vel: 0.5, density: 12, size: 320, spray: 0.03, octave: 0.02, echo: 0.3 } },
  Amen:       { low: { sound: "adte", level: [0.42, 0.21], vel: 0.55, density: 8, size: 480, spray: 0.03 },
                high: { sound: "factus", level: [0.42, 0], vel: 0.45, density: 7, size: 420, spray: 0.03, on: bar => bar < 8, echo: 0.25 } },
}
let cfg = null, last = null, count = 0
function section(name) { cfg = (plan[name] || {})[ROLE] || null; last = null }

function trigger(ev) {
  if (!cfg || ev.step % 16 !== 0) return
  const bar = song.bar
  if (cfg.on && !cfg.on(bar)) return
  const chant = shared.chants[cfg.sound]
  const lerp = (a, b) => a + (b - a) * song.progress
  // the level moves through the section, a bar at a time
  set({ level: lerp(cfg.level[0], cfg.level[1]) })
  const c = shared.chordFor(song.section, bar)
  if (c === last) return
  last = c
  count++
  // the grains: the places of the chant where it sings its home note, in another order for every chord
  const a = chant.tonic, b = cfg.both && chant.fifth ? chant.fifth : { pos: [], fix: [] }
  const pos = [], fix = []
  for (let i = 0; i < Math.max(a.pos.length, b.pos.length); i++) {
    if (i < a.pos.length) { pos.push(a.pos[i]); fix.push(a.fix[i]) }
    if (i < b.pos.length) { pos.push(b.pos[i]); fix.push(b.fix[i]) }
  }
  set({
    sound: cfg.sound, scan: 0, spray: cfg.spray, density: cfg.density, grainSize: cfg.size,
    octave: cfg.octave || (high ? 0.04 : 0.02),
    // every grain has a filter, opened a little and closed by a slow lfo (a breath of about 14 s); the high choir has a soft echo of itself
    filter: 1, cutoff: cfg.cut || (high ? 5500 : 3800), resonance: 0.1, keyTrack: 0.3,
    mod1Source: 1, mod1Dest: 5, mod1Amount: 0.3, mod1Rate: high ? 0.06 : 0.08,
    feedback: cfg.echo || 0, feedbackTime: 0.9, feedbackPitch: 0,
    positionSeq: shared.rot(pos, count * 7), pitchSeq: shared.rot(fix, count * 7),
  })
  // the voices: low ones around the home note of the chant, high ones a fifth above it
  const home = chant.root
  const all = high ? shared.colours(c, home + 4).concat(shared.voices(c, home + 4).slice(0, 2)) : shared.voices(c, home - 7)
  // lighter: the low choir sings the root and the fifth, the third only in the indwelling and the glory; the high one, two or three voices
  const voices = high ? all.slice(0, 3) : (["Indwelling", "Glory"].includes(song.section) ? all : all.slice(0, 2))
  const bars = shared.lasts(song.section, bar, song.bars)
  note(voices, cfg.vel, bars * 16 - 0.5)
}
