// LEAD: one voice of the shaper synth, and a different sound every section. The script plays the
// notes and plays the synth itself: set() sends it chains (as text), the macros m1..m4 that the
// chains read, the morph and the envelopes. The song module says which section this is.
const D = scale("D dorian")
const tones = D.notes(57, 83)                    // A3 to B5, the notes of D dorian in it
const high = tones.slice(8)

// a melody: a markov chain over the notes above, mostly steps, now and then a leap
function stepsOver(n) {
  const t = {}
  for (let i = 0; i < n; i++) {
    t[i] = {}
    for (const d of [-3, -2, -1, 0, 1, 2, 3]) {
      if (i + d >= 0 && i + d < n) t[i][i + d] = [1, 2, 4, 1, 4, 2, 1][d + 3]
    }
  }
  return t
}
const melody = genny.markov(stepsOver(tones.length), 8)
const next = () => tones[Number(melody.next())]

const walk = genny.rwalk(0.5, 0, 1, 0.12)       // a random walk, 0..1
const chaos = genny.logistic(3.9)               // a chaotic number, 0..1
const henon = genny.henon()                     // another, about -1.3..1.3
const sweep = genny.ramp(128, 0, 1, 1.6)        // 8 bars of 16 steps, rising
const rhythm = (hits, rot) => genny.euclid(hits, 16, rot || 0)

let eased = 0.5
const ease = (target, k) => (eased += (target - eased) * k)

// what every section starts from; a section puts its own on top
const base = {
  chainB: "", chainC: "", chainD: "", morph: 0, morphAmount: 0, shapeAmount: 0, macro1: 0.5,
  ampAttack: 3, ampDecay: 600, ampSustain: 0.7, ampRelease: 300,
  morphEnvAttack: 2, morphEnvDecay: 500, morphEnvSustain: 0.2, morphEnvRelease: 300,
  shapeEnvAttack: 2, shapeEnvDecay: 500, shapeEnvSustain: 0.2, shapeEnvRelease: 300,
  filterType: 0, vibratoDepth: 0.05, width: 0.4, level2: 0.3, octave2: 1, detune2: 6, level: 0.5,
}
const lowpass = (cutoff, resonance, amount) => ({ filterType: 1, cutoff, resonance, filterEnvAmount: amount, filterAttack: 3, filterDecay: 400, filterSustain: 0.5, filterRelease: 250 })

