// LAMPS: seven lamps of fire burning before the throne, which are the seven Spirits of God. The sevenLamps preset: a bell whose
// partials fall where no harmonic series puts them (one sine bends the phase of another at a ratio of seven to three),
// flickering as it rings. In the lamps section one lamp is lit for each chord of three bars (the seven chords), so the arpeggio goes
// round more tones each time, to seven, and the rhythm gains a hit of the twelve steps with every second lamp (one of twelve, then
// two, three, four) against the sixteen of the bar, so it comes round every three bars, as the chord does. Later it is a few bells.
const parts = {
  Lamps:     { trim: -13, lo: 57, hi: 96, lit: bar => 1 + Math.floor(bar / 3), vel: 0.32 },
  Glass:     { trim: -8, lo: 69, hi: 96, rhythm: [2, 12], lit: bar => 7, vel: 0.3, on: bar => bar >= 2 },
  Creatures: { trim: -15, lo: 69, hi: 96, rhythm: [3, 12], lit: bar => 7, vel: 0.22, on: bar => shared.beastOf(bar) === 3 },
  }
const shapes = genny.markov(shared.shapeTable, "run")
const rnd = genny.rand(0, 1)
let part = null, hits = null, k = 0, lastLit = 0, shape = "run", base = 0
const PHRASE = 6                      // notes in a phrase of the lamps, whatever the hits of the twelve steps
function section(name) {
  part = parts[name]
  hits = part && part.rhythm ? genny.euclid(part.rhythm[0], part.rhythm[1], 0) : null
  lastLit = 0
  k = 0
  if (part) set({ preset: "sevenLamps", trim: part.trim })
}

function trigger(ev) {
  if (!part) return
  const bar = song.bar
  // in the lamps section the hits of the twelve steps are the lamps lit: one more for every second lamp
  if (!part.rhythm && part.lit(bar) !== lastLit) { lastLit = part.lit(bar); hits = genny.euclid(Math.ceil(Math.min(lastLit, 7) / 2), 12, 0) }
  const hit = hits.next()
  if (!hit || (part.on && !part.on(bar))) return
  const tones = shared.chordFor(song.section, bar).tones(part.lo, part.hi)
  // the lit lamps: tones of the chord, up to seven, spread over its range
  const lit = Math.min(part.lit(bar), 7)
  const m = Math.min(lit, tones.length)
  const lamps = Array.from({ length: m }, (_, i) => tones[Math.floor(i * tones.length / m)])
  // the lamps go round in a shape of their own for six notes, then another: more lamps, more ways round
  if (k % PHRASE === 0) { shape = shapes.next(); base = Math.floor(rnd.next() * m) }
  const j = k++ % PHRASE
  const p = lamps[(base + shared.shapes[shape](j, PHRASE)) % lamps.length]
  const vel = part.vel * shared.swell(song.section, song.progress)
  note(part.double ? [p, p + 12] : p, vel * (j === 0 ? 1.15 : 1), 12)
}
