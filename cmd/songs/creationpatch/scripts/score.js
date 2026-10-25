// The score library of "Creation": the days, their chords and voicings (put in `shared` by the program before this script), the notation the melodies are
// written in, the tuning, and the player every part uses. It is the first script of the patch, so the others find all this in `shared` when they begin.
//
// A part is a score: a function of a day that gives its notes, bar by bar (see the parts). shared.register(name, options, score) makes the player of a part:
// `section(day)` when a day begins makes its notes, `trigger(song, note, ev)` on every sixteenth plays what begins there (the song and the note function are those of
// the script of the part).

const EVENING = 6 // bars of every day before the morning
const FINAL = 17 // the bar of the seventh day the last chord comes home on, after the Amen's IV
const DARK = 0.9 / 1.3 // keeps the evening of the first day as quiet as the darkness: the day sounds out (its dynamics) when the light comes

// the pure ratios of the twelve semitones above the tonic (E flat), as cents against the equal temperament: 16/15, 9/8, 6/5, 5/4, 4/3, 45/32, 3/2, 8/5, 5/3, 9/5, 15/8
const JUST = [0, 11.73, 3.91, 15.64, -13.69, -1.96, -9.78, 1.96, 13.69, -15.64, 17.6, -11.73]
const TONIC = 63

const mod12 = x => ((x % 12) + 12) % 12

// tune puts a pitch in just intonation (when the program says so): every note a pure ratio of the tonic, a fraction of a semitone is kept
shared.tune = p => {
  const ref = 12 * Math.log2(shared.a4 / 440) // the reference pitch of A4
  if (!shared.just) return p + ref
  return p + ref + JUST[mod12(Math.round(p) - TONIC)] / 100
}

shared.EVENING = EVENING
shared.FINAL = FINAL
shared.DARK = DARK

// --- the days and their bars ---

shared.day = name => {
  const d = shared.days.find(d => d.name === name)
  if (!d) throw new Error("no day " + name)
  return d
}
// bar b of a day: { c: chord name, r: root, iv: intervals, high, mid: the voicings, root: the bass }
shared.bar = (day, b) => shared.bars[day.start + b]
shared.morning = b => b >= EVENING
// whether a day is one of the days (by number, 1 to 7)
shared.in = (day, ...numbers) => numbers.includes(day.number)
// the chord of a bar close above a pitch, root position
shared.close = (br, lowest) => {
  const root = lowest + mod12(br.r - mod12(lowest))
  return br.iv.map(i => root + i)
}

// --- notes: { p: [pitches] (none or 0: a rest), beats, hold, vel, amp } ---

shared.N = (pitches, beats, extra) => Object.assign({ p: [].concat(pitches), beats }, extra)
shared.chord = (pitches, beats, extra) => shared.N(pitches, beats, extra)
shared.rest = bars => shared.N([], bars * shared.beats)
// notes of the same length: a 0 is a rest
shared.notes = (beats, ...pitches) => pitches.map(p => shared.N(p === 0 ? [] : [p], beats))
shared.join = (...lists) => [].concat(...lists)
shared.pitches = (pitches, semitones) => pitches.map(p => p + semitones)
shared.transpose = (list, semitones) => list.map(n => Object.assign({}, n, { p: n.p.map(p => p + semitones) }))

// parse reads notes from a notation: "G4:2 Bb4:1 | C5:2" - a name, the beats after a colon (the beats of the one before when it has none; 1 to begin), bars are only for reading
shared.parse = text => {
  const out = []
  let beats = 1
  for (const tok of text.split(/\s+/)) {
    if (!tok || tok === "|") continue
    const [name, len] = tok.split(":")
    if (len !== undefined) beats = Number(len)
    out.push(shared.N(name === "r" || name === "-" ? [] : [pitch(name)], beats))
  }
  return out
}

