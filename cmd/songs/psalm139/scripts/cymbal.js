// a swell of a cymbal, once at the start of the Wonder and once, softer, at the start of the Light: the moment the thought of Him is seen
clock("1/16")
const part = shared.part("cymbal", { raw: true, vel: 0.5, legato: 1 }, sec => {
  switch (sec.name) {
    case "Wonder": return [{ at: 0, p: shared.kits.cymbal.swell[0], beats: 8, vel: 0.5 }]
    case "Light": return [{ at: 0, p: shared.kits.cymbal.swell[0], beats: 8, vel: 0.32 }]
  }
  return []
})
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
