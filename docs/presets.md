# Presets

The presets of the synths, by family. The ones in [synths](synths.md) are the first of each synth; those listed here came with the later work (the modulation matrix, the second filter, the oscillator B and warp modes, the free operators, the physical models). Every preset is in the synth's `Presets` map, and can be loaded by name in the patcher, the editors and the examples.

They were designed from knowledge of the classic sounds and checked by measurement: each renders finite, non-silent sound over the keyboard, is calibrated to -12 LUFS or a -1 dB peak (`synths/levels`), and differs from every other preset of its synth by a spectral fingerprint. They have not been voiced by ear, so treat them as starting points.

## subtractive

### Basses

| Name | Function | Sound |
|---|---|---|
| moogBass | MoogBassPreset | Minimoog bass: two saws an octave down and a square, a 24 dB ladder snapping shut |
| pulseBass | PulseBassPreset | Thin pulse bass whose pulse width follows the velocity: a hard note is hollower and nasal |
| growlBass | GrowlBassPreset | Snarling bass: a saw phase modulated by its own octave, driven, the wheel pushes the FM |
| bass808 | Bass808Preset | Long sub of a 808: a sine with a short pitch dive at the start and a slow decay |
| acidSquare | AcidSquarePreset | Square-ish acid line: a pulse through a screaming ladder, glided, accents from the velocity |
| dubWobble | DubWobblePreset | Dubstep wobble: a low saw pair, a resonant ladder swept by a triangle LFO, the wheel speeds it up |
| pluckBass | PluckBassPreset | Short, woody bass pluck, the velocity opens the filter |
| wideReese | WideReesePreset | Reese of five detuned unison saws, drifting, the wheel spreads them further |

### Leads

| Name | Function | Sound |
|---|---|---|
| monoLead | MonoLeadPreset | Classic mono lead: two drifting saws through a ladder, glided, a late vibrato |
| squareLead | SquareLeadPreset | Hollow square lead with a narrow pulse an octave up and a soft vibrato |
| pwmLead | PWMLeadPreset | Pulse lead with the width swept by a slow LFO and two unison voices, a Pro-One style PWM |
| tranceLead | TranceLeadPreset | Big seven voice unison saw lead, bright and wide, a fifth and an octave in the second oscillator |
| portaLead | PortaLeadPreset | Legato lead that slides a long way between notes, the wheel adds a vibrato rate and bite |
| screamLead | ScreamLeadPreset | Hard synced lead: the velocity sweeps the synced oscillator, the harder the higher the scream |
| sineLead | SineLeadPreset | Soft, pure lead: a sine with a touch of triangle, a slow vibrato and a slight glide |
| brassLead | BrassLeadPreset | Brassy solo line: saws through a 12 dB ladder that blares open with the velocity |

### Pads

| Name | Function | Sound |
|---|---|---|
| warmPad | WarmPadPreset | Soft analog pad: triangle and saw, drifting, a 12 dB ladder opening slowly |
| glassPad | GlassPadPreset | Cold, airy pad of sines and an octave of triangle with the lows cut away |
| stringMachine | StringMachinePreset | String machine: three unison saws, an airy low pass and a slow swell |
| evolvingPad | EvolvingPadPreset | Pad that never stops moving: a slow LFO on the cutoff and the width, another on the resonance |
| ahChoir | AhChoirPreset | Human "aah": unison saws through the formant filter, the vowel moves a little with a slow LFO |
| vowelPad | VowelPadPreset | Gets its vowel colour from a second filter: a low pass and a resonant band beside it, an "oh" pad |
| darkPad | DarkPadPreset | Brooding pad: saws an octave down under a closed 24 dB ladder |
| brightPad | BrightPadPreset | Open pad of saw and pulse in unison with a high pass keeping it light |
| hugePad | HugePadPreset | Wall of seven unison voices spread wide, slow and enormous |
| junoPad | JunoPadPreset | Juno pad: a pulse swept by the LFO with a sub and saw, a high pass and the chorus-like unison |
| jupiterPad | JupiterPadPreset | Jupiter-8 pad: bright detuned saws, a resonant filter that opens and a second band for air |

### Keys, plucks and sequences

| Name | Function | Sound |
|---|---|---|
| clavinet | ClavinetPreset | Funky clavinet: a narrow pulse, a high pass taking away the body, a snap that follows the velocity |
| harpsichord | HarpsichordPreset | Plucked harpsichord: two bright saws an octave apart, a click and no sustain |
| wurli | WurliPreset | Wurlitzer style reed piano: a triangle with a sine an octave up and a bark that follows the velocity |
| marimba | MarimbaPreset | Wooden mallet: a sine with a fast dying high partial two octaves and a bit up |
| pizzicato | PizzicatoPreset | Plucked string section note: a short saw with the filter closing quickly |
| tubularBell | TubularBellPreset | Church bell: a ring modulated pair at an inharmonic ratio, FM on top, ringing out |
| analogSeq | AnalogSeqPreset | Step sequencer bass note: a short resonant saw with a random cutoff nudge on every step |

### Brass, strings, vintage

| Name | Function | Sound |
|---|---|---|
| junoBrass | JunoBrassPreset | Juno brass: a saw and a pulse in unison with a sub, a quick swell and a high pass |
| jupiterBrass | JupiterBrassPreset | Jupiter-8 brass: two driven saws in a 12 dB ladder, a bite of resonance at the swell |
| stringSection | StringSectionPreset | Full string section: five unison saws drifting with a breath of vibrato, a slow bow |
| obStab | OBStabPreset | Oberheim style stab: two saws, a short open ladder swell and a little drift |

### Organs and drones

| Name | Function | Sound |
|---|---|---|
| percOrgan | PercOrganPreset | Percussive organ: drawbar sounds at 8', 4' and 16' with a fast key click that fades |
| rockOrgan | RockOrganPreset | Overdriven rock organ, the wheel pushes it further into the drive |
| droneFifth | DroneFifthPreset | Held drone of an open fifth in unison saws, slowly breathing |
| darkDrone | DarkDronePreset | Low rumbling drone: a sub, a dark saw and a band of rustling noise |

### Effects

| Name | Function | Sound |
|---|---|---|
| gale | GalePreset | Howling wind: noise through a resonant band that a random sample and hold keeps moving |
| laser | LaserPreset | Sci-fi laser: a resonant saw falling several octaves in a flash |
| sirenFx | SirenFxPreset | Rescue siren: a wide triangle LFO rising and falling in pitch, the wheel speeds it |
| noiseTexture | NoiseTexturePreset | Tonal noise bed: noise through a swept notch and a high pass, shifting from note to note |

### Drums

| Name | Function | Sound |
|---|---|---|
| kick808 | Kick808Preset | Deep, round kick: a sine with a fast pitch fall and a long boomy tail, play around MIDI 33 |
| clap | ClapPreset | Hand clap: noise through a band pass around 1.2 kHz, with a quick double burst of the envelope |
| rimShot | RimShotPreset | Rim shot: two short, high pulses with a click of noise through a band pass |
| cowbell | CowbellPreset | Cowbell: two squares a tritone apart through a band pass, a short knock |

## supersaw

### Pads and strings

| Name | Function | Sound |
|---|---|---|
| stringMachine | StringMachine | Ensemble strings: a narrow supersaw through the three delay chorus, a slow swell |
| choirWash | ChoirWash | Vast, hazy wash: the cutoff low and slow, the ensemble wide and slow, the lows cut at 350 Hz |
| epicPad | EpicPad | Sixteen saws wide open, the wheel spreading them further and the velocity opening the filter |
| darkSwarm | DarkSwarm | Low, heavy swarm: a sub under a dark filter that opens and closes with a very slow LFO |
| shimmerPad | ShimmerPad | High, glittering pad: the lows cut at 900 Hz, an ensemble that breathes with LFO 2 |
| gatedPad | GatedPad | Trance gate: a pad chopped into eighth notes by the level, in time with the tempo |
| pumpPad | PumpPad | Sidechained pad: it ducks on every beat and swells back, the filter breathing with it |
| airPad | AirPad | Light, open pad: few detuned saws, the lows cut and the ensemble around it |
| analogPad | AnalogPad | Pad that is never the same twice: every note gets a random detune and a hair of random pitch |
| morphPad | MorphPad | Changes with the wheel: from a narrow, dry pad to a wide, thick ensemble |

