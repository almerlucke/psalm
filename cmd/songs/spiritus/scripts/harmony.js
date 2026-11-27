// HARMONY: the score. This script plays nothing: at its start it puts the chords of the piece and a small
// notation for lines into `shared`, the object every script of the patch sees. The piece is in D major, slow and
// open (sus and add chords, no hard edges), and in the Glory it lifts a whole step to E major: from glory to glory.
// The chords are the I, vi, IV and V of the key with a suspension before the V, and the harmony only
// moves every two to four bars, so the voices have time to bloom.
const progs = {
  Waiting:    [["Dsus2", 4], ["Gmaj7/D", 2], ["Dsus2", 2]],
  Wind:       [["Dsus2", 3], ["Gmaj9", 3], ["Dsus2", 2], ["Asus4", 2], ["A", 2]],
  Fire:       [["Dmaj9", 4], ["Bm9", 4], ["Gmaj7", 4], ["Asus4", 2], ["A", 2]],
  Indwelling: [["Dmaj9", 4], ["F#m7", 4], ["Bm9", 4], ["Gmaj9", 4], ["Asus4", 2], ["A", 2]],
  Glory:      [["Emaj9", 4], ["C#m7", 4], ["Amaj9", 4], ["F#m7", 4], ["Bsus4", 2], ["B", 2]],
  Amen:       [["Gmaj9", 4], ["Dmaj9", 4], ["Gmaj7", 2], ["Dadd9", 2]],
}
const perBar = {}
for (const [name, list] of Object.entries(progs)) {
  perBar[name] = list.flatMap(([symbol, bars]) => Array(bars).fill(symbol)).map(chord)
}

// the chord of a bar of a section (the same object while the chord goes on, so `===` says it did not change)
shared.chordFor = (section, bar) => {
  const bars = perBar[section] || perBar.Waiting
  return bars[bar % bars.length]
}

// how many bars from this one the chord lasts (at most to the end of the section)
shared.lasts = (section, bar, bars) => {
  const c = shared.chordFor(section, bar)
  let n = 1
  while (bar + n < bars && shared.chordFor(section, bar + n) === c) n++
  return n
}

// the notes of the chord in a window of an octave from `low`: the root, the fifth and a colour tone (a third, a second,
// a fourth), each once. A voice of a choir that sings them is not a cluster.
shared.voices = (c, low) => {
  const t = c.tones(low, low + 11.99), r = c.root
  const iv = n => (((n - r) % 12) + 12) % 12
  const pick = set => { for (const i of set) { const n = t.find(n => iv(n) === i); if (n !== undefined) return n } }
  return [pick([0]), pick([7]), pick([4, 3, 2, 5])].filter(n => n !== undefined)
}

// the ninth and the seventh of the chord, for the voices above
shared.colours = (c, low) => {
  const t = c.tones(low, low + 11.99), r = c.root
  const iv = n => (((n - r) % 12) + 12) % 12
  return t.filter(n => [2, 9, 10, 11].includes(iv(n)))
}

// a note fitted to a chord: a chord tone stays as it is, any other moves to the nearest chord tone within three semitones
shared.fit = (p, c) => {
  const near = c.tones(p - 3, p + 3)
  if (near.length === 0 || near.includes(p)) return p
  return near.reduce((best, t) => (Math.abs(t - p) < Math.abs(best - p) || (Math.abs(t - p) === Math.abs(best - p) && t < best)) ? t : best)
}

// a line in words: a note a step, `.` a rest, `-` the note before goes on. Gives an array with, for each
// step, null or {pitch, len} (len in steps)
shared.line = text => {
  const tokens = text.trim().split(/\s+/)
  const steps = tokens.map(() => null)
  let on = -1
  tokens.forEach((t, i) => {
    if (t === "-") { if (on >= 0) steps[on].len++ }
    else if (t === ".") on = -1
    else { steps[i] = { pitch: pitch(t), len: 1 }; on = i }
  })
  return steps
}

// a list turned k places (the voices of a chord, and the chords after it, start in other places of a chant)
shared.rot = (a, k) => { const n = a.length; if (!n) return a; const i = ((k % n) + n) % n; return a.slice(i).concat(a.slice(0, i)) }
