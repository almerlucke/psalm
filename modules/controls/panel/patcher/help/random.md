# random

A **random modulation source**: every so often it picks a new value between 0 and 1, and its `out` (a **mod** output) goes to it, jumping or gliding. Cable it to a setting for movement that never repeats. For a smooth repeating movement use an [lfo](lfo.md).

## Tutorial

1. Add a random and cable its `out` to a setting of a synth or effect (the inspector lists the settings it can move). The cable's **range** says which part of the setting 0 to 1 covers.
2. **rate** (Hz) or **beats** says how often a new value is drawn. With **beats** above 0 it follows the tempo (1 is a new value every beat) and the rate is ignored.
3. **smooth** says how it gets there: 0 jumps at once (a sample and hold), 1 glides all the way, so the value is always on its way to the next.

```example Steps, a new value every beat
[set]
beats = 1
smooth = 0
```

```example A slow wander
[set]
rate = 0.3
beats = 0      # use the rate
smooth = 1
```

4. Use several, with different rates, on different settings: a filter that drifts while the pan wanders slower, and nothing lines up.

Its values are not repeated from one run to the next.

## Settings

| | |
|---|---|
| rate | Hz, 0.01 to 40 |
| beats | beats between values (0 to 32); 0 uses the rate |
| smooth | 0 jumps, 1 glides all the way |
