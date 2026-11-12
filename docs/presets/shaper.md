# shaper

The chains of every preset are in the module's window in the patcher (see [synths](../synths/shaper.md#shaper)). `go run ./cmd/examples/presets -synth shaper` plays all of them, two bars each. Presets with chains of their own for line 2 (a different wave against the first, not a copy of it): warmPad, stringPad (the two lines morph opposite ways), darkPad, reese, squareLead, pwm (the pulse widths move opposite ways), fmLead (another ratio), brassLead, syncLead (another sync ratio) and duoLead. Every preset is held to the same measures by a test: no clipping, as loud as the others (a sustained sound at about -15 dB rms, a short one peaking near 0.5), under -18 dB of its energy above 6 kHz (the effects excepted), silent within 12 seconds of the key being let go, and not like any other preset.

## Basses

| Name | Function | Sound |
|---|---|---|
| subSine | SubSine | SubSine is a deep sine with a little thump at the start and a touch of odd harmonics, so a small speaker has something to play |
| kickSub | KickSub | KickSub is a kick drum: a sine whose pitch falls three octaves in a flash. Play it low (C1 to C2) |
| reese | Reese | Reese is two saws a few cents apart, beating slowly, under a low pass that breathes |
| acid | Acid | Acid is the squelch: a saw (or a square) into a resonant low pass whose envelope snaps open on every note, a harder note opening it further and going square; it plays one note at a time and slides |
| wobble | Wobble | Wobble is a saw and a fold through a low pass that an LFO sweeps up and down, about three times a second |
| pulseBass | PulseBass | PulseBass is a pulse whose width narrows over the note, through a snapping low pass |
| fmBass | FMBass | FMBass is a phase modulated sine with a lot of modulation at the start that falls away, a sine an octave under |
| growl | Growl | Growl is FM with a low modulator folded hard, swelling and falling twice a second |
| subFold | SubFold | SubFold is a deep sine folded into a growl by the filter's envelope, which here is the shape's |

## Leads

| Name | Function | Sound |
|---|---|---|
| supersawLead | SupersawLead | SupersawLead is the big trance saw: two saws a few cents apart, wide, through a low pass, a vibrato that comes in late |
| squareLead | SquareLead | SquareLead is a hollow pulse with its width moving, one note at a time, sliding |
| syncLead | SyncLead | SyncLead is a saw hard synced to a higher pitch that sweeps down as the note goes on |
| pluckLead | PluckLead | PluckLead is a bright saw that mellows at once, a filter snapping shut: a sequence lead |
| fmLead | FMLead | FMLead is an electric-organ-bright phase modulated lead: the harder the key the brighter, the wheel adds more |
| fluteLead | FluteLead | FluteLead is a nearly pure tone with a breath of the second and third harmonic, slow to speak, with a vibrato |
| brassLead | BrassLead | BrassLead swells in with its brightness: phase modulation that opens over the first tenth of a second |
| vowel | Vowel | Vowel moves between mouth shapes made of a few partials, no filter needed |
| reed | Reed | Reed goes from a soft triangle to a bright squeezed saw with how hard the key is struck |

## Pads

| Name | Function | Sound |
|---|---|---|
| warmPad | WarmPad | WarmPad is detuned saws behind a low pass that breathes, wide and slow |
| stringPad | StringPad | StringPad is saw and pulse together, the morph between them drifting |
| choirPad | ChoirPad | ChoirPad is mouth shapes moving slowly, a few people singing |
| airPad | AirPad | AirPad is high, soft and moving, sines and a fold that comes and goes |
| darkPad | DarkPad | DarkPad is a low pulse behind a closed filter that opens and shuts very slowly |
| evolve | Evolve | Evolve goes through four waves by itself on two slow LFOs that never line up: sine, a fold, a saw, a pulse; no two bars sound the same |
| glassPad | GlassPad | GlassPad is soft phase modulation, the glassy sheen of an old digital pad |
| foldPad | FoldPad | FoldPad is a sine folded back on itself more and more, slowly swelling and fading |
| morphPad | MorphPad | MorphPad goes through four waves, sine, triangle, saw, square, and back, on a slow LFO |
| swarm | Swarm | Swarm is a saw folded and detuned wide, restless on two LFOs |
| organ | Organ | Organ is drawbars as a sum of sines, the upper ones joining in with the morph |

