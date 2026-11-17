// The score library of "Creation" in jazz: the days, their chords and voicings (put in `shared` by the program before this script), the notation the melodies are
// written in, the tuning, and the player every part uses. It is the first script of the patch, so the others find all this in `shared` when they begin.
//
// A part is a score: a function of a day that gives its notes, bar by bar (see the parts). shared.register(name, options, score) makes the player of a part:
// `section(day)` when a day begins makes its notes, `trigger(song, note, ev)` on every sixteenth plays what begins there (the song and the note function are those of
// the script of the part).

const EVENING = 4 // the bar of the intro where the light comes
const FINAL = 8 // the bar of the tag the last chord comes home on, after the Amen's IV
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

// --- the head: the written melody of the tune, thirty-two bars (A A B A, the changes of the chorus in form.go), a waltz in E flat ---

// the first A over I vi ii V7 iii vi ii V7: it climbs on the tones of the chords to the ninth and falls back, the bridge sings higher over IV V7 iii vi ii V7 I V7
// the motif of the piece: the opening of the head, a rising E flat chord (G B flat | C E flat) over I vi. The head plays it, and a bell states it again in every section
// where the changes begin on I vi (see bells.js), so that all the sections are recognised by it
shared.motif = () => shared.parse("G4:2 Bb4:1 | C5:2 Eb5:1")
shared.headA = () => shared.join(shared.motif(), shared.parse("| F5:2 Eb5:1 | D5:3 | Bb4:2 D5:1 | C5:1 Eb5 G5 | F5:2 C5:1 | Bb4:3"))
shared.headA2 = () => shared.join(shared.motif(), shared.parse("| F5:2 Eb5:1 | D5:3 | Bb4:2 D5:1 | C5:1 Eb5 G5 | Ab4:2 C5:1 | Eb5:3"))
shared.headB = () => shared.parse("C5:2 Eb5:1 | D5:2 F5:1 | G5:2 F5:1 | Eb5:3 | C5:2 Ab4:1 | D5:1 F5 Ab5 | G5:3 | F5:2 D5:1")
shared.head = () => shared.join(shared.headA(), shared.headA(), shared.headB(), shared.headA2())

// the answers to the motif, one for every section, two bars over ii V7 (F minor, B flat seventh) that follow the motif and say which part of creation it is: the light
// rising (1), the firmament low under the head (2), the waters flowing (3), the creatures calling (4), the lights sparkling high (5), man standing tall (6), and the rest
// falling (7)
shared.answers = {
  1: () => shared.parse("F5:1 Ab5 C6 | D6:3"),
  2: () => shared.parse("F4:2 Ab4:1 | D4:3"),
  3: () => shared.parse("C5:1 F5 Ab5 | F5:1 D5 F5"),
  4: () => shared.parse("F5:0.5 F5 Ab5 r C6:1 | D6:0.5 D6 F5 r Ab5:1"),
  5: () => shared.parse("Ab5:0.5 C6 F6 C6 Ab5 F5 | D6:3"),
  6: () => shared.parse("F5:2 Ab5:1 | D6:3"),
  7: () => shared.parse("Ab5:3 | F5:3"),
}

// where the motif is stated in every section (by the number of the section): the bars [bar, semitones above the motif] and how loud, for the bell that states it, the
// marimba and the Word that answer it
shared.statements = {
  1: [[[4, 0]], 0.5],
  2: [[[0, 0], [8, 0], [24, 0]], 0.8],
  3: [[[0, 0], [8, 0], [24, 0]], 0.8],
  4: [[[2, 0], [10, 0]], 0.85],
  5: [[[0, 0], [16, 0]], 0.85],
  6: [[[0, 0], [8, 0], [24, 12]], 1],
  7: [[[0, 0], [10, 0]], 0.9],
}
// whether an answer begins at bar b of a day: two bars after a statement, with room for it (four bars from the statement to the end of the section)
shared.answerAt = (day, b) => shared.statements[day.number][0].some(([bar]) => bar + 2 === b && bar + 4 <= day.bars)

