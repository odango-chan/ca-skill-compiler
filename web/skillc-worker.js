// Skill Studio invokes the same Go compiler core inside a dedicated Worker.
// Request shape: { id, command: "check"|"build"|"inspect", request: {files:[{path,bytes:base64}],targets?} }.
// Never pass credentials, filesystem handles, or executable code to this bridge.
let boot = null;

async function ensureCompiler() {
  if (!boot) {
    boot = (async () => {
      importScripts("./wasm_exec.js");
      const go = new Go();
      const response = await fetch("./skillc.wasm");
      if (!response.ok) throw new Error("Unable to fetch skillc.wasm");
      const bytes = await response.arrayBuffer();
      const instance = await WebAssembly.instantiate(bytes, go.importObject);
      go.run(instance.instance); // intentionally long-lived Go callbacks
      for (let attempt = 0; attempt < 50; attempt++) {
        if (typeof self.skillcBuild === "function" && typeof self.skillcCheck === "function" && typeof self.skillcInspectDependencies === "function") return;
        await new Promise(resolve => setTimeout(resolve, 10));
      }
      throw new Error("Go/WASM compiler did not expose its callbacks");
    })();
  }
  return boot;
}

self.onmessage = async event => {
  const {id, command, request} = event.data ?? {};
  try {
    await ensureCompiler();
    if (!["check", "build", "inspect"].includes(command)) throw new Error("Unsupported command");
    const method = command === "check" ? self.skillcCheck : command === "inspect" ? self.skillcInspectDependencies : self.skillcBuild;
    const result = JSON.parse(method(JSON.stringify(request)));
    self.postMessage({id, result});
  } catch (error) {
    self.postMessage({id, error: String(error?.message ?? error)});
  }
};
