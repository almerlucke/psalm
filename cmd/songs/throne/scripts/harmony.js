// HARMONY: the score. This script plays nothing: at its start it puts the chords of the piece and a few helpers
// into `shared`, the object every script of the patch sees, and the other scripts ask it in the middle of their own work.
// The piece is in D major, all of it light: it is the throne room of God. The door opens on open fifths (a suspended second and
// fourth, neither major nor minor), the throne has a major seventh and ninth (the rainbow), and the four beasts keep to major
// chords with their ninths, high and clear; the elders and the holy are D major and the worthy ends in it. Every chord is held for a number of bars that is a holy one: 7 = 4 + 3 at the door, 3 for each
// of the seven lamps, 6 for each of the beasts, 2 for each of the 12 chords of the elders, 4 for each of the seven chords
// of the "holy".
const progs = {
  Door:      [["Dsus2", 4], ["Asus4", 3]],
  Throne:    [["Dsus2", 4], ["Dmaj9", 4], ["Gmaj7", 4]],
  Lamps:     [["D", 3], ["Bm7", 3], ["Gmaj7", 3], ["A", 3], ["F#m7", 3], ["Em7", 3], ["A7sus4", 3]],
  Glass:     [["Gmaj9", 4], ["Dmaj9", 4], ["Bm9", 4]],
  Creatures: [["Dmaj9", 6], ["Gmaj9", 6], ["Em9", 6], ["Aadd9", 6]],
  Elders:    [["D", 2], ["A/C#", 2], ["Bm7", 2], ["F#m7", 2], ["Gmaj7", 2], ["D/F#", 2], ["Em7", 2], ["A7sus4", 2], ["Bm7", 2], ["Gmaj7", 2], ["Asus4", 2], ["A", 2]],
  Holy:      [["D", 4], ["Gmaj7", 4], ["Bm7", 4], ["A", 4], ["D", 4], ["Gmaj7", 4], ["A7sus4", 4]],
  Worthy:    [["Dmaj9", 4], ["Gmaj7", 4], ["Em7", 2], ["A7sus4", 2], ["Dmaj9", 4]],
}

const perBar = {}
for (const [name, list] of Object.entries(progs)) {
  perBar[name] = list.flatMap(([symbol, bars]) => Array(bars).fill(symbol)).map(chord)
}

// the chord of a bar of a section
shared.chordFor = (section, bar) => {
  const bars = perBar[section] || perBar.Door
  return bars[bar % bars.length]
}

// a note fitted to a chord: a chord tone stays as it is, any other moves to the nearest chord tone within three
// semitones (the lower one when two are as near), so a line written for one chord does not clash with the next
shared.fit = (p, c) => {
  const near = c.tones(p - 3, p + 3)
  if (near.length === 0 || near.includes(p)) return p
  return near.reduce((best, t) => (Math.abs(t - p) < Math.abs(best - p) || (Math.abs(t - p) === Math.abs(best - p) && t < best)) ? t : best)
}

// the shapes an arpeggio can take over the tones of a chord (so it is not a scale going up and down): k is the note of the phrase
// (0 to h - 1), h the notes in it, and the result how many tones of the chord above the base of the phrase. Plain functions, used by the
// harp, the lamps and the stars, each of which chooses the next shape by a markov chain with weights 3, 7 and 12 (shared.shapeTable).
shared.shapes = {
  run:    (k, h) => k,                                        // up by steps
  thirds: (k, h) => 2 * k,                                    // up by thirds
  fall:   (k, h) => 2 * (h - 1 - k),                          // down by thirds from the top
  arch:   (k, h) => 2 * Math.min(k, h - 1 - k),               // up and down again
  wave:   (k, h) => [0, 2, 1, 3, 2, 4, 3, 5][k % 8],          // two forward, one back
  skip:   (k, h) => (k % 2) * 4 + (k >> 1),                   // a low line and a high line, turn and turn about
  pedal:  (k, h) => (k % 2) * (2 + (k >> 1)),                 // the low note again and again, the upper ones climbing
  spread: (k, h) => [0, 4, 2, 6, 1, 5, 3, 7][k % 8],          // broken wide
}
shared.shapeTable = {
  run:    { thirds: 12, arch: 7, wave: 7, fall: 3, skip: 3 },
  thirds: { arch: 12, wave: 7, spread: 7, fall: 3, pedal: 3 },
  fall:   { wave: 12, arch: 7, skip: 7, run: 3, pedal: 3 },
  arch:   { fall: 12, wave: 7, spread: 7, run: 3, pedal: 3, thirds: 3 },
  wave:   { arch: 12, skip: 7, pedal: 7, run: 3, fall: 3, spread: 3 },
  skip:   { wave: 12, spread: 7, thirds: 7, fall: 3, pedal: 3 },
  pedal:  { run: 12, wave: 7, spread: 7, arch: 3, fall: 3 },
  spread: { fall: 12, arch: 7, run: 7, wave: 3, skip: 3, thirds: 3 },
}

// how loud the lines play through a section: a little more as it goes, but the worthy, the last, only fades, so the piece comes back
// to the quiet it began in (the end joins the door when it loops)
shared.swell = (section, progress) => section === "Worthy" ? 1 - 0.75 * progress : 0.7 + 0.3 * progress

// the four beasts of the Creatures, six bars each: the lion, the calf (an ox), the man and the flying eagle, from the
// lowest to the highest register, the pitch in the middle of each one's range
shared.beastOf = bar => Math.min(3, Math.floor(bar / 6))
shared.beastCentre = [62, 66, 69, 79]
