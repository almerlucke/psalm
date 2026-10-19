// BASS: a second voice of the shaper synth, a sine under everything that grows a fold when it is asked
// to: chain B is a sine folded back on itself, and the morph from one to the other is the growl.
// The script follows the song: a root a bar (D, Bb, F, C), and a different way of playing it in each
// section.
const roots = [38, 34, 41, 36]
const euclid = genny.euclid(5, 16, 0)
const octave = genny.sequence([0, 0, 12, 0, 0, 12, 7, 0])

// how each section plays it: "held" a note a bar, "pulse" eighths with octaves, "euclid" a rhythm
const style = { Glass: "held", Fold: "pulse", Sync: "pulse", Pulse: "euclid", Morph: "held", Metal: "held",
                Vowel: "held", Crush: "euclid", Rewrite: "euclid", Chance: "euclid", Bells: "pulse", Finale: "held" }

const sound = {
  chainA: "sin", chainB: "sin | mult m1[1.5,5] | mirror -1 1 | softclip 1.3", macro1: 0.4, morph: 0,
  ampAttack: 4, ampDecay: 700, ampSustain: 0.8, ampRelease: 200, octave2: -1, level2: 0.5, detune2: 3, level: 0.9,
  filterType: 1, cutoff: 700, resonance: 0.2, filterEnvAmount: 1, filterAttack: 3, filterDecay: 300, filterSustain: 0.3, filterRelease: 150,
}

let how = "held"
function section(name) {
  how = style[name] || "held"
  octave.reset(); euclid.reset()
  set(sound)
}

function trigger(ev) {
  const k = ev.step % 16
  const root = roots[song.bar % 4]
  // the fold opens over the section, and falls back at the end
  if (k % 4 === 0) set("morph", 0.15 + 0.6 * Math.sin(Math.PI * song.progress))
  if (how === "held") {
    if (k === 0) note(root, 0.9, 15)
  } else if (how === "pulse") {
    if (k % 2 === 0) note(root + octave.next(), k % 4 === 0 ? 0.9 : 0.6, 1.6)
  } else if (euclid.next()) {
    note(root + (k % 8 === 7 ? 12 : 0), 0.85, 1.5)
  }
}
