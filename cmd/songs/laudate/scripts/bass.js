// BASS: a walking bass, a note to a beat (a clock of quarters), low down. The first beat of the bar is the root of the chord, the third a tone of it (the fifth or
// the third), the second a step of the scale between them, and the last an approach to the root of the next chord, a half tone above or below it, as a jazz
// bass does (in the last bar of a chord only; in the first the last beat is a tone of the chord). Where the bass is weak (the sanctuary, the end) it plays the root alone, a long note on the first beat of every second bar and the fifth on the third.
// The last bar of a group of four rests on its third and fourth beat for the breath.
clock("1/4")
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
const LOW = 31, HIGH = 50
const rootOf = bar => shared.chordAt(bar).bass(LOW)
const near = (list, p) => shared.nearest(list, p)
let prev = 36
function trigger(ev) {
  if (!shared.known()) return
  const s = shared.s(), bar = shared.bar(), beat = ev.step % 4
  const chd = shared.chordAt(bar), root = rootOf(bar)
  let pitch = -1, vel = 0.5, steps = 0.9
  if (s.bass < 0.7) {                                          // the root and the fifth
    if (beat === 0 && bar % 2 === 0) { note(root, 0.6 * s.bass, 14); prev = root }
    else if (beat === 2 && bar % 4 === 2 && s.bass >= 0.5) note(root + 7, 0.4 * s.bass, 4)
    return
  }
  if (bar % 4 === 3 && beat >= 2) return                        // the breath
  const tones = chd.tones(LOW, HIGH)
  if (beat === 0) { pitch = root; vel = 0.62 }
  else if (beat === 2) { pitch = near(tones, root + (R() < 0.6 ? 7 : 3 + (chd.contains(root + 4) ? 1 : 0))); vel = 0.5 }
  else if (beat === 1) {                                         // a step between the root and the tone of the third beat
    const sc = shared.scale.filter(x => x >= LOW && x <= HIGH)
    const target = near(tones, root + 7), from = sc.indexOf(near(sc, root))
    const dir = target >= root ? 1 : -1
    pitch = sc[Math.max(0, Math.min(sc.length - 1, from + dir * (R() < 0.7 ? 1 : 2)))]
    if (shared.clashes(chd, pitch)) pitch = near(tones, pitch)
    vel = 0.4
  } else if (bar % 2 === 0) {                                    // the chord goes on into the next bar: the fifth below, a tone of it
    pitch = near(tones, root + 7 - 12 < LOW ? root + 7 : root - 5)
    vel = 0.45
  } else {                                                       // the approach to the next chord, a half tone above or below its root
    const next = rootOf(bar + 1)
    pitch = next + (R() < 0.5 ? 1 : -1)
    if (pitch < LOW) pitch += 12
    vel = 0.45
  }
  if (beat === 0 && prev === pitch && bar % 2 === 1) steps = 1.5
  prev = pitch
  note(pitch, vel * s.bass, steps)
}
