"use strict";

const DEFAULT_CHARSET = ".,:;'\"`-_~=+*#%@/\\|()[]{}<>!1lIiO0";
const SAMPLES = ["gradient.png", "circle.png", "lineart.png", "portrait.png", "photo.png"];

const elements = {
  status: document.getElementById("status"),
  drop: document.getElementById("drop"),
  dropText: document.getElementById("dropText"),
  file: document.getElementById("file"),
  preview: document.getElementById("preview"),
  sourceName: document.getElementById("sourceName"),
  sourceSize: document.getElementById("sourceSize"),
  samples: document.getElementById("samples"),
  presets: document.getElementById("presets"),
  cols: document.getElementById("cols"),
  mode: document.getElementById("mode"),
  color: document.getElementById("color"),
  palette: document.getElementById("palette"),
  cellAspect: document.getElementById("cellAspect"),
  charset: document.getElementById("charset"),
  clipLow: document.getElementById("clipLow"),
  clipHigh: document.getElementById("clipHigh"),
  gamma: document.getElementById("gamma"),
  background: document.getElementById("background"),
  texture: document.getElementById("texture"),
  seed: document.getElementById("seed"),
  fontSize: document.getElementById("fontSize"),
  invert: document.getElementById("invert"),
  edges: document.getElementById("edges"),
  localContrast: document.getElementById("localContrast"),
  dither: document.getElementById("dither"),
  halfblock: document.getElementById("halfblock"),
  font: document.getElementById("font"),
  fontName: document.getElementById("fontName"),
  copy: document.getElementById("copy"),
  downloadTxt: document.getElementById("downloadTxt"),
  downloadHtml: document.getElementById("downloadHtml"),
  downloadPng: document.getElementById("downloadPng"),
  outputCode: document.getElementById("outputCode"),
};

const state = {
  image: null,
  imageName: "",
  previewUrl: "",
  font: null,
  fontName: "",
  pending: new Map(),
  nextId: 1,
  renderToken: 0,
  last: null,
};

elements.charset.value = DEFAULT_CHARSET;

const worker = new Worker("worker.js");

worker.onmessage = (event) => {
  const data = event.data;
  if (data.type === "ready") {
    setStatus("ready");
    render();
    return;
  }
  const entry = state.pending.get(data.id);
  if (!entry) return;
  state.pending.delete(data.id);
  if (data.type === "error") {
    entry.reject(new Error(data.error));
    return;
  }
  entry.resolve(data);
};

worker.onerror = (event) => {
  setStatus(event.message || "worker failed to load", true);
};

function setStatus(message, isError = false) {
  elements.status.textContent = message;
  elements.status.classList.toggle("is-error", isError);
}

function request(image, options) {
  return new Promise((resolve, reject) => {
    const id = state.nextId++;
    state.pending.set(id, { resolve, reject });
    worker.postMessage({ id, image, options });
  });
}

function number(value, fallback) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function collectOptions() {
  return {
    cols: Math.round(number(elements.cols.value, 100)),
    mode: elements.mode.value,
    color: elements.color.value,
    palette: elements.palette.value,
    charset: elements.charset.value,
    cellAspect: number(elements.cellAspect.value, 2),
    clipLow: number(elements.clipLow.value, 0.02),
    clipHigh: number(elements.clipHigh.value, 0.98),
    gamma: number(elements.gamma.value, 1),
    background: elements.background.value,
    texture: number(elements.texture.value, 0.6),
    seed: Math.round(number(elements.seed.value, 1)),
    invert: elements.invert.checked,
    edges: elements.edges.checked,
    localContrast: elements.localContrast.checked,
    dither: elements.dither.checked,
    halfblock: elements.halfblock.checked,
    fontData: state.font,
  };
}

let scheduleTimer = 0;
function scheduleRender() {
  window.clearTimeout(scheduleTimer);
  scheduleTimer = window.setTimeout(render, 120);
}

async function render() {
  if (!state.image) return;
  const token = ++state.renderToken;
  setStatus("rendering…");
  const started = performance.now();
  try {
    const result = await request(state.image, collectOptions());
    if (token !== state.renderToken) return;
    if (!result.ok) throw new Error(result.error);
    state.last = result;
    if (result.html) {
      elements.outputCode.innerHTML = result.html;
    } else {
      elements.outputCode.textContent = result.text;
    }
    const elapsed = Math.max(1, Math.round(performance.now() - started));
    setStatus(`${result.cols} × ${result.rows} · ${elapsed} ms`);
  } catch (error) {
    if (token !== state.renderToken) return;
    setStatus(error.message || "conversion failed", true);
  }
}

function showPreview(buffer, name) {
  if (state.previewUrl) URL.revokeObjectURL(state.previewUrl);
  const url = URL.createObjectURL(new Blob([buffer]));
  state.previewUrl = url;
  elements.preview.src = url;
  elements.preview.hidden = false;
  elements.sourceName.textContent = name;
  elements.sourceSize.textContent = "—";

  const image = new Image();
  image.onload = () => {
    if (state.previewUrl !== url) return;
    elements.sourceSize.textContent = `${image.naturalWidth} × ${image.naturalHeight}`;
  };
  image.src = url;
}

function useImage(buffer, name) {
  state.image = buffer;
  state.imageName = name;
  elements.dropText.textContent = name;
  showPreview(buffer, name);
  render();
}

async function loadFile(file) {
  if (!file) return;
  const buffer = await file.arrayBuffer();
  useImage(buffer, file.name);
}