### Leads

| Name | Function | Sound |
|---|---|---|
| anthemLead | AnthemLead | Stadium trance lead: a glide, a vibrato from LFO 2 and the wheel pulling the saws apart |
| screamingSaw | ScreamingSaw | Resonant, biting lead: the velocity opens it and pushes the resonance |
| hardstyleLead | HardstyleLead | Hard, screeching lead: seven fat saws with a sub, a short pitch kick at the start |
| trioLead | TrioLead | Lead of three saws: tight, a little glide and a vibrato |
| expressiveLead | ExpressiveLead | Answers the player: the velocity opens the filter and spreads the saws, the wheel brings in the ensemble |

### Basses

| Name | Function | Sound |
|---|---|---|
| dubstepGrowl | DubstepGrowl | Bass with its cutoff swept four times a beat, the wheel speeding the sweep |
| dropBass | DropBass | Sub drop: a sine-like low end that falls a fifth in the first moments |
| reeseBass | ReeseBass | Rolling reese: three wide saws and a sub under a filter moving slowly |
| gliderBass | GliderBass | Mono bass that slides between its notes, three saws and a sub |

### Plucks, stabs, brass

| Name | Function | Sound |
|---|---|---|
| arpSaw | ArpSaw | Short sequenced saw for arpeggios: a snappy filter, the velocity as the accent |
| houseStab | HouseStab | Chord stab with the lows cut at 150 Hz: short, bright and punchy, the velocity opens it |
| brassSection | BrassSection | Saw brass: five close saws, a swell of the filter that follows the velocity |
| ravePluck | RavePluck | Bright plucked chord with a touch of ensemble: the filter snaps shut |
| softPluck | SoftPluck | Gentle ensemble pluck: three saws, a mellow filter closing, the chorus wide |

### Effects

| Name | Function | Sound |
|---|---|---|
| dive | Dive | Effect that falls: the pitch drops an octave over the first seconds as the filter closes |
| riseUp | RiseUp | Riser: the pitch and the cutoff climb for three seconds |

## wavetable

### Basses

| Name | Function | Sound |
|---|---|---|
| subBass | SubBassPreset | A sine sub for the bottom of a mix: a barely brighter wave over a pure sine an octave down, a soft lowpass and a short glide |
| wobbleBass | WobbleBassPreset | The dubstep wobble: a slow LFO sweeps the cutoff of a resonant lowpass over a pair of detuned oscillators, the wheel speeds the wobble up |
| fmBass | FMBassPreset | A clanking FM bass: a sine whose phase is modulated by a sine at its own pitch, the filter envelope opening the modulation at every note, harder notes opening it more |
| syncBass | SyncBassPreset | A hard sync bass: the frame is played faster and faster at every note, the tearing sound of a sync sweep, closed down by a lowpass |
| hooverBass | HooverBassPreset | The rave hoover: seven oscillators far apart, the phase bent, and a pitch that scoops down into the note |

### Leads

| Name | Function | Sound |
|---|---|---|
| syncLead | SyncLeadPreset | A screaming sync lead: the wheel pushes the sync ratio up, harder notes sync further |
| mirrorLead | MirrorLeadPreset | A hollow, folding lead: the phase is played forward then backward, and a slow LFO moves how far |
| fmLead | FMLeadPreset | A glassy FM lead: a sine an octave up modulates the phase, a burst of modulation at the start of the note settles to a singing tone |
| superLead | SuperLeadPreset | A supersaw lead: seven oscillators spread wide and bright, for trance |
| thereminLead | ThereminLeadPreset | A theremin: a nearly pure tone, a late deep vibrato and a long glide |

### Pads

| Name | Function | Sound |
|---|---|---|
| glassPad | GlassPadPreset | A pad of glass: a thin second oscillator two octaves up in a shimmering stack, whose timbre a slow LFO moves |
| evolvingPad | EvolvingPadPreset | EvolvingPadPreset never stands still: one slow LFO travels the table, another slow one bends the phase, the wheel opens the filter further |
| voxPad | VoxPadPreset | A vocal-ish pad: a resonant bandpass that a slow LFO sweeps over the vowels of a wide stack |
| ensemblePad | EnsemblePadPreset | A string machine: two stacks an octave apart, one detuned wide, a soft lowpass and a slow bow |
| darkPad | DarkPadPreset | A brooding pad: low, with a sub, the filter nearly shut and slowly breathing |
| shimmerPad | ShimmerPadPreset | ShimmerPadPreset floats above the note: a second stack a twelfth up, fading in and out with a slow LFO, under a highpass that takes the weight away |
| choirPad | ChoirPadPreset | A choir: its own table of vowels ("ah" to "oh" to "oo") sung by four voices, vowels moving slowly. It ignores the table it is given |

### Plucks and keys

| Name | Function | Sound |
|---|---|---|
| syncPluck | SyncPluckPreset | A pluck whose sync sweeps down at every note, from tearing to round |
| fmPluck | FMPluckPreset | A plucked FM string: the phase modulation bursts and dies with the note |
| harpPluck | HarpPluckPreset | A harp: a bright start that falls up and down the table, a lowpass closing on the pluck and a soft octave overtone |
| marimba | MarimbaPreset | A marimba: a sine with a short ring two octaves up that dies in a blink |
| tineKeys | TineKeysPreset | An electric piano: a near sine whose phase an overtone three octaves up bends at the strike, the bark harder with the velocity, ringing out long |
| clavKeys | ClavKeysPreset | A clavinet: a bright, thin wave with a snap, no sustain to speak of |

### Bells

| Name | Function | Sound |
|---|---|---|
| musicBox | MusicBoxPreset | A music box: a sine with a thin tine three octaves and a fifth up, ringing short and high |
| fmBell | FMBellPreset | A DX style bell: a sine modulated by a sine at 3.5 times its pitch, the modulation dying away so the bell turns pure as it rings |
| tubularBell | TubularBellPreset | A tubular bell: a stack of sines with a strike partial at about 2.8 times the pitch, a long ring |

### Brass and strings

| Name | Function | Sound |
|---|---|---|
| brassSection | BrassSectionPreset | A big brass section: two stacks a few cents apart, the filter swelling into each note and the velocity digging in |
| synthBrass | SynthBrassPreset | The Jupiter brass: a bent saw with a squealing filter swell, solo |
| cello | CelloPreset | A solo cello: a warm saw with its octave under, a slow bow and a heavy vibrato |
| violin | ViolinPreset | A solo violin: bright, a quick vibrato following the wheel in the filter and a bow that bites |

### Organs and winds

| Name | Function | Sound |
|---|---|---|
| drawbarOrgan | DrawbarOrganPreset | A drawbar organ with a key click: its own table, from a soft registration to the full organ, the click a quick flick towards the full registration, and the slight wobble of a rotating speaker. It ignores the table it is given |
| churchOrgan | ChurchOrganPreset | A pipe organ: its own table from a flute to a full principal chorus, a 16' pedal under, a breath of attack and a long release. It ignores the table it is given |
| percOrgan | PercOrganPreset | A rock organ: the drawbars out and a decaying third harmonic percussion that bites at every note. It ignores the table it is given |
| clarinet | ClarinetPreset | A clarinet: its own table of odd harmonics, darkest in the low register and brighter as it is blown harder. It ignores the table it is given |
| flute | FlutePreset | A breathy flute: its own, nearly pure table, a puff of the second harmonic and the phase wobbled by a sine for air. It ignores the table it is given |

### Arps

| Name | Function | Sound |
|---|---|---|
| arpPluck | ArpPluckPreset | For arpeggios: short, the filter and the fold of the phase snapping closed, and a new random brightness at every note |
| arpGlass | ArpGlassPreset | A glassy arpeggio voice: a bright FM ping with a different amount of bite at every note |

### Drones

