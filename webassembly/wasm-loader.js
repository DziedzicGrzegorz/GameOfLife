const go = new Go();

async function initWasm() {
    try {
        const result = await WebAssembly.instantiateStreaming(fetch("json.wasm"), go.importObject);
        go.run(result.instance);
        console.log("WASM loaded successfully");
    } catch (err) {
        console.error("Failed to load WASM:", err);
    }
}

// Initialize WASM and wait for it to be ready
// const wasmPromise = initWasm();

(async func => {
    await initWasm();
    await import("./game.js").then(module => module.initCanvases());
})()
