"use strict";

importScripts("wasm_exec.js");

const go = new Go();
let instance = null;

async function start() {
  const response = await fetch("asciify.wasm");
  if (!response.ok) throw new Error(`could not load asciify.wasm (${response.status})`);
  const buffer = await response.arrayBuffer();
  const result = await WebAssembly.instantiate(buffer, go.importObject);
  instance = result.instance;
  go.run(instance);
  self.postMessage({ type: "ready" });
}

start().catch((error) => {
  self.postMessage({ type: "error", error: String((error && error.message) || error) });
});

self.onmessage = (event) => {
  const { id, image, options } = event.data;
  if (!instance) {
    self.postMessage({ type: "error", id, error: "WASM is still loading" });
    return;
  }
  try {
    const result = self.asciifyConvert(new Uint8Array(image), options);
    self.postMessage({ type: "result", id, ...result });
  } catch (error) {
    self.postMessage({
      type: "error",
      id,
      error: String((error && error.message) || error),
    });
  }
};
