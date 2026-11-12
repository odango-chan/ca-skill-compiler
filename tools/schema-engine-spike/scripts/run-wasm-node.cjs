const fs = require("node:fs");
const path = require("node:path");

const runtimePath = process.argv[2];
const wasmPath = process.argv[3];
if (!runtimePath || !wasmPath) {
  throw new Error("usage: node run-wasm-node.cjs <matching-wasm_exec.js> <candidate.wasm>");
}

require(path.resolve(runtimePath));
const go = new Go();
const wasm = fs.readFileSync(wasmPath);
WebAssembly.instantiate(wasm, go.importObject)
  .then(({instance}) => go.run(instance))
  .then(() => {
    if (globalThis.__schemaSpikeError) throw new Error(globalThis.__schemaSpikeError);
    const encoded = globalThis.__schemaSpikeResult;
    if (!encoded) throw new Error("WASM returned no result");
    const result = JSON.parse(encoded);
    if (result.vectors?.passed !== 25 || result.vectors?.failed !== 0) {
      throw new Error(`semantic vectors failed: ${JSON.stringify(result.vectors)}`);
    }
    if (result.security?.length !== 8 || result.security.some((item) => !item.passed)) {
      throw new Error(`reference security failed: ${JSON.stringify(result.security)}`);
    }
    if (result.regexRegexp2?.length !== 8 || result.regexRegexp2.some((item) => !item.conformsSample)) {
      throw new Error(`custom regex samples failed: ${JSON.stringify(result.regexRegexp2)}`);
    }
    process.stdout.write(JSON.stringify(result, null, 2));
  })
  .catch((error) => {
    console.error(error);
    process.exitCode = 1;
  });