// motifAt makes the notes of a part that plays the motif in a section, at the bars given as [bar, semitones above the motif] (two bars each, on the chords I vi), or, when
// `answer` is true, the answer of the section two bars after it (two bars, on ii V7), where there is room for it in the section; rests between them and after. The bell
// plays the motif and a marimba the answer: a call and its reply
shared.motifAt = (day, at, amp, answer) => {
  const out = []
  let pos = 0
  for (const [bar, shift] of at) {
    if (answer && bar + 4 > day.bars) continue
    const from = bar + (answer ? 2 : 0)
    if (from > pos) out.push(shared.rest(from - pos))
    const notes = answer ? shared.answers[day.number]() : shared.motif()
    out.push(...shared.transpose(notes, shift).map(n => Object.assign(n, { amp })))
    pos = from + 2
  }
  if (pos < day.bars) out.push(shared.rest(day.bars - pos))
  return out
}

// --- jazz: the parts that are not written, but played as the song goes ---

// the layers of the combo and the bars they play in; n is the number of the section: 1 intro, 2 head, 3 the steel pan's solo, 4 the percussion's break, 5 the solo of
// the flute and the vibraphone, 6 the head out, 7 the tag. The music starts with the first bar: the darbuka and the bass, the claves from the third bar, the piano and
// the shakers from the fifth. In the tag everything but the bass and the piano drops away to let the last chord ring.
const upTo = (d, b, n) => d.number !== 7 || b < n
shared.layers = {
  bass: (d, b) => (d.number === 4 ? b >= 4 : upTo(d, b, 10)),
  darbuka: (d, b) => upTo(d, b, 8),
  claves: (d, b) => (d.number === 1 ? b >= 2 : upTo(d, b, 8)),
  shakers: (d, b) => (d.number === 1 ? b >= 4 : upTo(d, b, 8)),
  congas: (d, b) => (d.number === 1 ? b >= 4 : d.number === 2 ? b >= 8 : upTo(d, b, 8)),
  bongos: (d, b) => (d.number === 3 ? b >= 8 : d.number >= 4 && d.number <= 6) || (d.number === 7 && b < 4),
  comp: (d, b) => (d.number === 1 ? b >= 4 : d.number === 4 ? b >= 8 : upTo(d, b, 10)),
  vibes: (d, b) => (d.number === 2 || d.number === 3 ? b >= 16 : d.number === 5 ? b >= 8 : d.number === 6 || (d.number === 7 && b < 8)),
  flute: (d, b) => (d.number === 5 && b < 16) || (d.number === 7 && b < 8),
  sea: (d, b) => d.number === 3 && b < 24,
  animals: (d, b) => (d.number === 4 && b >= 2) || (d.number === 6 && b >= 8 && b < 24),
  gulls: (d, b) => (d.number === 5 && b >= 4) || (d.number === 7 && b < 8),
  children: (d, b) => (d.number === 6 && b >= 24) || (d.number === 7 && b < 4),
  pan: (d, b) => d.number === 3 || (d.number === 5 && b >= 8),
}

// --- the tune: a tune is made for every bar, the voices play it in canon ---

// the tones of a chord a tune walks on, as semitones above its root: the root, the third, the fifth, the sixth, the octave and the third above it
const ISLAND_TONES = q => [0, q.third, 7, 9, 12, 12 + q.third]

// toneAt is the pitch of tone `pos` of the chord of a bar, the lowest tone at `lowest` or the first pitch of the chord above it (folded down when it gets too high)
shared.toneAt = (br, q, pos, lowest) => {
  const base = lowest + mod12(br.r - lowest)
  const p = base + ISLAND_TONES(q)[Math.max(0, Math.min(5, pos))]
  return p > lowest + 22 ? p - 12 : p
}

