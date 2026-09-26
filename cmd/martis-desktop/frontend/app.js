"use strict";

const api = window.go.main.App;
const $ = (id) => document.getElementById(id);
const LARGE = 1 << 20;     // ask before rendering bodies over 1 MB
const HIGHLIGHT = 200_000; // skip syntax colouring above this size
const MAX_EVENTS = 500;    // SSE events kept on screen

const state = {
  col: { name: "Collections", folders: [] },
  sel: null,       // { f, i } of the loaded collection item
  open: new Set(), // expanded folder ids
  last: null,      // latest response
  prev: null,      // previous successful response body
  diff: false,
  showLarge: false,
  resTab: "rbody",
  sending: false,
  events: 0,       // SSE events received for the request in flight
};

// ---------- collections ----------

async function loadCollections() {
  state.col = (await api.Collections()) || { name: "Collections", folders: [] };
  state.col.folders ||= [];
  state.col.folders.forEach((f) => f.is_expanded && state.open.add(f.id));
  renderTree();
}

function renderTree() {
  const q = $("tree-filter").value.trim().toLowerCase();
  const tree = $("tree");
  tree.textContent = "";
  state.col.folders.forEach((folder, f) => {
    const items = (folder.items || []).map((it, i) => ({ it, i }))
      .filter(({ it }) => !q || `${it.name} ${it.url}`.toLowerCase().includes(q));
    if (q && !items.length) return;
    const open = q || state.open.has(folder.id);
    const row = el("div", "folder", [el("span", "chev", [open ? "▾" : "▸"]), el("span", "name", [folder.name])]);
    row.onclick = () => { state.open.has(folder.id) ? state.open.delete(folder.id) : state.open.add(folder.id); renderTree(); };
    tree.append(row);
    if (!open) return;
    for (const { it, i } of items) {
      const item = el("div", "item", [el("span", `verb ${it.method}`, [it.method]), el("span", "name", [it.name])]);
      if (state.sel && state.sel.f === f && state.sel.i === i) item.classList.add("selected");
      item.onclick = () => loadItem(f, i);
      tree.append(item);
    }
  });
}

function loadItem(f, i) {
  const it = state.col.folders[f].items[i];
  state.sel = { f, i };
  $("method").value = it.method || "GET";
  $("url").value = it.url || "";
  $("name").value = it.name || "";
  const gql = it.body_type === "graphql";
  setBodyType(gql ? "graphql" : "raw");
  $("body").value = gql ? "" : it.body_raw || "";
  $("gql").value = gql ? it.body_raw || "" : "";
  $("gqlvars").value = it.variables || "";
  $("tests").value = it.assertions || "";
  // Fold the legacy single-header fields into the header list.
  const lines = [];
  if (it.header_key) lines.push(`${it.header_key}: ${it.header_val || ""}`);
  if (it.header_auth) lines.push(`Authorization: ${it.header_auth}`);
  for (const h of it.headers || []) lines.push(`${h.key}: ${h.value}`);
  $("headers").value = lines.join("\n");
  $("crumb").textContent = `${state.col.folders[f].name} / ${it.name}`;
  renderTree();
}

function newRequest() {
  state.sel = null;
  for (const id of ["url", "name", "body", "gql", "gqlvars", "headers", "tests"]) $(id).value = "";
  setBodyType("raw");
  $("method").value = "GET";
  $("crumb").textContent = "Untitled request";
  renderTree();
  $("url").focus();
}

function parseHeaders() {
  return $("headers").value.split("\n").map((l) => l.trim()).filter(Boolean).map((l) => {
    const at = l.indexOf(":");
    return at < 0 ? { key: l, value: "" } : { key: l.slice(0, at).trim(), value: l.slice(at + 1).trim() };
  });
}

async function save() {
  const edited = {
    method: $("method").value, url: $("url").value.trim(), headers: parseHeaders(),
    ...bodyFields(), assertions: $("tests").value,
    header_key: "", header_val: "", header_auth: "",
  };
  edited.name = $("name").value.trim() || `${edited.method} ${edited.url.split("?")[0].split("/").pop() || "request"}`;
  if (state.sel) {
    const items = state.col.folders[state.sel.f].items;
    items[state.sel.i] = { ...items[state.sel.i], ...edited }; // keep fields the desktop does not edit
  } else {
    if (!state.col.folders.length) state.col.folders.push({ id: `f${Date.now()}`, name: "Requests", is_expanded: true, items: [] });
    const folder = state.col.folders[0];
    folder.items ||= [];
    folder.items.push({ id: `r${Date.now()}`, ...edited });
    state.sel = { f: 0, i: folder.items.length - 1 };
    state.open.add(folder.id);
  }
  try {
    await api.SaveCollections(state.col);
    $("name").value = edited.name;
    $("crumb").textContent = `${state.col.folders[state.sel.f].name} / ${edited.name}`;
    flash("Saved");
  } catch (e) {
    flash(`Save failed: ${e}`, true);
  }
  renderTree();
}

