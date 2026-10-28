// struck glass: three threads of other lengths than the bar, five hits in twelve sixteenths, four in nine and three in seven (the five, four and three that make twelve), going
// round whether or not they sound, so they meet where they meet and the bar of sixteen and the nine and the seven only come back together after a long while. Where the drums and
// the eggs are unpitched, the glass sings the chord: each thread walks the notes of the chord of the bar in its own register, a step or two up or down at every hit, so the
// threads cross in the middle of the chord and are always in tune with it. They come in one by one in the Knit
clock("1/16")
const play = shared.player("glass")
const rand = shared.rng(shared.seed || 1, "glass-walk")
// the bar of the section each thread comes in at (never: 99)
const enter = { Knit: [4, 8, 12], Wonder: [0, 8, 99], Shadow: [0, 99, 99], Light: [0, 0, 4], Sand: [0, 0, 0] }
const threads = [
  { hits: 5, len: 12, rot: 0, lo: 72, hi: 86, vel: 0.36 },
  { hits: 4, len: 9, rot: 2, lo: 78, hi: 90, vel: 0.3 },
  { hits: 3, len: 7, rot: 1, lo: 72, hi: 82, vel: 0.28 },
]
let voices = null, current = null, last = -1
function section(name) {
  current = enter[name] ? shared.sec(name) : null
  last = -1
  voices = threads.map(t => ({ ...t, cycle: genny.euclid(t.hits, t.len, t.rot).take(t.len), at: Math.floor(rand() * 4) }))
}
function trigger(ev) {
  if (!current) return
  const k = shared.sixteenth(song)
  if (k === last) return
  last = k
  voices.forEach((v, i) => {
    const hit = v.cycle[k % v.len] // the cycles go round whether or not the glass sounds
    if (!hit || song.bar < enter[current.name][i]) return
    const tones = shared.chordFor(current.name, song.bar).tones(v.lo, v.hi)
    if (tones.length === 0) return
    v.at = Math.max(0, Math.min(tones.length - 1, v.at + Math.round((rand() - 0.5) * 4)))
    play(note, tones[v.at], (v.vel + 0.1 * rand()) * current.dyn, 4, ev)
  })
}