## Keys and plucks

| Name | Function | Sound |
|---|---|---|
| pluck | Pluck | Pluck is a bright start that falls to a mellow tone at once |
| harp | Harp | Harp is a plucked string: all the harmonics at the start, the high ones dying first |
| kalimba | Kalimba | Kalimba is a thumb piano: a quick ping on top of the note that falls away |
| marimba | Marimba | Marimba is wood: a sine with a hard partial at four times its pitch that dies in a moment |
| ePiano | EPiano | EPiano is a tine piano: phase modulation, brighter the harder the key, the bright part dying first, a second, softer layer ringing a little out of tune |
| clav | Clav | Clav is a clavinet: a narrow pulse and a filter that snaps shut in a few milliseconds |
| vibes | Vibes | Vibes is a vibraphone: a sine with a bright partial that fades, and a tremolo |
| stab | Stab | Stab is the chord stab of house: detuned saws, a filter that snaps shut, a short tail |
| musicBox | MusicBox | MusicBox is a comb of metal: a sine and a partial at five times its pitch, tinkling |
| pwm | PWM | PWM is a pulse whose width moves on an LFO, two lines apart |

## Bells

| Name | Function | Sound |
|---|---|---|
| bellMorph | BellMorph | BellMorph strikes a bell and lets it settle into a plain sine |
| glassBell | GlassBell | GlassBell is phase modulation at three times the pitch, falling to a pure tone: a struck glass |
| churchBell | ChurchBell | ChurchBell is partials that die one after the other: the high ones in a moment, the low ring on |
| gong | Gong | Gong is a swell of a very bright, very low sound that settles over many seconds |
| metal | Metal | Metal stacks whole number partials, brightening and fading: a struck pipe |
| glass | Glass | Glass is a sine that grows overtones as the note starts, folding into a glassy ring |

## Drones and textures

| Name | Function | Sound |
|---|---|---|
| drone5 | Drone5 | Drone5 is a note and its fifth, folded softly, breathing: a drone to sit under a piece |
| subDrone | SubDrone | SubDrone is a very low sine folded slowly into a growl and back |
| crush | Crush | Crush is a saw cut into steps, fewer of them as the macro goes up |

## Effects

| Name | Function | Sound |
|---|---|---|
| laser | Laser | Laser falls from far above the note in a moment, a hard synced saw |
| riser | Riser | Riser climbs for a long note: pitch, brightness and fold all going up, for the bars before a change |
| impact | Impact | Impact is a low hit: a fold falling from high to low and a sine thump under it, ringing out |
| zap | Zap | Zap is a short fall in pitch of an FM tone: a game sound |

## Cross modulation

| Name | Function | Sound |
|---|---|---|
| xmodBell | XModBell | XModBell is an FM bell: a silent line at 3.5 times the pitch bends the phase of a sine, and the bend dies with the note, so the strike is bright and metallic and the ring is a clean tone |
| xmodGrowl | XModGrowl | XModGrowl is a bass from a saw-like carrier bent by a line an octave under it, the bend breathing on a slow LFO: the growl of a talking bass |
| ringMetal | RingMetal | RingMetal is two lines a tritone apart multiplied together: the sum and the difference of their partials, a clang that belongs to no pitch |

## Beyond

Sounds only this synth makes: the wave itself travels, changes with the key or the touch, is drawn anew with every note, rings with partials that belong to no pitch, or is a drum.

