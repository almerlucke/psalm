// the flute sings the tune: A and B, the two halves that fit over each other, in turn. It is the first voice (the weft begins with it): alone in the Thread, in the Knit at
// bar 0 with the violin four bars behind it and the cello four more, and in the Light; high above the Wonder's second hymn; alone over the sand; and only the first two bars
// of it when the sleeper wakes
clock("1/16")
const part = shared.part("flute", { vel: 0.6, legato: 0.97 }, sec => {
  const bars = shared.beats
  switch (sec.name) {
    case "Thread": return shared.phrases([[4, "A"]], 0, { vel: 0.5 })
    case "Knit": return shared.phrases(shared.alternate(0, 16, "A"), 0, { vel: 0.62 })
    case "Wonder": return shared.melody("W", 8, 12, { vel: 0.4 })
    case "Light": return shared.phrases(shared.alternate(0, 12, "A"), 0, { vel: 0.6 })
    case "Sand": return shared.phrases([[4, "A"], [12, "A"]], 0, { vel: 0.5 })
    case "Awake": return shared.melody("A", 0, 0, { vel: 0.4 }).filter(e => e.at < 2 * bars)
  }
  return []
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
