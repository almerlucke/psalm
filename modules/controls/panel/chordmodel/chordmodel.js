// the chord model: a matrix of how likely each chord is to follow another (drag a cell up or down, the wheel changes it, a right click makes it never), a progression drawn
// from it, a card a chord (lock it to keep it when the others are drawn again, choose another from its list, set its bars), buttons that draw a new progression or only
// the chords not locked, hear it, and write it into the harmony of the patch. A progression typed in numerals teaches the model what follows what.
psalm.register("chordmodel", (p, api) => {
	const mk = (tag, props, ...kids) => { const e = document.createElement(tag); Object.assign(e, props); e.append(...kids); return e; };
	const root = mk("div", {className: "cm"});
	const el = api.labelled(p, root);
	el.classList.add("chordmodel");

	let st = null, rev = -1, hold = 0, seed = 1;
	let model = null; // the model as the page has it while it is being edited
	let dragging = false;
	const send = m => api.message(p.name, m);
	const setModel = () => { hold = performance.now() + 600; send({set: {model}}); };
	const sendStrip = () => { hold = performance.now() + 600; send({set: {strip: st.strip}}); };

	// --- the top: preset, key, scale, how the end goes ---
	const preset = mk("select", {title: "the model of a style"});
	const key = mk("select", {title: "the key the chords are in"});
	const scale = mk("select", {title: "the scale the numerals are read in"});
	const endSel = mk("select", {title: "the chord every progression ends on"});
	const memory = mk("input", {type: "range", min: 0, max: 1, step: 0.05, title: "0: only the matrix; 1: what was taught of the two chords before, where there is any"});
	const top = mk("div", {className: "cm-bar"}, mk("label", {}, "Style ", preset), mk("label", {}, "Key ", key), mk("label", {}, "Scale ", scale));
	preset.addEventListener("change", () => { if (preset.value) send({preset: preset.value}); });
	key.addEventListener("change", () => send({set: {key: Number(key.value)}}));
	scale.addEventListener("change", () => send({set: {scale: Number(scale.value)}}));
	endSel.addEventListener("change", () => { model.end = Number(endSel.value); setModel(); });
	memory.addEventListener("input", () => { model.memory = Number(memory.value); });
	memory.addEventListener("change", () => setModel());

	// --- the progression: a card for every chord ---
	const stripEl = mk("div", {className: "cm-strip"});
	const textEl = mk("code", {className: "cm-text"});
	const genB = mk("button", {textContent: "\u27f3 New", title: "draw a whole new progression of the bars to the right"});
	const rerollB = mk("button", {textContent: "Redraw the unlocked", title: "draw again the chords that are not locked (\u{1F512}), each fitting the ones next to it"});
	const playB = mk("button", {textContent: "\u25b6 Hear all", title: "play the progression through the synths this module is cabled to"});
	const barsIn = mk("input", {type: "number", min: 1, max: 128, step: 1, title: "how many bars a new progression has"});
	const target = mk("select", {title: "the section of the song the chords are written to"});
	const asSel = mk("select", {title: "write the chords as symbols (Dm7) or as numerals (ii7)"}, mk("option", {value: "symbols", textContent: "as chords"}), mk("option", {value: "numerals", textContent: "as numerals"}));
	const bakeB = mk("button", {className: "cm-primary", textContent: "Write to the harmony", title: "put this progression in the harmony module of the patch (the section chosen: its line)"});
	const status = mk("div", {className: "cm-status"});
	const nextSeed = () => { seed = Math.max(seed, st.seed); return seed++; }; // the server's next seed, or the one after when two clicks come before its answer
	genB.addEventListener("click", () => send({generate: nextSeed()}));
	rerollB.addEventListener("click", () => send({reroll: nextSeed()}));
	playB.addEventListener("click", () => send({preview: !st.playing}));
	barsIn.addEventListener("change", () => send({set: {bars: Number(barsIn.value)}}));
	target.addEventListener("change", () => send({set: {target: target.value}}));
	asSel.addEventListener("change", () => send({set: {as: asSel.value}}));
	bakeB.addEventListener("click", () => send({bake: true}));
	const liveBox = mk("input", {type: "checkbox", title: "while the song plays: when it leaves the section the chords are written to, draw the chords not locked again and write them to the harmony, so the next time through that section has other chords"});
	const everyIn = mk("input", {type: "number", min: 0, max: 128, step: 1, title: "without a song: new chords every so many bars (0: the length of the progression)"});
	liveBox.addEventListener("change", () => send({set: {live: liveBox.checked}}));
	everyIn.addEventListener("change", () => send({set: {every: Number(everyIn.value) || 0}}));
	const actions = mk("div", {className: "cm-bar"}, genB, rerollB, playB, mk("label", {}, "Bars ", barsIn));
	const live = mk("div", {className: "cm-bar"}, mk("label", {title: liveBox.title}, liveBox, " New chords after each section"), mk("label", {className: "cm-muted", title: everyIn.title}, " (without a song: every ", everyIn, " bars)"));
	const bake = mk("div", {className: "cm-bar cm-use"}, mk("span", {className: "cm-label", textContent: "Use these chords in"}), target, asSel, bakeB);

	const BARS = [0.5, 1, 2, 4];
	const barsText = n => (n === 0.5 ? "\u00bd" : String(n)) + (n === 1 ? " bar" : " bars");
	const drawStrip = () => {
		stripEl.replaceChildren();
		const total = st.strip.reduce((a, s) => a + s.bars, 0) || 1;
		st.strip.forEach((s, i) => {
			const sel = mk("select", {className: "cm-card-chord", title: "choose another chord"});
			st.model.states.forEach((n, j) => sel.append(mk("option", {value: j, textContent: st.names[j], selected: j === s.s})));
			sel.addEventListener("change", () => { s.s = Number(sel.value); s.lock = true; sendStrip(); });
			const hear = mk("button", {className: "cm-icon", textContent: "\u25b6", title: "hear this chord"});
			hear.addEventListener("click", () => send({hear: i}));
			const less = mk("button", {className: "cm-icon", textContent: "\u2212", title: "a shorter chord"});
			less.addEventListener("click", () => { const k = BARS.indexOf(s.bars); s.bars = BARS[Math.max(0, (k < 0 ? 1 : k) - 1)]; sendStrip(); });
			const more = mk("button", {className: "cm-icon", textContent: "+", title: "a longer chord"});
			more.addEventListener("click", () => { const k = BARS.indexOf(s.bars); s.bars = BARS[Math.min(BARS.length - 1, (k < 0 ? 1 : k) + 1)]; sendStrip(); });
			const lock = mk("button", {className: "cm-icon cm-lock" + (s.lock ? " on" : ""), textContent: s.lock ? "\u{1F512}" : "\u{1F513}", title: s.lock ? "locked: it stays when the others are drawn again (click to let go)" : "lock: keep this chord when the others are drawn again"});
			lock.addEventListener("click", () => { s.lock = !s.lock; sendStrip(); });
			const del = mk("button", {className: "cm-del", textContent: "\u00d7", title: "take the chord out"});
			del.addEventListener("click", () => { st.strip.splice(i, 1); sendStrip(); });
			const card = mk("div", {className: "cm-card" + (s.lock ? " locked" : "")}, mk("div", {className: "cm-roman", textContent: st.model.states[s.s]}), sel,
				mk("div", {className: "cm-length", textContent: barsText(s.bars)}), mk("div", {className: "cm-card-row"}, hear, less, more, lock), del);
			card.style.flexGrow = String(s.bars / total * 10);
			card.style.flexBasis = "104px";
			card.dataset.index = i;
			stripEl.append(card);
		});
		const add = mk("button", {className: "cm-add", textContent: "+", title: "a chord at the end"});
		add.addEventListener("click", () => { st.strip.push({s: 0, bars: 1, lock: false}); sendStrip(); });
		stripEl.append(add);
	};
	const markPlaying = () => {
		let at = 0;
		st.strip.forEach((s, i) => {
			const card = stripEl.children[i]; if (!card) return;
			card.classList.toggle("now", st.playing && st.pos >= at - 1e-9 && st.pos < at + s.bars - 1e-9);
			at += s.bars;
		});
	};

	// --- the matrix: how likely each chord is to follow each ---
	const matrixEl = mk("div", {className: "cm-matrix"});
	const rowSum = r => r.reduce((a, x) => a + Math.max(0, x), 0);
	const heat = (frac, on) => `color-mix(in srgb, var(--accent) ${Math.round(frac * 92)}%, var(--track))`;
	const paintCell = c => { // a cell as its row says: the chance of the chord in it
		const row = c._row, j = c._j, sum = rowSum(row) || 1, frac = Math.max(0, row[j]) / sum;
		c.classList.toggle("zero", frac < 0.005);
		c.textContent = frac < 0.005 ? "" : Math.round(frac * 100);
		c.style.background = heat(Math.sqrt(frac));
		c.title = `${c._label} → ${st.names[j]} ${model.states[j]}: ${Math.round(frac * 100)}%  (drag up or down, wheel, right click for never)`;
	};
	const refreshCells = () => matrixEl.querySelectorAll(".cm-cell").forEach(paintCell);
	const drawMatrix = () => {
		const n = model.states.length;
		matrixEl.style.gridTemplateColumns = `repeat(${n + 1}, minmax(34px, 1fr))`;
		matrixEl.replaceChildren();
		const head = (text, small, extra) => mk("div", {className: "cm-head" + (extra ? " " + extra : ""), textContent: text}, small ? mk("small", {textContent: small}) : "");
		matrixEl.append(mk("div", {className: "cm-corner", textContent: "from ↓  to →"}));
		model.states.forEach((s, j) => {
			const h = head(s, st.names[j]);
			const x = mk("button", {className: "cm-x", textContent: "×", title: "take this chord out of the model"});
			x.addEventListener("click", () => { send({set: {model: removeState(j)}}); });
			h.append(x);
			matrixEl.append(h);
		});
		const cell = (row, j, label) => {
			const c = mk("div", {className: "cm-cell"});
			c._row = row; c._j = j; c._label = label;
			paintCell(c);
			c.dataset.j = j;
			c.addEventListener("pointerdown", e => {
				if (e.button !== 0) return;
				c.setPointerCapture(e.pointerId); dragging = true;
				const y0 = e.clientY, w0 = row[j], base = Math.max(rowSum(row), 1);
				const move = ev => { row[j] = Math.max(0, w0 + (y0 - ev.clientY) / 60 * base); refreshCells(); };
				const up = () => { c.removeEventListener("pointermove", move); c.removeEventListener("pointerup", up); dragging = false; setModel(); };
				c.addEventListener("pointermove", move); c.addEventListener("pointerup", up);
				e.preventDefault();
			});
			c.addEventListener("contextmenu", e => { e.preventDefault(); row[j] = 0; refreshCells(); setModel(); });
			c.addEventListener("wheel", e => {
				e.preventDefault();
				row[j] = Math.max(0, row[j] + (e.deltaY < 0 ? 1 : -1) * 0.05 * Math.max(rowSum(row), 1));
				refreshCells(); clearTimeout(wheelT); wheelT = setTimeout(setModel, 300);
			}, {passive: false});
			return c;
		};
		let wheelT = 0;
		// the start row: how likely each chord is to begin
		matrixEl.append(mk("div", {className: "cm-head cm-start", textContent: "start"}));
		for (let j = 0; j < n; j++) matrixEl.append(cell(model.start, j, "start"));
		for (let i = 0; i < n; i++) {
			matrixEl.append(head(model.states[i], st.names[i], "cm-row"));
			for (let j = 0; j < n; j++) matrixEl.append(cell(model.w[i], j, model.states[i]));
		}
	};
	const removeState = j => { // a copy of the model without a chord (the server drops the learned pairs too)
		const m = JSON.parse(JSON.stringify(model));
		m.states.splice(j, 1); m.start.splice(j, 1); m.w.splice(j, 1); m.w.forEach(r => r.splice(j, 1)); delete m.pairs;
		m.end = m.end === j ? -1 : m.end > j ? m.end - 1 : m.end;
		return m;
	};

	// --- how long a chord lasts ---
	const lengthsEl = mk("div", {className: "cm-lengths"});
	const drawLengths = () => {
		lengthsEl.replaceChildren(mk("span", {textContent: "how long a chord lasts: "}));
		const top = Math.max(...model.lengths.map(l => l.weight), 1);
		model.lengths.forEach(l => {
			const bar = mk("div", {className: "cm-len-bar"}); bar.style.height = Math.max(2, 26 * l.weight / top) + "px";
			const box = mk("div", {className: "cm-len", title: "drag up or down: how often a chord lasts " + l.bars + " bar(s)"}, mk("div", {className: "cm-len-col"}, bar), mk("small", {textContent: l.bars === 0.5 ? "½" : String(l.bars)}));
			box.addEventListener("pointerdown", e => {
				box.setPointerCapture(e.pointerId); const y0 = e.clientY, w0 = l.weight;
				const move = ev => { l.weight = Math.max(0, w0 + (y0 - ev.clientY) / 26 * top); drawLengths(); };
				const up = () => { box.removeEventListener("pointermove", move); box.removeEventListener("pointerup", up); setModel(); };
				box.addEventListener("pointermove", move); box.addEventListener("pointerup", up);
				e.preventDefault();
			});
			lengthsEl.append(box);
		});
	};

	// --- teaching ---
	const learnText = mk("textarea", {rows: 3, placeholder: "I vi IV V\nii V I\nI bVII IV I", title: "progressions in numerals, a line each: what follows what is made likelier, chords the model doesn't have are added"});
	const rate = mk("input", {type: "number", min: 0.05, max: 2, step: 0.05, title: "how much a progression counts against what the matrix says (1 against the sum of a row)", value: 0.25});
	const learnB = mk("button", {textContent: "Teach it", title: "make what these progressions do likelier"});
	learnB.addEventListener("click", () => { send({set: {rate: Number(rate.value)}}); send({learn: learnText.value}); });
	const learn = mk("div", {className: "cm-learn"}, learnText, mk("div", {className: "cm-bar"}, learnB, mk("label", {}, "weight ", rate)));

	const endRow = mk("div", {className: "cm-bar"}, mk("label", {title: endSel.title}, "End on ", endSel), mk("label", {title: memory.title}, "Memory ", memory));
	const adv = mk("details", {className: "cm-adv"}, mk("summary", {textContent: "How it chooses (advanced)"}), endRow, mk("h4", {textContent: "What follows what"}), matrixEl, lengthsEl, mk("h4", {textContent: "Teach it a progression you like"}), learn);
	root.append(top, stripEl, actions, live, textEl, bake, status, adv);

	const fill = (select, options, value) => { // keep the choice as it is when nothing changed
		const k = JSON.stringify(options);
		if (select._k !== k) { select._k = k; select.replaceChildren(...options.map(([v, t]) => mk("option", {value: v, textContent: t}))); }
		select.value = String(value);
	};

	return {
		el,
		show() {},
		update(s) {
			if (dragging || performance.now() < hold) return;
			if (s.rev === rev && st) { st.playing = s.playing; st.pos = s.pos; markPlaying(); playB.textContent = s.playing ? "\u25a0 Stop" : "\u25b6 Hear all"; return; }
			rev = s.rev; st = s;
			model = JSON.parse(JSON.stringify(s.model));
			fill(preset, [["", s.preset ? "choose a style…" : "(taught)"]].concat(s.presets.map(n => [n, n])), s.preset || "");
			fill(key, s.keys.map((k, i) => [i, k]), s.key);
			fill(scale, s.scales.map((k, i) => [i, k]), s.scale);
			fill(endSel, [[-1, "(anything)"]].concat(s.names.map((n, i) => [i, `${n}  ${s.model.states[i]}`])), s.model.end);
			memory.value = s.model.memory;
			barsIn.value = s.bars; asSel.value = s.as; rate.value = s.rate; liveBox.checked = !!s.live; everyIn.value = s.every || 0;
			fill(target, [["", "the whole harmony"]].concat(s.sections.map(x => [x.name, x.name])), s.target);
			textEl.textContent = s.text || " ";
			status.textContent = s.error; status.classList.toggle("bad", !!s.error);
			playB.textContent = s.playing ? "\u25a0 Stop" : "\u25b6 Hear all";
			drawStrip(); drawMatrix(); drawLengths(); markPlaying();
		},
		restyle() {},
	};
});
