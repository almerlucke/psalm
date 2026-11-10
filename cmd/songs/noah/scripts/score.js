// SCORE: plays nothing. It puts in `shared` what the other scripts ask for, so that the form of the piece is read in one place. The piece is
// as long as the recording (HeavenlyNoah.wav, 6:24) and goes at 60 bpm in 4/4, so a bar is four seconds and everything below is written in seconds of the recording.
//   shared.time(song)        seconds since the start
//   shared.lv(part, t)       how loud a part is at that time (0: it does not play), from the curves below
//   shared.hush(t)           1, or less while the thunder speaks (the music leaves it alone)
//   shared.tones(lo, hi, t)  the notes of the scale of that time between two pitches
//   shared.pitch(p, t)       a pitch as it is played (the voice at the end is a little sharp: so is the music then)
//   shared.rng(seed)         a seeded random function
// What the recording does (found by looking at it): water from the start, the rain coming in from 8 s; thunder at 58, 124 and 138 s; granular bells
// in D (D E F# A B) from 72 to 190 s; the animals and shifting bells until 290 s; a storm from 300 to 340 s; the voice, around a C that is
// a third of a semitone sharp, from 340 s to the end.

const BAR = 4
const START = { Water: 0, Rain: 8, Bells: 18, Beasts: 47, Storm: 73, Voice: 85 }

shared.time = song => (START[song.section] + song.bar) * BAR + song.beat

// the scale: D major pentatonic while the bells are in D; D E G A C (no third: it is neither major nor minor) for the animals; C D F G for the storm and the voice
const scales = [
  { to: 188, pcs: [2, 4, 6, 9, 11] },
  { to: 324, pcs: [2, 4, 7, 9, 0] },
  { to: 1e9, pcs: [0, 2, 5, 7] },
]
shared.tones = (lo, hi, t) => {
  const s = scales.find(s => t < s.to)
  const out = []
  for (let p = lo; p <= hi; p++) if (s.pcs.includes(p % 12)) out.push(p)
  return out
}
shared.pitch = (p, t) => p + (t >= 336 ? 0.3 : 0)

// the loudness of every part: [time, level] points, joined by lines
const curves = {
  kalimba: [[30, 0], [34, 0.5], [72, 0.6], [80, 0.9], [185, 0.9], [200, 0.7], [290, 0.5], [320, 0.3], [336, 0]],
  marimba: [[60, 0], [70, 0.5], [100, 0.7], [185, 0.7], [200, 0.9], [290, 0.7], [310, 0.3], [330, 0]],
  koto:    [[110, 0], [120, 0.5], [186, 0.5], [192, 0.8], [290, 0.8], [306, 0.3], [324, 0]],
  bell:    [[20, 0], [26, 0.6], [300, 0.6], [330, 0.5], [372, 0.3], [384, 0]],
  wood:    [[14, 0.4], [60, 0.5], [188, 0.8], [290, 0.8], [300, 0.4], [340, 0.2], [372, 0.1]],
  frame:   [[36, 0], [44, 0.4], [188, 0.5], [200, 0.9], [290, 0.8], [306, 0.2], [320, 0]],
  bowl:    [[0, 1], [384, 1]],
}
shared.lv = (part, t) => {
  const c = curves[part]
  if (!c || t < c[0][0]) return 0
  for (let i = 1; i < c.length; i++) {
    if (t < c[i][0]) {
      const [t0, v0] = c[i - 1], [t1, v1] = c[i]
      return v0 + (v1 - v0) * (t - t0) / (t1 - t0)
    }
  }
  return c[c.length - 1][1]
}

// the thunder: [from, to] in seconds
const thunder = [[57, 61], [123, 125], [137, 139], [306, 308], [321, 332], [336, 339], [355, 359]]
shared.hush = t => (thunder.some(([a, b]) => t >= a - 1 && t <= b) ? 0.35 : 1)

// a small random function (mulberry32) for every part, so a render is the same every time
shared.rng = seed => {
  let a = (seed + (shared.seed || 0)) >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

// the eighth (half a second) the song is at; a part plays once on every one
shared.eighth = song => Math.round(shared.time(song) * 2)