function isGraphQL() { return $("bodytype").value === "graphql"; }

function setBodyType(type) {
  $("bodytype").value = type;
  $("body").hidden = type === "graphql";
  $("gql-pane").hidden = type !== "graphql";
}

// bodyFields returns the collection-item body fields for the active body type.
function bodyFields() {
  return isGraphQL()
    ? { body_type: "graphql", body_raw: $("gql").value, variables: $("gqlvars").value }
    : { body_type: "raw", body_raw: $("body").value, variables: "" };
}

function formatBody() {
  const body = isGraphQL() ? $("gqlvars") : $("body");
  if (!body.value.trim()) return;
  try {
    body.value = JSON.stringify(JSON.parse(body.value), null, 2);
  } catch {
    flash("Body is not valid JSON", true);
  }
}

// ---------- request / response ----------

async function send() {
  if (state.sending) return api.StopStream(); // Send doubles as Stop while a request is open
  const btn = $("send");
  state.sending = true;
  state.events = 0;
  btn.firstChild.textContent = "Stop ";
  btn.classList.add("stop");
  const b = bodyFields();
  const payload = {
    Method: $("method").value, URL: $("url").value.trim(), Headers: parseHeaders(),
    BodyType: b.body_type, BodyRaw: b.body_raw, Variables: b.variables, Assertions: $("tests").value,
  };
  const res = await api.Send(payload, $("env").value);
  state.sending = false;
  btn.firstChild.textContent = "Send ";
  btn.classList.remove("stop");
  if (!res.error && state.last && !state.last.error) state.prev = state.last.body;
  state.last = res;
  state.diff = false;
  state.showLarge = false;
  $("diff").classList.remove("on");
  renderResponse();
}

// onEvent shows an SSE event as soon as it arrives, keeping only the newest ones.
function onEvent(text) {
  if (!state.sending) return;
  const out = $("output");
  if (state.events === 0) {
    out.textContent = "";
    $("large").hidden = true;
    $("status").textContent = "Streaming";
    $("status").className = "status ok";
  }
  state.events++;
  $("meta").textContent = `${state.events} event${state.events === 1 ? "" : "s"}`;
  const line = document.createElement("span");
  line.className = "event";
  line.textContent = text.trimEnd();
  out.append(line);
  while (out.childElementCount > MAX_EVENTS) out.firstElementChild.remove();
  out.scrollTop = out.scrollHeight;
}

function renderResponse() {
  const r = state.last;
  const status = $("status");
  if (!r) return;
  if (r.error) {
    status.textContent = "Request failed";
    status.className = "status err";
    $("meta").textContent = "";
  } else {
    status.textContent = `${r.status} ${r.statusText}`;
    status.className = `status ${r.status < 300 ? "ok" : r.status < 400 ? "warn" : "err"}`;
    $("meta").textContent = r.stream
      ? `stream closed · ${state.events} events · ${(r.durationMs / 1000).toFixed(1)} s`
      : `${r.durationMs} ms · ${formatBytes(r.size)}`;
  }
  renderTests(r);
  renderOutput();
}

async function renderOutput() {
  const r = state.last;
  const out = $("output");
  $("large").hidden = true;
  if (!r) return;
  if (r.error) { out.textContent = r.error; return; }
  if (state.resTab === "rheaders") {
    out.textContent = Object.keys(r.headers).sort().map((k) => `${k}: ${r.headers[k]}`).join("\n");
    return;
  }
  if (state.diff && state.prev != null) {
    const diff = await api.Diff(state.prev, r.body);
    out.innerHTML = diff.split("\n").map((l) =>
      l.startsWith("+ ") ? `<span class="add">${esc(l)}</span>` : l.startsWith("- ") ? `<span class="del">${esc(l)}</span>` : esc(l) + "\n").join("");
    return;
  }
  const filter = $("filter").value.trim();
  if (filter.startsWith("json.")) {
    const [value, ok] = await api.JSONPath(r.body, filter);
    out.innerHTML = ok ? highlight(value) : esc(`No value at ${filter}`);
    return;
  }
  if (filter) {
    const q = filter.toLowerCase();
    out.textContent = r.body.split("\n").filter((l) => l.toLowerCase().includes(q)).join("\n");
    return;
  }
  // Files and large bodies ask first: download, or show (text only).
  if ((r.file || r.size > LARGE) && !state.showLarge) {
    const text = $("large-text");
    text.textContent = "";
    if (r.file) {
      text.append(el("span", "kind", [`${r.file.kind} file`]), ` · ${r.file.name} · ${formatBytes(r.size)}`);
      if (r.binary) text.append(document.createElement("br"), "Binary content is not shown. Download it to open.");
    } else {
      text.append(`This response is ${formatBytes(r.size)}. Rendering it may slow the window; download it, filter with json.path, or show it.`);
    }
    $("show-large").hidden = r.binary;
    $("large").hidden = false;
    out.textContent = "";
    return;
  }
  out.innerHTML = highlight(pretty(r.body));
}