// the notes of a section bar by bar: f(b, bar) gives the notes of a bar of the day. A bar without notes (null) rests until the end of the bar: a whole bar of rest,
// nothing when a longer note before it already lasts through the bar
shared.perBar = (day, f) => {
  const out = []
  let beats = 0
  for (let b = 0; b < day.bars; b++) {
    let notes = f(b, shared.bar(day, b))
    if (!notes) {
      const rest = (b + 1) * shared.beats - beats
      notes = rest > 1e-9 ? [shared.N([], rest)] : []
    }
    out.push(...notes)
    beats += notes.reduce((t, n) => t + n.beats, 0)
  }
  return out
}

// a chord in every bar of the seventh day, the last one ringing to the end
shared.ringing = (b, chord) => {
  if (b < FINAL) return [shared.chord(chord, shared.beats)]
  if (b === FINAL) return [shared.chord(chord, 12, { hold: 12 })]
  return null // the last chord still rings
}

// six eighth notes up and down a chord
shared.arp = pitches => {
  const ps = pitches.slice().sort((a, b) => a - b)
  return [0, 1, 2, 3, 2, 1].map(i => shared.N([ps[Math.min(i, ps.length - 1)]], 0.5))
}

// a bar of quick calls on the tones of a chord
shared.birdsong = (br, alt) => {
  const p = shared.close(br, 72)
  p.push(p[0] + 12)
  if (alt) return shared.join(shared.notes(0.25, p[3], p[2], p[3], p[2]), shared.notes(1, 0), shared.notes(0.5, p[1], p[3]))
  return shared.join(shared.notes(0.25, p[1], p[2]), shared.notes(0.5, p[3], 0), shared.notes(0.25, p[2], p[3]), shared.notes(1, p[1]))
}

// a bar of slow twinkling high in a chord
shared.stars = (br, b) => {
  const p = shared.close(br, 79)
  switch (b % 3) {
    case 0: return shared.notes(1, p[2], 0, p[0])
    case 1: return shared.join(shared.notes(0.5, 0, p[1]), shared.notes(2, p[2]))
  }
  return shared.notes(1.5, p[0] + 12, p[1])
}

// --- melodies ---

// the theme of man, made in the image of God (day 6, over I vi IV V IV I)
shared.theme = () => shared.parse("G4:2 Bb4:1 | C5:2 Eb5:1 | Eb5:1 D5 C5 | Bb4:2 D5:1 | C5:1 Bb4 Ab4 | G4:1 F4 Eb4")
// the horn answers: its fourth bar and fifth bar hold the figure of the cross (Bb G C Bb: the first and last note the beam, the second and third crossing it)
shared.answer = () => shared.parse("Eb3:3 | C3:3 | Ab2:3 | Bb2:1 G2 C3 | Bb2:3 | Eb3:3")
// the figure of the cross by itself, two bars
shared.cross = () => shared.parse("Bb3:1 G3 C4 | Bb3:3")
// the moon (day 4, over vi IV I V vi V)
shared.moon = () => shared.parse("Eb5:2 C5:1 | C5:2 Ab4:1 | Bb4:2 G4:1 | F4:2 Bb4:1 | C5:1 D5 Eb5 | D5:3")
// the canon of the plants, every phrase two bars over I and IV: the flute sings A B C, the oboe follows two bars later with A B, the clarinet two more with A, an octave down
shared.seedA = () => shared.parse("Bb4:1 G4 Eb4 | C5:2 Ab4:1")
shared.seedB = () => shared.parse("G4:2 Bb4:1 | Ab4:1 C5 Eb5")
shared.seedC = () => shared.parse("Eb5:2 D5:1 | C5:3")
// the Doxology: the Old Hundredth in E flat, a long first and last note in every line, three beats to the bar
shared.doxology = () => shared.parse(
  "Eb4:3 | Eb4:1 D4 C4 | Bb3:1 Eb4 F4 | G4:3 | " + // Praise God, from whom all blessings flow
  "G4:3 | G4:1 G4 F4 | Eb4:1 Ab4 G4 | F4:3 | " + // praise him, all creatures here below
  "Eb4:3 | F4:1 G4 F4 | Eb4:1 C4 D4 | Eb4:3 | " + // praise him above, ye heavenly host
  "Bb4:3 | G4:1 Eb4 F4 | Ab4:1 G4 F4 | Eb4:3") // praise Father, Son, and Holy Ghost

