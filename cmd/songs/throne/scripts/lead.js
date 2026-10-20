// LEAD: the voice of the piece, a different one in each part, all of the shaper. A voice as of a trumpet calls John up at the
// door, three notes of a growing touch (a breath, then brass). In the lamps and the sea of glass the eyes draw every note
// anew (manyEyes). The four beasts each have a voice of their own, six bars each: a flute for the lion, a marimba for the calf, a
// voice for the man, an airy pad for the eagle, the line rising from one to the next. In the elders a soft flute again, as in the throne. And in the holy
// the call is said by bells, three times, three notes at a time ("holy, holy, holy"), and answered.
// The tune is a walk of a markov chain over the tones of the chord of the bar, the weights holy numbers: staying on a note 12,
// a step 7, a leap of two 3 (after a leap the step back is likeliest), fitted to the chord (shared.fit).
// steps along the tones of the chord; the leaps of three are rare, and after one the step back is the likeliest
const stepOf = { dn3: -3, dn2: -2, dn1: -1, same: 0, up1: 1, up2: 2, up3: 3 }
const walk = genny.markov({
  dn3:  { up1: 12, up2: 7, same: 3 },
  dn2:  { up1: 12, same: 7, dn1: 3 },
  dn1:  { dn1: 7, same: 12, up1: 7, dn2: 3, up2: 3 },
  same: { same: 3, dn1: 7, up1: 7, dn2: 3, up2: 3, dn3: 3, up3: 3 },
  up1:  { up1: 7, same: 12, dn1: 7, up2: 3, dn2: 3 },
  up2:  { dn1: 12, same: 7, up1: 3 },
  up3:  { dn1: 12, dn2: 7, same: 3 },
}, "same")
// a tune is not only a walk: now and then the last three steps come again (a sequence, on the tones the walk has reached)
const recent = []
const rnd = genny.rand(0, 1)
let echo = [], rest = 0
function nextStep() {
  if (echo.length > 0) return echo.shift()
  const s = stepOf[walk.next()]
  recent.push(s)
  if (recent.length > 3) recent.shift()
  // one time in five the last three steps come again, once, and not again for a while
  if (rest > 0) rest--
  else if (recent.length === 3 && rnd.next() < 0.2) { echo = recent.slice(); recent.length = 0; rest = 8 }
  return s
}
let index = null            // where the tune is among the tones of the chord
const hits = {}             // the rhythms: euclidean ones, seven of twelve and so on, against the sixteenths
function euclid(name, h, n, r) { return hits[name] || (hits[name] = genny.euclid(h, n, r)) }

// a step of the walk along the tones of the chord of the bar, around a centre
function walkTo(centre, bar) {
  const tones = shared.chordFor(song.section, bar).tones(centre - 12, centre + 12)
  if (index === null || index >= tones.length) index = Math.floor(tones.length / 2)
  index = Math.max(0, Math.min(tones.length - 1, index + nextStep()))
  return tones[index]
}

const parts = {
  // "come up hither": D4 soft, A4, D5 louder, two bars between (a rising fifth and fourth, three notes), the first only a breath after
  // the drone has faded in
  Door: { preset: "trumpetVoice", trim: -5, mono: 0, legato: 0, calls: { 2: [62, 0.15, 28], 4: [69, 0.4, 28], 6: [74, 0.7, 20] } },
  Throne: { preset: "fluteLead", trim: -2, mono: 0, legato: 0, centre: 69, rhythm: [2, 12], on: bar => bar >= 6, len: 10, vel: 0.35 },
  Lamps: { preset: "manyEyes", trim: 0, mono: 0, legato: 0, centre: 72, rhythm: [2, 12], on: bar => bar >= 15, len: 10, vel: 0.4 },
  Glass: { preset: "manyEyes", trim: -6, mono: 0, legato: 0, centre: 79, rhythm: [2, 12], on: bar => true, len: 10, vel: 0.35 },
  Creatures: { preset: "fluteLead", trim: -2, mono: 0, legato: 0, beasts: true, rhythm: [3, 12], on: bar => true, len: 6, vel: 0.5 },
  Elders: { preset: "fluteLead", trim: -3, mono: 0, legato: 0, centre: 72, rhythm: [2, 12], on: bar => bar >= 4 && bar < 20, len: 10, vel: 0.35 },
  Holy: { preset: "glassBell", trim: -9, mono: 0, legato: 0, centre: 81, holy: true, vel: 0.55 },
  Worthy: { preset: "fluteLead", trim: -2, mono: 0, legato: 0, centre: 74, rhythm: [2, 12], on: bar => bar < 14, len: 14, vel: 0.45 },
}
// the four living creatures, six bars each, each its own voice and register: the lion a flute, the calf a marimba, the man a voice, the
// flying eagle an airy pad, high
const beasts = [
  { preset: "fluteLead", trim: -2, len: 6 },
  { preset: "marimba", trim: -5, len: 3 },
  { preset: "vowel", trim: -11, len: 8 },
  { preset: "airPad", trim: -7, len: 12 },
]
let part = null, holyStep = 0, holyTone = 0, beast = -1
function section(name) {
  part = parts[name]
  index = null
  holyStep = 0
  beast = -1
  if (part) set({ preset: part.preset, trim: part.trim, mono: part.mono, legato: part.legato, glide: 0 })
}

function trigger(ev) {
  if (!part) return
  const bar = song.bar, k = ev.step % 16
  const c = shared.chordFor(song.section, bar)
  if (part.calls) {
    const call = k === 0 ? part.calls[bar] : null
    if (call) note(call[0], call[1], call[2])
    return
  }
  if (part.holy) {
    // "Holy, holy, holy": the call said three times (the same note three times, each of 4 steps), then a long note that
    // answers; a phrase of 3 x 4 + 4 steps is a bar, and it comes round for 28 bars (seven phrases of 4 bars, the walk deciding where they go)
    if (k % 4 !== 0 || bar % 4 > 1) return                 // two bars of four sing, and two are a breath
    const beat = k / 4                                    // 0 to 3
    if (beat === 0) holyTone = walkTo(part.centre, bar)
    const answer = beat === 3
    const p = answer ? shared.fit(holyTone + (stepOf[walk.next()] > 0 ? 4 : -3), c) : holyTone
    note(p, part.vel * (answer ? 1 : 0.8 + 0.07 * beat), answer ? 3 : 3.2)
    return
  }
  if (!part.on(bar)) return
  const hit = euclid(song.section, part.rhythm[0], part.rhythm[1], 0).next()
  if (!hit) return
  let len = part.len
  if (part.beasts) {
    const b = shared.beastOf(bar)
    if (b !== beast) { beast = b; set({ preset: beasts[b].preset, trim: beasts[b].trim, mono: 0, legato: 0, glide: 0 }) }
    len = beasts[b].len
  }
  const centre = part.beasts ? shared.beastCentre[shared.beastOf(bar)] : part.centre
  note(shared.fit(walkTo(centre, bar), c), part.vel * shared.swell(song.section, song.progress), len)
}
