// The score of "Woven" (Psalm 139): the harmony of every section, the tunes the voices weave, the tuning and the players. The program puts the sections, the tuning and
// the keys of the percussion in `shared` before this text; this is the first script of the patch, so the others find all of it in `shared` when they begin.
//
// A part is a score: a function of a section that gives its events (`at` in beats from the start of the section, `p` a pitch or a chord, `beats`, and a `vel`).
// shared.part(name, options, score) makes the player of a part, with `section(name)` for the section function of its script and `trigger(ev)` for the trigger
// function (a clock of sixteenths): the events fall on the sixteenths they begin on, a little off the grid, as a player plays them.

const TONIC = 62 // the D every pure ratio is measured from
// the pure ratios of the twelve semitones above the tonic, as cents against the equal temperament: 16/15, 9/8, 6/5, 5/4, 4/3, 45/32, 3/2, 8/5, 5/3, 9/5, 15/8
const JUST = [0, 11.73, 3.91, 15.64, -13.69, -1.96, -9.78, 1.96, 13.69, -15.64, 17.6, -11.73]
const mod12 = x => ((x % 12) + 12) % 12
const clamp = (x, lo, hi) => Math.max(lo, Math.min(hi, x))

// tune puts a pitch in just intonation (when the program says so): every note a pure ratio of the tonic, a fraction of a semitone is kept
shared.tune = p => {
  const ref = 12 * Math.log2(shared.a4 / 440) // the reference pitch of A4
  if (!shared.just) return p + ref
  return p + ref + JUST[mod12(Math.round(p) - TONIC)] / 100
}

// --- the sections ---

shared.sec = name => {
  const s = shared.sections.find(s => s.name === name)
  if (!s) throw new Error("no section " + name)
  return s
}

// --- the harmony: a progression for every section, each chord for a number of bars. One loop of four bars (D Bm7 Gmaj7 A) is what the tune is written over, so the voices of
// the canon, a phrase or four bars apart, always sing over the same chords; the Shadow is the same loop a minor third lower, the Wonder a hymn
const rep = (list, n) => Array.from({ length: n }).flatMap(() => list)
const LOOP = [["D", 1], ["Bm7", 1], ["Gmaj7", 1], ["A", 1]]
const PROGS = {
  Thread: [["Dsus2", 4], ["D", 2], ["Dsus2", 2]],
  Knit: rep(LOOP, 4),
  Wonder: rep([["G", 1], ["D/F#", 1], ["Em7", 1], ["A", 1], ["Bm", 1], ["G", 1], ["A", 1], ["D", 1]], 2),
  Shadow: rep([["Bm", 1], ["G", 1], ["Em7", 1], ["A", 1]], 3),
  Light: rep(LOOP, 3),
  Sand: rep(LOOP, 4),
  Awake: [["Dsus2", 4], ["D", 8]],
}
const perBar = {}
for (const s of shared.sections) {
  const bars = (PROGS[s.name] || []).flatMap(([symbol, n]) => Array(n).fill(symbol))
  if (bars.length !== s.bars) throw new Error(s.name + ": " + bars.length + " chords for " + s.bars + " bars")
  perBar[s.name] = bars.map(chord)
}
shared.chordFor = (section, bar) => perBar[section][bar]

// --- the tunes. Names and the beats after a colon (the beats of the one before when there are none), `r` a rest, bars only for the eye. A and B are two halves that fit over
// each other: on the strong beats of every bar both are notes of the chord, so they may be sung together. W is the hymn of the Wonder, S the tune of the Shadow
const TUNES = {
  A: "A4:2 F#4:1 A4:1 | B4:1.5 A4:0.5 F#4:2 | G4:1 B4:1 D5:2 | C#5:1.5 B4:0.5 A4:2",
  B: "F#5:2 D5:1 A4:1 | D5:1.5 B4:0.5 D5:2 | B4:1 D5:1 G5:2 | E5:2 C#5:1 A4:1",
  W: "B4:2 D5:2 | F#5:3 E5:1 | G5:2 E5:1 D5:1 | E5:2 C#5:2 | D5:2 F#5:2 | G5:3 F#5:1 | E5:2 D5:1 C#5:1 | D5:4",
  S: "F#4:2 D5:1 B4:1 | B4:1.5 G4:0.5 D5:2 | E5:2 B4:1 G4:1 | E5:1.5 C#5:0.5 A4:2",
}

shared.parse = text => {
  const out = []
  let beats = 1
  for (const tok of text.split(/\s+/)) {
    if (!tok || tok === "|") continue
    const [name, len] = tok.split(":")
    if (len !== undefined) beats = Number(len)
    out.push({ p: name === "r" ? null : pitch(name), beats })
  }
  return out
}

// the events of a tune that begins at a bar of the section, moved by some semitones
shared.melody = (tune, bar, semitones, extra) => {
  const out = []
  let at = bar * shared.beats
  for (const n of shared.parse(TUNES[tune])) {
    if (n.p !== null) out.push({ at, p: n.p + (semitones || 0), beats: n.beats, ...extra })
    at += n.beats
  }
  return out
}
// a plan is [[bar, "A"], [bar, "B"] ...]: the tunes in turn
shared.phrases = (plan, semitones, extra) => plan.flatMap(([bar, tune]) => shared.melody(tune, bar, semitones, extra))
// the plan of a voice that sings A and B in turn from a bar to a bar (exclusive), starting with `first`
shared.alternate = (from, to, first) => {
  const plan = []
  let tune = first || "A"
  for (let b = from; b < to; b += 4) {
    plan.push([b, tune])
    tune = tune === "A" ? "B" : "A"
  }
  return plan
}

