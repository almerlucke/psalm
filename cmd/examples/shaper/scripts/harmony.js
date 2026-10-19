// HARMONY: the score. This script plays nothing: at its start it puts the chords of the piece and a small
// notation for lines into `shared`, the object every script of the patch sees, and the other scripts ask
// it in the middle of their own work. The piece is in D minor, dark and slow to change: a section has a
// progression, each chord for a number of bars (an Eb over the D in the bass is the dark one).
const progs = {
  Drone:    [["Dm9", 6], ["Ebmaj7/D", 3], ["Dm9", 3]],
  Pulse:    [["Dm7", 8], ["Ebmaj7/D", 4], ["Dm7", 4], ["Bbmaj7/D", 4]],
  Depth:    [["Dm7", 4], ["Gm9", 4], ["Ebmaj7", 4], ["Dm7", 4], ["A7sus4", 2], ["Dm7", 2]],
  Hollow:   [["Dm9", 4], ["Bbmaj7", 4], ["Dm9", 4]],
  Return:   [["Dm7", 4], ["Ebmaj7/D", 4], ["Dm7", 4], ["Gm9", 4], ["Dm7", 4]],
  Dissolve: [["Dm9", 6], ["Ebmaj7/D", 3], ["Dm9", 3]],
}
const progOf = { Drone: "Drone", Pulse: "Pulse", Depth: "Depth", Hollow: "Hollow", Return: "Return", Dissolve: "Dissolve" }

const perBar = {}
for (const [name, list] of Object.entries(progs)) {
  perBar[name] = list.flatMap(([symbol, bars]) => Array(bars).fill(symbol)).map(chord)
}

// the chord of a bar of a section
shared.chordFor = (section, bar) => {
  const bars = perBar[progOf[section] || "Drone"]
  return bars[bar % bars.length]
}

// a note fitted to a chord: a chord tone stays as it is, any other moves to the nearest chord tone within three
// semitones (the lower one when two are as near), so a figure written for one chord does not clash with the
// next: the A of a figure in D minor, a semitone from the Bb of an Eb chord, becomes that Bb
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

// the steps from step k to the next note or the end of the line (the length of a note that runs to it)
shared.until = (steps, k) => {
  for (let j = 1; j < steps.length; j++) if (steps[(k + j) % steps.length]) return j
  return steps.length
}