| Name | Function | Sound |
|---|---|---|
| darkDrone | DarkDronePreset | A cavern: seven slow oscillators, a sub, a second stack an octave down, the filter breathing almost shut |
| tensionDrone | TensionDronePreset | Uneasy: a second oscillator a tritone away, the phase folding slowly |

### Effects

| Name | Function | Sound |
|---|---|---|
| sirenFx | SirenFXPreset | An alarm: an LFO swings the pitch over an octave and a half, the wheel quickens it |
| laserFx | LaserFXPreset | A laser: the pitch falls two octaves in an instant, the phase modulation with it |
| windFx | WindFXPreset | Wind: heavy phase modulation turns seven oscillators to noise, a bandpass wanders over it |

### Percussion

| Name | Function | Sound |
|---|---|---|
| kickDrum | KickDrumPreset | A kick: a sine whose pitch falls fast from two octaves up, played low |
| tomDrum | TomDrumPreset | A tom: a slower, shallower fall of the pitch and a longer ring, with a detuned partner for the skin |
| snareHit | SnareHitPreset | A snare: a body and a burst of noise, the noise made from heavy phase modulation of a stack under a highpass |
| hatMetal | HatMetalPreset | A hi-hat: inharmonic phase modulation of five oscillators, only the top left |
| rimClick | RimClickPreset | A rim shot: a very short, resonant click |

## pulsar

### Formant pads

| Name | Function | Sound |
|---|---|---|
| ahPad | AhPadPreset | Broad "ah" pad: two open formants, a slow drift of the vowel and a wide spread |
| oohPad | OohPadPreset | Closed, dark "ooh": a low formant with a faint second one, hardly any movement |
| eerieChoir | EerieChoirPreset | Thin "ee" choir with a little breath: a very low first formant under a high second one, and a slow wandering of both |
| shimmerPad | ShimmerPadPreset | Glassy pad: narrow Gaussian pulses in the highs, the second formant swayed by the second LFO of the matrix so it glitters |
| darkPad | DarkPadPreset | Brooding pad: low formants through a ladder filter that opens slowly at every note |

### Vowel leads

| Name | Function | Sound |
|---|---|---|
| aeLead | AeLeadPreset | Lead that sings "a-e-a": the formants swing slowly between the vowels |
| wahLead | WahLeadPreset | Lead played with the mod wheel and the touch: the wheel opens the formant, a hard note opens it more, the saw pulses give the wah something to bite into |
| talkLead | TalkLeadPreset | Talking synth: a fast swing of the formants, a vowel every few hundred milliseconds, the pitch gliding between the notes |
| sawVowel | SawVowelPreset | Vowel lead from saw pulsarets: all the harmonics inside the formant, a low pass keeping the top down, a random touch of formant at every note |

### Granular clouds (low notes)

| Name | Function | Sound |
|---|---|---|
| grainCloud | GrainCloudPreset | Cloud of grains for notes around 40 Hz: soft Gaussian pulses, half of them left out, the period jittering, every grain in its own place; the formant drifts |
| shimmerDust | ShimmerDustPreset | Fine dust for notes around 60 Hz: tiny high grains in two bands, sparse and wide |
| lowCloud | LowCloudPreset | Rumbling cloud for notes around 25 Hz: long soft grains in the lows that overlap and drift in and out |
| scatterBells | ScatterBellsPreset | Bells thrown about for notes around 20 Hz: sparse struck pulses in the highs, each at a place and a pitch of its own, the formant wandering |
| swarmCloud | SwarmCloudPreset | Swarm for notes around 80 Hz: buzzing mid grains, the pulse rate swaying by whole semitones so the swarm swells and thins |

### Rhythmic bursts (low notes, chopLead is a note)

| Name | Function | Sound |
|---|---|---|
| stutterTick | StutterTickPreset | Stuttering hi-hat for notes around 12 Hz: three ticks and two rests, over and over |
| gallop | GallopPreset | Low gallop for notes around 10 Hz: three dull thuds and a rest |
| clavePulse | ClavePulsePreset | Clave pattern for notes around 8 Hz: two woody knocks and three rests, a five-step feel |
| machineGun | MachineGunPreset | Burst of fire for notes around 18 Hz: four hits with a falling, snapping formant, four rests |
| chopLead | ChopLeadPreset | Pitched lead gated by its own bursts: six pulses on, two off, so the tone flutters at a quarter of the pitch |

### Crackle, rain and insect textures

| Name | Function | Sound |
|---|---|---|
| fireCrackle | FireCracklePreset | Fire for notes around 45 Hz: most pulses gone, the rest in bright pops |
| crickets | CricketsPreset | Crickets for notes around 40 Hz: short chirps of three pulses in a high band with a rest between, the chirp rate wavering |
| cicadas | CicadasPreset | Tree full of cicadas for a note: a bright buzz pulsing in and out, the level swaying with the second LFO of the matrix |
| drizzle | DrizzlePreset | Light rain for notes around 15 Hz: soft drops, half missing, the pitch of every drop a little different |
| geigerTicks | GeigerTicksPreset | Counter for notes around 30 Hz: very short clicks, nine in ten gone |

### Tonal plucks

| Name | Function | Sound |
|---|---|---|
| pluckPulsar | PluckPulsarPreset | Tonal pluck: a struck pulse whose formant falls fast from high, like a string losing its brightness |
| harpPulsar | HarpPulsarPreset | Harp: narrow Gaussian pulses with a second formant an octave and a half up, ringing long and the notes spread over the stereo field |
| kalimba | KalimbaPreset | Thumb piano: soft triangle pulsarets, a woody tine with a short ring |

### Basses

| Name | Function | Sound |
|---|---|---|
| squareBass | SquareBassPreset | Hollow square bass: odd harmonics inside a formant that follows the pitch, a low pass that snaps |
| reeseBass | ReeseBassPreset | Thick, moving bass: a saw in a formant a little slow of the pitch, the vibrato wide and slow so the harmonics beat |
| acidPulsar | AcidPulsarPreset | Acid line: saw pulses through a resonant ladder snapping shut at every note, the notes sliding into each other |

### Drones

| Name | Function | Sound |
|---|---|---|
| deepDrone | DeepDronePreset | Drone in the cellar: a formant far under the pitch, the notes spread, a very slow swell of the formant |
| beatingDrone | BeatingDronePreset | Drone of two formants a hair apart, which beat against each other slowly; the vibrato is slow and shallow |
| ghostDrone | GhostDronePreset | Hollow drone with the breath of a ghost: some pulses gone, a band pass that sweeps slowly by itself |

### Noise sweeps

| Name | Function | Sound |
|---|---|---|
| noiseSweep | NoiseSweepPreset | Riser: a pulse train torn to noise by a full jitter, the formant climbing four octaves over three seconds |
| windGust | WindGustPreset | Wind: noisy pulses through a resonant band pass, the formant and the pass swaying about in slow, unrelated cycles |

### Buzzing leads

| Name | Function | Sound |
|---|---|---|
| hornet | HornetPreset | Buzzing lead: rectangular pulses in a bright formant that follows the notes, a quick vibrato like wings |
| sawLead | SawLeadPreset | Classic saw lead out of pulses: all the harmonics, a formant opening at the start, a touch of slide |
| zapLead | ZapLeadPreset | Lead with a zap in it: tapered pulses and a formant swung fast and wide by its LFO, pitch and wheel sending it further |

## shaper

The chains of every preset are in the module's side panel in the patcher (see [synths](synths.md#shaper)). `go run ./cmd/examples/presets -synth shaper` plays all of them, two bars each. Every preset is held to the same measures by a test: no clipping, as loud as the others (a sustained sound at about -15 dB rms, a short one peaking near 0.5), under -18 dB of its energy above 6 kHz (the effects excepted), silent within 12 seconds of the key being let go, and not like any other preset.

### Basses

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

### Leads

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

### Pads

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

### Keys and plucks

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

### Bells

