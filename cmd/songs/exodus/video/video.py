#!/usr/bin/env python3
"""A video of the Exodus song: public domain pictures (Doré, Tissot, Roberts, Poussin, fetched by images.py) with
slow movement, the verse as it is spoken with a black outline, chapter titles and closing credits.

    go run ./cmd/songs/exodus -lang nl -render exodus.wav -seed 7 > exodus.log
    python3 cmd/songs/exodus/video/images.py img
    python3 cmd/songs/exodus/video/video.py exodus.log exodus.wav nl exodus.mp4 --images img

usage: video.py LOG WAV LANG OUT.mp4 --images DIR [--still SECONDS OUT.png] [--from S --to S]
LOG is what the song printed while it rendered (the chapter times and the verses), needs numpy, Pillow and ffmpeg.
The clips of the voice are read from assets/sounds/exodus/LANG, the credits are for the languages nl and en."""
import os
import re, subprocess, sys, math
import numpy as np
from PIL import Image, ImageDraw, ImageFont, ImageFilter

W, H, FPS = 1920, 1080, 24
REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", ".."))
FONT = "/usr/share/fonts/truetype/freefont/FreeSerif.ttf"
FONTI = "/usr/share/fonts/truetype/freefont/FreeSerifItalic.ttf"

log, wav, lang, out = sys.argv[1:5]
args = sys.argv[5:]
still = None
IMG = args[args.index("--images") + 1].rstrip("/") + "/"
t_from, t_to = 0.0, None
if "--still" in args:
    i = args.index("--still"); still = (float(args[i + 1]), args[i + 2])
if "--from" in args: t_from = float(args[args.index("--from") + 1])
if "--to" in args: t_to = float(args[args.index("--to") + 1])

# ---- the schedule, titles and verses come from the song's own log
text = open(log).read()
times = [float(m.group(1)) for m in re.finditer(r"^\s*([\d.]+)s\s+(?:bar|end)", text, re.M)]
end_time = times[-1]
starts = times[:-1]
body = text.split("\n\n", 1)[1]
blocks = [b.strip().split("\n") for b in body.split("\n\n") if b.strip()]
blocks = [b for b in blocks if len(b) >= 2 and not b[0].startswith("rendered")][:11]
titles = [b[0] for b in blocks]
verses = [b[1] for b in blocks]
assert len(starts) == 11 and len(titles) == 11, (len(starts), len(titles))

bpms = [58, 72, 108, 126, 54, 132, 120, 52, 66, 50, 60]
vstart = [2, 1, 1, 1, 1, 0, 2, 2, 1, 2, 2]  # the bar the voice starts at (voice.go readings)
files = "00-egypt 01-night 02-exodus 03-pursuit 04-sea 05-waters 06-miriam 07-wilderness 08-sinai 09-years 10-land".split()

def duration(path):
    return float(subprocess.check_output(["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", path]))

speech = []
for i in range(11):
    s = starts[i] + vstart[i] * 240 / bpms[i]
    d = duration(f"{REPO}/assets/sounds/exodus/{lang}/{files[i]}.wav")
    speech.append((s - 0.3, s + d + 1.2))

