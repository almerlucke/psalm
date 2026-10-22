// VOICE: the long notes I sang (loofdeheer.wav, on the keys from 24, five apart; a key off by a step or two sings a little higher or lower). Once in four
// bars, mostly in the quiet parts, one of them: a tone of the chord, or a step or two off one, one of the three that lie nearest the note before.
clock("1/4")
const rnd = genny.rand(0, 1)
const R = () => rnd.next()
let prev = 48, last = -1
function trigger(ev) {
  if (!shared.known()) return
  const s = shared.s(), bar = shared.bar()
  if (ev.step % 4 !== 0 || bar % 4 !== 0 || s.voice === 0 || R() > 0.4 + s.voice) return
  const tones = shared.chordAt(bar).tones(36, 72), found = []
  shared.sounds.sung.forEach((n, i) => {
    for (let d = -2; d <= 2; d++) {
      const t = n.pitch + d
      if (!tones.includes(t)) continue
      found.push({ score: 2 * Math.abs(d) + Math.abs(t - prev) / 3 + (i === last ? 6 : 0) + (t === prev ? 5 : 0), i, d, t })
    }
  })
  if (!found.length) return
  found.sort((x, y) => x.score - y.score)
  const c = found[Math.min(found.length - 1, Math.floor(R() * R() * 3))]
  prev = c.t; last = c.i
  const n = shared.sounds.sung[c.i]
  note(n.key + c.d, Math.min(1, 0.55 * shared.evenOut(n.db, -16)) * s.voice, Math.max(3, Math.min(12, Math.floor(n.sec / ev.seconds) - 1)))     // never longer than the recording, or its end goes round
}
