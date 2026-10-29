// the editor of a melodist: what it plays now on top (the pass as a small piano roll with the chords over it and a playhead), the settings under it in plain words (a slider from one
// thing to its opposite), the shape of the line as pictures, its range on a keyboard, the sections it plays in, how much it changes from pass to pass, presets, and the buttons that draw
// it again, freeze it and keep it in a note editor.
psalm.register("melodist", (p, api) => {
	const mk = (tag, props, ...kids) => { const e = document.createElement(tag); Object.assign(e, props); e.append(...kids); return e; };
	const root = mk("div", {className: "md"});
	const el = api.labelled(p, root);
	el.classList.add("melodist");
	const send = m => api.message(p.name, m);

	let st = null, hold = 0, at = performance.now();
	const cfgSend = (key, value) => { hold = performance.now() + 500; if (st) st.cfg[key] = value; send({set: {[key]: value}}); };

	// --- the top: presets and the buttons ---
	const preset = mk("select", {title: "start from a kind of melody"});
	preset.addEventListener("change", () => { if (preset.value) { hold = performance.now() + 500; send({preset: preset.value}); preset.value = ""; } });
	const againB = mk("button", {textContent: "\u{1F3B2} Draw again", title: "forget what the sections had: the next time through each section is a new melody"});
	againB.addEventListener("click", () => send({action: "again"}));
	const freezeB = mk("button", {textContent: "Freeze", title: "keep the melody as it is: it plays the same every time through a section (until you let go)"});
	freezeB.addEventListener("click", () => cfgSend("freeze", st && st.cfg.freeze ? 0 : 1));
	const keepB = mk("button", {textContent: "Keep in the roll", title: "write the melody that plays into the clip of its section in the note editor of the patch, to edit it there"});
	keepB.addEventListener("click", () => send({action: "keep"}));
	const top = mk("div", {className: "md-bar"}, preset, againB, freezeB, keepB);
	const statusEl = mk("div", {className: "md-status"});

	// --- what it plays now ---
	const canvas = mk("canvas", {className: "md-roll", title: "the melody that plays now: the notes (louder ones brighter), the chords over them"});
	let colors = {};
	const restyle = () => { colors = {bg: api.css("track"), ink: api.css("ink"), muted: api.css("muted"), line: api.css("line"), accent: api.css("accent")}; };
	restyle();
	const drawRoll = () => {
		const dpr = window.devicePixelRatio || 1, w = canvas.clientWidth, h = canvas.clientHeight;
		if (!w || !h) return;
		if (canvas.width !== Math.round(w * dpr)) { canvas.width = Math.round(w * dpr); canvas.height = Math.round(h * dpr); }
		const c = canvas.getContext("2d");
		c.setTransform(dpr, 0, 0, dpr, 0, 0);
		c.clearRect(0, 0, w, h);
		if (!st || !st.beats || !(st.notes.length || st.counter.length)) {
			c.fillStyle = colors.muted; c.font = "12px system-ui"; c.textAlign = "center";
			c.fillText(st && st.status ? "waiting for the next pass…" : "it plays when the song does", w / 2, h / 2 + 4);
			return;
		}
		const top = 16, pad = 6, beats = st.beats;
		const all = st.notes.concat(st.counter);
		const lo = Math.min(...all.map(n => n[2])) - 1, hi = Math.max(...all.map(n => n[2])) + 1, span = Math.max(8, hi - lo);
		const x = b => pad + b / beats * (w - 2 * pad), y = k => h - pad - (k - lo + 0.5) / span * (h - top - pad);
		// the chords: a stripe each, its name on top
		st.chords.forEach((ch, i) => {
			c.globalAlpha = i % 2 ? 0.1 : 0.2; c.fillStyle = colors.line; c.fillRect(x(ch.at), top, x(ch.at + ch.len) - x(ch.at), h - top);
			c.globalAlpha = 1; c.fillStyle = colors.muted; c.font = "11px system-ui"; c.textAlign = "left";
			c.fillText(ch.name, x(ch.at) + 3, 11);
		});
		// the beats
		c.strokeStyle = colors.line; c.globalAlpha = 0.5; c.lineWidth = 1;
		for (let b = 0; b < beats; b += 4) { c.beginPath(); c.moveTo(x(b), top); c.lineTo(x(b), h); c.stroke(); }
		c.globalAlpha = 1;
		const bar = (n, alpha) => {
			c.globalAlpha = alpha * (0.35 + 0.65 * Math.min(1, n[3]));
			c.fillStyle = colors.accent;
			const bh = Math.max(3, Math.min(8, (h - top) / span * 0.8));
			c.beginPath(); c.roundRect(x(n[0]), y(n[2]) - bh / 2, Math.max(3, x(n[0] + n[1] * 0.95) - x(n[0])), bh, 2); c.fill();
		};
		st.counter.forEach(n => bar(n, 0.55));
		st.notes.forEach(n => bar(n, 1));
		c.globalAlpha = 1;
		// the playhead
		if (st.pos >= 0) {
			const b = (st.pos + (performance.now() - at) / 1000 * st.bps) % beats;
			c.strokeStyle = colors.ink; c.lineWidth = 1.5; c.beginPath(); c.moveTo(x(b), top); c.lineTo(x(b), h); c.stroke();
		}
	};
	const loop = () => { if (document.body.contains(canvas) || !st) drawRoll(); requestAnimationFrame(loop); };

	// --- where it plays ---
	const chips = mk("div", {className: "md-chips"});
	const drawChips = () => {
		chips.replaceChildren();
		if (!st.sections.length) { chips.append(mk("span", {className: "md-muted", textContent: "the patch has no song: it plays all the time, a pass of the length below"})); return; }
		const plays = st.plays && st.plays.length ? st.plays : null;
		const all = mk("button", {className: "md-chip" + (plays ? "" : " on"), textContent: "All", title: "play in every section"});
		all.addEventListener("click", () => { hold = performance.now() + 500; send({plays: []}); });
		chips.append(all);
		st.sections.forEach(s => {
			const on = !plays || plays.includes(s);
			const b = mk("button", {className: "md-chip" + (on ? " on" : ""), textContent: s, title: "play (or not) in the " + s});
			b.addEventListener("click", () => {
				const now = plays ? plays.slice() : st.sections.slice();
				const i = now.indexOf(s);
				if (i >= 0) now.splice(i, 1); else now.push(s);
				const next = now.length === st.sections.length ? [] : now;
				if (!next.length && now.length === 0) return; // not in no section
				hold = performance.now() + 500; send({plays: next});
			});
			chips.append(b);
		});
	};

	// --- sliders in plain words ---
	const rows = [];
	const slider = (label, left, right, get, put, tip, show) => {
		const min = 0, max = 1, step = 0.01;
		const input = mk("input", {type: "range", min, max, step, title: tip});
		const val = mk("span", {className: "md-val"});
		const row = mk("div", {className: "md-row"}, mk("div", {className: "md-label", textContent: label, title: tip}), mk("span", {className: "md-end", textContent: left}), input, mk("span", {className: "md-end", textContent: right}), val);
		input.addEventListener("input", () => { put(Number(input.value)); val.textContent = shown(Number(input.value)); });
		const shown = show || (v => Math.round(v * 100) + "%");
		rows.push(() => { input.value = String(get()); val.textContent = shown(Number(input.value)); });
		return row;
	};
	const logSlider = (label, left, right, get, put, tip) => { // note length: from 4 beats to a quarter of a beat, on a log scale
		const beats = v => { const g = 4 * Math.pow(1 / 16, v); return (g >= 1 ? String(Math.round(g * 10) / 10) : String(Math.round(g * 100) / 100)) + " beats"; };
		return slider(label, left, right, () => Math.log(get() / 4) / Math.log(1 / 16), v => put(4 * Math.pow(1 / 16, v)), tip, v => "≥ " + beats(v));
	};
	const feel = mk("div", {className: "md-group"},
		logSlider("Note length", "Long", "Short", () => st.cfg.grid, v => cfgSend("grid", Math.round(v * 100) / 100), "the shortest note the melody plays: long notes make a slow tune"),
		slider("Silence", "Little", "A lot", () => st.cfg.rest / 0.8, v => cfgSend("rest", v * 0.8), "how often it rests between the notes"),
		slider("Notes", "Calm", "Busy", () => st.cfg.density, v => cfgSend("density", v), "how many notes (the length of the notes matters more: see above)"),
		slider("Movement", "Steps", "Leaps", () => 1 - st.cfg.steps, v => cfgSend("steps", 1 - v), "moving to the next note of the scale, or jumping"),
		slider("Harmony", "Anywhere", "Chord tones", () => st.cfg.chord, v => cfgSend("chord", v), "how much the notes on the beat keep to the notes of the chord"),
		slider("Expression", "Even", "Expressive", () => st.cfg.dynamics, v => cfgSend("dynamics", v), "how much the notes differ in loudness: the beats accented, the line swelling in a phrase"),
		slider("Loudness", "Soft", "Loud", () => st.cfg.velocity, v => cfgSend("velocity", v), "how loud the melody plays"));

	// --- the shape ---
	const ICONS = {free: "∼", arch: "∩", rise: "↗", fall: "↘", wave: "∿"};
	const contourBox = mk("div", {className: "md-seg"});
	const phraseBox = mk("div", {className: "md-seg"});
	const rhythmBox = mk("div", {className: "md-seg"});
	const formSel = mk("select", {title: "how the phrases repeat and vary"});
	const FORMS = {free: "one phrase", "AA'": "a period: A A′", ABAB: "A B A B", AABA: "A A′ B A", call: "a question and its answer", grammar: "a grammar (below, under advanced)"};
	formSel.addEventListener("change", () => cfgSend("form", Number(formSel.value)));
	const PHRASES = [2, 4, 8, 16];
	const segment = (box, items, current, put) => {
		box.replaceChildren(...items.map(([v, label, title]) => { const b = mk("button", {className: "md-segb" + (v === current ? " on" : ""), textContent: label, title}); b.addEventListener("click", () => put(v)); return b; }));
	};
	const shape = mk("div", {className: "md-group"},
		mk("div", {className: "md-line"}, mk("span", {className: "md-label", textContent: "Contour"}), contourBox),
		mk("div", {className: "md-line"}, mk("span", {className: "md-label", textContent: "Rhythm"}), rhythmBox),
		mk("div", {className: "md-line"}, mk("span", {className: "md-label", textContent: "Phrase"}), phraseBox, mk("span", {className: "md-muted", textContent: "beats"})),
		mk("div", {className: "md-line"}, mk("span", {className: "md-label", textContent: "Form"}), formSel));

	// --- the range on a keyboard ---
	const keys = mk("canvas", {className: "md-keys", title: "drag the ends: the lowest and the highest note of the melody"});
	const KLO = 36, KHI = 108;
	let dragKey = null;
	const keyX = (k, w) => (k - KLO) / (KHI - KLO) * w;
	const drawKeys = () => {
		const dpr = window.devicePixelRatio || 1, w = keys.clientWidth, h = keys.clientHeight;
		if (!w || !h || !st) return;
		if (keys.width !== Math.round(w * dpr)) { keys.width = Math.round(w * dpr); keys.height = Math.round(h * dpr); }
		const c = keys.getContext("2d");
		c.setTransform(dpr, 0, 0, dpr, 0, 0);
		c.clearRect(0, 0, w, h);
		const black = k => [1, 3, 6, 8, 10].includes(((k % 12) + 12) % 12);
		const kw = w / (KHI - KLO);
		for (let k = KLO; k < KHI; k++) {
			const inside = k >= st.cfg.low && k <= st.cfg.high;
			c.globalAlpha = inside ? 1 : 0.35;
			c.fillStyle = black(k) ? colors.muted : colors.ink; c.globalAlpha *= black(k) ? 0.8 : 0.25;
			c.fillRect(keyX(k, w), black(k) ? 0 : 0, kw - 1, black(k) ? h * 0.6 : h);
			if (inside) { c.globalAlpha = 0.3; c.fillStyle = colors.accent; c.fillRect(keyX(k, w), 0, kw - 1, h); }
			if (k % 12 === 0) { c.globalAlpha = 0.8; c.fillStyle = colors.muted; c.font = "9px system-ui"; c.textAlign = "left"; c.fillText("C" + (k / 12 - 1), keyX(k, w) + 1, h - 2); }
		}
		c.globalAlpha = 1; c.fillStyle = colors.accent;
		for (const k of [st.cfg.low, st.cfg.high + 1]) c.fillRect(keyX(k, w) - 1.5, 0, 3, h);
	};
	const keyAt = e => { const r = keys.getBoundingClientRect(); return Math.round(KLO + (e.clientX - r.left) / r.width * (KHI - KLO)); };
	keys.addEventListener("pointerdown", e => {
		const k = keyAt(e), dl = Math.abs(k - st.cfg.low), dh = Math.abs(k - (st.cfg.high + 1));
		dragKey = dl <= dh ? "low" : "high";
		keys.setPointerCapture(e.pointerId);
		keys.dispatchEvent(new PointerEvent("pointermove", {clientX: e.clientX, pointerId: e.pointerId}));
	});
	keys.addEventListener("pointermove", e => {
		if (!dragKey) return;
		let k = Math.max(KLO, Math.min(KHI, keyAt(e)));
		if (dragKey === "low") k = Math.min(k, st.cfg.high - 4); else k = Math.max(k - 1, st.cfg.low + 4);
		st.cfg[dragKey] = k; hold = performance.now() + 800; drawKeys(); rangeText.textContent = noteName(st.cfg.low) + " – " + noteName(st.cfg.high);
	});
	keys.addEventListener("pointerup", () => { if (dragKey) { send({set: {low: st.cfg.low, high: st.cfg.high}}); dragKey = null; } });
	const NAMES = ["C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"];
	const noteName = k => NAMES[((k % 12) + 12) % 12] + (Math.floor(k / 12) - 1);
	const rangeText = mk("span", {className: "md-muted"});
	const scaleSel = mk("select", {title: "the scale the notes are in: the one of the harmony of the patch, or one of its own"});
	const keySel = mk("select", {title: "the key of the scale"});
	scaleSel.addEventListener("change", () => cfgSend("scale", Number(scaleSel.value)));
	keySel.addEventListener("change", () => cfgSend("key", Number(keySel.value)));
	const range = mk("div", {className: "md-group"}, keys, mk("div", {className: "md-line"}, mk("span", {className: "md-label", textContent: "Range"}), rangeText, mk("span", {className: "md-label md-gap", textContent: "Scale"}), scaleSel, keySel));

	// --- change over time ---
	const diceB = mk("button", {textContent: "\u{1F3B2}", title: "another seed: a different melody from the same settings"});
	diceB.addEventListener("click", () => { cfgSend("seed", Math.floor(Math.random() * 9999)); send({action: "again"}); });
	const change = mk("div", {className: "md-group"}, mk("div", {className: "md-line md-wide"},
		slider("Changes", "Same every time", "New every time", () => st.cfg.mutate, v => cfgSend("mutate", v), "how much of the melody changes the next time through a section: a little (the tune is the tune again and a little other) or all of it").firstChild.parentNode, diceB));

	// --- advanced ---
	const counterBox = mk("input", {type: "checkbox", title: "a second voice goes with the melody, against it"});
	counterBox.addEventListener("change", () => cfgSend("counter", counterBox.checked ? 1 : 0));
	const justBox = mk("input", {type: "checkbox", title: "the notes in pure ratios from the tonic instead of equal temperament"});
	justBox.addEventListener("change", () => cfgSend("just", justBox.checked ? 1 : 0));
	const lengthIn = mk("input", {type: "number", min: 2, max: 128, step: 1, title: "how many beats a pass has when the patch has no song"});
	lengthIn.addEventListener("change", () => cfgSend("length", Number(lengthIn.value)));
	const grammar = mk("textarea", {rows: 4, spellcheck: false, className: "md-gram", placeholder: "S -> A A' B A~", title: "rules that write the form: S -> A A' B A~ (A to D are phrases; ' a variation, ~ upside down, < backwards, ^ v a step up or down, ? ends open); or axiom: A, iterate: 3, A -> A B A'"});
	let gramT = 0;
	grammar.addEventListener("input", () => { clearTimeout(gramT); gramT = setTimeout(() => { hold = performance.now() + 600; send({grammar: grammar.value}); }, 500); });
	const gramErr = mk("div", {className: "md-err"});
	const advanced = mk("details", {className: "md-adv"}, mk("summary", {textContent: "More"}),
		mk("div", {className: "md-line"}, mk("label", {}, counterBox, " a second voice"), mk("label", {}, justBox, " just intonation"), mk("label", {}, "beats a pass without a song ", lengthIn)),
		mk("div", {className: "md-label", textContent: "Grammar for the form (used when the form is “a grammar”)"}), grammar, gramErr);

	const h = text => mk("h4", {textContent: text});
	root.append(top, statusEl, canvas, h("Where it plays"), chips, h("Feel"), feel, h("Shape"), shape, h("Range"), range, h("Over time"), change, advanced);

	const fill = (select, options, value) => {
		const k = JSON.stringify(options);
		if (select._k !== k) { select._k = k; select.replaceChildren(...options.map(([v, t]) => mk("option", {value: v, textContent: t}))); }
		select.value = String(value);
	};
	let drawn = "";
	requestAnimationFrame(loop);

	return {
		el,
		show() { drawRoll(); drawKeys(); },
		restyle() { restyle(); drawRoll(); drawKeys(); },
		update(s) {
			const first = !st;
			if (st && (performance.now() < hold)) { // our own change until the server has it: only what plays and where
				st.notes = s.notes; st.counter = s.counter; st.chords = s.chords; st.beats = s.beats; st.pos = s.pos; st.bps = s.bps; st.section = s.section; st.pass = s.pass; st.status = s.status;
				at = performance.now(); statusEl.textContent = s.status; return;
			}
			st = s; at = performance.now();
			fill(preset, [["", "presets…"]].concat(s.presets.map(n => [n, n])), "");
			statusEl.textContent = s.status + (s.kept ? " · " + s.kept : "");
			freezeB.classList.toggle("on", !!s.cfg.freeze); freezeB.textContent = s.cfg.freeze ? "Frozen" : "Freeze";
			rows.forEach(f => f());
			segment(contourBox, s.contours.map((c, i) => [i, ICONS[c] || c, c]), s.cfg.contour, v => cfgSend("contour", v));
			segment(rhythmBox, s.rhythms.map((c, i) => [i, c, c]), s.cfg.rhythm, v => cfgSend("rhythm", v));
			segment(phraseBox, PHRASES.map((b, i) => [i, String(b), b + " beats"]), s.cfg.phrase, v => cfgSend("phrase", v));
			fill(formSel, s.forms.map((f, i) => [i, FORMS[f] || f]), s.cfg.form);
			fill(scaleSel, s.scales.map((n, i) => [i, n]), s.cfg.scale);
			fill(keySel, s.keys.map((n, i) => [i, n]), s.cfg.key); keySel.hidden = s.cfg.scale === 0;
			rangeText.textContent = noteName(s.cfg.low) + " – " + noteName(s.cfg.high);
			counterBox.checked = !!s.cfg.counter; justBox.checked = !!s.cfg.just; lengthIn.value = s.cfg.length;
			if (document.activeElement !== grammar) grammar.value = s.grammar;
			gramErr.textContent = s.grammarErr || "";
			if (first && (s.forms[s.cfg.form] === "grammar" || s.grammarErr)) advanced.open = true;
			drawChips(); drawKeys(); drawRoll();
		},
	};
});