// --- jazz: the parts that are not written, but played as the song goes ---

// the layers of the jazz and the bars they play in: the bass walks from the third day on, the tabla keeps the bembe from the second, the vibraphone comps from the
// fourth, and in the Doxology all three play it as a waltz, until the last chord rings alone
shared.layers = {
  bass: (d, b) => (d.number >= 3 && d.number <= 6 && (shared.morning(b) || (d.number >= 5 && b >= 3))) || (d.number === 7 && b >= 2 && b < 16),
  vibes: (d, b) => (d.number >= 4 && d.number <= 6 && shared.morning(b)) || (d.number === 7 && b >= 4 && b < 16),
  tabla: (d, b) => (d.number >= 2 && d.number <= 6 && shared.morning(b) && (d.number > 2 || b >= 8)) || (d.number === 7 && b >= 2 && b < 16),
  // the percussion: the tambourine with the sun, the tubes with the land and the creatures, the hand drums with the beasts and in the praise
  tambourine: (d, b) => (d.number >= 4 && d.number <= 6 && shared.morning(b)) || (d.number === 7 && b >= 4 && b < 16),
  wood: (d, b) => (d.number === 3 && b >= 3) || (d.number >= 4 && d.number <= 6 && shared.morning(b)) || (d.number === 7 && b >= 6 && b < 16),
  hands: (d, b) => (d.number === 5 && shared.morning(b)) || (d.number === 6 && b >= 2) || (d.number === 7 && b >= 4 && b < 16),
}

// the feel of the players of the jazz: how far behind the beat (ms; the bass lays back, the ride is on top of it), how loose, how much louder and softer, how
// much the offbeat is swung
const PLAYERS = {
  "upright bass": { lay: 9, timing: 5, velocity: 0.1, swing: 0.62 },
  "vibes": { lay: 4, timing: 7, velocity: 0.14, swing: 0.62 },
  "tabla": { lay: -3, timing: 4, velocity: 0.12, swing: 0.62 },
  "tambourine": { lay: 0, timing: 7, velocity: 0.15, swing: 0.6 },
  "wood": { lay: 2, timing: 8, velocity: 0.15, swing: 0.58 },
  "hands": { lay: 3, timing: 8, velocity: 0.12, swing: 0.6 },
}

// player makes a player of a part that is not a score but played as the song goes (the jazz, the percussion): play(note, ev, sixteenth, pitches, velocity, steps, raw) puts a note where the player puts it
shared.player = name => {
  const feel = PLAYERS[name]
  const rand = rng((shared.seed || 1) + 7, name)
  const tri = () => rand() + rand() - 1
  return (note, ev, sixteenth, pitches, vel, steps, raw) => {
    const step = ev && ev.seconds > 0 ? ev.seconds : 0
    const v = Math.max(0.01, Math.min(1, vel * (shared.human ? 1 + tri() * feel.velocity : 1)))
    const ps = [].concat(pitches).map(p => (raw ? p : shared.tune(p)))
    if (!step || !shared.human) return note(ps.length === 1 ? ps[0] : ps, v, steps)
    const swing = shared.jazz && sixteenth % 4 === 2 ? (feel.swing - 0.5) * 4 * step * 1000 : 0
    ps.forEach(p => note(p, v, steps, Math.max(0, PAD + feel.lay + swing + tri() * feel.timing) / 1000 / step))
  }
}

// the quality of the chord of a bar, for the jazz to put sevenths and ninths on: the third, the seventh (the dominant on the fifth degree, the chord of the Doxology's
// V7, major or minor sevenths elsewhere), null for the bare fifths of the void
shared.quality = br => {
  const third = br.iv.includes(3) ? 3 : br.iv.includes(4) ? 4 : 0
  if (!third) return null
  const seventh = br.iv.includes(10) ? 10 : br.iv.includes(11) ? 11 : third === 3 ? 10 : mod12(br.r - TONIC) === 7 ? 10 : 11
  return { third, seventh, minor: third === 3, dominant: seventh === 10 && third === 4 }
}

