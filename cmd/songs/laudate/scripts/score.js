// SCORE: Psalm 150, "Praise ye the LORD". Plays nothing. At its start it puts what the others need into `shared` (before this line shared.sounds is there, the
// patch has put it: the pieces of the tubes, the percussion, the deep drums, the hats and the tambourine, the sung notes, the pitches of the glass and the harp). Where in the piece it is (the bar, the section), the amounts
// of each part in each section, the scale of F minor, and the chords: a markov chain of its own, one chord for two bars, that begins and ends on F minor so the
// piece loops. Each section is a verse of the psalm, and the verse is written to the log of the patch as it begins (the Log button).
//
//  Sanctuary   8 bars  "Praise God in his sanctuary: praise him in the firmament of his power"  a pad, a bass far down, glass bells, a long note sung
//  Greatness  12       "Praise him for his mighty acts: praise him according to his excellent greatness"  the piano comes in on the rhythm script, the harp
//  Dance      16       "the trumpet, the psaltery and harp, the timbrel and dance, stringed instruments and organs"  everything against everything
//  Cymbals    12       "Praise him upon the loud cymbals: upon the high sounding cymbals"  the most hits, ratchets, the piano high
//  Breath     12       "Let every thing that hath breath praise the LORD"  the rhythm thins, bells and the voice
//  Hallelujah  6       "Praise ye the LORD"  the pad and one long note; the last chord rings out
const starts = { Sanctuary: 0, Greatness: 8, Dance: 20, Cymbals: 36, Breath: 48, Hallelujah: 60 }
const bars = 66
shared.starts = starts
shared.known = () => song.section in starts
shared.bar = () => starts[song.section] + song.bar
// how much of each part (padp: the phrases the organ plays in; perc and tamb: the hand percussion and the tambourine loop), the euclidean hits of the lead's rhythm in sixteen steps, how many of them are ratchets, its register. The piece goes in phrases of two
// bars, four to a group: the first says something (A), the second says it again, the third is the answer (the lead rests, the harp or the bells play), the fourth is
// the first again, a little changed (A'). The lists say in which of the four phrases a part plays.
shared.sec = {
  Sanctuary:  { perc: 0,   pp: [0, 0, 0, 0], kit: 0,   kp: [0, 0, 0, 0], hat: 0.4, tamb: 0,   tp: [0, 0, 0, 0], lead: 0,   hits: 0, rat: 0,   lp: [0, 0, 0, 0], comp: 0,   glass: 0.5, gp: [1, 1, 1, 1], harp: 0,   hp: [0, 0, 0, 0], tubes: 0,   bass: 0.5, pad: 1, padp: [1, 1, 1, 1],   voice: 0.8, reg: [60, 76] },
  Greatness:  { perc: 0.5, pp: [1, 1, 0, 1], kit: 0.3, kp: [0, 0, 1, 0], hat: 0.6, tamb: 0,   tp: [0, 0, 0, 0], lead: 0.6, hits: 6, rat: 0.1, lp: [1, 1, 0, 1], comp: 0.5, glass: 0.6, gp: [0, 0, 1, 0], harp: 0.5, hp: [0, 0, 1, 0], tubes: 0,   bass: 0.8, pad: 1, padp: [1, 0, 1, 0],   voice: 0.6, reg: [60, 79] },
  Dance:      { perc: 0.7, pp: [1, 1, 1, 1], kit: 0.6, kp: [0, 1, 0, 1], hat: 0.8, tamb: 0.5, tp: [0, 1, 1, 1], lead: 0.8, hits: 7, rat: 0.3, lp: [1, 1, 0, 1], comp: 0.8, glass: 0.5, gp: [0, 0, 1, 0], harp: 0.8, hp: [0, 1, 1, 1], tubes: 0.8, bass: 1,   pad: 0.8, padp: [0, 1, 0, 1], voice: 0.4, reg: [57, 81] },
  Cymbals:    { perc: 1,   pp: [1, 1, 1, 1], kit: 1,   kp: [1, 1, 1, 1], hat: 1,   tamb: 1,   tp: [1, 1, 1, 1], lead: 1,   hits: 9, rat: 0.6, lp: [1, 1, 1, 1], comp: 0.6, glass: 0.4, gp: [0, 0, 0, 1], harp: 1,   hp: [1, 0, 1, 1], tubes: 1,   bass: 1,   pad: 0.7, padp: [1, 0, 0, 1], voice: 0,   reg: [62, 84] },
  Breath:     { perc: 0.5, pp: [1, 1, 0, 1], kit: 0.4, kp: [0, 0, 1, 0], hat: 0.6, tamb: 0.4, tp: [0, 0, 0, 1], lead: 0.6, hits: 6, rat: 0.2, lp: [1, 1, 0, 1], comp: 0.4, glass: 0.7, gp: [0, 1, 0, 1], harp: 0.5, hp: [0, 1, 0, 0], tubes: 0.5, bass: 0.6, pad: 1, padp: [1, 1, 0, 1],   voice: 0.5, reg: [60, 76] },
  Hallelujah: { perc: 0,   pp: [0, 0, 0, 0], kit: 0,   kp: [0, 0, 0, 0], hat: 0,   tamb: 0,   tp: [0, 0, 0, 0], lead: 0,   hits: 0, rat: 0,   lp: [0, 0, 0, 0], comp: 0,   glass: 0.5, gp: [1, 1, 1, 1], harp: 0,   hp: [0, 0, 0, 0], tubes: 0,   bass: 0.3, pad: 1, padp: [1, 1, 1, 1],   voice: 1,   reg: [60, 72] },
}
shared.phrase = () => Math.floor(shared.bar() / 2)          // the phrase of two bars
shared.q = () => shared.phrase() % 4                        // which of the four of its group
shared.lead = { on: false, kind: "new" }                    // what the rhythm script says of the phrase going on: the lead plays, new or again
shared.s = () => shared.sec[song.section]

