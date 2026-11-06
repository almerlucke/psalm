// SCORE: plays nothing. The program puts `const TUNES` in front of this text: the notes of the three jigs, as the files
// have them ([start, pitch, length, velocity], start and length in eighths). At its start this script makes three things for the
// other scripts and puts them in `shared`, the object every script of the patch sees:
//   shared.notesAt(section, bar, step)   the notes of the tune that start on an eighth of a bar
//   shared.chordAt(section, bar, step)   the chord of that eighth (a chord is a bar or half a bar)
//   shared.level(part, section, bar)     how loud a part is in a bar (0: it does not play)
// So the form of the piece is read here, in one place.

// the chords of the tunes, a bar each; "Am|D" is two chords, one for each half of the bar (three eighths)
const harm = {
  jig1: [
    // A, said twice
    ...Array(2).fill(["G", "Am|D", "G", "D", "Em|Am", "G", "C", "D|G"]).flat(),
    // B, said twice, the second one ends on the G
    "G", "Am|D", "Em", "D|G", "Bm|D", "G", "G", "D",
    "G", "Am|D", "Em", "D|G", "Bm|D", "G", "G", "G",
  ],
  jig2: [
    ...Array(2).fill(["G", "C", "G|D", "D", "G", "Em", "Em|D", "G"]).flat(),
    ...Array(2).fill(["Em|G", "D|G", "Am|C", "G|C", "G", "G|D", "D|Em", "D"]).flat(),
  ],
  jig3: [
    ...Array(2).fill(["G", "D|G", "C", "D|G", "G|D", "D|G", "C|D", "G|Em"]).flat(),
    ...Array(2).fill(["G", "D", "C", "Am|Em", "Am", "G|Em", "G", "C|G", "Am|G", "D|G", "D|G", "Em", "D", "G", "C|G", "D"]).flat(),
  ],
}
const symbols = {}
for (const [name, list] of Object.entries(harm)) symbols[name] = list.map(s => s.split("|").map(chord))

// which tune each section plays, and from which bar of it
const plan = {
  Hearth: { tune: "jig1", from: 0 },
  First:  { tune: "jig1", from: 0 },
  Break:  { tune: "jig1", from: 16 },   // the chords of the B part, without its tune
  Second: { tune: "jig2", from: 0 },
  Bridge: { tune: "jig3", from: 0 },
  Third:  { tune: "jig3", from: 0 },
  Outro:  { tune: "jig1", from: 0 },
}

// the layers: for each section, each part: a level, or [from, to, level] for the bars it plays
const layers = {
  Hearth: { whistle: 0.9, drone: 0.7, pad: [[2, 8, 0.5]], bodhran: [[4, 8, 0.5]], fx: 1 },
  First: {
    flute: 1, mandolin: [[0, 32, 0.9]], whistle: [[16, 32, 0.45]], drone: [[0, 8, 0.5]],
    bodhran: [[8, 32, 1]], bass: [[8, 32, 1]], hats: [[8, 16, 0.5], [16, 32, 0.8]], kick: [[16, 32, 1]], snare: [[24, 32, 1]],
    pad: [[16, 32, 0.7]], arp: [[16, 24, 0.5], [24, 32, 0.8]], fx: 1,
  },
  Break: {
    mandolin: [[0, 8, 0.8]], bass: [[0, 8, 1]], kick: [[0, 4, 0.9]], hats: [[4, 8, 0.8]], snare: [[6, 8, 0.8]],
    pad: [[0, 8, 0.8]], arp: [[0, 8, 0.9]], fx: 1,
  },
  Second: {
    flute: 1, mandolin: 1, whistle: [[0, 16, 0.35]], lead: [[8, 32, 0.8]],
    bodhran: 1, bass: 1, hats: 0.9, kick: 1, snare: [[4, 32, 1]],
    pad: 0.8, arp: [[0, 32, 0.9]], fx: 1,
  },
  Bridge: {
    whistle: 0.8, drone: 0.7, pad: [[0, 8, 0.6]], bodhran: [[2, 8, 0.7]], hats: [[4, 8, 0.4]], arp: [[4, 8, 0.4]], fx: 1,
  },
  Third: {
    flute: 1, mandolin: 1, whistle: [[8, 48, 0.5]], lead: [[8, 48, 0.9]],
    bodhran: 1, bass: 1, hats: 1, kick: 1, snare: 1, pad: 0.9, arp: 1, fx: 1,
  },
  Outro: {
    whistle: 0.9, flute: [[0, 8, 0.45]], drone: 0.8, pad: [[0, 8, 0.6]], mandolin: [[0, 8, 0.6]], bodhran: [[0, 4, 0.5]], fx: 1,
  },
}

// the notes of every bar of every tune
const bars = {}
for (const [name, list] of Object.entries(TUNES)) {
  const byBar = []
  for (const [start, pitch, len, vel] of list) {
    const b = Math.floor(start / 6 + 1e-6)
    const at = start - b * 6
    const step = Math.floor(at + 1e-6)
    ;(byBar[b] = byBar[b] || []).push({ step, delay: Math.max(0, at - step), pitch, len, vel })
  }
  bars[name] = byBar
}

const where = (section, bar) => {
  const p = plan[section] || plan.Hearth
  return { tune: p.tune, bar: (p.from + bar) % harm[p.tune].length }
}

shared.notesAt = (section, bar, step) => {
  const w = where(section, bar)
  return ((bars[w.tune] || [])[w.bar] || []).filter(n => n.step === step)
}

shared.chordAt = (section, bar, step) => {
  const w = where(section, bar)
  const list = symbols[w.tune][w.bar]
  return list[Math.min(list.length - 1, step < 3 ? 0 : 1)]
}

shared.level = (part, section, bar) => {
  const l = (layers[section] || {})[part]
  if (l === undefined) return 0
  let v = 0
  if (typeof l === "number") v = l
  else for (const [a, b, level] of l) if (bar >= a && bar < b) v = level
  // the end: everything fades out over the outro
  if (section === "Outro") v *= 1 - 0.75 * song.progress
  return v
}
