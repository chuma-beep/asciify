"use strict";

const DEFAULT_CHARSET = ".,:;'\"`-_~=+*#%@/\\|()[]{}<>!1lIiO0";
const SAMPLES = ["gradient.png", "circle.png", "lineart.png", "portrait.png", "photo.png"];

const elements = {
  status: document.getElementById("status"),
  drop: document.getElementById("drop"),
  dropText: document.getElementById("dropText"),
  file: document.getElementById("file"),
  samples: document.getElementById("samples"),
  cols: document.getElementById("cols"),
  mode: document.getElementById("mode"),
  color: document.getElementById("color"),
  cellAspect: document.getElementById("cellAspect"),
  charset: document.getElementById("charset"),
  clipLow: document.getElementById("clipLow"),
  clipHigh: document.getElementById("clipHigh"),
  invert: document.getElementById("invert"),
  edges: document.getElementById("edges"),
  localContrast: document.getElementById("localContrast"),
  dither: document.getElementById("dither"),
  halfblock: document.getElementById("halfblock"),
  font: document.getElementById("font"),
  fontName: document.getElementById("fontName"),
  copy: document.getElementById("copy"),
  downloadTxt: document.getElementById("downloadTxt"),
  downloadPng: document.getElementById("downloadPng"),
  outputCode: document.getElementById("outputCode"),
};

const state = {
  image: null,
  imageName: "",
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
    charset: elements.charset.value,
    cellAspect: number(elements.cellAspect.value, 2),
    clipLow: number(elements.clipLow.value, 0.02),
    clipHigh: number(elements.clipHigh.value, 0.98),
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
    elements.outputCode.textContent = result.text;
    const elapsed = Math.max(1, Math.round(performance.now() - started));
    setStatus(`${result.cols} × ${result.rows} · ${elapsed} ms`);
  } catch (error) {
    if (token !== state.renderToken) return;
    setStatus(error.message || "conversion failed", true);
  }
}

async function loadFile(file) {
  if (!file) return;
  const buffer = await file.arrayBuffer();
  state.image = buffer;
  state.imageName = file.name;
  elements.dropText.textContent = file.name;
  render();
}

async function loadSample(name) {
  const response = await fetch(`samples/${name}`);
  if (!response.ok) throw new Error(`could not load ${name}`);
  const buffer = await response.arrayBuffer();
  state.image = buffer;
  state.imageName = name;
  elements.dropText.textContent = name;
  render();
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