async function loadSample(name) {
  const response = await fetch(`samples/${name}`);
  if (!response.ok) throw new Error(`could not load ${name}`);
  const buffer = await response.arrayBuffer();
  useImage(buffer, name);
}

function baseName() {
  const name = state.imageName || "ascii";
  return name.replace(/\.[^.]+$/, "") + ".asciify";
}

function download(blob, filename) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

elements.file.addEventListener("change", () => loadFile(elements.file.files[0]));

["dragenter", "dragover"].forEach((type) => {
  elements.drop.addEventListener(type, (event) => {
    event.preventDefault();
    elements.drop.classList.add("is-over");
  });
});
["dragleave", "drop"].forEach((type) => {
  elements.drop.addEventListener(type, (event) => {
    event.preventDefault();
    elements.drop.classList.remove("is-over");
  });
});
elements.drop.addEventListener("drop", (event) => {
  loadFile(event.dataTransfer.files[0]);
});

elements.font.addEventListener("change", async () => {
  const file = elements.font.files[0];
  if (!file) return;
  state.font = await file.arrayBuffer();
  state.fontName = file.name;
  elements.fontName.textContent = file.name;
  render();
});

for (const input of document.querySelectorAll(".panel input, .panel select")) {
  // Font size only changes the preview scale; re-rendering would be wasted work.
  if (input === elements.fontSize) continue;
  input.addEventListener("input", scheduleRender);
  input.addEventListener("change", scheduleRender);
}

for (const name of SAMPLES) {
  const button = document.createElement("button");
  button.type = "button";
  button.className = "chip";
  button.textContent = name.replace(/\.png$/, "");
  button.addEventListener("click", () => {
    loadSample(name).catch((error) => setStatus(error.message, true));
  });
  elements.samples.appendChild(button);
}

const PRESETS = {
  lagoon: {
    cols: 120,
    mode: "paint",
    color: "truecolor",
    palette: "abyss",
    cellAspect: 2,
    gamma: 1.05,
    background: "auto",
    texture: 0.65,
    seed: 1,
    invert: false,
    edges: false,
    localContrast: false,
    dither: false,
    halfblock: false,
  },
  portrait: {
    cols: 100,
    mode: "paint",
    color: "ansi256",
    palette: "abyss",
    cellAspect: 2,
    gamma: 1.05,
    background: "auto",
    texture: 0.6,
    seed: 1,
    invert: false,
    edges: false,
    localContrast: false,
    dither: false,
    halfblock: false,
  },
  photo: {
    cols: 120,
    mode: "paint",
    color: "truecolor",
    palette: "none",
    cellAspect: 2,
    gamma: 1.05,
    background: "auto",
    texture: 0.55,
    seed: 1,
    invert: false,
    edges: false,
    localContrast: false,
    dither: false,
    halfblock: false,
  },
  lineart: {
    cols: 100,
    mode: "shape",
    color: "none",
    palette: "none",
    cellAspect: 2,
    gamma: 1,
    background: "white",
    invert: true,
    edges: true,
    localContrast: false,
    dither: false,
    halfblock: false,
  },
  blocks: {
    cols: 100,
    mode: "shape",
    color: "truecolor",
    palette: "none",
    cellAspect: 2,
    gamma: 1,
    background: "auto",
    invert: false,
    edges: false,
    localContrast: false,
    dither: false,
    halfblock: true,
  },
};

function applyPreset(name) {
  const preset = PRESETS[name];
  if (!preset) return;
  for (const [key, value] of Object.entries(preset)) {
    const control = elements[key];
    if (!control) continue;
    if (control.type === "checkbox") {
      control.checked = value;
    } else {
      control.value = String(value);
    }
  }
  scheduleRender();
}

for (const button of document.querySelectorAll("[data-preset]")) {
  button.addEventListener("click", () => applyPreset(button.dataset.preset));
}

elements.fontSize.addEventListener("input", () => {
  document.documentElement.style.setProperty("--output-size", `${elements.fontSize.value}px`);
});
document.documentElement.style.setProperty("--output-size", `${elements.fontSize.value}px`);

elements.copy.addEventListener("click", async () => {
  if (!state.last) return;
  try {
    await navigator.clipboard.writeText(state.last.text);
    setStatus("copied");
  } catch {
    setStatus("copy failed", true);
  }
});

elements.downloadTxt.addEventListener("click", () => {
  if (!state.last) return;
  const text = state.last.ansi || state.last.text;
  download(new Blob([text], { type: "text/plain" }), `${baseName()}.txt`);
});

function escapeHtml(value) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

elements.downloadHtml.addEventListener("click", () => {
  if (!state.last) return;
  const body = state.last.html || escapeHtml(state.last.text);
  const title = state.imageName || "asciify";
  const html = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>${escapeHtml(title)} — asciify</title>
<style>
body { margin: 0; padding: 24px; background: #070809; color: #ece8df; }
pre { margin: 0; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; line-height: 1.15; white-space: pre; }
</style>
</head>
<body>
<pre>${body}</pre>
</body>
</html>
`;
  download(new Blob([html], { type: "text/html" }), `${baseName()}.html`);
});

elements.downloadPng.addEventListener("click", async () => {
  if (!state.image) return;
  setStatus("rendering png…");
  try {
    const options = { ...collectOptions(), png: true };
    const result = await request(state.image, options);
    if (!result.ok) throw new Error(result.error);
    const bytes = Uint8Array.from(atob(result.png), (char) => char.charCodeAt(0));
    download(new Blob([bytes], { type: "image/png" }), `${baseName()}.png`);
    setStatus("png saved");
  } catch (error) {
    setStatus(error.message || "png export failed", true);
  }
});
