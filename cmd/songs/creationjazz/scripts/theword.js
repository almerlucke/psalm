// the voice of the Word: in the intro it says John 1:1 whole, twice, into the darkness (the vocoder says it through the pad of open fifths); afterwards it is cut into its
// syllables, and one is spoken on every note the pad sings (the notes are those of "the Word" in word.js) and on every beat of a chord it holds, a markov chain walking
// the syllables, so the verse goes through the band in pieces: the tune sung loud, the chords softer
clock("1/16")
const play = shared.player("word")
let said = 0, walk = null, starts = {}, last = -1
function section(name) {
  if (name === shared.days[0].name) said = 0
  starts = {}
  const day = shared.day(name)
  if (day.number <= 2) return
  for (const e of shared.compile("the word", name)) {
    const chord = e.pitches.length > 1
    // a chord that is held says a syllable on every beat; a note of the tune one
    for (let t = e.beat; t < e.beat + (chord ? Math.max(e.hold, 0.25) : 0.25) - 0.01; t += 1) starts[Math.round(t * 4)] = { chord }
  }
}
function trigger(ev) {
  if (song.index <= 1) return
  if (song.index === 0) {
    if ((said === 0 && song.bar >= 1) || (said === 1 && song.bar >= 3)) { said++; note(35, 1, 4) }
    return
  }
  const pieces = shared.pieces.word
  const k = Math.round((song.bar * shared.beats + song.beat) * 4)
  if (k === last || !pieces || !pieces.length || !starts[k]) return
  last = k
  walk = walk || shared.walker(pieces.length, 0)
  const s = Math.round(song.beat * 4)
  shared.piece(play, note, ev, s, pieces[walk()], starts[k].chord ? 0.95 : 1, -26)
}
