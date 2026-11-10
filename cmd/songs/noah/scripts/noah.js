// NOAH: the recording, HeavenlyNoah.wav, played once from the first step to its end by a streamer (it is read from the disk while it plays; the note is of a sound that
// is not held, so it plays to its end whatever the length of the note).
clock("1/8")
let started = false
function trigger() {
  if (started) return
  started = true
  note(36, 1, 1)
}