const scenes = {
  // a sine whose overtones grow as the note starts: the morph envelope walks it up through the chains
  Glass: {
    set: { chainA: "sin", chainB: "par(sin, mult 3 | sin | mult 0.4)", chainC: "sin | mult 3 | mirror -1 1",
           morphAmount: 0.85, morphEnvDecay: 900, morphEnvSustain: 0.05, ampDecay: 1800, ampSustain: 0.3, ampRelease: 700, level: 0.47 },
    hits: rhythm(3, 2),
    step(k) { if (this.hits.next()) note(next(), 0.7, 7) },
  },
  // a sine folded back on itself: the random walk turns the fold up and down, a step at a time
  Fold: {
    set: { chainA: "sin | mult m1[1,7] | mirror -1 1", chainB: "sin | mult m1[1,7] | mirror -1 1 | softclip 2", morph: 0.3,
           ampDecay: 350, ampSustain: 0.25, ampRelease: 400, level: 0.6 },
    arp: genny.arpeggio([tones[3], tones[5], tones[7], tones[9], tones[10], tones[12]], "alternate", "exclusive"),
    step(k) { set("macro1", walk.next()); if (k % 2 === 0) note(this.arp.next(), 0.6 + 0.3 * (k % 8 === 0), 2) },
  },
  // hard sync: a saw reset by a faster one, and the ratio sweeps up for the length of the section
  Sync: {
    set: { chainA: "mult m1[1,6] | mod 1 | bipolar", ...lowpass(4500, 0.2, 0.6), ampDecay: 500, ampSustain: 0.6, level: 0.52 },
    hits: rhythm(5, 0),
    step(k) { set("macro1", sweep.next()); if (this.hits.next()) note(next(), 0.8, 3) },
  },
  // a pulse whose width follows a chaotic number
  Pulse: {
    set: { chainA: "pulse m1[0.5,0.08] | bipolar", ...lowpass(3600, 0.15, 0.4), ampDecay: 300, ampSustain: 0.5, width: 0.7, detune2: 9, level2: 1, level: 0.45 },
    arp: genny.arpeggio(tones.slice(2, 10), "up", "inclusive"),
    step(k) { set("macro1", ease(chaos.next(), 0.5)); if (k % 2 === 0) note(this.arp.next(), k % 8 === 0 ? 0.9 : 0.6, 1.5) },
  },
  // four waves in a row, sine, triangle, saw, square: the morph goes through them in the length of the section
  Morph: {
    set: { chainA: "sin", chainB: "tri | bipolar", chainC: "bipolar", chainD: "pulse 0.5 | bipolar", ...lowpass(2800, 0.12, 0.5),
           ampAttack: 200, ampDecay: 600, ampSustain: 0.8, ampRelease: 900, vibratoDepth: 0.1, width: 0.8, level: 0.5 },
    step(k) { set("morph", song.progress); if (k % 8 === 0) note(next(), 0.7, 8) },
  },
  // whole number partials: struck pipes that start bright and settle
  Metal: {
    set: { chainA: "par(sin, mult 3 | sin | mult 0.2)", chainB: "par(sin, mult 3 | sin | mult 0.6, mult 7 | sin | mult 0.4)",
           chainC: "par(sin | mult 0.6, mult 5 | sin | mult 0.5, mult 9 | sin | mult 0.5, mult 13 | sin | mult 0.3)",
           morph: 0.2, morphAmount: 0.7, morphEnvDecay: 1400, morphEnvSustain: 0, ampDecay: 2600, ampSustain: 0, ampRelease: 900, level: 0.48 },
    pitches: genny.choice(tones.slice(0, 12), [4, 1, 3, 1, 3, 2, 1, 2, 1, 1, 1, 1]),
    hits: genny.bucket([1, 0, 0, 1, 0, 1, 0, 0], "shuffle"),
    step(k) { if (k % 2 === 0 && this.hits.next()) note(this.pitches.next(), 0.8, 12) },
  },
  // mouth shapes made of a few partials: the morph follows a chaotic number, smoothed, like talking
  Vowel: {
    set: { chainA: "par(sin, mult 2 | sin | mult 0.9, mult 3 | sin | mult 0.3, mult 8 | sin | mult 0.12)",
           chainB: "par(sin | mult 0.8, mult 2 | sin | mult 0.2, mult 4 | sin | mult 0.8, mult 5 | sin | mult 0.2)",
           chainC: "par(sin | mult 0.7, mult 3 | sin | mult 0.2, mult 6 | sin | mult 0.5, mult 9 | sin | mult 0.5)",
           ampAttack: 60, ampDecay: 400, ampSustain: 0.85, ampRelease: 300, vibratoDepth: 0.08, level: 0.45 },
    step(k) { set("morph", ease((henon.next() + 1.3) / 2.6, 0.3)); if (k % 4 === 0) note(next(), 0.7, 5) },
  },
  // a saw cut into steps: a rhythm of rough and clean, the hits of a euclidean rhythm set the macro
  Crush: {
    set: { chainA: "bipolar | quantize m1[0.5,0.04]", ...lowpass(4800, 0.1, 0.3), ampDecay: 200, ampSustain: 0.4, level: 0.62 },
    hits: rhythm(7, 3),
    arp: genny.arpeggio(tones.slice(4, 11), "converge"),
    step(k) { set("macro1", this.hits.next() ? 0.95 : 0.1); note(this.arp.next(), k % 4 === 0 ? 0.9 : 0.55, 1) },
  },
  // the script rewrites the chain itself, every bar another one from a list
  Rewrite: {
    set: { ...lowpass(5000, 0.12, 0.4), ampDecay: 400, ampSustain: 0.4, level: 0.55 },
    chains: genny.sequence([
      "bipolar | softclip 2",
      "sin | cheby 1:1 2:0.5 3:0.3 5:0.2",
      "tri | bipolar | mult 2 | mirror -1 1",
      "par(bipolar, mult 2 | bipolar | mult 0.5) | softclip 1.5",
      "sin | pow 0.5",
      "pulse 0.25 | bipolar | softclip 3",
    ]),
    arp: genny.arpeggio([tones[2], tones[4], tones[6], tones[8], tones[9]], "up", "exclusive"),
    step(k) {
      if (k % 16 === 0) { const c = this.chains.next(); set("chainA", c); log(c) }
      if (k % 2 === 0) note(this.arp.next(), 0.7, 1.5)
    },
  },
  // the script makes up chains: a base wave and one or two bends, picked by chance, every bar
  Chance: {
    set: { ...lowpass(4200, 0.15, 0.4), ampDecay: 500, ampSustain: 0.5, level: 0.5 },
    bases: genny.bucket(["sin", "bipolar", "tri | bipolar", "pulse 0.5 | bipolar", "par(sin, mult 2 | sin | mult 0.5)"]),
    bends: genny.bucket([k => `mult ${k} | mirror -1 1`, k => `softclip ${k}`, k => `mult ${k} | sin`, k => `tanh ${k}`, k => `quantize ${(1 / (k * 3)).toFixed(3)}`]),
    ks: genny.bucket([2, 3, 4, 5]),
    step(k) {
      if (k % 16 === 0) {
        let text = this.bases.next()
        for (let i = 0; i < 1 + (genny.rand(0, 1).next() > 0.5); i++) text += " | " + this.bends.next()(this.ks.next())
        set("chainA", text); log(text)
      }
      if (k % 2 === 0 && genny.rand(0, 1).next() < 0.7) note(next(), 0.7, 2)
    },
  },
  // a quick pluck: bright at the start, mellow at once, in runs of sixteenths
  Bells: {
    set: { chainA: "sin | mult 1.4 | softclip 1.5", chainB: "bipolar | softclip 2", chainC: "par(bipolar, mult 3 | bipolar | mult 0.5) | softclip 1.5",
           morphAmount: 1, morphEnvDecay: 260, morphEnvSustain: 0.05, ampDecay: 1100, ampSustain: 0.04, ampRelease: 260, level: 0.7 },
    arp: genny.arpeggio(high.slice(0, 8), "converge"),
    step(k) { if (k % 16 < 12) note(this.arp.next(), k % 4 === 0 ? 0.95 : 0.6, 1) },
  },
  // the sines again, slow, and the last note held out
  Finale: {
    set: { chainA: "sin", chainB: "par(sin, mult 3 | sin | mult 0.4)", morphAmount: 0.5, morphEnvDecay: 2500, morphEnvSustain: 0.15,
           ampAttack: 30, ampDecay: 3000, ampSustain: 0.4, ampRelease: 2500, level: 0.47 },
    step(k) { if (k % 32 === 0 && k < 96) note(high[(k / 32) * 2 + 1], 0.7, 28); if (k === 96) note(tones[14], 0.6, 60) },
  },
}

let scene = scenes.Glass
function section(name) {
  scene = scenes[name]
  if (!scene) return
  set({ ...base, ...scene.set })
  for (const g of Object.values(scene)) if (g && typeof g.reset === "function") g.reset()
  if (name !== "Morph" && name !== "Vowel") eased = 0.5
}

function trigger(ev) {
  if (scene && scene.step) scene.step.call(scene, song.bar * 16 + ev.step % 16)
}