| Name | Function | Sound |
|---|---|---|
| rainbow | Rainbow | a pad that travels through four colours in one held chord (a sine, a bend of two sines, a fold, the partials 1, 3, 7 and 12): the morph envelope takes six seconds and a slow wander rides on it |
| crystalSea | CrystalSea | a pad that does not repeat: waves of twelve and seven seconds move the morph and a macro, every note a third, so it never comes back to the same place |
| fourBeasts | FourBeasts | four instruments on the keys: the note pulls the morph, a lion (folded FM) low, an ox (a hollow square), a man (a voice with its overtones) and an eagle (bright partials) high |
| trumpetVoice | TrumpetVoice | a hollow breath played softly and brass played hard: the strength of the touch bends the wave, folds it and opens the filter, and the swell of the morph lags the note |
| sevenLamps | SevenLamps | a bell of fire: one sine bends the phase of another at seven to three, so its partials fall where no harmonic series puts them, and the bend flickers as it dies |
| crownsCast | CrownsCast | a clatter of gold: two lines multiplied, the sums and differences of all their partials, dying in a second |
| thunder | Thunder | a sine falling two octaves with a fold and a rumble: a kick low, a tom higher |
| lightning | Lightning | noise through a band that follows the key: a snare low, a hat high (the thunder and the lightning are a drum kit from one synth) |
| lionBass | LionBass | a purr when touched softly and a roar when hard: the velocity folds the wave |
| throneDrone | ThroneDrone | a drone of the partials 1, 3, 7 and 12 whose levels breathe against one another on waves of twelve and seven seconds |
| manyEyes | ManyEyes | a lead that draws itself anew with every note (morph, fold and bend from a random value for the note) |

## New shapers

The stages from genny's newer shapers (folds, the tube, dead zones, curves, smoothsteps, Casio CZ phase distortion, harmonics, polynomials, funnels); `go run ./cmd/examples/shapestages` plays them with their chains.

| Name | Function | Sound |
|---|---|---|
| czResonant | CZResonant | the Casio CZ resonant wave: a falling saw times a cosine of higher pitch, whose resonance rises as the note opens |
| tubeOrgan | TubeOrgan | a sine with its octave and fifth driven through an asymmetric tube; the touch leans the wave and adds even harmonics |
| foldLead | FoldLead | a sine folded more and more as the note opens: the harmonics climb like a west coast synth's |
| triFoldPad | TriFoldPad | a sine reflected at the edges by a gain that rises slowly: a pad that grows teeth |
| tubeBass | TubeBass | a sine driven hard into an asymmetric tube at the start of the note, then settling: a round bass with even harmonics |
| softDeadLead | SoftDeadLead | a sine with a dead zone that widens over the note: a thin, reedy lead |
| curveBass | CurveBass | a saw bent by a tension that moves from one side to the other: a bass that is hollow and then full |
| stepLead | StepLead | a triangle pressed towards a square by a smootherstep whose edge closes in: soft to bright |
| czSawLead | CZSawLead | a Casio CZ saw whose distortion grows during the note, with vibrato and a second line an octave down |
| czSquareBass | CZSquareBass | the CZ square: a cosine turned into a pulse with fast edges |
| czPulseArp | CZPulseArp | the CZ pulse: a cosine squeezed into the start of the cycle, plucked |
| harmPad | HarmPad | a sum of sines (1, 3, 5 and 8) as a waveshaper, held in a unison |
| polyLead | PolyLead | a saw through a cubic: rounded, soft at the peaks |
| legendrePad | LegendrePad | the Legendre polynomial of order five as a waveshaper: a strange, even stack of harmonics |
| funnelArp | FunnelArp | a saw snapped to six levels, plucked: a staircase with the grain of a low bit depth |
| duoLead | DuoLead | two different oscillators: a folded sine on line 1, a CZ saw an octave up on line 2 (chains of its own), both opening with the note |
| splitPad | SplitPad | a different wave on each side: a soft triangle on the left, a narrow pulse on the right (line 2 with chains of its own, width 1) |
| splitLead | SplitLead | a folded sine on the left and a CZ saw on the right, each opening with the note |
| wideFoldPad | WideFoldPad | one chain made wide: a fold that is deeper on the right (stereo m1), so the left is soft and the right bright |
| wideMorph | WideMorph | a pad that travels through a sine, a triangle, a bend of two sines and a narrow pulse, the left behind the right (stereo morph) |
