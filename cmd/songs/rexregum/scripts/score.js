// SCORE: plays nothing. The program puts `const SCORE` and `const SECTIONS` in front of this text: the notes of the file by part
// ([start, pitch, length, velocity], start and length in sixteenths, a bar of the 3/4 is twelve, the velocity 0-127) and the
// sections ([name, bars]). At its start this script makes three things for the other scripts and puts them in `shared`, the
// object every script of the patch sees:
//   shared.notesNow(part, ev)   the notes of a part that start on this step of the clock: {pitch, len, vel} (vel 0-1)
//   shared.heldAt(part, step)   the pitch a part holds at a step of the piece (-1: none), for the parts that follow another
//   shared.stepNow(ev)          the step of the piece (sixteenths from the start)
// A step of the piece is the bar of the song (the sections in a row) and the step of the clock in the bar.

const BAR = 12
const starts = []
{
  let bars = 0
  for (const [, n] of SECTIONS) { starts.push(bars); bars += n }
  shared.bars = bars
}

const byStep = {}
for (const [part, list] of Object.entries(SCORE)) {
  const m = (byStep[part] = {})
  for (const [start, pitch, len, vel] of list) (m[start] = m[start] || []).push({ pitch, len, vel: vel / 127 })
}

shared.stepNow = ev => (song.index < 0 ? -1 : (starts[song.index] + song.bar) * BAR + (ev.step % BAR))

shared.notesNow = (part, ev) => {
  const s = shared.stepNow(ev)
  return s < 0 ? [] : (byStep[part] || {})[s] || []
}

// what a part holds at every step
const held = {}
for (const [part, list] of Object.entries(SCORE)) {
  const h = (held[part] = new Array(shared.bars * BAR).fill(-1))
  for (const [start, pitch, len] of list) for (let i = start; i < start + len && i < h.length; i++) h[i] = pitch
}
shared.heldAt = (part, step) => (held[part] && held[part][step] !== undefined ? held[part][step] : -1)