# ---- the images: (file, mode, (cx, cy, zoom) from, (cx, cy, zoom) to); chapters get their slots in equal parts
S = {  # name -> (file, mode, crop to the picture itself (fractions) or None)
    "pharaoh": ("c03.jpg", "cover", None), "plague": ("c04.jpg", "cover", None), "dark": ("c05.jpg", "cover", None),
    "slain": ("c06.jpg", "cover", None), "depart": ("d36.jpg", "cover", None), "sea": ("c07.jpg", "cover", None),
    "law": ("c08.jpg", "cover", None), "down": ("c09.jpg", "cover", None), "rock": ("c10.jpg", "fit", None),
    "jordan": ("c11.jpg", "fit", None), "spies": ("c13.jpg", "fit", None), "korah": ("c14.jpg", "cover", None),
    "ark": ("c15.jpg", "fit", None), "serpent": ("d42.jpg", "cover", None), "jericho": ("d46.jpg", "cover", None),
    # Tissot's drawings for the Brunoff set (photographs of the framed drawings), Tissot's watercolours
    "t_pursue": ("td129.jpg", "fit", (.09, .30, .925, .70)),
    "t_divided": ("td130.jpg", "fit", (.07, .30, .915, .69)),
    "t_destroyed": ("td131.jpg", "fit", (.075, .265, .92, .685)),
    "t_songs": ("t_songs.jpg", "cover", None),
    "t_sketch_songs": ("td134.jpg", "fit", (.155, .18, .865, .805)),
    "t_manna": ("td132.jpg", "fit", (.15, .085, .82, .77)),
    "t_rock": ("td133.jpg", "fit", (.23, .10, .77, .81)),
    "t_promised_sketch": ("td175.jpg", "fit", (.21, .14, .82, .85)),
    "t_promised": ("t_promised.jpg", "fit", None),
    # David Roberts, 1839
    "r_sinai": ("r_sinai.jpg", "cover", (.105, .15, .865, .78)),
    "r_rock": ("r_rock.jpg", "fit", (.155, .12, .84, .875)),
    "r_jericho": ("r_jericho.jpg", "fit", (.11, .15, .915, .835)),
    "r_jordan": ("r_jordan.jpg", "cover", (.02, .06, .98, .94)),
    # Poussin, 1639
    "p_manna": ("p_manna.jpg", "cover", None),
}
chapters = [
    [("pharaoh", (.5, .28, 1), (.5, .62, 1)), ("plague", (.3, .5, 1.2), (.65, .5, 1.0))],
    [("dark", (.72, .5, 1.25), (.4, .5, 1.0)), ("slain", (.5, .22, 1), (.5, .6, 1))],
    [("depart", (.5, .3, 1), (.5, .72, 1))],
    [("t_pursue", (.4, .5, 1.0), (.6, .5, 1.15))],
    [("t_divided", (.4, .5, 1.0), (.55, .5, 1.12)), ("sea", (.42, .3, 1.6), (.5, .5, 1.0))],
    [("t_destroyed", (.4, .5, 1.0), (.6, .5, 1.15))],
    [("t_songs", (.5, .5, 1.0), (.55, .45, 1.2)), ("t_sketch_songs", (.5, .5, 1.0), (.5, .5, 1.1))],
    [("p_manna", (.5, .5, 1.0), (.6, .5, 1.2)), ("t_manna", (.5, .5, 1.0), (.5, .5, 1.1)), ("rock", (.5, .5, 1.0), (.5, .45, 1.12)),
     ("t_rock", (.5, .5, 1.0), (.5, .5, 1.1)), ("r_rock", (.5, .5, 1.0), (.5, .5, 1.1))],
    [("r_sinai", (.5, .5, 1.0), (.55, .5, 1.25)), ("law", (.5, .5, 1.0), (.72, .55, 1.3)), ("down", (.5, .3, 1), (.5, .62, 1))],
    [("spies", (.5, .5, 1.0), (.5, .5, 1.12)), ("korah", (.5, .25, 1), (.5, .7, 1)), ("r_jericho", (.5, .5, 1.0), (.5, .5, 1.12))],
    [("jordan", (.5, .5, 1.0), (.5, .5, 1.12)), ("jericho", (.5, .3, 1), (.5, .7, 1)), ("ark", (.5, .5, 1.0), (.5, .48, 1.12)),
     ("r_jordan", (.4, .5, 1.0), (.6, .5, 1.12)), ("t_promised_sketch", (.5, .5, 1.0), (.5, .5, 1.1)), ("t_promised", (.5, .5, 1.0), (.5, .5, 1.08))],
]
FADE = 3.0  # the crossfade between slots

slots = []  # (t0, t1, name, a, b)
for i, ch in enumerate(chapters):
    t0 = starts[i]
    t1 = starts[i + 1] if i < 10 else end_time
    n = len(ch)
    for j, (name, a, b) in enumerate(ch):
        slots.append((t0 + (t1 - t0) * j / n, t0 + (t1 - t0) * (j + 1) / n, name, a, b))
slots[-1] = slots[-1][:1] + (end_time + 10,) + slots[-1][2:]

