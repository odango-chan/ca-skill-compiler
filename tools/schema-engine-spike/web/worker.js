const candidate = new URL(self.location.href).searchParams.get("candidate");
if (!new Set(["san", "kaptin"]).has(candidate)) throw new Error("unknown candidate");
importScripts(`wasm_exec_${candidate}.js`);
(async () => {
  const start = performance.now();
  const response = await fetch(`${candidate}.wasm`);
  if (!response.ok) throw new Error(`WASM fetch failed: ${response.status}`);
  const bytes = await response.arrayBuffer();
  const go = new Go();
  const {instance} = await WebAssembly.instantiate(bytes, go.importObject);
  const instantiateMs = performance.now() - start;
  const memoryBeforeBytes = instance.exports.mem ? instance.exports.mem.buffer.byteLength : null;
  const runStart = performance.now();
  await go.run(instance);
  const goRunMs = performance.now() - runStart;
  if (globalThis.__schemaSpikeError) throw new Error(globalThis.__schemaSpikeError);
  const result = JSON.parse(globalThis.__schemaSpikeResult || "null");
  if (!result) throw new Error("WASM returned no result");
  self.postMessage({
    candidate,
    goVersion: result.goVersion,
    instantiateMs,
    goRunMs,
    memoryBeforeBytes,
    memoryAfterBytes: instance.exports.mem ? instance.exports.mem.buffer.byteLength : null,
    result
  });
})().catch((error) => { self.postMessage({candidate, error: String(error)}); });
