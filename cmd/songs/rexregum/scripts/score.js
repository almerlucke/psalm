// SCORE: plays nothing. The notes are played by the midiplayer, from the file, straight to the synths. This script is for the parts that are made
// from the file and not in it: the program puts `const SCORE` (the notes of the pedal only: [start, pitch, length, velocity], start and length in
// sixteenths, a bar of the 3/4 is twelve) and `const SECTIONS` ([name, bars]) in front of this text, and it makes things for the other scripts in `shared`,
// the object every script of the patch sees:
//   shared.heldAt(part, step)   the pitch a part holds at a step of the piece (-1: none), for the pulse that follows the pedal
//   shared.stepNow(ev)          the step of the piece (sixteenths from the start)
//   shared.fadeIn(ev, bars)     0 at the start to 1 after some bars, for the parts that come in slowly
// A step of the piece is the bar of the song (the sections in a row) and the step of the clock in the bar.

const BAR = 12
const starts = []
{
  let bars = 0
  for (const [, n] of SECTIONS) { starts.push(bars); bars += n }
  shared.bars = bars
}

shared.stepNow = ev => (song.index < 0 ? -1 : (starts[song.index] + song.bar) * BAR + (ev.step % BAR))

// what a part holds at every step
const held = {}
for (const [part, list] of Object.entries(SCORE)) {
  const h = (held[part] = new Array(shared.bars * BAR).fill(-1))
  for (const [start, pitch, len] of list) for (let i = start; i < start + len && i < h.length; i++) h[i] = pitch
}
shared.heldAt = (part, step) => (held[part] && held[part][step] !== undefined ? held[part][step] : -1)

// shared.fadeIn(ev, bars): 0 at the start of the piece to 1 after `bars` bars, along half a cosine (a slow start, a soft arrival)
shared.fadeIn = (ev, bars) => {
  const t = Math.min(1, Math.max(0, shared.stepNow(ev) / (BAR * bars)))
  return 0.5 - 0.5 * Math.cos(Math.PI * t)
}