function renderTests(r) {
  const el = $("tests-result");
  el.className = "";
  el.textContent = "";
  if (r.error || !$("tests").value.trim()) return;
  const parts = [];
  if (r.captured && r.captured.length) parts.push(`captured ${r.captured.join(", ")}`);
  if (r.failures && r.failures.length) {
    el.className = "pill err";
    el.title = r.failures.join("\n");
    el.textContent = `${r.failures.length} failed` + (parts.length ? ` · ${parts.join(" · ")}` : "");
  } else {
    el.className = "pill ok";
    el.textContent = ["tests passed", ...parts].join(" · ");
  }
}

// ---------- helpers ----------

function el(tag, cls, children) {
  const n = document.createElement(tag);
  n.className = cls;
  n.append(...children);
  return n;
}

function esc(s) {
  return s.replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[c]);
}

function pretty(body) {
  if (body.length > LARGE) return body;
  try { return JSON.stringify(JSON.parse(body), null, 2); } catch { return body; }
}

function highlight(text) {
  const safe = esc(text);
  if (text.length > HIGHLIGHT) return safe;
  return safe.replace(/("(?:\\.|[^"\\])*")(\s*:)?|\b(true|false|null)\b|-?\b\d+(?:\.\d+)?(?:[eE][+-]?\d+)?\b/g,
    (m, str, colon, lit) => str ? (colon ? `<span class="k">${str}</span>${colon}` : `<span class="s">${str}</span>`)
      : lit ? `<span class="b">${m}</span>` : `<span class="n">${m}</span>`);
}

function formatBytes(n) {
  return n < 1024 ? `${n} B` : n < LARGE ? `${(n / 1024).toFixed(1)} KB` : `${(n / LARGE).toFixed(1)} MB`;
}

let flashTimer;
function flash(text, isError) {
  const el = $("config-path");
  el.textContent = text;
  el.style.color = isError ? "var(--err)" : "";
  clearTimeout(flashTimer);
  flashTimer = setTimeout(() => { el.textContent = "~/martis"; el.style.color = ""; }, 2000);
}

function bindTabs(group, onChange) {
  const bar = document.querySelector(`.tabs[data-group="${group}"]`);
  bar.addEventListener("click", (e) => {
    const tab = e.target.closest(".tab");
    if (!tab) return;
    bar.querySelectorAll(".tab").forEach((t) => t.classList.toggle("active", t === tab));
    onChange(tab.dataset.tab);
  });
}

// ---------- wiring ----------

bindTabs("req", (name) => document.querySelectorAll(".request .pane").forEach((p) => { p.hidden = p.dataset.pane !== name; }));
bindTabs("res", (name) => { state.resTab = name; renderOutput(); });
$("send").onclick = send;
$("save").onclick = save;
$("format").onclick = formatBody;
$("bodytype").onchange = (e) => setBodyType(e.target.value);
window.runtime?.EventsOn("sse", onEvent);
$("new-request").onclick = newRequest;
$("tree-filter").oninput = renderTree;
$("show-large").onclick = () => { state.showLarge = true; renderOutput(); };
$("download").onclick = async () => {
  try {
    const path = await api.SaveResponse();
    if (path) flash(`Saved to ${path}`);
  } catch (e) {
    flash(`Download failed: ${e}`, true);
  }
};
$("url").addEventListener("keydown", (e) => { if (e.key === "Enter") send(); });
let filterTimer;
$("filter").oninput = () => { clearTimeout(filterTimer); filterTimer = setTimeout(renderOutput, 150); };
$("diff").onclick = () => {
  if (state.prev == null) return flash("No previous response to diff yet");
  state.diff = !state.diff;
  $("diff").classList.toggle("on", state.diff);
  renderOutput();
};

document.addEventListener("keydown", (e) => {
  const mod = e.metaKey || e.ctrlKey;
  if (!mod) return;
  const key = e.key.toLowerCase();
  if (key === "enter") { e.preventDefault(); send(); }
  else if (key === "s") { e.preventDefault(); save(); }
  else if (key === "d") { e.preventDefault(); $("diff").click(); }
  else if (key === "f" && e.shiftKey) { e.preventDefault(); formatBody(); }
  else if (key === "f") { e.preventDefault(); $("filter").focus(); }
  else if (key === "n") { e.preventDefault(); newRequest(); }
});

// Tab inserts two spaces in editors instead of moving focus.
document.querySelectorAll(".editor").forEach((t) => t.addEventListener("keydown", (e) => {
  if (e.key !== "Tab") return;
  e.preventDefault();
  t.setRangeText("  ", t.selectionStart, t.selectionEnd, "end");
}));

(async () => {
  for (const name of await api.Environments()) $("env").append(new Option(name, name));
  await loadCollections();
})();