| Name | Function | Sound |
|---|---|---|
| bellMorph | BellMorph | BellMorph strikes a bell and lets it settle into a plain sine |
| glassBell | GlassBell | GlassBell is phase modulation at three times the pitch, falling to a pure tone: a struck glass |
| churchBell | ChurchBell | ChurchBell is partials that die one after the other: the high ones in a moment, the low ring on |
| gong | Gong | Gong is a swell of a very bright, very low sound that settles over many seconds |
| metal | Metal | Metal stacks whole number partials, brightening and fading: a struck pipe |
| glass | Glass | Glass is a sine that grows overtones as the note starts, folding into a glassy ring |

### Drones and textures

| Name | Function | Sound |
|---|---|---|
| drone5 | Drone5 | Drone5 is a note and its fifth, folded softly, breathing: a drone to sit under a piece |
| subDrone | SubDrone | SubDrone is a very low sine folded slowly into a growl and back |
| crush | Crush | Crush is a saw cut into steps, fewer of them as the macro goes up |

### Effects

| Name | Function | Sound |
|---|---|---|
| laser | Laser | Laser falls from far above the note in a moment, a hard synced saw |
| riser | Riser | Riser climbs for a long note: pitch, brightness and fold all going up, for the bars before a change |
| impact | Impact | Impact is a low hit: a fold falling from high to low and a sine thump under it, ringing out |
| zap | Zap | Zap is a short fall in pitch of an FM tone: a game sound |

### Cross modulation

| Name | Function | Sound |
|---|---|---|
| xmodBell | XModBell | XModBell is an FM bell: a silent line at 3.5 times the pitch bends the phase of a sine, and the bend dies with the note, so the strike is bright and metallic and the ring is a clean tone |
| xmodGrowl | XModGrowl | XModGrowl is a bass from a saw-like carrier bent by a line an octave under it, the bend breathing on a slow LFO: the growl of a talking bass |
| ringMetal | RingMetal | RingMetal is two lines a tritone apart multiplied together: the sum and the difference of their partials, a clang that belongs to no pitch |

## dx

### Electric pianos

| Name | Function | Sound |
|---|---|---|
| tineKeys | TineKeys | TineKeys is an electric piano of a tine: a bright ratio 14 strike that dies in a moment over a warm body |
| fmPiano | FMPiano | FMPiano is the electric piano of the factory banks: carriers of the same pitch, their modulators at ratio 1 that bark and settle, a high strike on top |
| softPiano | SoftPiano | SoftPiano is a mellow piano of two chains, soft in the attack and long in the decay |
| dynoKeys | DynoKeys | DynoKeys is a Rhodes of the 70s with bark: an octave modulator with a hard velocity and the feedback loop of algorithm 6 |
| wurlyKeys | WurlyKeys | WurlyKeys is a reed piano: a sustaining body, a fierce feedback and a soft tremolo |
| brightGrand | BrightGrand | BrightGrand is a piano of four carriers in algorithm 21: hammer partials above the string, strong key scaling |

### Basses

| Name | Function | Sound |
|---|---|---|
| slapFM | SlapFM | SlapFM is a slapped bass: the pop of a fast modulator over a round body in algorithm 16 |
| synthBass | SynthBass | SynthBass is a saw like bass: a heavy feedback modulator over a sub octave pair (algorithm 1) |
| fretlessFM | FretlessFM | FretlessFM is a fretless bass: a slow growl as the modulators settle, a little vibrato |
| fmBass | FMBass | FMBass is the DX7 bass: a sub carrier under a stack that clacks and settles (algorithm 10) |
| acidBass | AcidBass | AcidBass is a resonant bass: a log saw carrier through the 24 dB ladder whose envelope snaps shut |
| rubberBass | RubberBass | RubberBass is a rubbery bass: a pitch envelope that snaps down into the note on a sine and a square pair |
| subBass | SubBass | SubBass is a low sine with just a breath of feedback, an octave down |

### Brass

| Name | Function | Sound |
|---|---|---|
| brassSection | BrassSection | BrassSection is a brass section: a modulator that swells over four detuned carriers (algorithm 23) |
| soloTrumpet | SoloTrumpet | SoloTrumpet is a trumpet: a bright blare that breaks in with the attack, then a vibrato that comes in late |
| hornSolo | HornSolo | HornSolo is a horn: mellow, a slow attack, and the modulator on a lower level |
| synthBrass | SynthBrass | SynthBrass is an analogue like brass: a seven feedback modulator on a stack of five detuned carriers (algorithm 24) |
| sforzandoBrass | SforzandoBrass | SforzandoBrass is a stab: the modulators open wide at the start and close again, harder when played hard |

### Strings and pads

| Name | Function | Sound |
|---|---|---|
| warmStrings | WarmStrings | WarmStrings is a warm string section: slow attacks, detuned carriers over a soft modulator (algorithm 26) |
| glassPad | GlassPad | GlassPad is a pad of odd partials: ratios 1, 3 and 5 as pairs, a slow swell and a high shimmer (algorithm 6) |
| icePad | IcePad | IcePad is a cold pad: carriers at 1, 2 and the top (algorithm 28) with high inharmonic modulators |
| evolvingPad | EvolvingPad | EvolvingPad is a pad that changes: modulators that swell and fade at different speeds, a slow LFO |
| choirPad | ChoirPad | ChoirPad is a choir of "ah": detuned voices with a formant like pair on two ratios (algorithm 20) |
| sweepStrings | SweepStrings | SweepStrings is a string machine: six detuned saw and pulse carriers (algorithm 32) through a low pass that opens and closes |

### Mallets

| Name | Function | Sound |
|---|---|---|
| woodMarimba | WoodMarimba | WoodMarimba is a marimba: a sine with a short ratio 4 knock, and a sub octave warmth (algorithm 5) |
| vibesFM | VibesFM | VibesFM is a vibraphone: a long ring with a ratio 4 strike and a motor tremolo (algorithm 5) |
| thumbKalimba | ThumbKalimba | ThumbKalimba is a thumb piano: a hollow tone from the even sine, a tick of an inharmonic partial |
| steelCelesta | SteelCelesta | SteelCelesta is a celesta an octave up: soft bell partials, no sustain (algorithm 5) |
| crystalGlock | CrystalGlock | CrystalGlock is a glockenspiel: ratio 3.5 strikes, a ring that lasts (algorithm 31) |
| chanceMallet | ChanceMallet | ChanceMallet is a marimba whose strike changes with every note: the matrix moves the knock by a random value |

### Bells

| Name | Function | Sound |
|---|---|---|
| tubeChime | TubeChime | TubeChime is a tubular bell: strike partials of 1, 2.76 and 5.4 that ring for seconds (algorithm 5) |
| churchBell | ChurchBell | ChurchBell is a big bell: a low hum, the minor third and a strike of inharmonic partials (algorithm 1) |
| musicBoxFM | MusicBoxFM | MusicBoxFM is a music box: the tines of a comb, high and plain with a faint ratio 7 sparkle |
| gamelanBell | GamelanBell | GamelanBell is a gamelan gong chime: pairs a few cents apart beat slowly (algorithm 4) |
| bellShimmer | BellShimmer | BellShimmer is a pad of bells: slow attacks of inharmonic partials with an LFO that makes them shimmer (algorithm 10) |

### Plucks

| Name | Function | Sound |
|---|---|---|
| harpFM | HarpFM | HarpFM is a harp: a soft strike, a ring and a rising pitch at the very start (algorithm 5) |
| nylonPick | NylonPick | NylonPick is a nylon string guitar: the body of the first partials and a soft pick (algorithm 2) |
| kotoPluck | KotoPluck | KotoPluck is a koto: a bright pluck that bends down into the pitch, with a bite of the square wave (algorithm 4) |
| banjoTwang | BanjoTwang | BanjoTwang is a banjo: a bright, tinny head, fast to die away (algorithm 3) |
| baroqueHarpsi | BaroqueHarpsi | BaroqueHarpsi is a harpsichord: plucked plectra of every harmonic, with a release click of the jack (algorithm 4) |
| funkClav | FunkClav | FunkClav is a clavinet: a rasp of the feedback and a pickup that is hit hard, short (algorithm 5) |

### Organs

