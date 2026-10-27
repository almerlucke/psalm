// the controls of a synth or an effect, by group: knobs, switches, choices and its envelopes
psalm.register("editor", (p, api) => {
	"use strict";
	const d = p.data;
	const byKey = {};
	for (const pt of p.parts) byKey[pt.name.slice(p.name.length + 1)] = pt;
	const partWidgets = {};
	const root = document.createElement("div");
	root.className = "editor";
	const cards = {};
	// a synth with very many settings (a plugin's): the groups of more than a few are closed, a click on the title opens one; the first group stays open
	const crowded = d.params.length > 100;
	d.groups.forEach((g, i) => {
		const card = document.createElement("div");
		card.className = "ed-group";
		const h = document.createElement("div");
		h.className = "ed-title";
		h.textContent = g;
		const row = document.createElement("div");
		row.className = "ed-controls";
		card.append(h, row);
		if (crowded) {
			const n = d.params.filter(q => q.group === g).length;
			h.textContent = `${g} (${n})`;
			h.classList.add("ed-fold");
			card.classList.toggle("closed", i > 0);
			h.addEventListener("click", () => card.classList.toggle("closed"));
		}
		root.append(card);
		cards[g] = row;
	});
	const add = (group, w) => (cards[group] || root).append(w.el);

	// the presets: the factory ones and your own, saved under a name
	const bar = document.createElement("div");
	bar.className = "ed-presets";
	const pick = document.createElement("select");
	pick.title = "a preset: the synth's own, or one of yours";
	const nameIn = document.createElement("input");
	nameIn.placeholder = "name"; nameIn.maxLength = 64;
	const save = document.createElement("button"); save.textContent = "Save";
	save.title = "save the settings as they are under the name (one of yours of that name is replaced)";
	const del = document.createElement("button"); del.textContent = "Delete";
	del.title = "delete this preset of yours";
	const note = document.createElement("span"); note.className = "ed-note";
	bar.append(pick, nameIn, save, del, note);
	root.prepend(bar);
	let st = {factory: [], mine: [], keeps: false, current: "", note: ""}, shown = "";
	const fill = () => {
		const key = JSON.stringify([st.factory, st.mine]);
		if (key !== shown) {
			shown = key;
			pick.innerHTML = "";
			const none = document.createElement("option"); none.value = ""; none.textContent = "preset…"; pick.append(none);
			for (const [label, names, mine] of [["mine", st.mine, true], ["factory", st.factory, false]]) {
				if (!names || !names.length) continue;
				const g = document.createElement("optgroup"); g.label = label;
				for (const n of names) { const o = document.createElement("option"); o.value = (mine ? "m:" : "f:") + n; o.textContent = n; g.append(o); }
				pick.append(g);
			}
		}
		const mineNow = st.mine && st.mine.includes(st.current);
		pick.value = st.current ? (mineNow ? "m:" : "f:") + st.current : "";
		if (pick.value === "" && st.current) pick.value = "f:" + st.current;
		del.disabled = !mineNow;
		for (const e of [nameIn, save, del]) e.hidden = !st.keeps;
		if (document.activeElement !== nameIn && mineNow) nameIn.value = st.current;
		note.textContent = st.note || "";
		bar.hidden = !st.keeps && !(st.factory && st.factory.length);
	};
	pick.addEventListener("change", () => {
		if (!pick.value) return;
		api.message(p.name, {load: pick.value.slice(2), mine: pick.value.startsWith("m:")});
	});
	const doSave = () => { const n = nameIn.value.trim(); if (n) api.message(p.name, {save: n}); };
	save.addEventListener("click", doSave);
	nameIn.addEventListener("keydown", e => { if (e.key === "Enter") doSave(); });
	del.addEventListener("click", () => {
		const n = pick.value.slice(2);
		if (n && confirm(`Delete the preset "${n}"?`)) api.message(p.name, {delete: n});
	});
	fill();

	for (const s of d.params) {
		const pt = byKey[s.key];
		if (!pt) continue;
		const q = Object.assign({}, pt, {label: s.label});
		let w;
		if (s.kind === "switch") w = api.control("toggle", q);
		else if (s.kind === "choice") w = api.control("select", Object.assign(q, {options: s.options}));
		else w = api.control("knob", q);
		partWidgets[pt.name] = w;
		api.bindable(w, pt);
		add(s.group, w);
	}

	// envelopes: the graph to drag when it has all four stages (the widgets' envelope), else knobs
	for (const e of d.envelopes) {
		const keys = {A: "Attack", D: "Decay", S: "Sustain", R: "Release"};
		const parts = [...e.stages].map(c => byKey[e.prefix + keys[c]]).filter(Boolean);
		if (e.stages === "ADSR" && parts.length === 4 && api.has("envelope")) {
			const [a, dd, ss, r] = parts;
			const q = Object.assign({}, a, {label: e.label, parts: [dd, ss, r]});
			const w = api.control("envelope", q);
			partWidgets[a.name] = {el: document.createElement("div"), show: v => w.show(v)};
			for (const x of [dd, ss, r]) partWidgets[x.name] = {el: document.createElement("div"), show: v => w.part(x.name, v)};
			w.el.classList.add("ed-envelope");
			add(e.group, w);
			continue;
		}
		const box = document.createElement("div");
		box.className = "ed-env-knobs";
		const t = document.createElement("div");
		t.className = "name"; t.textContent = e.label;
		const row = document.createElement("div");
		row.className = "ed-controls";
		box.append(t, row);
		for (const pt of parts) {
			const w = api.control("knob", pt);
			partWidgets[pt.name] = w;
			api.bindable(w, pt);
			row.append(w.el);
		}
		add(e.group, {el: box});
	}

	return {
		el: api.labelled(p, root),
		show() {},
		update(state) { if (state) { st = state; fill(); } },
		part(name, v) { if (partWidgets[name]) partWidgets[name].show(v); },
		partWidgets,
		restyle() { for (const w of Object.values(partWidgets)) if (w.restyle) w.restyle(); },
	};
});
