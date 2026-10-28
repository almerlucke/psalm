// the violin is the second voice of the weft: it takes the tune up four bars behind the flute, so that where the flute sings the second half the violin sings the first, and
// they cross in thirds and sixths. In the Wonder it has the hymn
clock("1/16")
const part = shared.part("violin", { vel: 0.6, legato: 0.97 }, sec => {
  switch (sec.name) {
    case "Knit": return shared.phrases(shared.alternate(4, 16, "A"), 0, { vel: 0.6 })
    case "Wonder": return shared.melody("W", 0, 0, { vel: 0.5 }).concat(shared.melody("W", 8, 0, { vel: 0.6 }))
    case "Light": return shared.phrases(shared.alternate(4, 12, "A"), 0, { vel: 0.6 })
  }
  return []
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