// the scale: F natural minor
const pcs = [5, 7, 8, 10, 0, 1, 3]
shared.scale = []
for (let p = 36; p <= 96; p++) if (pcs.includes(p % 12)) shared.scale.push(p)

// the chords: a chain, one for two bars
// jazz changes: the sevenths and ninths of F minor and the dominants that lead home (Gm7b5 to C7 to Fm9, Eb9 to Abmaj7, Bb9 to Ebmaj7)
const next = genny.markov({
  Fm9:    { Dbmaj7: 5, Abmaj7: 3, Bbm7: 4, Cm7: 2, Ebmaj7: 2, Gm7b5: 2 },
  Dbmaj7: { Abmaj7: 4, Fm9: 2, Ebmaj7: 3, Bbm7: 3, Gm7b5: 2 },
  Abmaj7: { Ebmaj7: 3, Dbmaj7: 4, Fm9: 2, Bbm7: 2, Cm7: 2, Gm7b5: 1 },
  Bbm7:   { Eb9: 5, Fm9: 3, Cm7: 3, Dbmaj7: 2, C7: 2 },
  Cm7:    { Fm9: 3, Abmaj7: 3, Bbm7: 2, Dbmaj7: 2, Bb9: 2 },
  Ebmaj7: { Abmaj7: 4, Fm9: 2, Dbmaj7: 3, Cm7: 2, Bb9: 1 },
  Gm7b5:  { C7: 8, Cm7: 1 },
  C7:     { Fm9: 8, Abmaj7: 3, Dbmaj7: 2 },
  Eb9:    { Abmaj7: 7, Dbmaj7: 2, Fm9: 2 },
  Bb9:    { Ebmaj7: 6, Eb9: 2, Fm9: 2 },
}, "Fm9")
const symbols = []
for (let i = 0; i < bars / 2; i++) symbols.push(next.next())
symbols[0] = "Fm9"; symbols[symbols.length - 1] = "Fm9"
const chords = symbols.map(chord)
shared.chordAt = bar => chords[Math.max(0, Math.min(chords.length - 1, Math.floor(bar / 2)))]
shared.chordName = bar => symbols[Math.max(0, Math.min(symbols.length - 1, Math.floor(bar / 2)))]
// does a pitch clash with a chord: a half tone from one of its tones and not one itself (the E over C7 against the Eb of the scale, the A over Bbm7)
shared.clashes = (chd, p) => !chd.contains(p) && chd.tones(0, 127).some(t => { const d = ((t - p) % 12 + 12) % 12; return d === 1 || d === 11 })
// the note of a list nearest to a pitch
shared.nearest = (list, p) => list.reduce((a, b) => Math.abs(b - p) < Math.abs(a - p) ? b : a)
// a level for a piece of a recording: quiet ones told a little louder, loud ones softer (the velocity goes in squares)
shared.evenOut = (db, ref) => Math.max(0.5, Math.min(1.6, Math.pow(10, (ref - db) / 40)))
log("chords:", symbols.join(" "))

// the verse of the psalm (KJV) as its section begins, in the log
const verses = {
  Sanctuary: "Psalm 150:1 Praise ye the LORD. Praise God in his sanctuary: praise him in the firmament of his power.",
  Greatness: "Psalm 150:2 Praise him for his mighty acts: praise him according to his excellent greatness.",
  Dance: "Psalm 150:3-4 Praise him with the sound of the trumpet: praise him with the psaltery and harp. Praise him with the timbrel and dance: praise him with stringed instruments and organs.",
  Cymbals: "Psalm 150:5 Praise him upon the loud cymbals: praise him upon the high sounding cymbals.",
  Breath: "Psalm 150:6 Let every thing that hath breath praise the LORD.",
  Hallelujah: "Praise ye the LORD.",
}
function section(name) { log(verses[name]) }
