// the window of a module of the patcher (opened by a double click on the module): its texts, each in a box of lines (what is wrong or what it says below them), its
// settings (its editor, served by the patcher's mount for this module: mount/<name>@<id>/) below. The patch lives on the page that opened it: this window asks the
// page's server for the patch as it changes (server sent events), and tells it what is typed. Query: base (the folder of the page), id, patcher (the name of the
// patcher on the page), mount, theme, mode
"use strict";
(() => {
	const q = new URLSearchParams(location.search);
	const base = q.get("base") || "../../", id = q.get("id") || "", patcher = q.get("patcher") || "", mount = q.get("mount") || "";
	const theme = q.get("theme") || "", mode = q.get("mode") || "", store = q.get("store") || "";
	const $ = name => document.getElementById(name);
	const el = (tag, cls, text) => { const e = document.createElement(tag); if (cls) e.className = cls; if (text !== undefined) e.textContent = text; return e; };
	if (mode) document.documentElement.dataset.mode = mode;
	$("theme").href = base + "theme.css" + (theme ? "?name=" + encodeURIComponent(theme) : "");

	const root = document.querySelector(".mw");
	let node = null, boxes = {}, log = null, built = "", framed = false, texts = [];
	const send = data => fetch(base + "api/message", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({name: patcher, data}), keepalive: true}).catch(() => {});
	const commitText = (key, value) => send({text: {id, key, value}});
	// what is typed taken: every text (only those changed, when closing)
	function commit(changed) {
		for (const [key, ta] of Object.entries(boxes)) {
			const t = texts.find(x => x.key === key);
			if (!changed || !t || ta.value !== t.value) commitText(key, ta.value);
		}
	}
	function help(text) {
		const list = el("ul", "mw-help");
		for (const line of text.split("\n")) {
			const li = el("li");
			line.split("`").forEach((part, i) => { if (part) li.append(i % 2 ? el("code", "", part) : document.createTextNode(part)); });
			list.append(li);
		}
		return list;
	}
	// the boxes are made when the module has other texts than they were made for
	function build(n) {
		const key = JSON.stringify([n.kind, n.preset, (n.texts || []).map(t => [t.key, t.label, t.lines])]) + (n.log !== undefined && n.log !== null) + !!n.page;
		if (key === built) return;
		built = key;
		boxes = {}; log = null;
		const holder = $("boxes");
		holder.innerHTML = "";
		texts = n.texts || [];
		for (const t of texts) {
			const wrap = el("label", "mw-box");
			if (texts.length > 1) wrap.append(el("span", "mw-label", t.label));
			if (t.help) wrap.append(help(t.help));
			const ta = el("textarea", "mw-text"); ta.spellcheck = false; ta.wrap = "off"; ta.value = t.value;
			ta.addEventListener("keydown", e => {
				e.stopPropagation();
				if (e.key === "Tab") { e.preventDefault(); document.execCommand("insertText", false, "\t"); }
				else if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) { e.preventDefault(); commit(); }
				else if (e.key === "Escape") { e.preventDefault(); commit(true); window.close(); }
			});
			wrap.append(ta);
			holder.append(wrap);
			boxes[t.key] = ta;
		}
		if (n.log !== undefined && n.log !== null) {
			for (const w of holder.children) w.style.flex = "0 0 auto";
			const wrap = el("label", "mw-box");
			const ta = el("textarea", "mw-text"); ta.readOnly = true; ta.spellcheck = false; ta.wrap = "off";
			wrap.append(ta); holder.append(wrap);
			log = ta;
		}
		$("code").hidden = $("run").hidden = !texts.length;
		$("settings").hidden = !n.page;
		const script = texts.some(t => t.lines > 1);
		root.classList.toggle("both", !!texts.length && !!n.page && script);
		root.classList.toggle("short", !!texts.length && !!n.page && !script);
		document.title = $("title").textContent = n.kind + (n.preset ? " · " + n.preset : "");
		const first = holder.querySelector("textarea");
		if (first) first.focus();
	}
	function show(n) {
		$("gone").hidden = !!n;
		if (!n) { root.querySelector(".mw-head").hidden = $("code").hidden = $("settings").hidden = true; return; }
		root.querySelector(".mw-head").hidden = false;
		if (!node || node.id !== n.id) framed = false;
		node = n;
		build(n);
		if (n.page && !framed) {
			framed = true;
			$("frame").src = base + "mount/" + encodeURIComponent(mount) + "@" + encodeURIComponent(id) + "/?embed&id=" + encodeURIComponent(id) + "&theme=" + encodeURIComponent(theme);
		}
		if (log && typeof n.log === "string" && log.value !== n.log) { // the newest lines stay in view
			const atEnd = log.scrollTop + log.clientHeight >= log.scrollHeight - 4;
			log.value = n.log;
			if (atEnd) log.scrollTop = log.scrollHeight;
		}
		let say = "";
		for (const [key, ta] of Object.entries(boxes)) { // what is wrong with a text: below the boxes
			const e = n.errors && n.errors[key];
			ta.classList.toggle("bad", !!e);
			if (e && !say) say = "error: " + (Object.keys(boxes).length > 1 ? key + ": " : "") + e;
		}
		if (!say) say = n.status || "";
		$("status").textContent = say;
		$("status").classList.toggle("bad", /^error/i.test(say));
	}

	// the settings frame tells its height; it gets the theme and the mode like the frames of the page
	window.addEventListener("message", e => {
		const m = e.data || {};
		if (e.source === $("frame").contentWindow && m.psalm === "height") $("frame").style.height = m.height + "px";
	});
	$("frame").addEventListener("load", () => {
		const w = $("frame").contentWindow;
		if (w && theme) w.postMessage({psalm: "theme", name: theme}, "*");
		if (w && mode) w.postMessage({psalm: "mode", mode}, "*");
	});
	// where the window is and how big: kept, for the next window of a module of this kind (the page that opens it reads it back). Nothing tells when a window is moved:
	// it is looked at now and then
	let kept = "";
	function keep() {
		if (!store || window.innerWidth < 100 || window.innerHeight < 100) return;
		const v = JSON.stringify({x: window.screenX, y: window.screenY, w: window.innerWidth, h: window.innerHeight});
		if (v === kept) return;
		kept = v;
		try { localStorage.setItem(store, v); } catch (e) { /* no storage: the window opens as it does */ }
	}
	setInterval(keep, 600);
	window.addEventListener("resize", keep);
	$("run").addEventListener("click", () => commit());
	window.addEventListener("pagehide", () => commit(true));

	// the patch, as it changes: the server sends the state of the patcher when it does
	const events = new EventSource(base + "api/events");
	events.onopen = () => $("live").classList.add("on");
	events.onerror = () => $("live").classList.remove("on");
	events.onmessage = e => {
		$("live").classList.add("on");
		const u = JSON.parse(e.data);
		const s = u.states && u.states[patcher];
		if (s && s.nodes) show(s.nodes.find(n => n.id === id) || null);
	};
})();
