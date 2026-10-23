// mankind: the hands of man on the piano, in the morning of the sixth day, the sustain pedal changed at every bar line
clock("1/16")
const part = shared.register("piano", { legato: 1 }, day => {
  if (day.number !== 6) return null
  return shared.perBar(day, (b, br) => shared.morning(b) ? shared.arp([br.root + 12, br.root + 19, ...br.mid]) : null)
})
function section(name) { part.section(name) }
function trigger(ev) {
  part.trigger(song, note, ev)
  if (song.section === shared.days[5].name && song.bar >= shared.EVENING) { // up for a moment at the bar line, down for the rest of the bar
    set("sustain", song.beat / shared.beats < 0.05 ? 0 : 1)
  } else {
    set("sustain", 0)
  }
}
