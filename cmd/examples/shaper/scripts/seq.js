// SEQ: the sequencer, a short cycle of notes round the root of the chord, in sixteenths, with rests in it, so
// it is a pulse more than a tune. A cycle of 8 notes comes round twice a bar; a cycle of 7 against 16 steps
// shifts a step every bar and takes seven bars to come back, which is the whole of the movement. The sound is
// a preset of the shaper synth for each section (set({preset})), a sub sine at first, then a narrowing pulse, an FM
// bass, and the pulse bass again, faster, in the lift; the script moves its filter, very little, and its macro.
// Every note is fitted to the chord (shared.fit).
const walk = genny.rwalk(0.5, 0, 1, 0.08)

// cycles: semitones above the root of the chord (the root is its bass, around D3: low enough to be dark, high
// enough for a small speaker to play the notes and not only their attacks); null a rest
const cycles = {
  eight: [0, null, 7, null, 0, 12, null, 5],
  seven: [0, null, 7, 0, null, 12, 5],
  nine:  [0, null, 7, null, 12, 0, null, 10, 5],
}

// what each section does: the preset and where it sits (trim, dB), which cycle, the cutoff from where to where (Hz),
// the density by bar
const parts = {
  Drone: { preset: "subSine", trim: -3, cycle: "eight", cut: [350, 500], macro: p => 0.3, hits: 3, bars: bar => bar < 8 ? 0 : 0.25 },
  Pulse: { preset: "pulseBass", trim: -6, cycle: "eight", cut: [500, 1500], macro: p => 0.2 + 0.5 * walk.next(), hits: 5,
           bars: bar => bar < 4 ? 0.25 : bar < 8 ? 0.5 : 1, tweak: { filterEnvAmount: 1, filterAttack: 6 } },
  Depth: { preset: "fmBass", trim: -7, cycle: "seven", cut: [700, 1800], macro: p => 0.1 + 0.3 * p, hits: 5, bars: bar => 1 },
  Hollow: { preset: "subSine", trim: -6, cycle: "eight", cut: [350, 700], macro: p => 0.3, hits: 3, bars: bar => bar < 8 ? 0 : 0.25 },
  Lift: { preset: "pulseBass", trim: -7, cycle: "nine", cut: [900, 2400], macro: p => 0.3 + 0.3 * p, hits: 5,
          bars: bar => bar < 2 ? 0.5 : 1, tweak: { filterEnvAmount: 1, filterAttack: 6 } },
  Glow: { preset: "pulseBass", trim: -7, cycle: "seven", cut: [800, 2000], macro: p => 0.2 + 0.4 * p, hits: 5,
          bars: bar => bar < 2 ? 0.5 : 1, tweak: { filterEnvAmount: 1, filterAttack: 6 } },
  Bloom: { preset: "pulseBass", trim: -7, cycle: "nine", cut: [1200, 3000], macro: p => 0.4 + 0.4 * p, hits: 7,
           bars: bar => 1, tweak: { filterEnvAmount: 1, filterAttack: 6 } },
  Dissolve: { preset: "pulseBass", trim: -8, cycle: "eight", cut: [1200, 300], macro: p => 0.2, hits: 3,
              bars: bar => bar < 2 ? 1 : bar < 5 ? 0.5 : bar < 8 ? 0.25 : 0, tweak: { filterEnvAmount: 1, filterAttack: 6 } },
}

let part = null, pitches = null, accents = null
function section(name) {
  part = parts[name]
  if (!part) return
  pitches = genny.sequence(cycles[part.cycle])
  set({ preset: part.preset, mono: part.mono ? 1 : 0, legato: part.mono ? 1 : 0, trim: part.trim, ...(part.tweak || {}) })
}

function trigger(ev) {
  if (!part) return
  const k = ev.step % 16, bar = song.bar
  if (k === 0) accents = genny.euclid(part.hits, 16, bar)         // turned a step every bar
  const accent = accents.next()
  const semis = pitches.next()                                      // the cycle goes on through rests
  const density = part.bars(bar)
  const every = density >= 1 ? 1 : density >= 0.5 ? 2 : density > 0 ? 4 : 0
  if (every === 0) return
  // the cutoff: a curve through the section, with a little accent
  const cutoff = part.cut[0] * Math.pow(part.cut[1] / part.cut[0], Math.pow(song.progress, 1.3))
  set({ cutoff: cutoff * (accent ? 1.3 : 1), macro1: part.macro(song.progress) })
  if (semis === null || k % every !== 0) return
  const c = shared.chordFor(song.section, bar)
  note(shared.fit(c.bass(45) + semis, c), accent ? 0.95 : 0.6, 1.3)
}