// the harmonic series puts the weight of an interval: the ratio p:q (1:1 the root, 2:1 the octave, 3:2 the fifth, 5:4 and 6:5 the thirds, 9:5 and 15:8 the sevenths)
// is as likely as 1/(p*q), the simpler the ratio the more often the bass lands on it
const RATIOS = { 0: [1, 1], 3: [6, 5], 4: [5, 4], 7: [3, 2], 10: [9, 5], 11: [15, 8], 12: [2, 1] }

// walkChain is a markov chain over the tones of a chord (the semitones above its root, 0 to 12) for a walking bass: the weight of a move is the weight of the ratio of
// the tone it goes to, less for a big leap and little for the same tone again
shared.walkChain = q => {
  const tones = [0, q.third, 7, q.seventh, 12]
  const table = {}
  tones.forEach((a, i) => {
    table[a] = {}
    tones.forEach((b, j) => {
      table[a][b] = (1 / (RATIOS[b][0] * RATIOS[b][1])) * (i === j ? 0.15 : 1 / (1 + Math.abs(i - j)))
    })
  })
  return genny.markov(table, "0")
}

// voice puts the tones of a chord in a close voicing above a pitch, each above the one before: type "A" is the third, fifth, seventh and ninth, "B" the seventh,
// ninth, third and fifth (the root left to the bass)
shared.voice = (br, q, type, lowest) => {
  const degrees = type === "A" ? [q.third, 7, q.seventh, 14] : [q.seventh, 14, q.third + 12, 19]
  const out = []
  let floor = lowest
  for (const d of degrees) {
    const pc = mod12(br.r + d)
    const p = floor + mod12(pc - floor)
    out.push(p)
    floor = p + 1
  }
  return out
}

// a level for a piece of a recording: quiet ones told a little louder, loud ones softer (the velocity goes in squares)
shared.evenOut = (db, ref) => Math.max(0.5, Math.min(1.6, Math.pow(10, (ref - db) / 40)))

// walker is a markov chain that walks the pieces of a recording: mostly on to the next, now and then two on, one back or the same again; step() gives the place
shared.walker = (length, start) => {
  const chain = genny.markov({
    next: { next: 14, skip: 4, back: 2, again: 3 },
    skip: { next: 12, skip: 2, back: 2, again: 2 },
    back: { next: 12, again: 3, skip: 2 },
    again: { next: 12, skip: 3, back: 2 },
  }, "next")
  let pos = start || 0
  return () => {
    const a = chain.next()
    if (a === "next") pos++
    else if (a === "skip") pos += 2
    else if (a === "back") pos = Math.max(0, pos - 1)
    pos %= length
    return pos
  }
}

// a piece played: the key of it, as loud as the velocity says once it is evened out, as long as it lasts (at most four steps)
shared.piece = (play, note, ev, sixteenth, p, vel, ref) => {
  const steps = ev && ev.seconds > 0 ? Math.max(1, Math.min(p.sec / ev.seconds, 4)) : 1
  play(note, ev, sixteenth, p.key, Math.min(1, vel * shared.evenOut(p.db, ref)), steps, true)
}

// --- percussion: bars of sixteenths ---

// hits is a bar of three beats in sixteenths: twelve characters, a dot is a rest, any other character a hit of the kind it names in `kinds` ({x: shared.kits.hands.conga}),
// one of the recordings of the kind in turn (by the bar and the place in it, so it is the same every time), a capital letter louder
shared.hits = (pattern, kinds, bar, vel) => {
  if (pattern.length !== shared.beats * 4) throw new Error("a bar of " + shared.beats * 4 + " sixteenths, not " + pattern.length + ": " + pattern)
  return [...pattern].map((c, i) => {
    const keys = kinds[c.toLowerCase()]
    if (c === ".") return shared.N([], 0.25)
    if (!keys) throw new Error("no kind " + c)
    return shared.N([keys[(bar * 5 + i * 3) % keys.length]], 0.25, { vel: (vel || 0.5) * (c === c.toUpperCase() ? 1.3 : 0.8) })
  })
}