| Name | Function | Sound |
|---|---|---|
| fmOrgan | FMOrgan | FMOrgan is an organ of six drawbars at 1/2, 1, 2, 3, 4 and 6 times the note (algorithm 32) |
| pipeOrgan | PipeOrgan | PipeOrgan is a pipe organ: sines with a chiff on the breath of the first moment (algorithm 31) |
| drawbarOrgan | DrawbarOrgan | DrawbarOrgan is a tonewheel organ: three drawbars with the percussion of the third harmonic (algorithm 29) |
| reedOrgan | ReedOrgan | ReedOrgan is a reed organ, the harmonium: the square wave of the TX81Z and a soft attack |

### Winds

| Name | Function | Sound |
|---|---|---|
| airFlute | AirFlute | AirFlute is a flute: a sine with a breathy edge from a feedback modulator that comes in with the attack (algorithm 3) |
| woodClarinet | WoodClarinet | WoodClarinet is a clarinet: odd harmonics from the square wave, mellowed by a low modulator |
| nasalOboe | NasalOboe | NasalOboe is an oboe: a narrow, reedy tone with partials around 3 and 4 (algorithm 14) |
| tenorSax | TenorSax | TenorSax is a tenor saxophone: a throaty tone with a growl as the feedback modulator is hit (algorithm 3) |
| panPipes | PanPipes | PanPipes is a pan flute: sines with a puff of high noise like breath at the start (algorithm 32) |

### Leads

| Name | Function | Sound |
|---|---|---|
| syncLead | SyncLead | SyncLead is a hard sync like lead: the log saw wave with a modulator that gives the tearing sweep (algorithm 1) |
| pwmLead | PWMLead | PWMLead is a pulse width lead: two squares a few cents apart beat to make the width move (algorithm 32) |
| pureSine | PureSine | PureSine is a sine lead, only a vibrato that comes in late |
| fmLead | FMLead | FMLead is an FM lead with a clean attack: a bell like bite that fades into a singing line (algorithm 5) |
| glideLead | GlideLead | GlideLead is a mono lead that slides from note to note, with a vibrato of its own |

### Percussion

| Name | Function | Sound |
|---|---|---|
| tomFM | TomFM | TomFM is a tom: a body that falls in pitch, a click of the stick (algorithm 1) |
| kickFM | KickFM | KickFM is a kick drum: a sine that drops fast from a high start with a click on top (algorithm 3) |
| timpaniFM | TimpaniFM | TimpaniFM is a timpani: a harmonic series from 1 to 2.1 that fades at different speeds and a pitch that settles (algorithm 5) |
| snareFM | SnareFM | SnareFM is a snare: a body and noise like bursts, a high modulator fed back on itself (algorithm 4) |
| hatFM | HatFM | HatFM is a hi hat: inharmonic square waves at fixed frequencies, a short burst (algorithm 32) |
| woodBlockFM | WoodBlockFM | WoodBlockFM is a wood block: a short dry knock of two inharmonic partials (algorithm 5) |

### Effects

| Name | Function | Sound |
|---|---|---|
| riserSweep | RiserSweep | RiserSweep is a riser: a pitch that climbs over seconds and a filter that opens |
| laserZap | LaserZap | LaserZap is a laser: a very high pitch that falls through the sound in a flash (algorithm 5) |
| windGust | WindGust | WindGust is wind: the noise of a high feedback modulator through a band pass that moves at random |
| vowelAh | VowelAh | VowelAh is a vowel: a modulator at the pitch of the note drives carriers fixed at the formants of "ah" (algorithm 22) |
| alienBubbles | AlienBubbles | AlienBubbles is a bubbling texture: a sample and hold LFO moves the pitch in steps over a metal bell (algorithm 15) |

### The waves of the tx81z

| Name | Function | Sound |
|---|---|---|
| hollowReed | HollowReed | HollowReed is a hollow reed: the pulse and even waves give the nasal odd partial colour (algorithm 11) |
| logSawBrass | LogSawBrass | LogSawBrass is a brass of the log saw: it bites, and its modulators swell with the attack (algorithm 22) |
| halfSineOrgan | HalfSineOrgan | HalfSineOrgan is an organ of the half sine: a wire like tone with even partials only |
| evenAbsPad | EvenAbsPad | EvenAbsPad is a pad of the double speed waves: sounds an octave above the note with a soft grain |
| pulseClav | PulseClav | PulseClav is a plucked pulse: the pulse sine wave on a short envelope, a funk riff voice |

### The matrix and the filter

| Name | Function | Sound |
|---|---|---|
| wheelKeys | WheelKeys | WheelKeys is an electric piano that the mod wheel brightens: it opens the modulators and adds a vibrato |
| velocityFM | VelocityFM | VelocityFM is a piano where velocity opens the modulators: soft notes are plain, hard notes ring with bells |
| filterPluck | FilterPluck | FilterPluck is a synthetic pluck: a saw through a resonant low pass whose envelope falls, brighter on hard notes |
| wheelPad | WheelPad | WheelPad is a pad whose filter the mod wheel opens, and a slow LFO that breathes on it |

## organ

### Soul, jazz and gospel

| Name | Function | Sound |
|---|---|---|
| jimmySmith | JimmySmith | 884002000 with a long third harmonic percussion, a little drive and the Leslie turning up to the chorale: the soul jazz trio organ |
| funkyStab | FunkyStab | 8', 4' and 2 2/3' with the fast second percussion and a lot of click, dry: a clipped, clavinet like chop |
| gospelFast | GospelFast | 888800000 with the deep chorus into a Leslie at tremolo: the church band at full gallop |
| gospelSlow | GospelSlow | 886000000 on the light chorus and the Leslie at its chorale: a sweet, round shout |
| bluesGrit | BluesGrit | 868000000 with a bit of drive, the light chorus and a slow Leslie: a smoky club organ |
| ballpark | Ballpark | 838380000 with the fast second percussion and the tremolo Leslie: the stadium organ |

### Rock and overdrive

| Name | Function | Sound |
|---|---|---|
| rockOverdrive | RockOverdrive | 838808000 into a preamp driven hard and bright, the Leslie at tremolo, all the click |

### Solo, flute and reed voices

| Name | Function | Sound |
|---|---|---|
| brightSolo | BrightSolo | High drawbars of a lead line, 008888000 with the second percussion and a bright preamp |
| softFlute | SoftFlute | 8', 4' and 2' very softly, 004840000, with the vibrato of a flute stop |
| nasalReed | NasalReed | 008060606: the odd mixtures over the 8', a hollow, oboe like reed |
| pipeFlute | PipeFlute | Single 4', through a low pass: a hollow wooden pipe |
| glassOrgan | GlassOrgan | Only the highest drawbars, 000008806: a glassy, whistling organ |
| stringOrgan | StringOrgan | Keen high drawbars of an organ string stop, 003003868, with the chorus |

### Church, theatre and pads

| Name | Function | Sound |
|---|---|---|
| softPad | SoftPad | Slow pad, 808000000, in a chorale Leslie |
| theatreTibia | TheatreTibia | Heavy 8' tibia with the 16' and a deep slow vibrato: the cinema organ |
| churchPlenum | ChurchPlenum | Whole chorus of a church organ, the 8' and 4' strong, the mixtures filling in, dry |
| principalChoir | PrincipalChoir | Principals, 004865432: the 4' leading with no 16' under it |
| diapason | Diapason | Foundation of a church organ, 006866420: round, full and without the 16' |
| cathedral | Cathedral | Low, dark registration, 863000600, a long way off: the highs rolled away |
| mellowFull | MellowFull | Every drawbar out, through a low pass: all the weight and none of the edge |

### Scanner and Leslie variants

| Name | Function | Sound |
|---|---|---|
| chorusV1 | ChorusV1 | 338600000 with the lightest vibrato, V1, and no Leslie: the plain scanner |
| warmChorus | WarmChorus | 848000000 on the middle chorus, C2, straight into the amp |
| deepVibrato | DeepVibrato | 008060000 on the deepest vibrato, V3 doubled, without the dry sound |
| leslieStopped | LeslieStopped | 884400000 with the rotors parked: the cabinet's tone with no movement |
| leslieFast | LeslieFast | 888800000 with the rotors at tremolo, the microphones far apart |

