#!/usr/bin/env python3
"""Measure the plucked notes of a recording: the reference.csv of this package.

    ffmpeg -i recording.wav -ac 1 -c:a pcm_f32le mono.wav
    python3 -I measure.py mono.wav reference.csv [sample rate [seconds of the notes [floor]]]

reference.csv was made so, with 44100, 180 and 0.003, of a recording of the nylon string guitar that is
modelled: single plucked notes, each let ring out, in the first 180 s of the file (the rest of it, taps on the
body, is left out with the seconds of the notes). The notes that have less than 2.4 s before the next one are
left out too. Per note: the onset, the pitch of the fundamental, and

  drop_dbps  the fall of the level (dB per second) from 0.3-0.5 s to 2.0-2.2 s after the onset
  high_db    the energy above 2 kHz against all energy from 70 Hz up (dB) in a window of 32768 samples (0.74 s
             at 44.1 kHz) from 0.25 s on
  b80 ... b5120  the same for the octave bands 80-160 Hz, 160-320, ... 5120-10240 Hz: the energy of the
             band against all (dB)

  margin_db  the energy above 2 kHz against that of the noise floor of the recording, in dB: near 0 the
             high_db and the bands above 1280 Hz are the hiss of the recording, not the guitar

The noise floor is the mean of the three quietest windows of the recording (it needs a few seconds of silence
in it, at the start or end of the notes). It is taken out of everything above: the power of the noise is
subtracted from the level, and from every bin of the spectrum (smoothed over 65 bins), and what is left is at
least 0.25 (0.3 for a bin) of the noise: a measure below the floor is "at most about the noise", with a margin
near 0. Before it was taken out, a quick decay measured as slower than it was, and the quiet high notes were
as bright as the hiss. Only the standard library is used.
"""
import array, math, cmath, sys

wav, out = sys.argv[1], sys.argv[2]
sr = int(sys.argv[3]) if len(sys.argv) > 3 else 44100
until = float(sys.argv[4]) if len(sys.argv) > 4 else 1e9
level_floor = float(sys.argv[5]) if len(sys.argv) > 5 else 0.0
b = open(wav, 'rb').read()
i = b.find(b'data')
a = array.array('f')
a.frombytes(b[i + 8:][:(len(b) - i - 8) // 4 * 4])

# onsets: a 10 ms frame well above the 100 ms before it
H = sr // 100
env = [math.sqrt(sum(v * v for v in a[k:k + H]) / H) for k in range(0, int(min(len(a), until * sr)) - H, H)]
floor = level_floor or 0.02 * max(env)
on, last = [], -100
for k in range(10, len(env)):
    prev = max(env[k - 10:k - 1]) if k > 10 else 0
    if env[k] > floor and env[k] > 3 * prev + 1e-6 and k - last > 25:
        on.append(k * 0.01)
        last = k


def fft(v):
    n = len(v)
    if n == 1:
        return v
    e, o = fft(v[0::2]), fft(v[1::2])
    t = [cmath.exp(-2j * math.pi * k / n) * o[k] for k in range(n // 2)]
    return [e[k] + t[k] for k in range(n // 2)] + [e[k] - t[k] for k in range(n // 2)]


N = 32768 if sr > 30000 else 16384

# the noise floor: the three quietest windows (not overlapping, within the notes), their power in time and
# per bin of the spectrum
quiet = sorted((sum(v * v for v in a[s0:s0 + N]), s0) for s0 in range(0, int(min(len(a), until * sr)) - N, N))[:3]
noise_t = sum(q for q, _ in quiet) / (3 * N)
hann = [0.5 - 0.5 * math.cos(2 * math.pi * k / N) for k in range(N)]
noise_f = [0.0] * (N // 2)
for _, s0 in quiet:
    F = fft([a[s0 + k] * hann[k] for k in range(N)])
    for k in range(N // 2):
        noise_f[k] += abs(F[k]) ** 2 / 3
# the spectrum of a noise is as rough as the noise: smoothed, it is the floor
SM = 32
run = [0.0]
for x in noise_f:
    run.append(run[-1] + x)
noise_f = [(run[min(N // 2, k + SM + 1)] - run[max(0, k - SM)]) / (min(N // 2, k + SM + 1) - max(0, k - SM)) for k in range(N // 2)]


def level(t0, t1):
    seg = a[int(t0 * sr):int(t1 * sr)]
    p = sum(v * v for v in seg) / max(1, len(seg))
    return 10 * math.log10(max(p - noise_t, 0.25 * noise_t) + 1e-12)


EDGES = [80, 160, 320, 640, 1280, 2560, 5120]
rows = []
for j, t in enumerate(on):
    nxt = on[j + 1] if j + 1 < len(on) else len(a) / sr
    s = int((t + 0.25) * sr)
    if s + N > len(a) or nxt - t < 2.4:
        continue
    w = [0.5 - 0.5 * math.cos(2 * math.pi * k / N) for k in range(N)]
    F = fft([a[s + k] * w[k] for k in range(N)])
    mag = [abs(F[k]) for k in range(N // 2)]
    k0, k1 = int(70 * N / sr), int(1400 * N / sr)
    kp = max(range(k0, k1), key=lambda k: mag[k])
    kp = min((k for k in range(k0, k1) if mag[k] > 0.25 * mag[kp] and mag[k] >= mag[k - 1] and mag[k] >= mag[k + 1]), default=kp)
    al, be, ga = (math.log(mag[kp + d] + 1e-12) for d in (-1, 0, 1))
    f = (kp + 0.5 * (al - ga) / (al - 2 * be + ga)) * sr / N
    m = 69 + 12 * math.log2(f / 440)
    net = [max(mag[k] ** 2 - noise_f[k], 0.3 * noise_f[k]) for k in range(N // 2)]
    top = int(12000 * N / sr)  # the band of the recording
    tot = sum(net[k0:top])
    hi = sum(net[int(2000 * N / sr):top])
    margin = 10 * math.log10(sum(x * x for x in mag[int(2000 * N / sr):top]) / sum(noise_f[int(2000 * N / sr):top]) + 1e-12)
    drop = (level(t + 0.3, t + 0.5) - level(t + 2.0, t + 2.2)) / 1.7
    bands = [10 * math.log10(sum(net[int(lo * N / sr):int(2 * lo * N / sr)]) / tot + 1e-12) for lo in EDGES]
    rows.append((t, f, round(m), (m - round(m)) * 100, level(t + 0.3, t + 0.5), drop, 10 * math.log10(hi / tot + 1e-12)) + tuple(bands) + (margin,))

with open(out, 'w') as fh:
    fh.write('onset_s,hz,pitch,cents,level_db,drop_dbps,high_db,' + ','.join('b%d' % e for e in EDGES) + ',margin_db\n')
    for r in rows:
        fh.write('%.2f,%.2f,%d,%.0f,%.1f,%.2f,%.1f' % r[:7] + ',%.1f' * (len(EDGES) + 1) % r[7:] + '\n')
print(len(rows), 'notes ->', out, '(noise floor %.1f dB rms)' % (10 * math.log10(noise_t)))
