// PIANO: the lead. A phrase is either comped or a line (the section says how often comped), and its repeats keep to it. Comping is the jazz piano: at every trigger
// of the rhythm script (the syncopations, the ratchets) a stab of the chord, rootless as a jazz pianist plays it, three or four of its upper tones (the third, the fifth,
// the seventh, the ninth) close together around the middle of the keyboard, short and soft, the bass having the root. A line is a melody by a markov chain. Every trigger of the rhythm script (a hit, one of a ratchet, a late one) is a note: the chain says how it goes from the
// one before, in steps of the scale (up one, down two, the same again) and now and then a leap of a fifth; at the edge of the register of the section it turns back.
// On the strong beats (the first of a quarter) a note that is not in the chord of the bar is moved to the nearest one that is, mostly, so the line leans on the harmony.
// It says things again: the phrase after a new one is the same notes (the strong beats moved to the new chord), and the fourth of a group is that again with a
// third of its notes drawn anew, so the ear can follow it.
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const moves = { same: 0, up1: 1, down1: -1, up2: 2, down2: -2, leapUp: 4, leapDown: -4 }
const chain = genny.markov({
  same:     { up1: 4, down1: 4, up2: 2, down2: 2, same: 1 },
  up1:      { up1: 3, down1: 2, up2: 2, same: 1, leapDown: 1, down2: 1 },
  down1:    { down1: 3, up1: 2, down2: 2, same: 1, leapUp: 1, up2: 1 },
  up2:      { down1: 4, down2: 2, up1: 2, leapDown: 1 },
  down2:    { up1: 4, up2: 2, down1: 2, leapUp: 1 },
  leapUp:   { down1: 4, down2: 3, same: 1 },
  leapDown: { up1: 4, up2: 3, same: 1 },
}, "same")
let pos = 8, phrase = -1, idx = 0, told = [], now = [], comping = false, lastStep = -1
// a stab: the upper tones of the chord of the bar, three or four close together, the middle of them nearest to the same place whatever the chord
function stab(ev, s) {
  const chd = shared.chordAt(shared.bar()), rootPc = chd.bass(0) % 12
  const tones = chd.tones(46, 78).filter(t => t % 12 !== rootPc)
  let at = 0
  for (let i = 0; i + 3 <= tones.length; i++) if (Math.abs(tones[i + 1] - 62) < Math.abs(tones[at + 1] - 62)) at = i
  const strong = ev.step % 4 === 0, grace = ev.step === lastStep
  lastStep = ev.step
  const voicing = tones.slice(at, at + (strong || R() < 0.4 ? 4 : 3))
  if (grace) return note(voicing[voicing.length - 1], 0.3, 1)
  note(voicing, (strong ? 0.5 : 0.4) + 0.08 * R(), 1.5)
}
function trigger(ev) {
  if (!shared.known()) return
  const s = shared.s(), sc = shared.scale, kind = shared.lead.kind
  const p = shared.phrase()
  if (p !== phrase) { phrase = p; idx = 0; if (kind === "new") { told = []; comping = R() < s.comp } now = [] }
  if (comping) return stab(ev, s)
  const lo = sc.findIndex(x => x >= s.reg[0]), hi = sc.length - 1 - [...sc].reverse().findIndex(x => x <= s.reg[1])
  const strong = ev.step % 4 === 0, chd = shared.chordAt(shared.bar())
  let pitch
  if (kind !== "new" && told[idx] !== undefined && !(kind === "var" && R() < 0.33)) {
    pitch = told[idx]
    if (strong && !chd.contains(pitch)) pitch = shared.nearest(chd.tones(s.reg[0], s.reg[1]), pitch)
  } else {
    let to = pos + moves[chain.next()]
    if (to < lo || to > hi) to = pos - (to - pos)       // back from the edge
    pos = Math.max(lo, Math.min(hi, to))
    pitch = sc[pos]
    if (strong && !chd.contains(pitch) && R() < 0.8) pitch = shared.nearest(chd.tones(s.reg[0], s.reg[1]), pitch)
  }
  const at = sc.indexOf(pitch)
  if (at >= 0) pos = at
  now.push(pitch)
  if (kind === "new" || told[idx] === undefined) told[idx] = pitch
  idx++
  note(pitch, (strong ? 0.6 : 0.46) + 0.1 * R(), 4)
}
