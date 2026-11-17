// the void: "the earth was without form, and void; and darkness was upon the face of the deep", an open fifth under the intro; it returns, softer and as a bare fifth, under the last chord of the
// tag. The darkness lifts at "Let there be light": the cutoff of the synth, on the mod output (a cable to it)
clock("1/16")
const part = shared.register("void", { legato: 1 }, day => {
  switch (day.number) {
    case 1: return shared.join([shared.chord([39, 46], 12, { hold: 13, amp: shared.DARK })], shared.rest(4))
    case 7: return shared.join(shared.rest(shared.FINAL), [shared.chord([39, 46], 12, { hold: 12, amp: 0.3 })])
  }
  return null
})
function section(name) { part.section(name) }
function trigger(ev) {
  part.trigger(song, note, ev)
  // 0 to 1 over the range of the cutoff: dark (300 Hz) through the first bars, open within a bar of the light
  let open = 1
  if (song.section === shared.days[0].name) {
    const bar = song.bar + song.beat / shared.beats
    open = bar < shared.EVENING ? 0 : Math.min(1, (bar - shared.EVENING) / 1)
  }
  mod(open)
}
