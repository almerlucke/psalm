// PIANO: the one instrument that is not granular, a modelled grand that plays along through the whole piece, gentle and glad, and never
// loud: the pedal is down and changes with the chords, the velocities are soft, and a bar is a breath. It does three things, a section
// at a time: a LEFT HAND (the root of the chord, and its fifth above it, once a bar or twice), an ARPEGGIO (the tones of the chord, up and back,
// a note every so many sixteenths, the first of a beat a little louder: the Spirit moving) and a MELODY (a slow tune of four bars that
// is said again and again, each time fitted to the chord that is playing, so it always belongs). In the Waiting it only touches a few keys;
// in the Fire it moves in eighths; in the Indwelling it sings the tune; in the Glory it lifts a step with the key and plays the tune
// in octaves with the arpeggio flowing under it; in the Amen it says the tune once more, alone, and rests on the last chord.
const C = 60 - pitch("C4") // the names below are the names where middle C is C4, whatever the script thinks
const P = name => pitch(name) + C
const tune = shared.line(`
  F#4 - - - A4 - - - B4 - - - - - - -
  A4 - - - F#4 - - - E4 - - - D4 - - -
  F#4 - - - A4 - - - D5 - - - C#5 - - -
  B4 - - - A4 - - - F#4 - - - - - - -
`)
const glad = shared.line(`
  A4 - B4 - D5 - - - C#5 - B4 - A4 - - -
  B4 - C#5 - E5 - - - D5 - C#5 - B4 - - -
  A4 - B4 - D5 - E5 - F#5 - - - E5 - - -
  D5 - C#5 - B4 - A4 - B4 - - - - - - -
`)
// the up and back through the tones of a chord, as steps in its list
const arch = [0, 1, 2, 3, 2, 1, 2, 3, 4, 3, 2, 1]
const plan = {
  Waiting:    { left: [0], arp: { every: 8, from: 55, to: 72, vel: 0.4 }, on: bar => bar >= 1, leftVel: 0.4 },
  Wind:       { left: [0], arp: { every: 4, from: 55, to: 76, vel: 0.3 }, on: bar => bar >= 1, leftVel: 0.35 },
  Fire:       { left: [0, 8], arp: { every: 2, from: 57, to: 79, vel: 0.33 }, tune: { line: tune, vel: 0.42, off: 0, on: bar => bar >= 4 }, leftVel: 0.4 },
  Indwelling: { left: [0], arp: { every: 4, from: 55, to: 74, vel: 0.28 }, tune: { line: tune, vel: 0.5, off: 0 }, leftVel: 0.38 },
  Glory:      { left: [0, 8], arp: { every: 2, from: 57, to: 79, vel: 0.36 }, tune: { line: glad, vel: 0.55, off: 2, octave: true }, leftVel: 0.45 },
  Amen:       { left: [0], arp: { every: 8, from: 55, to: 72, vel: 0.24 }, tune: { line: tune, vel: 0.4, off: 0, on: bar => bar < 4 }, leftVel: 0.32, on: bar => bar < 10 },
}
let cfg = null, last = null
// the level of the piano in each section: near the others in the Waiting, a little under them in the Glory, where everything is full
const gain = { Waiting: 5, Wind: 16, Fire: 13, Indwelling: 11.5, Glory: 6.5, Amen: 14 }
function section(name) { cfg = plan[name]; last = null; set({ level: gain[name] || 16 }) }
// every touch a hair different, in a way that does not depend on chance (a piano that is played, not struck by a machine)
const human = k => 1.35 + 0.15 * Math.sin(k * 2.399)
function trigger(ev) {
  if (!cfg || (cfg.on && !cfg.on(song.bar))) return
  const step = ev.step % 16
  const c = shared.chordFor(song.section, song.bar)
  const bars = shared.lasts(song.section, song.bar, song.bars)
  // the pedal: up where the chord changes, down again a sixteenth later
  if (step === 0 && c !== last) { set({ sustain: 0 }); last = c }
  if (step === 1 && song.bar === 0 || step === 1 && shared.chordFor(song.section, song.bar - 1) !== c) set({ sustain: 1 })
  const k = song.bar * 16 + step
  // left hand
  if (cfg.left.includes(step)) {
    const b = c.bass(40)
    note(step === 0 ? [b, b + 7] : b + 7, cfg.leftVel * human(k), 14)
  }
  // arpeggio
  const a = cfg.arp
  if (step % a.every === 0) {
    const t = c.tones(a.from, a.to)
    if (t.length) {
      const i = arch[(k / a.every) % arch.length | 0] % t.length
      note(t[i], a.vel * (step % 4 === 0 ? 1.15 : 0.85) * human(k + 3), a.every * 2)
    }
  }
  // the tune
  const m = cfg.tune
  if (m && !(m.on && !m.on(song.bar))) {
    const n = m.line[k % m.line.length]
    if (n) {
      const p = shared.fit(n.pitch + C + m.off, c)
      note(m.octave ? [p, p - 12] : p, m.vel * human(k + 7), Math.max(2, n.len * 2 - 0.5))
    }
  }
}