// --- chords held: the chord of every bar voice led from the one before (each chord moves as little as it can), held while it lasts, between two bars (inclusive, exclusive)
shared.holdChords = (section, lowest, from, to, extra) => {
  const out = []
  let voicing = null
  for (let b = from; b < to;) {
    const c = shared.chordFor(section, b)
    let n = 1
    while (b + n < to && shared.chordFor(section, b + n) === c) n++
    voicing = voicing ? c.voiceLead(voicing) : c.close(lowest)
    out.push({ at: b * shared.beats, p: voicing, beats: n * shared.beats, ...extra })
    b += n
  }
  return out
}

// --- how a part plays: players are never exactly on the beat, an ensemble never starts a chord together. Milliseconds of timing (the notes of a chord move together), spread
// (each note of a chord), strum (the notes of a chord one after the other, low to high) and push (every note moved: ahead or behind), and +- of velocity (0.1 is 10%)
const FEELS = {
  "heart": { timing: 4, velocity: 0.06 },
  "harp": { timing: 7, velocity: 0.1 },
  "celesta": { timing: 8, velocity: 0.12 },
  "flute": { timing: 12, push: 8, velocity: 0.08 },
  "alto flute": { timing: 12, push: 8, velocity: 0.08 },
  "violin": { timing: 12, push: 6, velocity: 0.08 },
  "cello": { timing: 12, push: 10, velocity: 0.08 },
  "strings": { timing: 14, spread: 22, velocity: 0.05 },
  "choir": { timing: 14, spread: 28, velocity: 0.05 },
  "piano": { timing: 8, strum: 14, velocity: 0.07 },
  "pizzicato": { timing: 8, velocity: 0.12 },
  "shaker": { timing: 7, velocity: 0.2 },
  "glass": { timing: 8, velocity: 0.15 },
  "hands": { timing: 8, velocity: 0.12 },
  "cymbal": { timing: 6 },
}
// a note can only be late, never early: every note waits this long (milliseconds) so the ones that play ahead of the beat have room
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
shared.rng = rng

// the sixteenth of the section the song is at (the song and the note function are those of the script of the part: each script has its own)
const sixteenth = song => Math.round((song.bar * shared.beats + song.beat) * 4)
shared.sixteenth = sixteenth

// a player: play(note, pitches, velocity, steps, ev, raw) puts a note where the part's feel puts it, with the note function of the script that plays it (raw: a key of
// a recording, not a pitch to tune)
shared.player = name => {
  const feel = shared.human ? FEELS[name] || {} : {}
  const rand = rng(shared.seed || 1, name)
  const tri = () => rand() + rand() - 1 // -1 to 1, most often near 0
  return (note, pitches, vel, steps, ev, raw) => {
    const step = ev && ev.seconds > 0 ? ev.seconds : 0 // how long a sixteenth is, to put milliseconds in steps
    const v = clamp(vel * (feel.velocity ? 1 + tri() * feel.velocity : 1), 0.02, 1)
    const ps = [].concat(pitches).map(p => (raw ? p : shared.tune(p)))
    if (!step || !shared.human) return note(ps.length === 1 ? ps[0] : ps, v, steps)
    const off = (feel.push || 0) + (feel.timing ? tri() * feel.timing : 0)
    ps.forEach((p, i) => {
      const ms = PAD + off + Math.abs(feel.strum || 0) * i + (feel.spread ? tri() * feel.spread : 0)
      note(p, v, steps, Math.max(0, ms) / 1000 / step)
    })
  }
}

// part makes a part from its score: section(name) makes the events of the section, trigger(ev) plays what falls on the sixteenth the song is at
shared.part = (name, options, score) => {
  const o = { legato: 0.95, vel: 0.7, ...options }
  const play = shared.player(name)
  let events = {}
  let last = -1
  let current = null
  return {
    section(sectionName) {
      current = shared.sec(sectionName)
      events = {}
      last = -1
      for (const e of score(current)) {
        const k = Math.round(e.at * 4)
        ;(events[k] = events[k] || []).push(e)
      }
    },
    trigger(song, note, ev) {
      const k = sixteenth(song)
      if (k === last || !current) return
      last = k
      for (const e of events[k] || []) {
        const v = (e.vel || o.vel) * current.dyn
        play(note, e.p, v, Math.max(0.3, e.beats * 4 * (e.legato || o.legato)), ev, o.raw)
      }
    },
  }
}

// --- the sections are written to the log as they begin
function section(name) {
  const s = shared.sec(name)
  log(s.name + "\n" + s.verse)
}

// --- percussion: the pieces of a recording ---

// a level for a piece of a recording: quiet ones told a little louder, loud ones softer (the velocity goes in squares)
shared.evenOut = (db, ref) => Math.max(0.5, Math.min(1.6, Math.pow(10, (ref - db) / 40)))

// walker is a markov chain that walks the pieces of a recording: mostly on to the next, now and then two on, one back or the same again; the call gives the place
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
shared.piece = (play, note, ev, p, vel, ref) => {
  const steps = ev && ev.seconds > 0 ? Math.max(1, Math.min(p.sec / ev.seconds, 4)) : 1
  play(note, p.key, Math.min(1, vel * shared.evenOut(p.db, ref)), steps, ev, true)
}
