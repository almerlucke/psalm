// BELLS: a second synth that nobody hears: bells that say a slow figure of the pentatonic scale of the key (a note every few beats,
// every bar fitted to the chord), into a grain buffer (bells). The grains of the SPARKLE play the bells as clear
// notes at first, then as a shower. The figure is a lift of the heart, up and back, said again and again.
const figure = shared.line(`
  F#5 - - - - - - - A5 - - - - - - -
  B5 - - - - - - - A5 - - - F#5 - - -
  E5 - - - - - - - F#5 - - - - - - -
  D5 - - - - - - - . . . . . . . .
`)
const high = shared.line(`
  A5 - - - - - B5 - - - D6 - - - - -
  C#6 - - - A5 - - - F#5 - - - - - . .
  B5 - - - - - A5 - - - F#5 - - - E5 -
  D5 - - - - - - - . . . . . . . .
`)
const plan = {
  Wind:       { line: figure, on: bar => bar >= 6, preset: "glassBell", vel: 0.45 },
  Fire:       { line: figure, preset: "churchBell", vel: 0.5 },
  Indwelling: { line: high, preset: "glassBell", vel: 0.45 },
  Glory:      { line: high, up: 2, preset: "churchBell", vel: 0.5 },
  Amen:       { line: figure, on: bar => bar < 6, preset: "glassBell", vel: 0.4 },
}
let part = null
function section(name) { part = plan[name]; if (part) set({ preset: part.preset, trim: -6 }) }
function trigger(ev) {
  if (!part || (part.on && !part.on(song.bar))) return
  const n = part.line[(song.bar * 16 + ev.step % 16) % part.line.length]
  if (!n) return
  note(shared.fit(n.pitch + (part.up || 0), shared.chordFor(song.section, song.bar)), part.vel, Math.max(1, n.len - 0.5))
}
