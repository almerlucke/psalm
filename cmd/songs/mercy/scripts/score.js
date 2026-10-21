// SCORE: Psalm 136, twenty-six verses of four bars (the length of the tabla loop), each ending with the same refrain, "for his mercy
// endureth for ever". This script plays nothing: at its start it puts what the others need into `shared` (before this line shared.slices
// is there, the patch has put it: where the pieces of every cut sound are, in sixteenths of the tempo). Where in the psalm the piece is
// (shared.verse, shared.verseBar, shared.step), the chord of each verse, how much of each layer plays in it, how the recording is told over the verses,
// and the fade at the start and the end, so the piece loops without a seam.
//
//  1-3    give thanks, to the God of gods, the Lord of lords              Thanks     12 bars
//  4-9    great wonders: the heavens, the earth on the waters, the lights, the sun by day, the moon and stars by night    Creation  24
//  10-15  Egypt smitten, Israel brought out, the Red Sea divided and crossed, Pharaoh overthrown     Egypt  24
//  16-22  the wilderness, great kings smitten, Sihon and Og, the land a heritage for Israel          Wilderness  28
//  23-25  he remembered us in our low estate, redeemed us, gives food to all flesh                    Remembered  12
//  26     give thanks unto the God of heaven                                                          Heaven  4
const starts = { Thanks: 0, Creation: 12, Egypt: 36, Wilderness: 60, Remembered: 88, Heaven: 100 }
shared.globalBar = () => starts[song.section] + song.bar
shared.verse = () => Math.min(25, Math.floor(shared.globalBar() / 4))
shared.verseBar = () => shared.globalBar() % 4
// the step in the verse, 0 to 63
shared.step = ev => shared.verseBar() * 16 + (ev.step % 16)
// (the coda, the last four bars of Heaven, is no verse: the scripts keep quiet and only what was started rings out)
shared.coda = () => song.section === "Heaven" && song.bar >= 4
shared.known = () => song.section in starts && !(song.section === "Heaven" && song.bar >= 4)

// the chord of each verse (F minor): the bass of the verse is its root
const symbols = [
  "Fm9", "Dbmaj7", "Abmaj7",                                            // thanks
  "Abmaj7", "Dbmaj7", "Abmaj7", "Fm9", "Dbmaj9", "Cm7",                 // creation
  "Fm7", "Gbmaj7", "Fm7", "Ebsus2", "Dbmaj7", "Fm7",                    // egypt: the Gb, the sea divided on an Eb, back to F
  "Fm9", "Dbmaj7", "Bbm7", "Fm9", "Eb", "Abmaj7", "Dbmaj7",             // wilderness
  "Bbm7", "Dbmaj7", "Abmaj7",                                           // remembered
  "Fm9",                                                                // heaven
]
const chords = symbols.map(chord)
shared.chordOf = v => chords[Math.max(0, Math.min(25, v))]

// how much the tabla loop is remixed in a verse (0 to 4 operations)
shared.heat = [0, 1, 2, 1, 1, 2, 2, 2, 2, 3, 3, 3, 4, 3, 4, 2, 2, 2, 2, 2, 2, 2, 1, 1, 1, 0]
// how much of each layer plays in each verse
shared.layer = {
  perc:  [0, 0, 0.4, 0.3, 0.4, 0.4, 0.5, 0.5, 0.5, 0.8, 0.8, 0.8, 0.3, 0.8, 1, 0.5, 0.7, 0.7, 0.7, 0.7, 0.5, 0.4, 0.2, 0.2, 0.2, 0.4],
  tubes: [0, 0, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.7, 0.3, 0.3, 0.3, 0.2, 0.3, 0.3, 0.4, 0.3, 0.3, 0.3, 0.3, 0.4, 0.5, 0.4, 0.4, 0.4, 0.5],
  harp:  [0, 0, 0, 0.5, 0.5, 0.5, 0.6, 0.7, 0.7, 0, 0, 0, 0, 0.3, 0, 0.5, 0.5, 0.5, 0.5, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.7],
  bed:   [0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.6, 0.6, 0.6, 0.4, 0.4, 0.4, 0.3, 0.4, 0.4, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5],
}
// the verses the birds sing in: the earth and the waters, the lights, the land, the food of all flesh
shared.birds = [4, 5, 7, 8, 21, 22, 24, 25]
// the horn calls (a fifth up and a fourth) and the shofar sounds
shared.horn = [0, 9, 12, 14, 25]
shared.shofar = { 0: 0, 9: 0, 12: 0, 14: 3, 25: 0 }       // the verse: the bar it sounds in

// the fade: the piece begins out of the quiet it ended in, and ends in it
shared.fade = () => {
  const v = shared.verse(), b = shared.verseBar()
  if (v === 0) return [0.6, 0.75, 0.9, 1][b]               // (the sampler's velocity goes in squares: 0.6 is a third of the level)
  if (v === 25) return [1, 0.9, 0.75, 0.6][b]
  return 1
}
// the voice does not fade in the last verse: its refrain is the end, and has to be heard
shared.voiceFade = () => shared.verse() === 25 ? 1 : shared.fade()

// THE RECORDING: the psalm spoken, the 26 verses each followed by its refrain, cut at the pauses (52 pieces, patch.go): verse n of the song says verse n of
// the recording, at the start of the verse (after the shofar when that sounds in the first bar), and its refrain in the last bar, the refrain bar, over the choir and the tabla loop as it was.
// A piece that is quiet is told a little louder, a loud one a little softer (the velocity goes in squares, 40 dB to a factor ten).
shared.loud = p => p.db > -50
shared.evenOut = (p, ref) => Math.max(0.6, Math.min(1.4, Math.pow(10, (ref - p.db) / 40)))
const pieces = shared.slices.chant
shared.calls = []
for (let v = 0; v < 26; v++) {
  const verse = pieces[2 * v], refrain = pieces[2 * v + 1]
  shared.calls.push([
    { key: verse.key, at: shared.shofar[v] === 0 ? 16 : 2, len: Math.min(verse.len, 44), gain: shared.evenOut(verse, -22) },
    { key: refrain.key, at: 48, len: Math.min(refrain.len, 16), gain: shared.evenOut(refrain, -22) },
  ])
}