### Bass and character

| Name | Function | Sound |
|---|---|---|
| subBass | SubBass | 16' and the 5 1/3' alone, 870000000, dry and with plenty of click |
| carnival | Carnival | Fairground organ, 008806000 on the fast vibrato at 8 Hz, a little bright |
| horror | Horror | Drooping 808000600 with the scanner at 3 Hz, deep: a haunted house organ |

### Matrix and filter

| Name | Function | Sound |
|---|---|---|
| wheelLeslie | WheelLeslie | 888800000 in the Leslie: the mod wheel speeds the rotors from the chorale up to the tremolo |
| wheelDrive | WheelDrive | 808800000 clean until the mod wheel pushes the preamp into saturation |
| wahOrgan | WahOrgan | Sweeps a resonant low pass over 888600000 with a slow LFO: a wah organ |
| swellFilter | SwellFilter | Opens a low pass over 888888888 with the first key of a phrase, like a swell pedal |

## waveguide

| Name | Function | Sound |
|---|---|---|
| nylonClassical | NylonClassical | A classical guitar played with the fingertips of the right hand: round and warm, the place of the pluck changing a little from note to note and the harder touch brighter |
| steelAcoustic | SteelAcoustic | A steel string flat top played with the fingers (fingerpicking): the bright ring of the wire without the click of a pick |
| twelveString | TwelveString | A guitar with its strings in pairs and octaves: the two of a pair never quite in unison (beats), the octave strings sounding along |
| palmMuted | PalmMuted | An electric guitar chugged with the heel of the hand on the strings at the bridge: a thick, short thud with a pick attack |
| jazzArchtop | JazzArchtop | A hollow body guitar with flatwound strings, played with the thumb: dark, round and smooth, little sustain |
| electricPick | ElectricPick | A solid body electric guitar through a pickup: picked, long and clear, with a little twang on a hard note; the pickup is a low pass, the touch opens it |
| resonator | Resonator | A dobro: a steel guitar with a metal cone in its body, picked and slid. A tinny, nasal ring, and the notes glide into each other on a mono voice |
| pinchHarmonic | PinchHarmonic | A pick that brushes the string with the thumb: an artificial harmonic squealing two octaves above the string's note, on a node a quarter of the string away |
| twelfthHarmonic | TwelfthHarmonic | The harmonic at the seventh fret of a nylon string: a soft bell a twelfth above the open string, a finger laid on a third of its length |

| Name | Function | Sound |
|---|---|---|
| slapBass | SlapBass | An electric bass slapped with the thumb and popped by the finger: bright, with a click, the string slapping the frets |
| fretlessBass | FretlessBass | A bass without frets: the string sings against the fingerboard, the "mwah" of a buzz that comes with the note, and a slide from note to note on a mono voice |
| pickBass | PickBass | A bass played with a plectrum: a hard attack and a clear, even sustain |
| uprightWalking | UprightWalking | An upright bass in a walking line: the finger pulls the thick gut string, a woody note that fades before the next |

| Name | Function | Sound |
|---|---|---|
| harpHarmonics | HarpHarmonics | Are the harmonics of a harp: a string stopped in the middle by the palm and plucked, the octave ringing like a small bell |
| celticHarp | CelticHarp | A small harp with brass strings plucked with the nails: bright, bell like, ringing on |
| banjo | Banjo | A steel string over a drum head: short and bright, with a metallic ring and the pluck right at the bridge |
| mandolin | Mandolin | Plucked with a plectrum on pairs of steel strings that are a little out of tune with each other: bright, quickly fading, shimmering |
| mandolinTremolo | MandolinTremolo | The tremolo of the mandolin: the plectrum going back and forth so fast that the notes join, a string kept sounding, 12 times a second |
| ukulele | Ukulele | A small nylon guitar strummed by the thumb: soft, plunky, quickly gone |
| sitar | Sitar | Has a flat bridge that the string is lifted from and falls back on: the string buzzes against it with every swing (the jawari), a plectrum on the main string, the drone strings sounding along |
| koto | Koto | A long zither of silk strings on movable bridges, plucked with a plectrum on the thumb and bent after the pluck by pressing the string behind the bridge |
| oud | Oud | The fretless lute of the Middle East: paired gut strings plucked with a long plectrum (risha), a deep round bowl |
| lute | Lute | A renaissance lute: courses of gut strings plucked with the fingers, the soft, woody sound of the back of its large bowl |
| dulcimer | Dulcimer | A hammered dulcimer: courses of wires struck with light hammers, every note shimmering and the open strings sounding along |
| zither | Zither | A flat box of strings plucked with a ring on the thumb, ringing into each other |
| harpsichord | Harpsichord | Has a quill in a jack that plucks the string as the key goes down and a damper that stops it as it goes up: bright, the same loudness whatever the touch |
| clavinet | Clavinet | A clavichord for funk: a string pressed onto an anvil and picked up electrically, a bright, stinging, tight note |
| pianoString | PianoString | A piano string alone, struck by a felt hammer an eighth of the way along: stiff partials a little sharp, a long ring, the soundboard under it |
| snapPizzicato | SnapPizzicato | A Bartok pizzicato on a cello: the string pulled up and let go so that it snaps against the fingerboard |
| colLegno | ColLegno | The wood of the bow struck on a violin string: a dry, short, clicking tone, hardly any ring |

| Name | Function | Sound |
|---|---|---|
| violinSolo | ViolinSolo | A solo violin with a vibrato the wheel deepens and a harder bow with the velocity |
| violinSection | ViolinSection | A section of violins played together: a slower start, a smaller and later vibrato, the grain of the rosin of many bows |
| viola | Viola | A viola: the string longer and thicker, a darker, reedier bow, the vibrato slower |
| contrabass | Contrabass | A double bass bowed: slow to speak, big and woody, a heavy bow |
| legatoViolin | LegatoViolin | A violin that changes the note without a new bow: on a mono voice the next note glides in |
| staccatoViolin | StaccatoViolin | The bow stopped on the string after each note: it speaks at once and stops short |
| spiccato | Spiccato | A bow that bounces: a sharp speech, a dry rub, gone as soon as it comes |
| tremoloStrings | TremoloStrings | Are the bow moved back and forth so fast that the sound shivers: the loudness swinging 10 times a second |
| hurdyGurdy | HurdyGurdy | A rosined wheel turned against the strings: a steady, nasal drone that rattles, a buzzing bridge, the drone strings sounding along |
| erhu | Erhu | The Chinese two string fiddle: a thin, nasal, vocal tone with a vibrato that comes in late |
| rebab | Rebab | A bowed lute with a skin for a belly: a rough, breathy and nasal fiddle |
| bowedGlass | BowedGlass | A string that sings like a rubbed glass: a light bow far from the bridge, a slow swell and a slow fade, a shimmer of strings tuned to its octaves |
| bowedPsaltery | BowedPsaltery | A psaltery bowed on the edge: a small triangle of strings, thin and sweet |
| celloDrone | CelloDrone | The open string of a cello bowed with a slow, steady stroke: a long, deep drone |
| singingSaw | SingingSaw | A saw bowed on its blade: a thin, wavering tone that slides from note to note |

| Name | Function | Sound |
|---|---|---|
| kickDrum | KickDrum | A large drum struck hard: the pitch drops fast after the hit, a short, deep thump |
| tomHigh | TomHigh | A small rack tom: a short, round and ringing tone |
| tomLow | TomLow | A mid tom, a stick on a loose head: long and dark |
| congaOpen | CongaOpen | The open tone of a conga: the hand bounces off the rim, a clear ring |
| congaSlap | CongaSlap | The slap: the fingers whipped across the head, bright and short |
| congaBass | CongaBass | The bass tone: the palm in the middle of the head, a dull thud |
| bongo | Bongo | A pair of small drums on the knees: high, dry and tight |
| djembe | Djembe | A goblet drum struck by the hand: a deep bass tone with a bright crack |
| bayan | Bayan | The left drum of the tabla: a large head that bends far upwards under the heel of the hand |
| dayan | Dayan | The right drum of the tabla: high, with the black spot of the head that gives a clear, ringing tone |
| bodhran | Bodhran | The Irish goatskin frame drum struck by a tipper: a hollow, dry knock |
| taiko | Taiko | A huge Japanese drum with a thick skin struck by heavy sticks: it booms |
| snareTight | SnareTight | A small snare tuned high, rimmed and hit hard: a crack |
| snareFat | SnareFat | A deep snare with loose wires: a thick, long backbeat |
| snareBrush | SnareBrush | A snare swept with brushes: a soft hit and a hush of the wires |
| timbale | Timbale | A shallow metal-shelled drum hit with a stick: tight, high and ringing |
| cajon | Cajon | A wooden box sat on and hit by the hand: a bass thud with a snare-like slap |
| tambourine | Tambourine | A frame with a head and jingles: a short slap and a shimmer of metal |