canvases = {}
def canvas(name):
    if name in canvases: return canvases[name]
    f, mode, crop = S[name]
    im = Image.open(IMG + f).convert("RGB")
    if crop:
        w, h = im.size
        im = im.crop((int(crop[0] * w), int(crop[1] * h), int(crop[2] * w), int(crop[3] * h)))
    gray = im.convert("L").convert("RGB")
    im = Image.blend(gray, im, 0.4)  # most of the colour out, the pictures sit together
    if mode == "cover":
        c = im
    else:
        bg = im.copy()
        s = max(W / bg.width, H / bg.height)
        bg = bg.resize((int(bg.width * s) + 1, int(bg.height * s) + 1)).filter(ImageFilter.GaussianBlur(40))
        bg = bg.crop(((bg.width - W) // 2, (bg.height - H) // 2, (bg.width - W) // 2 + W, (bg.height - H) // 2 + H))
        bg = Image.fromarray(np.clip(np.asarray(bg, dtype=np.float32) * 0.42 + np.random.default_rng(1).normal(0, 1.6, (H, W, 3)), 0, 255).astype(np.uint8))
        k = min(W * 0.92 / im.width, H * 0.9 / im.height)
        fg = im.resize((int(im.width * k), int(im.height * k)), Image.LANCZOS)
        bg.paste(fg, ((W - fg.width) // 2, (H - fg.height) // 2 - 12))
        c = bg
    canvases[name] = c
    return c

def view(name, a, b, u):
    """the window at fraction u of the way from a to b, as a W x H float array"""
    c = canvas(name)
    u = u * u * (3 - 2 * u)
    cx, cy, z = (a[k] + (b[k] - a[k]) * u for k in range(3))
    base_w = c.width
    base_h = c.width * H / W
    if base_h > c.height:
        base_h = c.height; base_w = c.height * W / H
    ww, hh = base_w / z, base_h / z
    x0 = min(max(cx * c.width - ww / 2, 0), c.width - ww)
    y0 = min(max(cy * c.height - hh / 2, 0), c.height - hh)
    return np.asarray(c.resize((W, H), Image.BICUBIC, box=(x0, y0, x0 + ww, y0 + hh)), dtype=np.float32)

yy, xx = np.mgrid[0:H, 0:W].astype(np.float32)
vig = 1 - 0.55 * (((xx - W / 2) / (W / 2)) ** 2 * 0.6 + ((yy - H / 2) / (H / 2)) ** 2 * 0.6) ** 1.5
vig = np.clip(vig, 0.25, 1)[..., None]
tint = np.array([1.04, 0.97, 0.84], dtype=np.float32)
band = np.clip((yy - (H - 450)) / 450, 0, 1) ** 1.1  # the dark gradient under the verse
top = np.clip(1 - yy / 220, 0, 1) ** 1.5

def wrap(draw, s, font, width):
    lines, cur = [], ""
    for w in s.split():
        t = (cur + " " + w).strip()
        if draw.textlength(t, font=font) <= width:
            cur = t
        else:
            lines.append(cur); cur = w
    if cur: lines.append(cur)
    return lines

def text_mask(lines_spec, stroke=5):
    """lines_spec: [(text, font, y)] centred; returns (fill, outline) masks, H x W"""
    fill = Image.new("L", (W, H), 0)
    edge = Image.new("L", (W, H), 0)
    df, de = ImageDraw.Draw(fill), ImageDraw.Draw(edge)
    for t, f, y in lines_spec:
        x = (W - df.textlength(t, font=f)) / 2
        df.text((x, y), t, font=f, fill=255)
        de.text((x, y), t, font=f, fill=255, stroke_width=stroke, stroke_fill=255)
    edge = edge.filter(ImageFilter.GaussianBlur(1.2))
    return np.asarray(fill, dtype=np.float32)[..., None] / 255, np.clip(np.asarray(edge, dtype=np.float32)[..., None] / 255 * 1.6, 0, 1)

def put(img, mask, f, color, shade=0.9):
    """text with a black outline over the picture"""
    fill, edge = mask
    img = img * (1 - shade * edge * f)
    return img * (1 - fill * f) + fill * f * np.array(color, dtype=np.float32)

vfont = ImageFont.truetype(FONT, 42)
rfont = ImageFont.truetype(FONTI, 32)
tfont = ImageFont.truetype(FONT, 110)
sfont = ImageFont.truetype(FONTI, 38)
cfont = ImageFont.truetype(FONT, 34)
dummy = ImageDraw.Draw(Image.new("L", (10, 10)))

verse_masks = []
for v in verses:
    m = re.match(r"(.*?)(?:\s*\(([^()]*)\))?$", v)
    body_t, ref = m.group(1), m.group(2)
    lines = wrap(dummy, body_t, vfont, 1500)
    y = H - 70 - (len(lines) * 56) - (44 if ref else 0)
    spec = [(l, vfont, y + k * 56) for k, l in enumerate(lines)]
    if ref: spec.append((ref, rfont, y + len(lines) * 56 + 8))
    verse_masks.append(text_mask(spec))

title_masks = [text_mask([(t, tfont, H // 2 - 120)], 7) for t in titles]
number_masks = [text_mask([(f"{i + 1} / 11", sfont, H // 2 + 30)], 3) for i in range(11)]
main_title = text_mask([("EXODUS", ImageFont.truetype(FONT, 190), H // 2 - 190)], 8)
main_sub = text_mask([("psalm", sfont, H // 2 + 60)], 3)
credits_t = {
    "nl": ["Afbeeldingen: Gustave Doré (1866), James Tissot (1896-1902), David Roberts (1839), Nicolas Poussin (1639)",
           "Alle afbeeldingen zijn publiek domein, via Wikimedia Commons",
           "Tekst: Canisius (1939) · Stem: Almer Lucke",
           "Muziek: psalm, een modulaire synthesizer in Go"],
    "en": ["Images: Gustave Doré (1866), James Tissot (1896-1902), David Roberts (1839), Nicolas Poussin (1639)",
           "All images are in the public domain, via Wikimedia Commons",
           "Text: King James Version · Voice: Piper en_GB-alba-medium (University of Edinburgh)",
           "Music: psalm, a modular synthesizer in Go"],
}[lang]
credits = text_mask([(t, cfont, H // 2 - 100 + 56 * k) for k, t in enumerate(credits_t)], 3)

def smooth(x):
    x = min(max(x, 0.0), 1.0)
    return x * x * (3 - 2 * x)

def pulse(t, a, b, fi, fo):
    return smooth((t - a) / fi) * (1 - smooth((t - (b - fo)) / fo)) if b > a else 0

def frame(t):
    # the images: every slot is shown from FADE/2 before it starts to FADE/2 after it ends
    img = None
    for (t0, t1, name, a, b) in slots:
        lo, hi = t0 - FADE / 2, t1 + FADE / 2
        if t < lo or t > hi: continue
        al = smooth((t - lo) / FADE) if t0 > 0 else 1.0
        v = view(name, a, b, min(max((t - t0) / (t1 - t0), 0), 1))
        img = v if img is None else img * (1 - al) + v * al
    img = img * tint * vig
    # a fade from and to black
    img *= smooth(t / 3.0) * (1 - smooth((t - (end_time + 0.5)) / 3.0))
    # the verse
    for i in range(11):
        a, b = speech[i]
        if a <= t <= b:
            f = pulse(t, a, b, 0.8, 1.0)
            img = img * (1 - 0.8 * band[..., None] * f)
            img = put(img, verse_masks[i], f, (238, 231, 214))
    # the titles
    for i in range(11):
        a, b = starts[i] + (6.5 if i == 0 else 0.4), starts[i] + (11.5 if i == 0 else 5.6)
        if a <= t <= b:
            f = pulse(t, a, b, 1.0, 1.4)
            img = img * (1 - 0.4 * top[..., None] * f)
            img = put(img, title_masks[i], f, (235, 226, 208))
            img = put(img, number_masks[i], f * 0.85, (235, 226, 208))
    if t < 6.5:
        f = pulse(t, 0.5, 6.5, 1.5, 1.5)
        img = img * (1 - 0.35 * f)
        img = put(img, main_title, f, (240, 236, 224))
        img = put(img, main_sub, f, (240, 236, 224))
    ct = end_time + 3.0
    if t >= ct:
        f = pulse(t, ct, ct + 12.5, 1.5, 2.0)
        img = put(img, credits, f, (225, 220, 208), 0.0)
    return np.clip(img, 0, 255).astype(np.uint8)

if still:
    Image.fromarray(frame(still[0])).save(still[1])
    sys.exit()

total = duration(wav)
t_to = t_to or total
cmd = ["ffmpeg", "-y", "-v", "error", "-f", "rawvideo", "-pix_fmt", "rgb24", "-s", f"{W}x{H}", "-r", str(FPS), "-i", "-",
       "-ss", str(t_from), "-t", str(t_to - t_from), "-i", wav,
       "-c:v", "libx264", "-preset", "medium", "-crf", "26", "-tune", "grain", "-pix_fmt", "yuv420p",
       "-c:a", "aac", "-b:a", "256k", "-shortest", "-movflags", "+faststart", out]
p = subprocess.Popen(cmd, stdin=subprocess.PIPE)
n0, n1 = int(t_from * FPS), int(t_to * FPS)
for n in range(n0, n1):
    p.stdin.write(frame(n / FPS).tobytes())
    if n % 480 == 0: print(f"{n / FPS:.0f}s / {t_to:.0f}s", flush=True)
p.stdin.close(); p.wait()
