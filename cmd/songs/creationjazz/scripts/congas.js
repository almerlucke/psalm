// the congas play the tumbao of the waltz: the low drum on the first beat, the high one and the muted ones around it (l low, o open, h high, m muted; a capital
// letter louder). It begins soft, four bars into the intro, and fills with the sections: the break and the head out add the open tones; in the tag it is three beats
clock("1/16")
const k = shared.kits.congas
const play = shared.player("congas")
const kinds = { l: k.low, o: k.open, h: k.high, m: k.muted }
const P = ["l..h..m.o...", "l..h..m.o...", "l.hm.hl.hm.h", "L.hmohL.hmoH", "l.hm.hl.hm.h", "L.hmohL.hmoh", "l..h..L..h.."]
let last = -1, n = 0
function trigger(ev) {
  const day = shared.days[song.index]
  const s = Math.round(song.beat * 4)
  const at = song.bar * 12 + s
  if (!day || at === last) return
  last = at
  const pat = P[day.number - 1]
  if (pat.length !== 12) throw new Error("a bar of twelve sixteenths: " + pat)
  const c = pat[s]
  if (c === "." || !shared.layers.congas(day, song.bar)) return
  const keys = kinds[c.toLowerCase()]
  const loud = c === c.toUpperCase() ? 1.25 : 1
  play(note, ev, s, keys[n++ % keys.length], (s % 4 === 0 ? 0.5 : 0.38) * loud * (day.number === 1 ? 0.7 : 1), 2, true)
}
