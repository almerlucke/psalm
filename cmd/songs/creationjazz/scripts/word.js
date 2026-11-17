// the Word sounds through the vocoder: the pad is the carrier, the voice of John 1:1 the modulator (a syllable of it on every beat of what the pad sings, see theword.js).
// In the intro the pad holds open fifths while the voice says the verse whole, twice. Then it is part of the band: it sings the chord of every bar (the close voicing of the
// harmony), held in the heads, on the long and the syncopated beat in the solo of the steel pan, in stabs in the break, in the Amen of the tag; and in the bridge of the head
// and the bridge and last A of the head out it sings the tune, an octave under the flute, and in every section it sings the answer to the motif with the marimba
clock("1/16")
const AMP = 0.6
const sing = (notes) => notes.map(n => Object.assign(n, { amp: n.amp || AMP }))

// the figures of the chord in a bar (three beats): long, a dotted rhythm, a syncopation, and stabs on the sixteenths where the pattern has an x
const figures = {
  long: br => [shared.chord(br.high, 3)],
  dotted: br => [shared.chord(br.high, 2), shared.chord(br.high, 1)],
  sync: br => [shared.N([], 0.5), shared.chord(br.high, 1.5), shared.chord(br.high, 1)],
  stabs: (br, b) => [...(b % 2 ? "x..x..x.x..." : "x..x..x...x.")].map(c => (c === "x" ? shared.chord(br.high, 0.25, { hold: 0.2 }) : shared.N([], 0.25))),
}
// the chords of the bars from to (excluded) of a day, in the figure `f(b)` names; where the motif is answered (see shared.statements) the Word sings the answer instead, an
// octave under the marimba that plays it
const chords = (day, from, to, f) => {
  const out = []
  for (let b = from; b < to; b++) {
    if (shared.answerAt(day, b) && b + 2 <= to) {
      out.push(...shared.transpose(shared.answers[day.number](), -12).map(n => Object.assign(n, { amp: 0.9 })))
      b++
    } else {
      out.push(...sing(figures[f(b)](shared.bar(day, b), b)))
    }
  }
  return out
}

const part = shared.register("the word", { legato: 1 }, day => {
  switch (day.number) {
    case 1: case 2: return null // the Word is silent in the intro and the head: it comes in with the solo
    case 3: return chords(day, 0, 32, b => (b % 4 === 3 ? "sync" : "long"))
    case 4: return chords(day, 0, 16, b => (b < 2 ? "long" : "stabs"))
    case 5: return chords(day, 0, 24, b => (b % 2 ? "long" : "dotted"))
    case 6: return shared.join(chords(day, 0, 16, () => "dotted"), shared.transpose(shared.headB(), -12), shared.transpose(shared.headA2(), -12))
    case 7: { // the Amen: the chords long, then IV and I, and the last chord rings under the void
      const amen = [shared.chord(shared.bar(day, 8).high, 3), shared.chord(shared.bar(day, 9).high, 9, { hold: 9 })]
      return shared.join(chords(day, 0, 8, () => "long"), sing(amen))
    }
  }
  return null
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