// the hits of a tune a bar: how many of the twelve sixteenths each day (a tune is made for every day)
const TUNE_HITS = { 3: 7, 5: 5, 7: 3 }
const phrases = {}

// phrase is the tune of bar b of a day, the same for every voice that plays it (the others follow it in canon, on the chords of the bars they play it in): a euclidean
// rhythm turned a little every bar (the ratio of hits to twelve sixteenths is the day's), and a walk over the tones of the chord that mostly steps, now and then leaps,
// and is led back to the middle of the tones. Every fourth bar the tune breathes. It goes on from the tone the bar before ended on. {end, hits: [{s, pos, v}]}
shared.phrase = (day, b) => {
  const key = day.number * 1000 + b
  if (phrases[key]) return phrases[key]
  const prev = b > 0 ? shared.phrase(day, b - 1) : null
  let pos = prev ? prev.end : 2
  const k = TUNE_HITS[day.number] || 0
  const rot = (day.number * 5 + b * 7) % 12
  const r = rng(shared.seed || 1, "phrase" + key)
  const hits = []
  if (k && b % 4 !== 3) {
    for (let i = 0; i < 12; i++) {
      const j = (i + rot) % 12
      if (Math.floor(((j + 1) * k) / 12) === Math.floor((j * k) / 12)) continue
      const moves = [-2, -1, 1, 2], weights = [1, 3, 3, 1]
      const pull = (2 - pos) * 0.3
      const w = moves.map((m, n) => weights[n] * Math.max(0.15, m > 0 ? 1 + pull : 1 - pull))
      let x = r() * w.reduce((a, c) => a + c, 0), n = 0
      while (n < 3 && x >= w[n]) x -= w[n++]
      pos = Math.max(0, Math.min(5, pos + moves[n]))
      hits.push({ s: i, pos, v: i % 4 === 0 ? 0.55 : i % 2 === 0 ? 0.45 : 0.38 })
    }
  }
  return (phrases[key] = { end: pos, hits })
}

// islandVoice is a voice of the canon: it plays the tune of the bar `lag` bars before the bar it plays in, on the chord of this bar, from `lowest` up, softer by `vel`,
// where `on(day, bar)` lets it
shared.islandVoice = (name, o) => {
  const play = shared.player(name)
  let last = -1
  return (song, note, ev) => {
    const day = shared.days[song.index]
    const s = Math.round(song.beat * 4)
    const at = song.bar * 12 + s
    if (!day || at === last) return
    last = at
    if (song.bar < o.lag || !o.on(day, song.bar)) return
    const h = shared.phrase(day, song.bar - o.lag).hits.find(h => h.s === s)
    if (!h) return
    const br = shared.bar(day, song.bar)
    const q = shared.quality(br)
    if (q) play(note, ev, s, shared.toneAt(br, q, h.pos, o.lowest), h.v * o.vel, 3)
  }
}

// the feel of the players of the jazz: how far behind the beat (ms; the bass lays back, the ride is on top of it), how loose, how much louder and softer, how
// much the offbeat is swung
const PLAYERS = {
  "upright bass": { lay: 9, timing: 5, velocity: 0.1, swing: 0.62 },
  "vibes": { lay: 4, timing: 7, velocity: 0.14, swing: 0.62 },
  "flute": { lay: 2, timing: 7, velocity: 0.12, swing: 0.62 },
  "clarinet": { lay: 6, timing: 7, velocity: 0.12, swing: 0.62 },
  "piano": { lay: 5, timing: 8, velocity: 0.14, swing: 0.62 },
  "steel pan": { lay: 3, timing: 7, velocity: 0.14, swing: 0.62 },
  "congas": { lay: 2, timing: 6, velocity: 0.14, swing: 0.6 },
  "bongos": { lay: 0, timing: 6, velocity: 0.15, swing: 0.6 },
  "claves": { lay: -2, timing: 4, velocity: 0.12, swing: 0.6 },
  "shakers": { lay: 3, timing: 7, velocity: 0.18, swing: 0.62 },
  "darbuka": { lay: 0, timing: 5, velocity: 0.12, swing: 0.6 },
  "word": { lay: 0, timing: 4, velocity: 0.1, swing: 0.6 },
  "animals": { lay: 4, timing: 7, velocity: 0.12, swing: 0.6 },
  "gulls": { lay: 2, timing: 9, velocity: 0.12, swing: 0.58 },
  "sea and birds": { lay: 3, timing: 8, velocity: 0.1, swing: 0.58 },
  "children": { lay: 0, timing: 6, velocity: 0.1, swing: 0.6 },
}