// groove is the percussion of a day: f(b, bar) gives { pattern, kinds, vel } for a bar, null for a rest. It is untuned (the keys are those of the
// recordings): a part with it is registered with { raw: true }
shared.groove = (day, f) => shared.perBar(day, (b, br) => {
  const g = f(b, br)
  return g ? shared.hits(g.pattern, g.kinds, b, g.vel) : null
})

// --- the sounds of creation: recordings faded in and out ---

// fade is the gain (1 is 0 dB) of a recording at a bar of a day (b, with the beats as a fraction): db at its height, faded in at bar `from` and out at bar `to`
// over a bar (or a quarter of them when they are fewer), silent outside them. A bar beyond the day (from -1, to the bars + 1) leaves out the fade at that edge,
// for a sound that carries on from the day before or into the day after
shared.fade = (db, b, from, to) => {
  const edge = Math.min(1, (to - from) / 4)
  if (b < from || b > to || edge <= 0) return 0
  const g = Math.min((b - from) / edge, (to - b) / edge, 1)
  return g <= 0 ? 0 : Math.pow(10, db / 20) * g
}
// where the song is in its day, in bars with the part of a bar as a fraction
shared.at = song => song.bar + song.beat / shared.beats

// --- the parts ---

const parts = {}
shared.parts = parts

// compile makes the notes of a part in a day, as events: where in the day (beats), the pitches, how long they sound (beats), the velocity and the amplitude
shared.compile = (name, dayName) => {
  const part = parts[name]
  const day = shared.day(dayName)
  const list = part.score(day)
  if (!list) return []
  const out = []
  let beat = 0
  for (const n of list) {
    if (n.p.length) {
      out.push({
        beat, pitches: n.p.slice().sort((a, b) => a - b), hold: n.hold || n.beats * part.legato, vel: n.vel || 0,
        amp: (n.amp || 1) * (day.dyn !== 1 ? day.dyn : 1),
      })
    }
    beat += n.beats
  }
  const expect = day.bars * shared.beats
  if (Math.abs(beat - expect) > 1e-6) throw new Error(name + " lasts " + beat + " beats in " + dayName + ", expected " + expect + " (" + day.bars + " bars)")
  return out
}

// the light comes up gently: the morning of the first day grows from four fifths to its full sound across its bars (at trigger time, the notes themselves are as written)
shared.dawn = song => {
  if (song.index !== 0 || song.bar < EVENING) return 1
  const day = shared.days[0]
  return 0.8 + 0.2 * Math.min(1, (song.bar + song.beat / shared.beats - EVENING) / (day.bars - EVENING))
}

// --- how a part plays: ensembles never start a chord exactly together, soloists breathe ---

// the feel of every part (as in the score package of psalm): timing, spread, strum and push in milliseconds (timing moves the notes of a chord together, spread
// each note of a chord, strum puts a chord's notes after each other, low to high, push lays every note back or ahead), velocity (+-, 0.1 is +-10%) and the
// accents through the bar (the beats of an accent, the beats of the bar)
const FEELS = {
  "violins": { timing: 8, spread: 12, velocity: 0.06 },
  "celli": { timing: 8, spread: 10, velocity: 0.06 },
  "pizzicato": { swing: 0.62, timing: 6, velocity: 0.1, accents: [1, 0.8, 0.85], bar: 3 },
  "harp": { swing: 0.62, timing: 5, velocity: 0.1 },
  "timpani": { timing: 4, velocity: 0.08 },
  "flute": { swing: 0.62, timing: 10, push: 6, velocity: 0.08 },
  "oboe": { swing: 0.62, timing: 10, push: 6, velocity: 0.08 },
  "clarinet": { swing: 0.62, timing: 10, push: 6, velocity: 0.08 },
  "horns": { timing: 10, spread: 15, velocity: 0.06 },
  "horn": { swing: 0.62, timing: 12, push: 4, velocity: 0.08 },
  "soprano": { timing: 10, push: 6, velocity: 0.08 },
  "spirit": { timing: 10, spread: 20 },
  "choir": { timing: 10, spread: 20 },
  "organ": { timing: 4, spread: 6 },
  "celesta": { swing: 0.62, timing: 6, velocity: 0.12 },
  "bell pad": { strum: 25, velocity: 0.05 },
  "bells": { strum: 40 },
  "piano": { swing: 0.62, timing: 6, strum: 18, velocity: 0.06 },
  "the deep": { timing: 20 },
  "shaker": { timing: 7, velocity: 0.18 },
  "tambourine": { timing: 7, velocity: 0.15 },
  "wood": { timing: 6, velocity: 0.15 },
  "hands": { timing: 8, velocity: 0.12 },
  "cymbal": { timing: 10 },
}
// a note can only be late, never early: every note of every part waits this long (milliseconds) so the ones that play ahead of the beat have room
const PAD = 32

