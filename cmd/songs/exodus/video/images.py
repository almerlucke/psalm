#!/usr/bin/env python3
"""Fetches the pictures of the Exodus video from Wikimedia Commons into a directory (all in the public domain):
python3 images.py DIR"""
import os, subprocess, sys, time, urllib.parse

PICTURES = {  # file here: file on Commons
    # Gustave Doré, La Sainte Bible (1866)
    "c01.jpg": "030.The Child Moses on the Nile.jpg",
    "c02.jpg": "031.The Finding of Moses.jpg",
    "c03.jpg": "032.Moses and Aaron Appear before Pharaoh.jpg",
    "c04.jpg": "033.The Fifth Plague. Livestock Disease.jpg",
    "c05.jpg": "034.The Ninth Plague. Darkness.jpg",
    "c06.jpg": "035.The Firstborn of the Egyptians Are Slain.jpg",
    "d36.jpg": "036.The Egyptians Ask Moses to Depart.jpg",
    "c07.jpg": "037.The Egyptians Drown in the Sea.jpg",
    "c08.jpg": "038.The Giving of the Law on Mount Sinai.jpg",
    "c09.jpg": "039.Moses Comes Down from Mount Sinai.jpg",
    "c10.jpg": "Dore Moses Striking the Rock in Horeb.jpg",
    "c11.jpg": "Dore joshua crossing.jpg",
    "c13.jpg": "Dore Return of the Spies From the Land of Promise.jpg",
    "c14.jpg": "Dore Death of Korah, Dathan and Abiram.jpg",
    "c15.jpg": "Gustave Doré - Ark of the Covenant.jpg",
    "d42.jpg": "042.The Bronze Serpent.jpg",
    "d46.jpg": "046.The Walls of Jericho Fall Down.jpg",
    # James Tissot (the drawings for the Brunoff set are photographs of the framed drawings), Tissot's watercolours
    "td129.jpg": "Tissot Drawing 129 Pharaoh pursues the Israelites (Exodus 14 7) for Brunoff 123 Pharaon poursuit les Israélites.jpg",
    "td130.jpg": "Tissot Drawing 130 The waters are divided (Exodus 14 22) for Brunoff 124 Les eaux s'entr'ouvrent.jpg",
    "td131.jpg": "Tissot Drawing 131 The Egyptians are destroyed (Exodus 14 28) for Brunoff 125 L'armée égyptienne engloutie.jpg",
    "td132.jpg": "Tissot Drawing 132 The gathering of the manna (Exodus 16 17) for Brunoff 128 La récolte de la manne.jpg",
    "td133.jpg": "Tissot Drawing 133 Moses smites the rock in the desert (Exodus 17 6) for Brunoff 129 L'eau sortant du rocher.jpg",
    "td134.jpg": "Tissot Drawing 134 The songs of joy (Exodus 15;20) for Brunoff 126 Les chants d'allégresse.jpg",
    "td175.jpg": "Tissot Drawing 175 Moses sees the promised land from afar (Numbers 27 12) for Brunoff 164 Moïse voit de loin la terre promise.jpg",
    "t_songs.jpg": "The Songs of Joy, by James Tissot.jpg",
    "t_promised.jpg": "Tissot Moses Sees the Promised Land from Afar.jpg",
    # David Roberts, The Holy Land (1839)
    "r_sinai.jpg": "Approach to Mount Sinai Wady Barah Feby 17th 1839 - David Roberts, R.A. LCCN2002717566.jpg",
    "r_rock.jpg": "Rock of Moses Wady-el-Lega Mount Horeb Feby 22nd 1839 - David Roberts, R.A. LCCN2002717556.jpg",
    "r_jericho.jpg": "Jericho April 3rd 1839 LCCN2002717496.jpg",
    "r_jordan.jpg": "David Roberts - Banks of the Jordan - 1927.78 - Cleveland Museum of Art.jpg",
    # Nicolas Poussin, The Jews Gathering the Manna in the Desert (1637-39)
    "p_manna.jpg": "Poussin, Nicolas - The Jews Gathering the Manna in the Desert -1637 - 1639.jpg",
}

out = sys.argv[1]
os.makedirs(out, exist_ok=True)
for name, title in PICTURES.items():
    path = os.path.join(out, name)
    if os.path.exists(path) and os.path.getsize(path) > 0:
        continue
    url = "https://commons.wikimedia.org/wiki/Special:FilePath/" + urllib.parse.quote(title) + "?width=1920"
    subprocess.run(["curl", "-sfL", "-m", "120", "-A", "psalm-video/1.0", "-o", path, url], check=True)
    print(name)
    time.sleep(1)