// player makes a player of a part that is not a score but played as the song goes (the combo): play(note, ev, sixteenth, pitches, velocity, steps, raw) puts a note where the player puts it
shared.player = name => {
  const feel = PLAYERS[name]
  const rand = rng((shared.seed || 1) + 7, name)
  const tri = () => rand() + rand() - 1
  return (note, ev, sixteenth, pitches, vel, steps, raw) => {
    const step = ev && ev.seconds > 0 ? ev.seconds : 0
    const v = Math.max(0.01, Math.min(1, vel * (shared.human ? 1 + tri() * feel.velocity : 1)))
    const ps = [].concat(pitches).map(p => (raw ? p : shared.tune(p)))
    if (!step || !shared.human) return note(ps.length === 1 ? ps[0] : ps, v, steps)
    const swing = sixteenth % 4 === 2 ? (feel.swing - 0.5) * 4 * step * 1000 : 0
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
// ninth, third and fifth (the root left to the bass), "C" the sixth chord of the islands: the third, fifth, sixth and ninth
shared.voice = (br, q, type, lowest) => {
  const degrees = type === "A" ? [q.third, 7, q.seventh, 14] : type === "C" ? [q.third, 7, 9, 14] : [q.seventh, 14, q.third + 12, 19]
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

// walker is a markov chain that walks the pieces of a recording: mostly on to the next, now and then two on, one back or the same again; the function gives the place
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

// calls makes a recording that plays like percussion: the pieces of it (shared.pieces[name]) walked by a markov chain on a euclidean rhythm of o.hits[section number]
// hits in twelve sixteenths, turned by o.turn every bar (0: the same every bar), where o.on(day, bar) lets it; o.vel is how loud, o.ref the level (dB) the pieces are
// evened out to. Returns what to call on every sixteenth: calls(song, note, ev)
shared.calls = (name, o) => {
  const play = shared.player(name)
  let walk = null, last = -1
  return (song, note, ev) => {
    const day = shared.days[song.index]
    const s = Math.round(song.beat * 4)
    const at = song.bar * 12 + s
    const pieces = shared.pieces[name]
    if (!day || at === last || !pieces || !pieces.length) return
    last = at
    const k = o.hits[day.number]
    if (!k || !o.on(day, song.bar)) return
    const j = (s + day.number * 5 + song.bar * (o.turn || 0)) % 12
    if (Math.floor(((j + 1) * k) / 12) === Math.floor((j * k) / 12)) return
    walk = walk || shared.walker(pieces.length, 0)
    shared.piece(play, note, ev, s, pieces[walk()], o.vel * (s % 4 === 0 ? 1.15 : 0.85), o.ref)
  }
}

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
  "void": { timing: 20 },
  "the word": { timing: 6, spread: 6, velocity: 0.06 },
  "flute": { swing: 0.62, timing: 8, push: 4, velocity: 0.08 },
  "bells": { timing: 6, strum: 12, velocity: 0.06 },
  "marimba": { swing: 0.6, timing: 7, velocity: 0.08 },
  "steel pan": { swing: 0.62, timing: 8, push: 2, velocity: 0.08 },
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
  const swing = shared.human ? (FEELS[name] || {}).swing || 0 : 0 // the offbeat eighths of a swinging part come late (0.5 straight, 0.67 triplets)
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
