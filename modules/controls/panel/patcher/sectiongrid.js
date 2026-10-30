// the sections of the song as a grid: a row for every module that has something to play per section (a clock's pattern, a part's notes, a sequence's values, the chords of the harmony, a
// curve of a lane), a column for every section of the song, a pattern in every cell. A cell is the line of its section in the text of the module (`Chorus: x . x x`); one that has none
// plays what the lines without a section say (shown pale), or nothing. Click a cell to edit it (Save, Vary: a nearby pattern, Clear, Copy to another section), drag a cell onto another of
// its row to copy it. psalm.sectionGrid(send) returns {el, update(st), toggle(), onToggle}.
psalm.sectionGrid = function (send) {
	"use strict";
	const el = (tag, cls, text) => { const e = document.createElement(tag); if (cls) e.className = cls; if (text !== undefined) e.textContent = text; return e; };
	const LABEL = /^\s*([A-Za-z][\w ]*?)\s*:(.*)$/;
	const RANK = {harmony: 0, clock: 2, part: 3, sequence: 4};

	const root = el("div", "pt-grid"); root.hidden = true;
	const head = el("div", "pt-grid-head");
	const title = el("b", "", "Sections");
	const hint = el("span", "pt-grid-hint", "a pattern a cell: click to edit, drag onto another cell of the row to copy");
	const close = el("button", "pt-x", "×"); close.title = "close the sections";
	head.append(title, hint, close);
	const body = el("div", "pt-grid-body");
	const empty = el("div", "pt-grid-empty", "the patch has no song: add one (Song › song) and its sections are the columns");
	root.append(head, body, empty);

	let state = null, editing = null, drag = null;

	// what a text says: its lines, each a section's or no section's
	const parse = text => (text || "").split("\n").map(l => { const m = LABEL.exec(l); return m ? {label: m[1], text: m[2].trim()} : {label: null, text: l}; });
	const cellText = (lines, section) => { const l = lines.find(l => l.label === section); return l ? l.text : null; };
	const othersText = lines => lines.filter(l => l.label === null).map(l => l.text.trim()).filter(Boolean).join(" ");
	const write = (lines) => lines.map(l => l.label === null ? l.text : l.label + ": " + l.text).join("\n");
	const withCell = (text, section, value) => { // the text with the line of a section set (null: taken away)
		const lines = parse(text);
		const i = lines.findIndex(l => l.label === section);
		if (value === null || value === "") { if (i >= 0) lines.splice(i, 1); }
		else if (i >= 0) lines[i].text = value;
		else lines.push({label: section, text: value});
		return write(lines.filter((l, k) => l.label !== null || l.text.trim() !== "" || k === 0 && lines.length === 1));
	};

	// a nearby pattern: the rhythm functions turned or reseeded, else a hit moved a step (or two chords swapped)
	const vary = text => {
		const rnd = n => 1 + Math.floor(Math.random() * n);
		let done = false;
		const bump = arg => { // an argument that may go on with the passes: 5, +1, 3+1, 2/4
			const a = arg.trim();
			let m;
			if ((m = /^(\d*)\+(\d*)$/.exec(a))) { done = true; return String(Number(m[1] || 0) + rnd(7)) + "+" + (m[2] || "1"); }
			if ((m = /^(\d+)\/(\d+)$/.exec(a))) { done = true; return (Number(m[1]) + rnd(9)) + "/" + m[2]; }
			if (/^-?\d+$/.test(a)) { done = true; return String(Number(a) + rnd(4)); }
			return arg;
		};
		let out = text.replace(/\b(E|CA)\(([^)]*)\)/g, (all, name, args) => {
			const parts = args.split(",");
			const i = name === "E" ? 2 : 2; // the rotation of E(k,n,rot), the generation of CA(rule,n,gen)
			if (parts.length <= i) { parts.push("+"); }
			parts[i] = bump(parts[i]);
			if (parts[i] === parts[i].trim() && !/[+\/]/.test(parts[i])) parts[i] = String(Number(parts[i]) || 0);
			done = true;
			return name + "(" + parts.join(",") + ")";
		});
		out = out.replace(/\bM\(([^;)]*);([^;)]*);([^;)]*)\)/g, (all, ex, n, seed) => { done = true; return "M(" + ex + ";" + n + ";" + bump(seed) + ")"; });
		if (done) return out;
		const toks = text.trim().split(/\s+/);
		if (toks.length < 2) return text;
		const hits = toks.map((t, i) => [t, i]).filter(([t]) => t !== "." && t !== "-" && t !== "_");
		for (let tries = 0; tries < 12 && hits.length; tries++) { // a hit next to a rest moves there
			const [t, i] = hits[Math.floor(Math.random() * hits.length)], j = i + (Math.random() < 0.5 ? -1 : 1);
			if (j >= 0 && j < toks.length && (toks[j] === "." || toks[j] === "_")) { [toks[i], toks[j]] = [toks[j], toks[i]]; return toks.join(" "); }
		}
		const i = Math.floor(Math.random() * (toks.length - 1)); // else two next to each other swap
		[toks[i], toks[i + 1]] = [toks[i + 1], toks[i]];
		return toks.join(" ");
	};

	// the instruments of the patch: what takes notes and makes sound
	const instruments = st => st.nodes.filter(t => (t.ins || []).some(p => p.type === "notes") && (t.outs || []).some(p => p.type === "audio"));
	const instLabel = t => (t.name ? t.name + " (" + t.kind + ")" : t.kind + (t.preset ? " \u00b7 " + t.preset : "")) + " " + t.id;
	// where a module plays, found along its notes cables (a clock through its notes module, a part at once): the instrument and the cable that reaches it
	const route = (n, st) => {
		const insts = new Set(instruments(st).map(t => t.id));
		const seen = new Set([n.id]);
		let frontier = [n.id];
		for (let hop = 0; hop < 4 && frontier.length; hop++) {
			const next = [];
			for (const id of frontier) for (const c of st.cables) if (c.from === id && c.type !== "mod" && !seen.has(c.to)) {
				if (insts.has(c.to) && c.type === "notes") return {inst: st.nodes.find(x => x.id === c.to), cable: c};
				seen.add(c.to); next.push(c.to);
			}
			frontier = next;
		}
		return null;
	};
	// plays on: the cable of the notes goes to another instrument (the module that sends them stays)
	const playOn = (n, st, id) => {
		const r = route(n, st);
		let from = r ? r.cable.from : null, out = r ? r.cable.out : 0;
		if (!r) { const o = (n.outs || []).findIndex(p => p.type === "notes"); if (o < 0) return; from = n.id; out = o; }
		const inst = st.nodes.find(t => t.id === id);
		if (r) send({disconnect: r.cable.id});
		if (!inst) return;
		const i = inst.ins.findIndex(p => p.type === "notes");
		if (i >= 0) send({connect: {from, out, to: id, in: i}});
	};

	const rows = () => {
		const out = [];
		for (const n of state.nodes) for (const key of (state.sectionTexts[n.kind] || [])) {
			const t = (n.texts || []).find(t => t.key === key);
			if (t) out.push({n, key, text: t.value, error: n.errors && n.errors[key] || ""});
		}
		out.sort((a, b) => (a.n.kind in RANK ? RANK[a.n.kind] : 9) - (b.n.kind in RANK ? RANK[b.n.kind] : 9) || Number(a.n.id.replace(/\D/g, "")) - Number(b.n.id.replace(/\D/g, "")));
		return out;
	};
	const setText = (r, text) => { // the page follows at once: the next edit builds on this one, not on the state that is still on its way
		const t = (r.n.texts || []).find(t => t.key === r.key);
		if (t) t.value = text;
		send({text: {id: r.n.id, key: r.key, value: text}});
		build();
	};

	const closeEditor = () => { editing = null; build(); };
	function build() {
		body.replaceChildren();
		if (!state) return;
		const sections = state.sections || [];
		empty.hidden = sections.length > 0;
		if (!sections.length) return;
		const table = el("div", "pt-grid-table");
		table.style.gridTemplateColumns = `minmax(130px, 170px) repeat(${sections.length}, minmax(120px, 1fr))`;
		table.append(el("div", "pt-grid-corner"));
		for (const s of sections) { const h = el("div", "pt-grid-col", s.name); h.append(el("small", "", s.bars + " bars")); table.append(h); }
		for (const r of rows()) {
			const label = el("div", "pt-grid-row");
			label.append(el("b", "", (r.n.name ? r.n.name : r.n.kind) + (r.n.name ? "" : r.key === "pattern" || r.key === "notes" || r.key === "values" || r.key === "chords" || r.key === "curves" ? "" : " " + r.key)), el("small", "", (r.n.name ? r.n.kind + " " : "") + r.n.id));
			label.title = "double-click to give it a name";
			label.addEventListener("dblclick", () => { const name = window.prompt("A name for this module (empty: none)", r.n.name || ""); if (name !== null) send({name: {id: r.n.id, value: name}}); });
			if (r.error) { label.classList.add("bad"); label.title = r.error; }
			if (r.n.kind !== "harmony") {
				const rt = route(r.n, state), pick = el("select", "pt-grid-on");
				pick.title = "plays on: the instrument these notes go to";
				pick.dataset.on = rt ? rt.inst.id : "";
				pick.append(new Option(rt ? "plays on\u2026" : "plays on\u2026 (none yet)", ""));
				for (const t of instruments(state)) pick.append(new Option(instLabel(t), t.id));
				if (rt) { pick.append(new Option("none", "-")); pick.value = rt.inst.id; }
				if (!rt && !(r.n.outs || []).some(p => p.type === "notes")) { pick.disabled = true; pick.title = "no notes come out of this module: add a notes module after it first"; }
				pick.addEventListener("change", () => { if (pick.value === "") return; playOn(r.n, state, pick.value === "-" ? "" : pick.value); });
				label.append(pick);
			}
			table.append(label);
			const lines = parse(r.text), others = othersText(lines);
			for (const s of sections) {
				const own = cellText(lines, s.name);
				const cell = el("div", "pt-grid-cell" + (own === null ? (others ? " inherited" : " silent") : ""));
				const text = own !== null ? own : others;
				cell.append(el("code", "", text || "—"));
				cell.title = (own !== null ? "" : others ? "no line of its own: it plays the lines without a section\n" : "silent here\n") + text;
				cell.draggable = own !== null;
				cell.addEventListener("click", () => { editing = {row: r, section: s.name, text: own !== null ? own : ""}; build(); });
				cell.addEventListener("dragstart", e => { drag = {row: r, section: s.name, text: own}; e.dataTransfer.setData("text/plain", own || ""); e.dataTransfer.effectAllowed = "copy"; });
				cell.addEventListener("dragover", e => { if (drag && drag.row.n.id === r.n.id && drag.row.key === r.key && drag.section !== s.name) { e.preventDefault(); cell.classList.add("over"); } });
				cell.addEventListener("dragleave", () => cell.classList.remove("over"));
				cell.addEventListener("drop", e => {
					cell.classList.remove("over");
					if (!drag || drag.row.n.id !== r.n.id || drag.row.key !== r.key) return;
					e.preventDefault(); setText(r, withCell(r.text, s.name, drag.text)); drag = null;
				});
				if (editing && editing.row.n.id === r.n.id && editing.row.key === r.key && editing.section === s.name) cell.classList.add("editing");
				table.append(cell);
			}
		}
		body.append(table);
		if (editing) body.append(editor());
	}
	function editor() {
		const r = rows().find(x => x.n.id === editing.row.n.id && x.key === editing.row.key);
		if (!r) { editing = null; return el("div"); }
		const box = el("div", "pt-grid-edit");
		const lines = parse(r.text), cur = cellText(lines, editing.section);
		const area = el("textarea", "pt-grid-text"); area.spellcheck = false; area.rows = 2; area.value = editing.text;
		area.placeholder = "what " + r.n.kind + " plays in the " + editing.section + " (empty: nothing, or the lines without a section)";
		const save = () => { editing.text = area.value.replace(/\n+/g, " ").trim(); setText(r, withCell(r.text, editing.section, editing.text)); };
		area.addEventListener("keydown", e => { e.stopPropagation(); if (e.key === "Enter" && (e.ctrlKey || e.metaKey || !e.shiftKey)) { e.preventDefault(); save(); } else if (e.key === "Escape") closeEditor(); });
		area.addEventListener("input", () => { editing.text = area.value; });
		const saveB = el("button", "", "Save"); saveB.title = "Enter takes it"; saveB.addEventListener("click", save);
		const varyB = el("button", "", "Vary"); varyB.title = "a nearby pattern: the rhythm functions turned or reseeded, a hit moved a step";
		varyB.addEventListener("click", () => { editing.text = vary(area.value || (cur !== null ? cur : othersText(lines))); area.value = editing.text; save(); });
		const clearB = el("button", "", "Clear"); clearB.title = "take the line of this section away: it plays the lines without a section, or nothing";
		clearB.addEventListener("click", () => { editing.text = ""; setText(r, withCell(r.text, editing.section, null)); });
		const copy = el("select"); copy.title = "copy this pattern to another section";
		copy.append(new Option("Copy to…", ""));
		for (const s of state.sections) if (s.name !== editing.section) copy.append(new Option(s.name, s.name));
		copy.append(new Option("all the others", "*"));
		copy.addEventListener("change", () => {
			const v = area.value.replace(/\n+/g, " ").trim() || cur || "";
			let t = r.text;
			for (const s of state.sections) if (s.name !== editing.section && (copy.value === "*" || copy.value === s.name)) t = withCell(t, s.name, v);
			setText(r, t); copy.value = "";
		});
		const done = el("button", "", "Done"); done.addEventListener("click", closeEditor);
		const err = el("div", "pt-grid-err", r.error || "");
		const row = el("div", "pt-grid-edit-row"); row.append(saveB, varyB, clearB, copy, done);
		box.append(el("div", "pt-grid-edit-title", r.n.kind + " \u00b7 " + editing.section), area, row, err);
		setTimeout(() => area.focus(), 0);
		return box;
	}

	close.addEventListener("click", () => api.toggle());
	const api = {
		el: root,
		get isOpen() { return !root.hidden; },
		toggle() { root.hidden = !root.hidden; if (!root.hidden) build(); if (api.onToggle) api.onToggle(!root.hidden); },
		update(st) {
			state = st;
			if (root.hidden) return;
			if (editing && document.activeElement && document.activeElement.classList && document.activeElement.classList.contains("pt-grid-text")) { // being typed: the cells under it follow, the box stays
				const keep = document.activeElement, v = keep.value, s = keep.selectionStart;
				const sig = JSON.stringify([st.sections, rows().map(r => [r.n.id, r.n.name, r.key, r.text, r.error, (route(r.n, st) || {inst: {id: ""}}).inst.id, instruments(st).map(t => t.id + (t.name || "") + (t.preset || ""))])]);
				if (sig === api._sig) return;
				api._sig = sig; build(); const k = body.querySelector(".pt-grid-text"); if (k) { k.value = v; k.focus(); k.setSelectionRange(s, s); }
				return;
			}
			const sig = JSON.stringify([st.sections, rows().map(r => [r.n.id, r.n.name, r.key, r.text, r.error, (route(r.n, st) || {inst: {id: ""}}).inst.id, instruments(st).map(t => t.id + (t.name || "") + (t.preset || ""))]), editing && [editing.row.n.id, editing.section]]);
			if (sig === api._sig) return;
			api._sig = sig; build();
		},
		onToggle: null,
		_sig: "",
	};
	return api;
};
