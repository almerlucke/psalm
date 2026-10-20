// SPARKLE: the stars, "when the morning stars sang together": a few bells very high, on a euclidean rhythm of three in seven steps
// (a cycle of seven against the sixteen of the bar, so it comes round every seven bars), each a tone of the chord of the bar
// taken in turn from the top of the range. In the throne it is a music box and plays as it always does; from then on it evolves, all
// through the piece, and in the worthy it comes home to the music box of the throne:
//   - the line drifts: every cycle of seven steps the figure may move along the tones of the chord by one or two (a random walk,
//     the chance a little more in each part), so the same stars are never in the same place for long;
//   - a star may have a twin: a second bell of the chord a third or a fifth above, struck with it;
//   - the bell changes: a music box, then a kalimba, vibes, a music box again, one for each of the four beasts, vibes for the elders, a
//     music box for the holy (more stars, four of seven, and more twins: the worship is the stars), and the music box again;
//   - the worthy pulls the drift back to nothing, a step a cycle, so the loop starts where it began.
const rnd = genny.rand(0, 1)
const parts = {
  Door:      { preset: "musicBox", trim: -11, rhythm: [2, 7], vel: 0.22, on: bar => bar < 2 || bar >= 5 },   // the stars of the worthy go on over the seam of the loop
  Throne:    { preset: "musicBox", trim: -9, rhythm: [3, 7], vel: 0.32, on: bar => bar >= 4 },
  Creatures: { beasts: [
                 { preset: "musicBox", trim: -10, rhythm: [2, 7] }, { preset: "kalimba", trim: -9, rhythm: [3, 7] },
                 { preset: "vibes", trim: -13, rhythm: [3, 7] }, { preset: "musicBox", trim: -11, rhythm: [3, 7] }],
               vel: 0.28, on: bar => true, drift: 0.35, twin: 0.1 },
  Elders:    { preset: "vibes", trim: -9, rhythm: [4, 7], vel: 0.3, on: bar => bar >= 2 && bar < 20, drift: 0.5, twin: 0.25 },
  Holy:      { preset: "musicBox", trim: -9, rhythm: [4, 7], vel: 0.3, on: bar => true, drift: 0.6, twin: 0.25 },
  Worthy:    { preset: "musicBox", trim: -9, rhythm: [3, 7], vel: 0.28, on: bar => true, pull: true },
}
const shapes = genny.markov(shared.shapeTable, "run")
let part = null, hits = null, count = 0, steps = 0, shift = 0, beast = -1, figure = "run", pos = 0, cycleStart = true
function use(preset, trim, rhythm) {
  hits = genny.euclid(rhythm[0], rhythm[1], 0)
  steps = 0
  set({ preset, trim })
}
function section(name) {
  part = parts[name]
  beast = -1
  if (part && !part.beasts) use(part.preset, part.trim, part.rhythm)
}

function trigger(ev) {
  if (!part) return
  const bar = song.bar
  if (part.beasts) {
    const b = shared.beastOf(bar)
    if (b !== beast) { beast = b; use(part.beasts[b].preset, part.beasts[b].trim, part.beasts[b].rhythm) }
  }
  const rhythm = part.beasts ? part.beasts[beast].rhythm : part.rhythm
  // a new cycle: the figure may move along the chord
  if (steps++ % rhythm[1] === 0) {
    cycleStart = true
    if (part.pull) shift -= Math.sign(shift)
    else if (part.drift && rnd.next() < part.drift) shift = Math.max(-4, Math.min(4, shift + (rnd.next() < 0.5 ? -1 : 1) * (rnd.next() < 0.3 ? 2 : 1)))
  }
  const hit = hits.next()
  if (!hit || !part.on(bar)) return
  const tones = shared.chordFor(song.section, bar).tones(79, 103)
  const n = tones.length
  if (n === 0) return
  // the throne's stars rise by thirds, as they always did; the later ones develop: every cycle of the rhythm is a small figure, the
  // next shape of the chain, on the tones the line has drifted to
  let i
  if (part.drift) {
    if (cycleStart) { figure = shapes.next(); cycleStart = false; pos = 0 }
    i = (((shift + shared.shapes[figure](pos++, rhythm[0])) % n) + n) % n
  } else {
    i = (((count++ * 2 + shift) % n) + n) % n
  }
  const vel = part.vel * shared.swell(song.section, song.progress)
  // a twin: the tone two places up the chord (a third or a fifth over the star)
  note(part.twin && rnd.next() < part.twin && n > 2 ? [tones[i], tones[(i + 2) % n]] : tones[i], vel, 10)
}