| Name | Function | Sound |
|---|---|---|
| altoFlute | AltoFlute | A flute in G, a fourth lower: wide, dark and airy, soft to start |
| piccolo | Piccolo | The small flute that plays an octave over the flute: bright, thin, piercing |
| shakuhachi | Shakuhachi | The Japanese bamboo flute with no reed or lip plate: very much air, a slow head shake for a vibrato and a note that swells |
| ocarina | Ocarina | A vessel flute of clay: a pure, hooting tone with hardly any overtones |
| recorder | Recorder | A block flute: the air is led onto the edge by a duct so that it speaks at once and the breath hardly changes the tone |
| bassClarinet | BassClarinet | A clarinet an octave below: a deep, dark and woody reed |
| chalumeau | Chalumeau | The low register of the clarinet: hollow, rich and woody, a little rough |
| breathyFlute | BreathyFlute | A flute blown with more air than tone: the sound of a whisper with a pitch to it |
| overblownFlute | OverblownFlute | A flute pushed to the edge of the next register: loud, edgy and unstable |
| whistle | Whistle | A pure whistled tone with a wide vibrato |
| bansuri | Bansuri | The Indian bamboo flute: warm and breathy, with a slow swell |
| tinWhistle | TinWhistle | The penny whistle of Irish music: a bright and reedy, slightly raw tone |

## modal

### Marimba and xylophone

| Name | Function | Sound |
|---|---|---|
| softMarimba | SoftMarimba | Marimba played with soft yarn mallets, heavy and slow: round, almost without attack |
| hardMarimba | HardMarimba | Marimba struck with hard rubber, near the end of the bar: clacking, bright, short |
| marimbaEdge | MarimbaEdge | Marimba struck almost at its end, where the fundamental hardly sounds: thin and woody |
| xyloWarm | XyloWarm | Rosewood xylophone with medium mallets, a little undercut: mellow, a short ring |
| toyXylophone | ToyXylophone | Child's xylophone: hard, thin, clicking |

### Vibraphone

| Name | Function | Sound |
|---|---|---|
| bowedVibraphone | BowedVibraphone | Vibraphone bar drawn with a bow: the note swells in and rings as long as it is held |
| vibeMotorFast | VibeMotorFast | Vibraphone with the motor at full speed, a deep fast tremolo |
| vibeDamped | VibeDamped | Vibraphone with the pedal down hard, hard mallets: a short, dry, bright note |
| vibeBell | VibeBell | Vibraphone struck like a bell: long modes in a stretched stack, the tremolo slow |
| wheelVibe | WheelVibe | Vibraphone whose motor the mod wheel turns on: the wheel brings in the tremolo and hardens the mallet |

### Mallet-bell hybrids and tines

| Name | Function | Sound |
|---|---|---|
| bellMarimba | BellMarimba | Mallet bell hybrid: the marimba bar's 1 : 4 : 10 with the partials of a bar bell above, struck with a medium mallet |
| musicBox | MusicBox | Comb of steel tines plucked by a pin: high, thin, ringing for a few seconds |
| kalimbaSoft | KalimbaSoft | Thumb piano played with the pad of the thumb, near the base of the tine: dull and woody |

### Bells and chimes

| Name | Function | Sound |
|---|---|---|
| handbellSoft | HandbellSoft | Handbell with a soft clapper |
| handbellChoir | HandbellChoir | Handbells with the pairs split wide, ringing together: a shimmering, beating chime |
| templeBell | TempleBell | Big bronze bell struck by a log: a very long hum under the strike, the pairs beating slowly |
| carillon | Carillon | Small bells of a tower: the church bell tuned high, hard and quick to die |
| tubularSoft | TubularSoft | Chimes struck with a leather hammer: the strike tone mellow, ringing and ringing |
| doorChime | DoorChime | Short tube chime: hard struck, a quick two second ring |
| keyedBell | KeyedBell | Tubular bell whose ring is shorter the higher the note, the key moving where it is struck |

### Gamelan

| Name | Function | Sound |
|---|---|---|
| gamelanSaron | GamelanSaron | Bronze bar with a hard beater, the pairs tuned to beat: bright and shimmering |
| gamelanGender | GamelanGender | Gender: thin keys over resonators, a soft disc beater, a low, humming bloom |

### Gongs

| Name | Function | Sound |
|---|---|---|
| gongAgeng | GongAgeng | Great gong ageng: huge, low, a soft beater and a long bloom |
| tamTam | TamTam | Orchestral tam-tam hit hard: it roars up from the low modes into a wash |
| gongSoft | GongSoft | Small gong tapped with a soft mallet: a low gong note and little else |
| bowedGong | BowedGong | Gong drawn with a bow on its edge: a slow swell of metal |

### Bowls, glass and bottles

| Name | Function | Sound |
|---|---|---|
| tunedBowlHigh | TunedBowlHigh | Small brass bowl struck sharply on the rim: clear and high, ringing a long time |
| tunedBowlDeep | TunedBowlDeep | Big bowl hit with a padded striker on its side: low, warm, slowly beating |
| bowedBowl | BowedBowl | Singing bowl sounded by rubbing the rim: it swells and sings as long as it is held |
| wobbleBowl | WobbleBowl | Singing bowl whose pitch the slow LFO sways, a few cents: a bowl on a rocking boat |
| glassHarmonica | GlassHarmonica | Wet glass bowls rubbed with a finger: pure, high, floating |
| bottleBow | BottleBow | Bottle sounded by a bow on the glass: the resonance of the body swells and trembles |
| bottleTap | BottleTap | Bottle tapped with a knuckle, soft: a dull, hollow plonk |

### Steel pans

| Name | Function | Sound |
|---|---|---|
| steelPanSoft | SteelPanSoft | Steel pan played with rubber-tipped sticks, rolling in a slow shimmer |
| steelPanBright | SteelPanBright | Steel pan hit hard near the rim of the note: bright and short |

### Metallic percussion and cymbals

| Name | Function | Sound |
|---|---|---|
| triangle | Triangle | Steel triangle, bright and inharmonic, shimmering at the top |
| anvil | Anvil | Steel plate hit with a hammer: a hard, ringing clang |
| crashCymbal | CrashCymbal | Crash: struck hard, the wash blooming and fading over four seconds |
| rideCymbal | RideCymbal | Ride struck on the bow with a stick: a ping over a soft, steady wash |
| hihatClosed | HihatClosed | Pair of cymbals pressed together: a tick, a breath of hiss |
| hihatOpen | HihatOpen | Same pair loose: a sizzle that rings a second |
| brushedCymbal | BrushedCymbal | Cymbal swept with a brush: a burst of noise into the modes, a soft swish |
| thunderSheet | ThunderSheet | Sheet of steel shaken: a burst of noise into a plate, rumbling |

### Wood

| Name | Function | Sound |
|---|---|---|
| claves | Claves | Are two sticks of hardwood knocked together: a bright, dry tock |
| slitDrum | SlitDrum | Hollow log struck with a padded stick: a deep, round knock |

## vocal

### Solo voices

