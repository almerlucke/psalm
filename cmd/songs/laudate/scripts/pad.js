// PAD: the organ (the diapason, the foundation of a church organ). It does not hold a chord all the time: it plays in the phrases the score lists (padp: in which of
// the four phrases of a group, the chord of two bars) and rests in the others, and a markov chain says how it comes in: held for the two bars, late (from the third
// beat, a bar and a half), or short (a bar), so the chords come and go under the rest. The chord is four neighbouring tones whose middle is nearest to the same place (58)
// whatever the chord, so the voicing never jumps.
clock("1/4")
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const how = genny.markov({ held: { held: 3, late: 3, short: 2 }, late: { held: 4, short: 2, late: 1 }, short: { held: 4, late: 3, short: 1 } }, "held")
let style = "held", phrase = -1
function trigger(ev) {
  if (!shared.known()) return
  const s = shared.s(), bar = shared.bar(), beat = ev.step % 4
  if (bar % 2 !== 0 || !s.padp[shared.q()]) return
  if (beat === 0 && shared.phrase() !== phrase) { phrase = shared.phrase(); style = song.section === "Sanctuary" || song.section === "Hallelujah" ? "held" : how.next() }
  if (style === "late" ? beat !== 2 : beat !== 0) return
  const tones = shared.chordAt(bar).tones(46, 70)
  let at = 0
  for (let i = 0; i + 4 <= tones.length; i++) if (Math.abs(tones[i + 1] / 2 + tones[i + 2] / 2 - 58) < Math.abs(tones[at + 1] / 2 + tones[at + 2] / 2 - 58)) at = i
  note(tones.slice(at, at + 4), (style === "short" ? 0.4 : 0.45) * s.pad, style === "held" ? 8.5 : style === "late" ? 5.5 : 4.5)
}