// the random numbers of a part: the same every time for a seed (shared.seed), every part its own
function rng(seed, name) {
  let a = seed >>> 0
  for (const c of name) a = (Math.imul(a ^ c.charCodeAt(0), 2654435761) + 0x9e3779b9) >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

// register makes a part: the player with the section and trigger functions to call from the script of the part (a clock of sixteenths)
shared.register = (name, options, score) => {
  const part = { legato: 1, vel: 0.8, ...options, score }
  const feel = shared.human ? FEELS[name] || {} : {}
  const accents = (FEELS[name] || {}).accents
  const swing = shared.jazz && shared.human ? (FEELS[name] || {}).swing || 0 : 0 // the offbeat eighths of a swinging part come late (0.5 straight, 0.67 triplets)
  const rand = rng(shared.seed || 1, name)
  const tri = () => rand() + rand() - 1 // -1 to 1, most often near 0
  parts[name] = part
  let events = {}
  let last = -1
  return {
    section(dayName) {
      events = {}
      last = -1
      for (const e of shared.compile(name, dayName)) {
        const k = Math.round(e.beat * 4)
        ;(events[k] = events[k] || []).push(e)
        // light: what is plucked dies away, so a chord held over a bar and a half is struck again every bar, a little softer each time
        if (shared.pluck && e.hold >= 5) {
          for (let j = 1; j * shared.beats < e.hold - 1; j++) {
            const kk = Math.round((e.beat + j * shared.beats) * 4)
            ;(events[kk] = events[kk] || []).push({ ...e, hold: shared.beats, vel: (e.vel || part.vel) * Math.pow(0.82, j) })
          }
        }
      }
    },
    trigger(song, note, ev) {
      const k = Math.round((song.bar * shared.beats + song.beat) * 4)
      if (k === last) return
      last = k
      const step = ev && ev.seconds > 0 ? ev.seconds : 0 // how long a sixteenth is, to put milliseconds in steps
      for (const e of events[k] || []) {
        let v = (e.vel || part.vel) * e.amp * shared.dawn(song)
        if (accents) { // the accent of the beat in the bar
          const inBar = e.beat - Math.floor(e.beat / (FEELS[name].bar || 4)) * (FEELS[name].bar || 4)
          v *= accents[Math.floor(inBar + 1e-9) % accents.length]
        }
        if (feel.velocity) v *= 1 + tri() * feel.velocity
        v = Math.max(0.01, Math.min(1, v))
        const hold = Math.max(0.05, e.hold * 4)
        if (!step || !shared.human) {
          note(part.raw ? e.pitches : e.pitches.map(shared.tune), v, hold)
          continue
        }
        const late = swing && k % 4 === 2 ? (swing - 0.5) * 4 * step * 1000 : 0
        const offset = (feel.push || 0) + late + (feel.timing ? tri() * feel.timing : 0)
        e.pitches.forEach((p, i) => {
          const ms = PAD + offset + Math.abs(feel.strum || 0) * i + (feel.spread ? tri() * feel.spread : 0)
          note(part.raw ? p : shared.tune(p), v, hold, Math.max(0, ms) / 1000 / step)
        })
      }
    },
  }
}

// the verse of every day is written to the log of the patch as the day begins
function section(name, index) {
  const d = shared.day(name)
  log(d.name + "\n" + d.verse)
}