| Name | Function | Sound |
|---|---|---|
| bassAh | BassAh | One bass singing "a", a natural vibrato late in coming |
| bassOo | BassOo | One bass humming a closed "u", dark and close |
| tenorOh | TenorOh | One tenor on a round "o", an LF source |
| tenorEe | TenorEe | One tenor on a bright "i", pressed |
| altoAh | AltoAh | One alto on "a", warm, with the singer's formant |
| altoEh | AltoEh | One alto on an open "e", a little breath |
| sopranoAh | SopranoAh | One soprano on "a", the first formant following the pitch up: the open head voice |
| sopranoEe | SopranoEe | One soprano on a thin, high "i" |
| childOo | ChildOo | Child singing "u": small, pure, almost without edge |
| childAh | ChildAh | Child shouting "a": bright, open, the voice a little pressed |

### Choirs

| Name | Function | Sound |
|---|---|---|
| bassChoir | BassChoir | Six basses on "o", slow in and slow out |
| tenorChoir | TenorChoir | Five tenors on "e" with an LF source, forward and wide |
| altoChoir | AltoChoir | Eight altos on "e": a thick middle voice |
| sopranoChoir | SopranoChoir | Ten sopranos on "i" with the singer's formant: a high, silvery section |
| childrenChoir | ChildrenChoir | Twelve children on "u", pure and a little out of tune |
| gospelChoir | GospelChoir | Six voices wide open on "a", bright and loud, a fast shouting vibrato |

### Humming and whisper

| Name | Function | Sound |
|---|---|---|
| humming | Humming | One voice with the lips closed: the nose open, an "m" with a pitch |
| hummingChoir | HummingChoir | Eight voices humming together, close and warm |
| whisperSoft | WhisperSoft | Quiet whisper of an "a": air and no pitch |
| whisperBreath | WhisperBreath | Half whispered "o": the voice under a sigh of air |

### Words and syllables

| Name | Function | Sound |
|---|---|---|
| chantMantra | ChantMantra | Sings "mo ma ni pa me u" one syllable a note, low and even |
| kyrie | Kyrie | Sings "ki ri e e le i so ne" slowly, a tenor on the beat |
| spokenWords | SpokenWords | Talks "ne lo wa ta ma ni": an alto speaking on the beat, no vibrato, no song |
| syllableStab | SyllableStab | Short "ta" stabbed on the note: a tenor, a hard attack, gone at once |
| consonantStab | ConsonantStab | Choir spitting "ke po": hard stops before an "o", short |
| liquidLa | LiquidLa | Sings "la le li lo lu ra re": liquids and glides moving into their vowels, one a note |

### Trained and pop voices

| Name | Function | Sound |
|---|---|---|
| operaTenor | OperaTenor | Trained tenor: the singer's formant, a wide natural vibrato, a full LF source |
| operaSoprano | OperaSoprano | Trained soprano whose first formant follows the note: an open, ringing "a" with a natural vibrato |
| operaBass | OperaBass | Trained bass, "o" ringing in the singer's formant, a slow wide vibrato |
| popBreathy | PopBreathy | Close, intimate pop voice: an LF source with a high Rd and a lot of air |
| popBright | PopBright | Forward, pressed pop voice, an LF source at a low Rd |

### Drones and pads

| Name | Function | Sound |
|---|---|---|
| monkDrone | MonkDrone | Four monks on a closed "o", a pitch held still for a long time |
| throatLow | ThroatLow | Giant's throat: the formants shifted far down, a rough, pressed source |
| overtoneSweep | OvertoneSweep | Low voice whose vowel sweeps slowly up and down: the overtones whistle out of the drone |
| padWarm | PadWarm | Ten voices on "o" opening slowly to "a": a warm, wide pad |
| padDark | PadDark | Dark pad of low "u" voices, slow |
| padAir | PadAir | Pad of breath: a whisper of "e" voices, wide, with a faint tone under it |

### Diphthongs and matrix

| Name | Function | Sound |
|---|---|---|
| diphthongAi | DiphthongAi | Glides each note from "a" to "i", slowly: a long, open "ai" |
| diphthongOu | DiphthongOu | Glides each note from "o" to "u", a choir closing its mouth |
| wheelVowel | WheelVowel | Sings an "a" whose vowel the mod wheel moves toward "u" through the vowels between |
| wheelWhisper | WheelWhisper | Sung "e" the mod wheel turns into a whisper |

## piano


### Concert grands

| Name | Function | Sound |
|---|---|---|
| steinway | SteinwayPreset | New York grand: brilliant singing treble, strong clear bass with bark, from the hall |
| bosendorfer | BosendorferPreset | Vienna grand: dark, warm, long resonant bass, soft hammers |
| yamaha | YamahaPreset | Clear, even and focused, little bark, clean sustain, studio |
| fazioli | FazioliPreset | Brilliant, glassy, very long ring, shining top octaves |

### Recording styles

| Name | Function | Sound |
|---|---|---|
| closeMic | CloseMicPreset | Microphones inches from the strings: tight, dry, hammer and damper detail |
| roomAmbient | RoomAmbientPreset | Grand in a big room, everything ringing along, blurred attack |
| popCompressed | PopCompressedPreset | Bright, even in level, short and punchy, no room |
| jazzMellow | JazzMellowPreset | Trio piano: round soft hammers, warm midrange, relaxed treble |
| jazzDry | JazzDryPreset | Dry club piano: thick woody midrange, quick dampers, little ring |
| filmWide | FilmWidePreset | Cinema piano: very soft hammers, long room, keys spread over the whole image |

### Uprights

| Name | Function | Sound |
|---|---|---|
| schoolUpright | SchoolUprightPreset | Worn school-room upright: dull, boxy, pleasantly out of tune |
| tack | TackPreset | Tack piano: hard metallic plink, short and bright |
| barPiano | BarPianoPreset | Untuned barroom upright: strings cents apart, worn action, dull and thin |
| jangle | JanglePreset | Wide even detune: a chorus shimmer on bright hammers |
| ragtime | RagtimePreset | Honky ragtime upright: brisk, dry, hard hammers |

### Historic

| Name | Function | Sound |
|---|---|---|
| fortepiano | FortepianoPreset | Mozart fortepiano: leather hammers, thin light strings, nasal, fast damping |
| cristofori | CristoforiPreset | First piano: very hard thin hammers, clipped, plucked, near harpsichord |
| squarePiano | SquarePianoPreset | Small box, single string, soft thin tinkling guitar-like tone |
| harpsichordish | HarpsichordishPreset | Plucked, bright, clipped, same loudness for all velocities |

### Bells and boxes

| Name | Function | Sound |
|---|---|---|
| musicBox | MusicBoxPreset | One thin string hit hard in the middle, hollow, bodiless |
| celesta | CelestaPreset | Soft chime on short strings, fast fade, mellow |
| glassHammer | GlassHammerPreset | Hardest hammer on stiff thin strings: glassy inharmonic shimmer |

### Prepared and lo-fi

| Name | Function | Sound |
|---|---|---|
| muffled | MuffledPreset | Blanket over the strings: highs gone at once, a pitched thud |
| satie | SatiePreset | Intimate quiet piano: soft felt, close mic, breathing mechanics |
| tapePiano | TapePianoPreset | Old tape piano: detuned strings, muffled top, every key different |

### Tuning, stereo, bridge, noise, pedals

| Name | Function | Sound |
|---|---|---|
| stretchedWide | StretchedWidePreset | Tuned wider than a tuner would: sharp treble (about +45 ct at key 100) |
| equalTempered | EqualTemperedPreset | No stretch, near harmonic partials: pure |
| widePiano | WidePianoPreset | Keys spread over the full stereo image |
| monoPiano | MonoPianoPreset | All in the middle |
| bridgeRich | BridgeRichPreset | Strong bridge coupling, full wound bass, precursor and bark: blooming chords |
| bridgeDry | BridgeDryPreset | No bridge coupling, plain strings, no precursor |
| silentAction | SilentActionPreset | No key, damper or pedal noise |
| noisyAction | NoisyActionPreset | Keybed thuds, dampers and pedal up front |
| pedalWash | PedalWashPreset | Sustain pedal always down: very long damper time, full resonance |
| unaCorda | UnaCordaPreset | Soft pedal always down: two strings, softer felt, veiled |
