// PAD: a third voice of the shaper synth playing the chords, four bars of Dm, Bb, F, C over and over
// (Dm: D F A C, Bb: Bb D F A, F: F A C E, C: C E G B). What it plays on changes with the section:
// the script picks a family of chains (sine, folded, organ, vowel, a saw with a filter) for each and
// sets them with set().
const chords = [["D3", "F3", "A3", "C4"], ["Bb2", "D3", "F3", "A3"], ["F3", "A3", "C4", "E4"], ["C3", "E3", "G3", "B3"]]
const walk = genny.rwalk(0.5, 0, 1, 0.2)

const families = {
  sines: { chainA: "sin", chainB: "par(sin, mult 2 | sin | mult 0.4)", chainC: "sin | mult 2 | mirror -1 1", chainD: "", filterType: 0 },
  fold:  { chainA: "sin | mult m1[1.5,5] | mirror -1 1", chainB: "sin | mult m1[2,6] | mirror -1 1 | softclip 1.5", chainC: "", chainD: "", filterType: 1, cutoff: 3000, resonance: 0.15 },
  organ: { chainA: "par(sin, mult 2 | sin | mult 0.5)", chainB: "par(sin, mult 2 | sin | mult 0.5, mult 3 | sin | mult 0.4, mult 4 | sin | mult 0.3)",
           chainC: "par(sin, mult 2 | sin | mult 0.8, mult 3 | sin | mult 0.7, mult 4 | sin | mult 0.6, mult 6 | sin | mult 0.5)", chainD: "", filterType: 0 },
  vowel: { chainA: "par(sin | mult 0.8, mult 2 | sin | mult 0.2, mult 4 | sin | mult 0.8, mult 5 | sin | mult 0.2)",
           chainB: "par(sin | mult 0.7, mult 3 | sin | mult 0.2, mult 6 | sin | mult 0.5, mult 9 | sin | mult 0.5)", chainC: "", chainD: "", filterType: 0 },
  saws:  { chainA: "bipolar | softclip 1.2", chainB: "bipolar | mult m1[1,2.5] | mirror -1 1", chainC: "", chainD: "", filterType: 1, cutoff: 1800, resonance: 0.25 },
}
const family = { Glass: "sines", Fold: "fold", Sync: "saws", Pulse: "fold", Morph: "sines", Metal: "organ",
                 Vowel: "vowel", Crush: "saws", Rewrite: "organ", Chance: "fold", Bells: "vowel", Finale: "sines" }

let name = "sines"
function section(sectionName) {
  name = family[sectionName] || "sines"
  set({
    macro1: 0.5, morph: 0.4, morphAmount: 0, shapeAmount: 0, width: 0.9, level2: 0.8, detune2: 9, octave2: 0, vibratoDepth: 0.04,
    ampAttack: 700, ampDecay: 900, ampSustain: 0.85, ampRelease: 1500, level: 0.4,
    filterEnvAmount: 0.6, filterAttack: 600, filterDecay: 800, filterSustain: 0.6, filterRelease: 1200,
    ...families[name],
  })
}

function trigger(ev) {
  const k = ev.step % 16
  if (k === 0) {
    // the chord of the bar, and a new place in the morph and the macro to play it at
    set({ morph: 0.15 + 0.7 * walk.next(), macro1: song.progress })
    note(chords[song.bar % 4], 0.5, 15.5)
  }
}
