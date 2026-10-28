// a note editor: a piano roll of a loop. The notes are [start, length, key, velocity] (start and length in beats). It works like the piano roll of a sequencer:
//
//   select tool (V)  a drag on empty space draws a rectangle that selects notes; a click on a note selects it (shift adds), a drag moves the notes selected (alt copies them),
//                    the right edge of a note resizes all the notes selected; a double click on empty space puts a note
//   draw tool (D)    a click puts a note, a drag makes it longer
//   erase tool (E)   a drag takes the notes it crosses away; a right click takes a note away with every tool
//   the ruler        a click puts the cursor (where paste goes), a drag chooses a region of the loop: copy it, duplicate it (ctrl+D repeats it), delete it
//   the velocity lane under the notes: drag a bar up or down, or paint across the notes
//   ctrl+A all, ctrl+C/X/V copy, cut, paste (at the cursor), ctrl+D duplicate (after the notes or region), ctrl+Z/ctrl+shift+Z undo, redo, Delete,
//   arrows move the notes (left/right by the grid, shift by a beat; up/down a semitone, shift an octave), Q quantizes the start, L makes them legato
//   the wheel scrolls the keys, shift+wheel the beats, ctrl+wheel zooms; the page keeps its own copy while you work and sends all of it when you let go
//
// The Melody panel makes melodies (tools/melody) for the region chosen in the ruler, the notes selected, or the whole clip: in the scale of the harmony (or one of its own), over the
// chords, in phrases that repeat and vary; Generate puts one in (undo takes it away), Variations shows six to hear and choose from; Invert, Reverse, scale steps, Humanize and
// Thin out change the notes selected.
//
// A song has a clip for every section: the clip chosen above the roll (the loop plays when a section has none) is the one edited, and the song plays the clip of the section it is in
// from the section's start. A clip can be copied to other sections. With a harmony module its chords show in the ruler, and the keys that are tones of the chord are lit under them
psalm.register("noteroll", (p, api) => {
	const canvas = document.createElement("canvas");
	const bar = document.createElement("div"); bar.className = "nr-bar";
	const bar2 = document.createElement("div"); bar2.className = "nr-bar";
	const hint = document.createElement("div"); hint.className = "nr-hint";
	hint.textContent = "V select, D draw, E erase; double click puts a note; ctrl+C/X/V/D/A/Z; arrows move; Q quantize, L legato; ctrl+wheel zooms, shift+wheel scrolls";
	const el = api.labelled(p, canvas, hint);
	el.classList.add("noteroll");
	canvas.parentNode.insertBefore(bar, canvas);
	canvas.parentNode.insertBefore(bar2, canvas);

	// the notes are objects here: {s: start, l: length, k: key, v: velocity}
	let notes = [], length = 8, pos = 0, bps = 2, at = performance.now();
	const sel = new Set();
	let tool = "select";
	let snap = 0.25;
	let lastLen = 1, lastVel = 0.8;
	let lo = 48; // the lowest key shown
	let pxb = 0, x0 = 0; // pixels a beat, the first beat shown
	let cursor = null; // where paste goes (beats)
	let region = null; // [a, b] in beats
	let askedView = null; // the clip we asked for, until the server shows it
	let viewName = "", sections = [], have = [""], playing = "", chords = [], waitView = false; // the clip edited, the sections of the song, the clips there are, the clip playing, the chords over it
	let grooves = [], scaleInfo = null, scaleLock = false, genId = 0, genPending = null, genSeed = Math.floor(Math.random() * 100000); // the scale of the harmony; keys held to it; the melodies asked for
	let first = true, showKeys = null; // the key to have in the middle of the roll, until it has a size
	let hold = 0; // until when the page's own notes stay (the server's come back after a send)
	let drag = null, band = null;
	const undo = [], redo = [];
	let base = ""; // the last state sent, as text

	const gutter = 52, rulerH = 34, velH = 70, rowH = 13, gap = 6;
	const MAXBEATS = 64;

	// --- the bar ---
	const mk = (tag, props, ...kids) => { const e = document.createElement(tag); Object.assign(e, props); e.append(...kids); return e; };
	const toolBtns = {};
	for (const [id, label, title] of [["select", "Select", "select and move notes (V)"], ["draw", "Draw", "click to put notes (D)"], ["erase", "Erase", "drag over notes to take them away (E)"]]) {
		toolBtns[id] = mk("button", {textContent: label, title, className: "nr-tool"});
		toolBtns[id].dataset.tool = id;
		toolBtns[id].addEventListener("click", () => setTool(id));
	}
	const setTool = t => {
		tool = t;
		for (const [id, b] of Object.entries(toolBtns)) b.classList.toggle("on", id === t);
		canvas.style.cursor = t === "erase" ? "not-allowed" : "crosshair";
	};
	const snapSel = mk("select", {title: "what the notes snap to"});
	for (const [name, v] of [["off", 0], ["1 beat", 1], ["1/2", 0.5], ["1/4 (16th)", 0.25], ["1/8", 0.125], ["1/3 (triplet)", 1 / 3], ["1/6", 1 / 6]]) {
		const o = mk("option", {value: v, textContent: name}); if (v === snap) o.selected = true; snapSel.append(o);
	}
	const lenIn = mk("input", {type: "number", min: 1, max: MAXBEATS, step: 1, value: length, title: "how many beats the loop has"});
	const dbl = mk("button", {textContent: "×2", title: "double the loop: its notes are copied into the second half"});
	const half = mk("button", {textContent: "½", title: "halve the loop: the second half is cut off"});
	const zoom = mk("input", {type: "range", min: 0, max: 1, step: 0.01, value: 0, title: "zoom (ctrl+wheel)", className: "nr-zoom"});
	const undoB = mk("button", {textContent: "Undo", title: "ctrl+Z"});
	const redoB = mk("button", {textContent: "Redo", title: "ctrl+shift+Z"});
	const quantB = mk("button", {textContent: "Quantize", title: "put the start of the notes (selected, or all) on the grid (Q)"});
	const legB = mk("button", {textContent: "Legato", title: "make the notes (selected, or all) last until the next one (L)"});
	const clear = mk("button", {textContent: "Clear", title: "take every note away"});
	const selInfo = mk("span", {className: "nr-sel"});
	const viewSel = mk("select", {title: "the clip edited: a song plays the clip of the section it is in; the loop plays where a section has no clip"});
	const copySel = mk("select", {title: "copy this clip to other sections"});
	const dropB = mk("button", {textContent: "Delete clip", title: "take the clip of this section away (the loop plays there instead)"});
	const viewLab = mk("label", {className: "nr-view"}, "Clip ", viewSel);
	bar.append(viewLab, copySel, dropB, ...Object.values(toolBtns), mk("label", {}, "Grid ", snapSel), mk("label", {}, "Beats ", lenIn), dbl, half, mk("label", {}, "Zoom ", zoom));
	bar2.append(undoB, redoB, quantB, legB, clear, selInfo);
	setTool("select");

	// --- the melody panel ---
	const MODES = {major: [0, 2, 4, 5, 7, 9, 11], minor: [0, 2, 3, 5, 7, 8, 10], dorian: [0, 2, 3, 5, 7, 9, 10], mixolydian: [0, 2, 4, 5, 7, 9, 10], lydian: [0, 2, 4, 6, 7, 9, 11],
		phrygian: [0, 1, 3, 5, 7, 8, 10], "harmonic minor": [0, 2, 3, 5, 7, 8, 11], "major pentatonic": [0, 2, 4, 7, 9], "minor pentatonic": [0, 3, 5, 7, 10], blues: [0, 3, 5, 6, 7, 10]};
	const KEYS = ["C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"];
	const gp = {grammar: "S -> A A' B A~\n# A B C D are phrases; ' a variation, ~ upside down, < backwards, ^ v a step up or down, ? ends open\n# an L-system: axiom: A, iterate: 3, A -> A B A', B -> A", form: "free", contour: "free", rhythm: "mixed", phrase: 4, density: 0.5, rest: 0.15, steps: 0.7, chord: 0.7, low: 55, high: 79, root: 0, mode: ""};
	const pick = (label, key, options, title) => {
		const sel = mk("select", {title});
		for (const [v, t] of options) sel.append(mk("option", {value: v, textContent: t, selected: String(gp[key]) === String(v)}));
		sel.addEventListener("change", () => { gp[key] = typeof gp[key] === "number" ? Number(sel.value) : sel.value; });
		return mk("label", {}, label + " ", sel);
	};
	const slide = (label, key, title) => {
		const i = mk("input", {type: "range", min: 0, max: 1, step: 0.05, value: gp[key], title});
		i.addEventListener("input", () => { gp[key] = Number(i.value); });
		return mk("label", {}, label + " ", i);
	};
	const rangeIn = key => { const i = mk("input", {type: "number", min: 0, max: 127, step: 1, value: gp[key], title: "key (60 is middle C)"}); i.addEventListener("change", () => { gp[key] = Math.max(0, Math.min(127, Number(i.value) || 0)); }); return i; };
	const modeSel = mk("select", {title: "the scale the melody is in"}, mk("option", {value: "", textContent: "the harmony's"}), ...Object.keys(MODES).map(m => mk("option", {value: m, textContent: m})));
	const rootSel = mk("select", {title: "the key of the scale"}, ...KEYS.map((k, i) => mk("option", {value: i, textContent: k})));
	modeSel.addEventListener("change", () => { gp.mode = modeSel.value; rootSel.disabled = !gp.mode; });
	rootSel.addEventListener("change", () => { gp.root = Number(rootSel.value); });
	rootSel.disabled = true;
	const genB = mk("button", {textContent: "Generate", title: "make a melody for the region of the ruler, the notes selected, or the whole clip (undo takes it away)"});
	const varB = mk("button", {textContent: "Variations", title: "make six melodies to hear and choose from"});
	const allB = mk("button", {textContent: "Every section", title: "make a melody for the clip of every section of the song, over the chords of each (clips that are there are replaced)"});
	allB.addEventListener("click", () => {
		if (have.length > 1 && !confirm("Replace the clips of the sections?")) return;
		api.message(p.name, {genall: {seed: genSeed, params: {...gp, grid: snap > 0 ? snap : 0.25, velocity: lastVel}}});
		genSeed++;
	});
	let counterEvery = 0;
	const everySel = mk("select", {title: "how often the counter-line plays a note"}, ...[[0, "a note under each"], [1, "a note a beat"], [2, "every 2 beats"], [4, "a note a bar"]].map(([v, t]) => mk("option", {value: v, textContent: t})));
	everySel.addEventListener("change", () => { counterEvery = Number(everySel.value); });
	const counterB = mk("button", {textContent: "Counter-line", title: "add a second voice under the notes selected (or the region, or the whole clip): consonant with them, moving against them, with no parallel fifths or octaves"});
	counterB.addEventListener("click", () => {
		const ts = targets(); if (!ts.length) return;
		const from = Math.min(...ts.map(n => n.s)), to = Math.min(length, Math.max(...ts.map(n => n.s + n.l)));
		genId++; genPending = {id: genId, apply: "add", from, to};
		api.message(p.name, {counter: {id: genId, from, to, notes: ts.map(n => [n.s, n.l, n.k, n.v]), every: counterEvery, motion: 0.6, params: {...gp, velocity: lastVel}}});
	});
	// grooves: the feel of the sixteenths of a bar (some later, some softer), applied to the notes on the grid, or taken from the notes selected
	const grooveSel = mk("select", {title: "a groove: the feel of the sixteenths"});
	const fillGrooves = () => { const cur = grooveSel.value; grooveSel.replaceChildren(...grooves.map((g, i) => mk("option", {value: i, textContent: g.name}))); if (cur) grooveSel.value = cur; };
	const grooveB = mk("button", {textContent: "Groove", title: "put the groove on the notes selected (or the region, or all): the sixteenths a little late and soft as it says"});
	grooveB.addEventListener("click", () => {
		const g = grooves[Number(grooveSel.value)]; const ts = targets(); if (!g || !ts.length || !g.shift.length) return;
		for (const n of ts) {
			const k = Math.round(n.s * 4); if (Math.abs(n.s * 4 - k) > 0.2) continue; // only notes on the grid
			const i = ((k % g.shift.length) + g.shift.length) % g.shift.length;
			n.s = Math.max(0, k / 4 + g.shift[i] / 4); n.v = Math.max(0.05, Math.min(1, n.v * g.vel[i]));
		}
		for (const n of notes) clipLen(n);
		commit();
	});
	const takeB = mk("button", {textContent: "Take groove", title: "take the feel of the notes selected (how late and how loud each sixteenth of a bar is) as a groove of your own, to put on other notes"});
	takeB.addEventListener("click", () => {
		const ts = targets(); if (ts.length < 4) return;
		const sh = Array(16).fill(0), vl = Array(16).fill(0), cnt = Array(16).fill(0);
		const mean = ts.reduce((a, n) => a + n.v, 0) / ts.length || 1;
		for (const n of ts) { const k = Math.round(n.s * 4), i = ((k % 16) + 16) % 16; sh[i] += n.s * 4 - k; vl[i] += n.v / mean; cnt[i]++; }
		const g = {name: "mine " + (grooves.filter(x => x.name.startsWith("mine")).length + 1), shift: sh.map((v, i) => cnt[i] ? Math.max(-0.5, Math.min(0.5, v / cnt[i])) : 0), vel: vl.map((v, i) => cnt[i] ? v / cnt[i] : 1)};
		grooves.push(g); fillGrooves(); grooveSel.value = grooves.length - 1;
	});
	const mutate = (label, title, f) => { const b = mk("button", {textContent: label, title}); b.addEventListener("click", f); return b; };
	const candEl = mk("div", {className: "nr-cands"});
	const lockBox = mk("input", {type: "checkbox", title: "the notes you put and move keep to the scale"});
	lockBox.addEventListener("change", () => { scaleLock = lockBox.checked; });
	const panelEl = mk("details", {className: "nr-gen"}, mk("summary", {textContent: "Melody generator (make notes for this clip)"}),
		mk("div", {className: "nr-bar"}, pick("Form", "form", [["free", "free"], ["AA'", "a period (A A')"], ["ABAB", "A B A B"], ["AABA", "A A' B A"], ["call", "question and answer"], ["grammar", "a grammar or L-system…"]], "how the phrases repeat and vary"),
			pick("Contour", "contour", [["free", "free"], ["arch", "arch"], ["rise", "rise"], ["fall", "fall"], ["wave", "wave"]], "the shape of the line over a phrase"),
			pick("Rhythm", "rhythm", [["mixed", "mixed"], ["even", "even"], ["syncopated", "syncopated"], ["dotted", "dotted"], ["triplet", "triplets"]], "the rhythm of the notes"),
			pick("Phrase", "phrase", [[2, "2 beats"], [4, "4 beats"], [8, "8 beats"], [16, "16 beats"]], "how long a phrase is")),
		mk("div", {className: "nr-bar"}, slide("Busy", "density", "how many notes"), slide("Rests", "rest", "how often it rests"), slide("Smooth", "steps", "steps against leaps"), slide("Chord tones", "chord", "how much the notes on the beat keep to the chord")),
		mk("div", {className: "nr-bar"}, mk("label", {}, "Scale ", rootSel, modeSel), mk("label", {}, "From ", rangeIn("low")), mk("label", {}, "to ", rangeIn("high")), mk("label", {}, lockBox, " keep to the scale")),
		mk("div", {className: "nr-bar"}, genB, varB, allB, counterB, everySel,
			mutate("Invert", "turn the notes selected upside down (in the scale)", () => mutateNotes("invert")), mutate("Reverse", "play the notes selected backwards", () => mutateNotes("reverse")),
			mutate("Step up", "move the notes selected a step up the scale", () => mutateNotes("up")), mutate("Step down", "move them a step down", () => mutateNotes("down")),
			grooveSel, grooveB, takeB, mutate("Humanize", "a little off the grid and the loudness", () => mutateNotes("humanize")), mutate("Thin out", "take some of the short notes away", () => mutateNotes("thin"))),
		candEl);
	const gramBox = mk("textarea", {className: "nr-gram", rows: 4, spellcheck: false, value: gp.grammar, title: "rules that write the form of the melody: S -> A A' B A, with ' ~ < ^ v ? after a phrase; or axiom: A, iterate: 3 and A -> A B A' for an L-system"});
	const gramErr = mk("div", {className: "nr-gram-err"});
	gramBox.addEventListener("input", () => { gp.grammar = gramBox.value; });
	const gramRow = mk("div", {className: "nr-bar nr-gram-row"}, gramBox, gramErr);
	const showGram = () => { gramRow.style.display = gp.form === "grammar" ? "" : "none"; };
	panelEl.querySelector("select").addEventListener("change", showGram);
	panelEl.querySelector(".nr-bar").after(gramRow);
	showGram();
	canvas.after(panelEl); // under the roll, so the notes are the first thing to see

	// the scale the melody and the keys lit are in: its own, or the harmony's
	const scalePcs = () => gp.mode ? MODES[gp.mode].map(x => (x + gp.root) % 12) : scaleInfo ? scaleInfo.pcs : null;
	const inScale = k => { const pcs = scalePcs(); return !pcs || pcs.includes(((k % 12) + 12) % 12); };
	const snapKey = k => { if (!scaleLock || inScale(k)) return k; for (let d = 1; d < 12; d++) { if (inScale(k - d)) return k - d; if (inScale(k + d)) return k + d; } return k; };
	const stepKey = (k, n) => { // a number of steps through the scale
		const pcs = scalePcs(); if (!pcs) return k + n;
		let q = k; const dir = Math.sign(n);
		if (!inScale(q)) q = snapKey2(q);
		for (let i = 0; i < Math.abs(n); i++) { do { q += dir; } while (q > 0 && q < 127 && !inScale(q)); }
		return Math.max(0, Math.min(127, q));
	};
	const snapKey2 = k => { for (let d = 0; d < 12; d++) { if (inScale(k - d)) return k - d; if (inScale(k + d)) return k + d; } return k; };

	// --- the notes, and what is sent ---
	const toArrays = () => notes.map(n => [n.s, n.l, n.k, n.v]);
	const fromArrays = a => a.map(n => ({s: n[0], l: n[1], k: n[2], v: n[3]}));
	const key = () => JSON.stringify([length, notes.map(n => [+n.s.toFixed(4), +n.l.toFixed(4), n.k, +n.v.toFixed(3)]).sort((a, b) => a[0] - b[0] || a[2] - b[2])]);
	base = key();
	let heard = -1; // the key last played (a note moved over keys plays each once)
	const hear = (k, v) => { if (k === heard) return; heard = k; api.message(p.name, {play: {key: k, vel: v === undefined ? lastVel : v}}); };
	const send = () => {
		hold = performance.now() + 700;
		api.message(p.name, {set: {clip: viewName, length, notes: toArrays()}});
	};
	// commit: what changed becomes one step of the history, and goes to the server
	const state = () => ({length, notes: notes.map(n => ({...n}))});
	let was = state(); // the notes before the step that is under way
	const commit = () => {
		const now = key();
		if (now === base) return;
		undo.push(was); if (undo.length > 100) undo.shift();
		redo.length = 0;
		base = now; was = state();
		send(); updateButtons();
	};
	const restore = s => {
		notes = s.notes.map(n => ({...n})); length = s.length; lenIn.value = length;
		sel.clear(); base = key(); was = state(); fit(); send(); updateButtons();
	};
	const doUndo = () => { if (!undo.length) return; redo.push(state()); restore(undo.pop()); };
	const doRedo = () => { if (!redo.length) return; undo.push(state()); restore(redo.pop()); };
	const updateButtons = () => { undoB.disabled = !undo.length; redoB.disabled = !redo.length; };
	updateButtons();

	undoB.addEventListener("click", doUndo);
	redoB.addEventListener("click", doRedo);
	snapSel.addEventListener("change", () => { snap = Number(snapSel.value); });
	lenIn.addEventListener("change", () => {
		length = Math.max(1, Math.min(MAXBEATS, Math.round(Number(lenIn.value)) || 1)); lenIn.value = length;
		notes = notes.filter(n => n.s < length); for (const n of notes) n.l = Math.min(n.l, length - n.s);
		sel.clear(); fit(); commit();
	});
	dbl.addEventListener("click", () => {
		if (length * 2 > MAXBEATS) return;
		const copies = notes.map(n => ({...n, s: n.s + length}));
		length *= 2; lenIn.value = length; notes.push(...copies); fit(); commit();
	});
	half.addEventListener("click", () => {
		if (length < 2) return;
		length = Math.max(1, Math.round(length / 2)); lenIn.value = length;
		notes = notes.filter(n => n.s < length); for (const n of notes) n.l = Math.min(n.l, length - n.s);
		sel.clear(); fit(); commit();
	});
	clear.addEventListener("click", () => { notes = []; sel.clear(); region = null; commit(); });
	quantB.addEventListener("click", () => quantize());
	legB.addEventListener("click", () => legato());
	zoom.addEventListener("input", () => {
		const g = geom(), c = x0 + (g.vw / g.pxb) / 2;
		pxb = zoomPx(Number(zoom.value)); x0 = c - g.vw / pxb / 2; clampView();
	});

	// --- the clips: one for every section of a song ---
	const setView = name => {
		if (name === viewName) return;
		viewName = name; waitView = true; askedView = name;
		notes = []; sel.clear(); region = null; cursor = null; undo.length = 0; redo.length = 0; base = key(); was = state(); updateButtons();
		api.message(p.name, {view: name});
	};
	viewSel.addEventListener("change", () => setView(viewSel.value));
	copySel.addEventListener("change", () => {
		const to = copySel.value === "*" ? sections.map(s => s.name).filter(n => n !== viewName) : copySel.value ? [copySel.value] : [];
		copySel.value = "";
		if (to.length) { commitNow(); api.message(p.name, {copy: {from: viewName, to}}); }
	});
	dropB.addEventListener("click", () => { waitView = true; hold = 0; api.message(p.name, {drop: viewName}); });
	const commitNow = () => { if (key() !== base) commit(); };
	let optionsKey = "";
	const showOptions = () => { // the choices of the clip and of the copy: the sections of the song, a mark on the ones that have a clip
		const k = JSON.stringify([sections.map(s => s.name), have, viewName]);
		if (k === optionsKey) return;
		optionsKey = k;
		viewSel.replaceChildren(mk("option", {value: "", textContent: (have.includes("") && viewName !== "" ? "● " : "") + "the loop"}));
		for (const s of sections) viewSel.append(mk("option", {value: s.name, textContent: (have.includes(s.name) ? "● " : "") + s.name}));
		viewSel.value = viewName;
		copySel.replaceChildren(mk("option", {value: "", textContent: "copy to…"}));
		if (sections.length > 1) copySel.append(mk("option", {value: "*", textContent: "all the others"}));
		for (const s of sections) if (s.name !== viewName) copySel.append(mk("option", {value: s.name, textContent: s.name}));
		const song = sections.length > 0;
		viewLab.style.display = copySel.style.display = dropB.style.display = song ? "" : "none";
		dropB.disabled = viewName === "" || !have.includes(viewName);
	};
	showOptions();

	// --- the view: the beats shown (zoom, scroll), the keys shown ---
	const name = k => ["C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"][k % 12] + (Math.floor(k / 12) - 1);
	const black = k => [1, 3, 6, 8, 10].includes(k % 12);
	const geom = () => {
		const w = canvas.clientWidth, h = canvas.clientHeight;
		const vw = w - gutter, areaTop = rulerH, areaH = h - rulerH - velH - gap, rows = Math.max(4, Math.floor(areaH / rowH));
		return {w, h, vw, pxb: pxb || vw / length, areaTop, areaH, rows, velTop: h - velH, rowH: areaH / rows};
	};
	const minPx = () => (canvas.clientWidth - gutter) / length; // the whole loop fits
	const maxPx = 240;
	const zoomPx = z => { const a = minPx(); return a * Math.pow(maxPx / a, z); };
	const fit = () => { // the loop changed: the view keeps within it
		const a = minPx(); if (!pxb || pxb < a) pxb = a;
		clampView(); zoom.value = Math.log(pxb / a) / Math.log(maxPx / a) || 0;
	};
	const clampView = () => {
		const vw = canvas.clientWidth - gutter, span = vw / pxb;
		x0 = Math.max(0, Math.min(x0, Math.max(0, length - span)));
	};
	const xOf = (g, b) => gutter + (b - x0) * g.pxb;
	const bOf = (g, x) => x0 + (x - gutter) / g.pxb;
	const keyAt = (g, y) => lo + g.rows - 1 - Math.floor((y - g.areaTop) / g.rowH);
	const yOf = (g, k) => g.areaTop + (lo + g.rows - 1 - k) * g.rowH;
	const snapTo = b => snap > 0 ? Math.round(b / snap) * snap : Math.round(b * 1000) / 1000;
	const snapDown = b => snap > 0 ? Math.floor(b / snap + 1e-9) * snap : b;
	const minLen = () => snap > 0 ? snap : 0.05;
	const head = () => { const t = (performance.now() - at) / 1000; return ((pos + bps * t) % length + length) % length; };

	let colors = {};
	const restyle = () => { colors = {bg: api.css("track"), ink: api.css("ink"), muted: api.css("muted"), line: api.css("line"), accent: api.css("accent")}; };
	restyle();

	const outOfChord = n => { // a note that is no tone of the chord it starts in (a harmony module: its chords are shown)
		for (const sp of chords) if (n.s >= sp.at - 1e-6 && n.s < sp.at + sp.len - 1e-6) return !sp.pcs.includes(n.k % 12);
		return false;
	};
	const noteAt = (g, x, y) => { // the note at a place (the last drawn is on top)
		const k = keyAt(g, y), b = bOf(g, x);
		for (let i = notes.length - 1; i >= 0; i--) { const n = notes[i]; if (n.k === k && b >= n.s && b <= n.s + n.l) return n; }
		return null;
	};
	const onEdge = (g, n, x) => xOf(g, n.s + n.l) - x <= Math.min(8, n.l * g.pxb / 3);

	// --- drawing ---
	const draw = () => {
		if (!canvas.isConnected) return;
		const g = geom(), dpr = window.devicePixelRatio || 1;
		if (!pxb) fit();
		if (showKeys !== null && g.h > 0) { lo = Math.max(0, Math.min(128 - g.rows, showKeys - Math.floor(g.rows / 2))); showKeys = null; }
		if (canvas.width !== Math.round(g.w * dpr) || canvas.height !== Math.round(g.h * dpr)) {
			canvas.width = Math.round(g.w * dpr); canvas.height = Math.round(g.h * dpr);
		}
		const c = canvas.getContext("2d");
		c.setTransform(dpr, 0, 0, dpr, 0, 0);
		c.clearRect(0, 0, g.w, g.h);
		const live = pos >= 0, hd = live ? head() : -1;
		c.save(); c.beginPath(); c.rect(gutter, 0, g.vw, g.h); c.clip();
		// the rows
		for (let k = lo; k < lo + g.rows; k++) {
			const y = yOf(g, k);
			c.globalAlpha = black(k) ? 0.5 : 0.18; c.fillStyle = colors.line; c.fillRect(gutter, y, g.vw, g.rowH);
			if (scalePcs() && inScale(k)) { c.globalAlpha = 0.06; c.fillStyle = colors.accent; c.fillRect(gutter, y, g.vw, g.rowH); }
			if (k % 12 === 0) { c.globalAlpha = 0.5; c.fillRect(gutter, y + g.rowH - 1, g.vw, 1); }
		}
		c.globalAlpha = 1;
		// the chords of the harmony: the keys that are tones of the chord are lit under it
		for (const sp of chords) {
			const x1 = xOf(g, sp.at), x2 = xOf(g, Math.min(length, sp.at + sp.len));
			if (x2 < gutter || x1 > g.w) continue;
			c.fillStyle = colors.accent;
			for (let k = lo; k < lo + g.rows; k++) if (sp.pcs.includes(k % 12)) { c.globalAlpha = 0.13; c.fillRect(x1, yOf(g, k), x2 - x1, g.rowH); }
			c.globalAlpha = 0.45; c.fillRect(Math.round(x1), g.areaTop, 1, g.areaH);
		}
		c.globalAlpha = 1;
		// the region, under the grid
		if (region) { c.globalAlpha = 0.16; c.fillStyle = colors.accent; c.fillRect(xOf(g, region[0]), g.areaTop, (region[1] - region[0]) * g.pxb, g.h - g.areaTop); c.globalAlpha = 1; }
		// the grid: the grid lines, the beats, the bars (4 beats); the loop's end
		const visFrom = Math.floor(x0), visTo = Math.min(length, Math.ceil(x0 + g.vw / g.pxb));
		if (snap > 0 && snap * g.pxb >= 6) {
			c.globalAlpha = 0.18; c.fillStyle = colors.line;
			for (let b = Math.ceil(x0 / snap) * snap; b <= visTo + 1e-9; b += snap) c.fillRect(Math.round(xOf(g, b)), g.areaTop, 1, g.areaH);
		}
		for (let b = visFrom; b <= visTo; b++) {
			c.globalAlpha = b % 4 === 0 ? 0.9 : 0.4; c.fillStyle = colors.line;
			c.fillRect(Math.round(xOf(g, b)), g.areaTop, 1, g.h - g.areaTop);
		}
		c.globalAlpha = 0.35; c.fillStyle = colors.bg; c.fillRect(xOf(g, length), g.areaTop, g.w, g.h - g.areaTop); // past the end of the loop
		c.globalAlpha = 1;
		// the notes
		for (const n of notes) {
			if (n.k < lo || n.k >= lo + g.rows) continue;
			const x = xOf(g, n.s), w = Math.max(3, n.l * g.pxb - 1), y = yOf(g, n.k);
			if (x + w < gutter || x > g.w) continue;
			const on = live && n.s <= hd && hd < n.s + n.l, s = sel.has(n), out = outOfChord(n);
			c.globalAlpha = (on ? 1 : 0.5 + 0.4 * n.v) * (out ? 0.7 : 1);
			c.fillStyle = colors.accent; c.fillRect(x, y + 1, w, g.rowH - 2);
			c.globalAlpha = 1;
			if (out) { c.strokeStyle = colors.muted; c.setLineDash([3, 2]); c.strokeRect(x + 0.5, y + 1.5, w - 1, g.rowH - 3); c.setLineDash([]); }
			if (s) { c.strokeStyle = colors.ink; c.lineWidth = 2; c.strokeRect(x + 1, y + 2, w - 2, g.rowH - 4); }
			if (w > 28 && g.rowH >= 11) { c.font = "9px sans-serif"; c.textBaseline = "middle"; c.fillStyle = colors.bg; c.globalAlpha = 0.85; c.fillText(name(n.k), x + 3, y + g.rowH / 2); c.globalAlpha = 1; }
		}
		// the velocity lane
		c.fillStyle = colors.line; c.globalAlpha = 0.25; c.fillRect(gutter, g.velTop, g.vw, velH); c.globalAlpha = 1;
		for (const n of notes) {
			const x = xOf(g, n.s); if (x < gutter - 4 || x > g.w) continue;
			const h = Math.max(2, n.v * (velH - 8));
			c.fillStyle = sel.has(n) ? colors.ink : colors.accent; c.globalAlpha = sel.has(n) ? 1 : 0.7;
			c.fillRect(x, g.velTop + velH - 2 - h, Math.min(4, Math.max(2, n.l * g.pxb - 1)), h);
		}
		c.globalAlpha = 1;
		// the cursor, the rubber band, the playhead
		if (cursor !== null) { c.fillStyle = colors.ink; c.globalAlpha = 0.6; c.fillRect(Math.round(xOf(g, cursor)) - 1, g.areaTop, 1, g.h - g.areaTop); c.globalAlpha = 1; }
		if (band) {
			const bx = Math.min(band.x1, band.x2), by = Math.min(band.y1, band.y2), bw = Math.abs(band.x2 - band.x1), bh = Math.abs(band.y2 - band.y1);
			c.fillStyle = colors.accent; c.globalAlpha = 0.15; c.fillRect(bx, by, bw, bh); c.globalAlpha = 1; c.strokeStyle = colors.accent; c.strokeRect(bx + 0.5, by + 0.5, bw, bh);
		}
		if (live) { c.fillStyle = colors.accent; c.fillRect(Math.round(xOf(g, hd)) - 1, rulerH - 4, 2, g.h - rulerH + 4); }
		c.restore();
		// the ruler
		c.fillStyle = colors.line; c.globalAlpha = 0.35; c.fillRect(gutter, 0, g.vw, rulerH); c.globalAlpha = 1;
		if (region) { c.fillStyle = colors.accent; c.globalAlpha = 0.5; c.fillRect(xOf(g, region[0]), 0, (region[1] - region[0]) * g.pxb, rulerH); c.globalAlpha = 1; }
		c.save(); c.beginPath(); c.rect(gutter, 0, g.vw, rulerH); c.clip();
		c.font = "10px sans-serif"; c.textBaseline = "middle"; c.fillStyle = colors.ink;
		for (let b = visFrom; b <= visTo; b++) {
			const x = Math.round(xOf(g, b));
			c.globalAlpha = 0.5; c.fillRect(x, b % 4 === 0 ? 3 : rulerH - 7, 1, b % 4 === 0 ? rulerH - 3 : 7);
			if (b % 4 === 0 || g.pxb > 40) { c.globalAlpha = 0.9; c.fillText(b % 4 === 0 ? String(b / 4 + 1) : "." + (b % 4 + 1), x + 3, 8); }
		}
		for (const sp of chords) { // the chords, in the lower half of the ruler
			const x = Math.round(xOf(g, sp.at));
			c.globalAlpha = 0.5; c.fillRect(x, 15, 1, rulerH - 15);
			c.globalAlpha = 0.95; c.font = "bold 11px sans-serif"; c.fillText(sp.name, x + 3, 25); c.font = "10px sans-serif";
		}
		c.restore(); c.globalAlpha = 1;
		// the keys
		c.font = "10px sans-serif"; c.textBaseline = "middle";
		for (let k = lo; k < lo + g.rows; k++) {
			const y = yOf(g, k);
			c.fillStyle = colors.ink; c.globalAlpha = 0.8; c.fillRect(0, y + 0.5, gutter - 2, g.rowH - 1);
			if (black(k)) { c.fillStyle = colors.bg; c.globalAlpha = 0.85; c.fillRect(0, y + 0.5, gutter * 0.62, g.rowH - 1); }
			c.globalAlpha = 1;
			if (k % 12 === 0) { c.fillStyle = colors.bg; c.fillText(name(k), gutter - 24, y + g.rowH / 2); }
		}
		c.fillStyle = colors.muted; c.fillText("velocity", 4, g.velTop + 10);
		const n1 = sel.size;
		selInfo.textContent = n1 === 1 ? (() => { const n = [...sel][0]; return `${name(n.k)}  ${+n.s.toFixed(3)} + ${+n.l.toFixed(3)}  v${Math.round(n.v * 127)}`; })()
			: n1 > 1 ? `${n1} notes` : region ? `region ${+region[0].toFixed(3)} – ${+region[1].toFixed(3)}` : `${notes.length} notes`;
		requestAnimationFrame(draw);
	};
	requestAnimationFrame(draw);

	// --- editing ---
	const clipLen = n => { n.s = Math.max(0, Math.min(n.s, length - minLen())); n.l = Math.max(0.01, Math.min(n.l, length - n.s)); n.k = Math.max(0, Math.min(127, n.k)); };
	const targets = () => sel.size ? [...sel] : region ? notes.filter(n => n.s >= region[0] - 1e-9 && n.s < region[1] - 1e-9) : notes;
	const quantize = () => { for (const n of targets()) { n.s = Math.min(snapTo(n.s), length - minLen()); } commit(); };
	const legato = () => {
		const ts = targets(), starts = [...new Set(notes.map(n => n.s))].sort((a, b) => a - b);
		for (const n of ts) { const next = starts.find(s => s > n.s + 1e-6); n.l = Math.max(0.01, (next === undefined ? length : next) - n.s); }
		commit();
	};
	const removeSelected = () => {
		if (sel.size) { notes = notes.filter(n => !sel.has(n)); sel.clear(); commit(); }
		else if (region) { notes = notes.filter(n => !(n.s >= region[0] - 1e-9 && n.s < region[1] - 1e-9)); commit(); }
	};
	const clipboard = () => (window.__psalmRollClipboard = window.__psalmRollClipboard || {notes: [], span: 0});
	const copy = () => {
		const ts = sel.size ? [...sel] : region ? targets() : [];
		if (!ts.length) return false;
		const from = sel.size ? Math.min(...ts.map(n => n.s)) : region[0];
		const span = sel.size ? Math.max(...ts.map(n => n.s + n.l)) - from : region[1] - region[0];
		window.__psalmRollClipboard = {notes: ts.map(n => ({s: n.s - from, l: n.l, k: n.k, v: n.v})), span};
		return true;
	};
	// put notes in, extending the loop when they run past its end (to the most it can be)
	const insert = (list, shift) => {
		const out = list.map(n => ({...n, s: n.s + shift}));
		const end = Math.max(0, ...out.map(n => n.s + n.l));
		if (end > length + 1e-9 && length < MAXBEATS) { length = Math.min(MAXBEATS, Math.ceil(end)); lenIn.value = length; fit(); }
		const fits = out.filter(n => n.s < length - 1e-9);
		for (const n of fits) n.l = Math.min(n.l, length - n.s);
		notes.push(...fits);
		return fits;
	};
	const paste = () => {
		const cb = clipboard(); if (!cb.notes.length) return;
		const to = cursor !== null ? cursor : region ? region[0] : 0;
		const put = insert(cb.notes, to);
		sel.clear(); for (const n of put) sel.add(n);
		cursor = Math.min(length, to + cb.span);
		commit();
	};
	const duplicate = () => {
		if (sel.size) {
			const ts = [...sel], from = Math.min(...ts.map(n => n.s)), end = Math.max(...ts.map(n => n.s + n.l));
			const span = snap > 0 ? Math.ceil((end - from) / snap - 1e-9) * snap : end - from;
			const put = insert(ts, span);
			sel.clear(); for (const n of put) sel.add(n);
			commit();
		} else if (region) {
			const [a, b] = region, ts = notes.filter(n => n.s >= a - 1e-9 && n.s < b - 1e-9);
			insert(ts, b - a);
			region = [b, Math.min(length, b + (b - a))]; cursor = region[0];
			if (region[1] <= region[0]) region = null;
			commit();
		}
	};
	const move = (dk, db) => {
		const ts = [...sel]; if (!ts.length) return;
		const mins = Math.min(...ts.map(n => n.s)), maxe = Math.max(...ts.map(n => n.s + n.l)), mink = Math.min(...ts.map(n => n.k)), maxk = Math.max(...ts.map(n => n.k));
		db = Math.max(-mins, Math.min(db, length - maxe)); dk = Math.max(-mink, Math.min(dk, 127 - maxk));
		for (const n of ts) { n.s += db; n.k += dk; }
		if (dk !== 0) { heard = -1; hear(Math.max(...ts.map(n => n.k))); }
		commit();
	};

	// --- melodies ---
	const genSpan = () => region ? [region[0], region[1]] : sel.size ? [Math.floor(Math.min(...[...sel].map(n => n.s))), Math.min(length, Math.ceil(Math.max(...[...sel].map(n => n.s + n.l))))] : [0, length];
	const ask = (count, apply) => {
		const [from, to] = genSpan();
		genId++; genPending = {id: genId, apply, from, to};
		const params = {...gp, grid: snap > 0 ? snap : 0.25, velocity: lastVel};
		api.message(p.name, {gen: {id: genId, from, to, count, seed: genSeed, params}});
		genSeed += count;
	};
	const replaceRegion = (from, to, melody) => { // the notes of a stretch are the melody's
		notes = notes.filter(n => !(n.s >= from - 1e-9 && n.s < to - 1e-9));
		const made = melody.map(m => ({s: m[0], l: m[1], k: m[2], v: m[3]}));
		notes.push(...made); sel.clear(); for (const n of made) sel.add(n);
		commit();
	};
	const drawCandidate = (cv, melody, from, to) => {
		const c = cv.getContext("2d"), w = cv.width, h = cv.height;
		c.clearRect(0, 0, w, h);
		if (!melody.length) return;
		const lo2 = Math.min(...melody.map(m => m[2])), hi2 = Math.max(...melody.map(m => m[2])), span = Math.max(6, hi2 - lo2 + 1);
		c.fillStyle = colors.accent;
		for (const m of melody) c.fillRect((m[0] - from) / (to - from) * w, h - 4 - (m[2] - lo2 + 0.5) / span * (h - 8), Math.max(2, m[1] / (to - from) * w - 1), 3);
	};
	const showCandidates = (melodies, from, to) => {
		candEl.replaceChildren();
		melodies.forEach((melody, i) => {
			const cv = mk("canvas", {width: 150, height: 46, className: "nr-cand-cv"});
			const hear = mk("button", {textContent: "Hear", title: "play it"});
			hear.addEventListener("click", () => api.message(p.name, {preview: {notes: melody.map(m => [m[0] - from, m[1], m[2], m[3]]), length: to - from}}));
			const use = mk("button", {textContent: "Use", title: "put it in the clip"});
			use.addEventListener("click", () => replaceRegion(from, to, melody));
			candEl.append(mk("div", {className: "nr-cand"}, cv, mk("div", {className: "nr-cand-row"}, hear, use)));
			drawCandidate(cv, melody, from, to);
		});
	};
	genB.addEventListener("click", () => ask(1, true));
	varB.addEventListener("click", () => ask(6, false));
	const mutateNotes = how => {
		const ts = targets(); if (!ts.length) return;
		const first = [...ts].sort((a, b) => a.s - b.s)[0];
		const a = Math.min(...ts.map(n => n.s)), b = Math.max(...ts.map(n => n.s + n.l));
		switch (how) {
		case "invert": { const pivot = first.k; for (const n of ts) n.k = Math.max(0, Math.min(127, snapKey2(2 * pivot - n.k))); break; }
		case "reverse": for (const n of ts) n.s = a + b - (n.s + n.l); break;
		case "up": for (const n of ts) n.k = stepKey(n.k, 1); break;
		case "down": for (const n of ts) n.k = stepKey(n.k, -1); break;
		case "humanize": for (const n of ts) { n.s = Math.max(0, n.s + (Math.random() - 0.5) * 0.06); n.v = Math.max(0.05, Math.min(1, n.v + (Math.random() - 0.5) * 0.16)); } break;
		case "thin": { const drop = new Set(ts.filter(n => n.l < 0.75 && Math.abs(n.s - Math.round(n.s)) > 1e-6 && Math.random() < 0.4)); notes = notes.filter(n => !drop.has(n)); for (const n of drop) sel.delete(n); break; }
		}
		for (const n of notes) clipLen(n);
		commit();
	};

	// --- the mouse ---
	const where = e => { const r = canvas.getBoundingClientRect(); return [e.clientX - r.left, e.clientY - r.top]; };
	const zone = (g, y) => y < rulerH ? "ruler" : y >= g.velTop ? "vel" : "notes";
	const inBand = (g, r, n) => { // does a note touch a rectangle (pixels)
		const x = xOf(g, n.s), w = Math.max(3, n.l * g.pxb), y = yOf(g, n.k);
		return x < r.x2 && x + w > r.x1 && y < r.y2 && y + g.rowH > r.y1;
	};
	const eraseAt = (g, x, y) => { const n = noteAt(g, x, y); if (n) { notes.splice(notes.indexOf(n), 1); sel.delete(n); drag.moved = true; } };

	canvas.addEventListener("contextmenu", e => {
		e.preventDefault();
		const g = geom(), [x, y] = where(e), n = zone(g, y) === "notes" ? noteAt(g, x, y) : null;
		if (n) { notes.splice(notes.indexOf(n), 1); sel.delete(n); commit(); }
	});
	canvas.addEventListener("dblclick", e => {
		const g = geom(), [x, y] = where(e);
		if (zone(g, y) === "ruler") { region = null; return; }
		if (zone(g, y) !== "notes" || x < gutter || noteAt(g, x, y)) return;
		const s = Math.max(0, Math.min(snapDown(bOf(g, x)), length - minLen()));
		const n = {s, l: Math.min(lastLen, length - s), k: snapKey(keyAt(g, y)), v: lastVel};
		notes.push(n); sel.clear(); sel.add(n); heard = -1; hear(n.k); commit();
	});
	canvas.addEventListener("pointerdown", e => {
		if (e.button !== 0) return;
		canvas.focus();
		const g = geom(), [x, y] = where(e), z = zone(g, y);
		if (x < gutter) { if (z === "notes") { heard = -1; hear(keyAt(g, y), 0.8); } return; } // a key of the keyboard: hear it
		const b = bOf(g, x);
		if (z === "ruler") {
			const s = Math.max(0, Math.min(snapTo(b), length));
			drag = {mode: "region", from: s, moved: false}; cursor = s; sel.clear();
			canvas.setPointerCapture(e.pointerId); return;
		}
		if (z === "vel") { // a bar of the velocity lane: set it, drag to paint across the notes
			drag = {mode: "vel", moved: false};
			setVel(g, x, y, e.shiftKey);
			canvas.setPointerCapture(e.pointerId); return;
		}
		let n = noteAt(g, x, y);
		if (tool === "erase") { drag = {mode: "erase", moved: false}; eraseAt(g, x, y); canvas.setPointerCapture(e.pointerId); return; }
		if (!n && tool === "draw") { // a new note: dragging makes it longer
			const s = Math.max(0, Math.min(snapDown(b), length - minLen()));
			n = {s, l: Math.min(lastLen, length - s), k: snapKey(keyAt(g, y)), v: lastVel};
			notes.push(n); sel.clear(); sel.add(n); cursor = s; heard = -1; hear(n.k);
			drag = {mode: "resize", fresh: true, moved: false, grab: n, b0: b, orig: [{n, s: n.s, l: n.l}]};
			canvas.setPointerCapture(e.pointerId); return;
		}
		if (!n) { // empty space: a rectangle that selects (shift keeps what is selected)
			if (!e.shiftKey) sel.clear();
			cursor = Math.max(0, Math.min(snapTo(b), length)); region = null;
			band = {x1: x, y1: y, x2: x, y2: y, keep: new Set(sel)};
			drag = {mode: "band", moved: false};
			canvas.setPointerCapture(e.pointerId); return;
		}
		if (e.shiftKey) { if (sel.has(n)) { sel.delete(n); return; } sel.add(n); }
		else if (!sel.has(n)) { sel.clear(); sel.add(n); }
		lastVel = n.v; heard = -1; hear(n.k, n.v);
		if (e.altKey && !onEdge(g, n, x)) { // a copy is dragged, the notes stay
			const origs = [...sel], copies = origs.map(m => ({...m})), at0 = origs.indexOf(n);
			notes.push(...copies); sel.clear(); for (const m of copies) sel.add(m); n = copies[Math.max(0, at0)];
		}
		const orig = [...sel].map(m => ({n: m, s: m.s, l: m.l, k: m.k}));
		drag = {mode: onEdge(g, n, x) ? "resize" : "move", moved: false, grab: n, b0: b, k0: keyAt(g, y), orig, copied: e.altKey};
		canvas.setPointerCapture(e.pointerId);
	});
	const setVel = (g, x, y, all) => {
		const v = Math.max(0.01, Math.min(1, (g.velTop + velH - 2 - y) / (velH - 8)));
		const b = bOf(g, x), tol = Math.max(3 / g.pxb, 0.02);
		let hit = notes.filter(n => Math.abs(n.s - b) <= tol).sort((a, c) => Math.abs(a.s - b) - Math.abs(c.s - b));
		if (!hit.length) return;
		const target = hit[0];
		if (sel.has(target) && sel.size > 1 && !all) { // the notes selected move together by the same amount
			const d = v - target.v;
			for (const n of sel) n.v = Math.max(0.01, Math.min(1, n.v + d));
		} else for (const n of all ? notes.filter(m => sel.size === 0 || sel.has(m)) : [target]) n.v = v;
		lastVel = target.v; drag && (drag.moved = true);
	};
	canvas.addEventListener("pointermove", e => {
		const g = geom(), [x, y] = where(e);
		if (!drag) { // the cursor at a note's end
			const n = zone(g, y) === "notes" && x >= gutter ? noteAt(g, x, y) : null;
			canvas.style.cursor = tool === "erase" ? "not-allowed" : n ? (onEdge(g, n, x) ? "ew-resize" : "grab") : zone(g, y) === "vel" ? "ns-resize" : zone(g, y) === "ruler" ? "text" : "crosshair";
			return;
		}
		const b = bOf(g, x);
		switch (drag.mode) {
		case "region": {
			const s = Math.max(0, Math.min(snapTo(b), length));
			if (Math.abs(s - drag.from) > 1e-9 || region) { region = [Math.min(s, drag.from), Math.max(s, drag.from)]; drag.moved = true; }
			if (region && region[1] - region[0] < 1e-9) region = null;
			break;
		}
		case "vel": setVel(g, x, y, false); break;
		case "erase": eraseAt(g, x, y); break;
		case "band":
			band.x2 = x; band.y2 = y; drag.moved = true;
			{ const r = {x1: Math.min(band.x1, band.x2), x2: Math.max(band.x1, band.x2), y1: Math.min(band.y1, band.y2), y2: Math.max(band.y1, band.y2)};
			  sel.clear(); for (const n of band.keep) sel.add(n); for (const n of notes) if (inBand(g, r, n)) sel.add(n); }
			break;
		case "resize": {
			const o = drag.orig.find(r => r.n === drag.grab) || drag.orig[0];
			if (drag.fresh && Math.abs(b - drag.b0) * g.pxb < 3) break; // a click: the note as long as the last one
			const end = drag.fresh ? snapTo(b) : snapTo(o.s + o.l + (b - drag.b0)), want = Math.max(minLen(), end - o.s), delta = want - o.l;
			for (const r of drag.orig) r.n.l = Math.max(0.01, Math.min(r.l + delta, length - r.n.s));
			drag.moved = true;
			break;
		}
		case "move": {
			const o = drag.orig.find(r => r.n === drag.grab) || drag.orig[0];
			let db = snapTo(o.s + (b - drag.b0)) - o.s, dk = keyAt(g, y) - drag.k0;
			const mins = Math.min(...drag.orig.map(r => r.s)), maxe = Math.max(...drag.orig.map(r => r.s + r.l)), mink = Math.min(...drag.orig.map(r => r.k)), maxk = Math.max(...drag.orig.map(r => r.k));
			db = Math.max(-mins, Math.min(db, length - maxe)); dk = Math.max(-mink, Math.min(dk, 127 - maxk));
			for (const r of drag.orig) { r.n.s = r.s + db; r.n.k = snapKey(r.k + dk); }
			if (dk !== 0) hear(drag.grab.k);
			drag.moved = true;
			break;
		}
		}
	});
	const up = () => {
		if (!drag) return;
		const d = drag; drag = null; band = null;
		if (d.mode === "resize" && d.moved) { const n = d.grab; if (n) lastLen = n.l; }
		if (d.mode === "resize" && d.fresh) lastLen = d.grab.l;
		if (d.mode === "region" && !d.moved) region = null; // a click: only the cursor
		for (const n of notes) clipLen(n);
		commit();
	};
	canvas.addEventListener("pointerup", up);
	canvas.addEventListener("pointercancel", up);
	canvas.addEventListener("wheel", e => {
		e.preventDefault();
		const g = geom();
		if (e.ctrlKey || e.metaKey) { // zoom around the pointer
			const [x] = where(e), b = bOf(g, Math.max(gutter, x)), z = Math.max(0, Math.min(1, Number(zoom.value) + (e.deltaY < 0 ? 0.06 : -0.06)));
			pxb = zoomPx(z); x0 = b - (Math.max(gutter, x) - gutter) / pxb; clampView(); zoom.value = z;
		} else if (e.shiftKey || Math.abs(e.deltaX) > Math.abs(e.deltaY)) {
			x0 += (e.shiftKey ? e.deltaY : e.deltaX) / g.pxb; clampView();
		} else lo = Math.max(0, Math.min(128 - g.rows, lo + (e.deltaY < 0 ? 2 : -2)));
	}, {passive: false});

	// --- the keyboard (when the roll has the focus) ---
	canvas.tabIndex = 0;
	canvas.addEventListener("keydown", e => {
		const mod = e.ctrlKey || e.metaKey, k = e.key.toLowerCase();
		const used = () => e.preventDefault();
		if (mod && k === "z") { used(); e.shiftKey ? doRedo() : doUndo(); }
		else if (mod && k === "y") { used(); doRedo(); }
		else if (mod && k === "a") { used(); sel.clear(); for (const n of notes) sel.add(n); }
		else if (mod && k === "c") { used(); copy(); }
		else if (mod && k === "x") { used(); if (copy()) removeSelected(); }
		else if (mod && k === "v") { used(); paste(); }
		else if (mod && k === "d") { used(); duplicate(); }
		else if (e.key === "Delete" || e.key === "Backspace") { used(); removeSelected(); }
		else if (e.key === "Escape") { sel.clear(); region = null; }
		else if (e.key === "ArrowUp") { used(); move(e.shiftKey ? 12 : 1, 0); }
		else if (e.key === "ArrowDown") { used(); move(e.shiftKey ? -12 : -1, 0); }
		else if (e.key === "ArrowLeft") { used(); move(0, -(e.shiftKey ? 1 : (snap || 0.25))); }
		else if (e.key === "ArrowRight") { used(); move(0, e.shiftKey ? 1 : (snap || 0.25)); }
		else if (!mod && k === "q") { used(); quantize(); }
		else if (!mod && k === "l") { used(); legato(); }
		else if (!mod && k === "v") { used(); setTool("select"); }
		else if (!mod && k === "d") { used(); setTool("draw"); }
		else if (!mod && k === "e") { used(); setTool("erase"); }
	});

	canvas.rollState = () => ({view: viewName, chords: chords.length, sections: sections.length, notes: notes.length, selected: sel.size, cursor, region, length, tool, x0, pxb, lo, band: band && {x1: band.x1, y1: band.y1, x2: band.x2, y2: band.y2}, list: notes.map(n => [n.s, n.k])}); // for the tests of the page
	return {
		el,
		show() { fit(); },
		update(st) {
			pos = st.pos; bps = st.bps; at = performance.now();
			if (st.view !== viewName) {
				if (askedView !== null) return; // the answer to a clip we have left
				viewName = st.view; waitView = true; // the server shows another clip (a patch that was made with one chosen, a patch loaded): follow it
			} else askedView = null;
			sections = st.sections || []; have = st.clips || [""]; playing = st.playing; chords = st.chords || []; scaleInfo = st.scale || null;
			if (st.grooves && st.grooves.length > grooves.filter(x => !x.name.startsWith("mine")).length) { grooves = [...st.grooves, ...grooves.filter(x => x.name.startsWith("mine"))]; fillGrooves(); }
			showOptions();
			if (first) { // the keys shown are where the notes are (once the roll has its size: the first draw)
				first = false;
				if (!st.notes.length) panelEl.open = true;
				if (st.notes.length) { const ks = st.notes.map(n => n[2]); showKeys = Math.floor((Math.min(...ks) + Math.max(...ks)) / 2); }
			}
			if (waitView) { // the clip we asked for: its notes are ours
				waitView = false; hold = 0;
				if (!st.notes.length) panelEl.open = true; // an empty clip: the way to fill it is in view
				notes = fromArrays(st.notes); length = st.length; lenIn.value = length; sel.clear(); base = key(); was = state(); undo.length = 0; redo.length = 0; fit(); updateButtons();
				return;
			}
			if (drag || performance.now() < hold) return; // our own notes until the server has them
			if (genPending && st.gen && st.gen.id === genPending.id) { // the melodies asked for are there
				const ask1 = genPending; genPending = null;
				gramErr.textContent = st.gen.err || "";
				if (st.gen.err) return;
				if (ask1.apply === "add") { const made = (st.gen.notes[0] || []).map(m => ({s: m[0], l: m[1], k: m[2], v: m[3]})); notes.push(...made); sel.clear(); for (const n of made) sel.add(n); commit(); }
				else if (ask1.apply) { if (st.gen.notes[0] && st.gen.notes[0].length) replaceRegion(ask1.from, ask1.to, st.gen.notes[0]); }
				else showCandidates(st.gen.notes, ask1.from, ask1.to);
				return;
			}
			const theirs = JSON.stringify([st.length, st.notes.map(n => [+n[0].toFixed(4), +n[1].toFixed(4), n[2], +n[3].toFixed(3)]).sort((a, b) => a[0] - b[0] || a[2] - b[2])]);
			if (theirs !== key()) { // somewhere else (a patch loaded): theirs
				notes = fromArrays(st.notes); length = st.length; lenIn.value = length; sel.clear(); region = null; cursor = null;
				base = key(); was = state(); undo.length = 0; redo.length = 0; fit(); updateButtons();
			}
		},
		restyle,
	};
});
