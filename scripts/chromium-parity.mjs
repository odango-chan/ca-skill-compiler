// Chromium Worker parity harness (WP8/WP11).
//
// It builds the Go/WASM compiler, serves a static page, launches real headless
// Chromium, runs Check/Build/Inspect through the same Web Worker transport used by
// Skill Studio, and compares the browser results with the Native Go reference for
// the same BuildRequest. The v1alpha5 flow
// compares full projections, including successful-Build artifact byte parity
// and Go Core artifact-integrity verification inside the Chromium Worker. The
// v1alpha5 trusted fixture is SYNTHETIC and compiler-owned. Run from the
// repository root:
//
//   GOTOOLCHAIN=go1.27.4 node scripts/chromium-parity.mjs
//
// Exit code 0 only when every compared projection is identical.

import { spawn, spawnSync, execFileSync } from 'node:child_process';
import { createServer } from 'node:http';
import { copyFileSync, mkdirSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import path from 'node:path';

const repoRoot = process.cwd();
const webRoot = path.join(repoRoot, '.parity-web');
const projectRoot = path.join(repoRoot, 'web', 'parity-project');
const v1alpha5FixtureRoot = path.join(repoRoot, 'web', 'parity-v1alpha5');
const chromePath = process.env.CHROME_PATH || 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const env = { ...process.env, GOTOOLCHAIN: 'go1.27.4', GOWORK: 'off', CGO_ENABLED: '0' };

const v1alpha5RequestAPIVersion = 'connectionagent.io/compiler-build-request/v1alpha5';
const v1alpha5LanguageTarget = 'connectionagent.io/v1alpha5';
const v1alpha5ResultAPIVersion = 'connectionagent.io/compiler-build-result/v1alpha5';

function fail(message) {
  console.error(`FAIL: ${message}`);
  process.exit(1);
}

function buildWasm() {
  rmSync(webRoot, { recursive: true, force: true });
  mkdirSync(webRoot, { recursive: true });
  const wasmPath = path.join(webRoot, 'skillc.wasm');
  execFileSync('go', ['build', '-o', wasmPath, './cmd/skillc-wasm'], {
    cwd: repoRoot,
    env: { ...env, GOOS: 'js', GOARCH: 'wasm' },
    stdio: 'inherit',
  });
  const goroot = execFileSync('go', ['env', 'GOROOT'], { env }).toString().trim();
  copyFileSync(path.join(goroot, 'lib', 'wasm', 'wasm_exec.js'), path.join(webRoot, 'wasm_exec.js'));
  copyFileSync(path.join(repoRoot, 'web', 'skillc-worker.js'), path.join(webRoot, 'skillc-worker.js'));
  copyFileSync(path.join(repoRoot, 'web', 'parity.html'), path.join(webRoot, 'parity.html'));
  copyFileSync(path.join(repoRoot, 'web', 'studio.html'), path.join(webRoot, 'studio.html'));
}

function collectFiles(directory, prefix = '') {
  const files = [];
  for (const name of readdirSync(directory).sort()) {
    const absolute = path.join(directory, name);
    const relative = prefix ? `${prefix}/${name}` : name;
    if (statSync(absolute).isDirectory()) files.push(...collectFiles(absolute, relative));
    else files.push({ path: relative, bytes: readFileSync(absolute).toString('base64') });
  }
  return files;
}

function writeRequest() {
  const request = {
    apiVersion: v1alpha5RequestAPIVersion,
    languageTarget: v1alpha5LanguageTarget,
    files: collectFiles(projectRoot),
    resolvedDependencies: [],
  };
  writeFileSync(path.join(webRoot, 'parity-request.json'), JSON.stringify(request));
}

function nativeReference() {
  const stdout = execFileSync('go', ['run', './cmd/parity-replay', projectRoot], { cwd: repoRoot, env }).toString();
  return JSON.parse(stdout);
}

// writeV1Alpha5Requests derives explicit BuildRequest JSON files from the
// checked-in complete trusted v1alpha5 fixture. The full-language pipeline runs
// the inherited package/dependency/Owner/Tool/Capability/Schema/Predicate/Script/
// Procedure semantics, so the valid and invalid cases must keep the complete
// package and trusted context and only vary the Procedure. The fixture is
// SYNTHETIC and compiler-owned, not a real Owner artifact.
function writeV1Alpha5Requests() {
  const requests = {};
  const trustedSource = readFileSync(path.join(v1alpha5FixtureRoot, 'trusted-build-request.json'));
  const trustedRequest = JSON.parse(trustedSource.toString('utf8'));

  const withProcedure = (procedure) => {
    const request = JSON.parse(JSON.stringify(trustedRequest));
    request.files = request.files.map(file => file.path.endsWith('procedure.yaml')
      ? { ...file, bytes: Buffer.from(procedure).toString('base64') }
      : file);
    return request;
  };

  const validProcedure = `start: complete-without-wait
nodes:
  complete-without-wait:
    operation:
      tool: tools::smf.update_timeout.handle
      arguments: {}
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
  final:
    postcondition:
      predicates:
        - predicate: session.goal_satisfied
          arguments: []
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
`;
  const invalidProcedure = readFileSync(path.join(v1alpha5FixtureRoot, 'invalid-procedure.yaml'));

  for (const [name, procedure] of [['valid', validProcedure], ['invalid', invalidProcedure]]) {
    const file = `parity-v1alpha5-${name}-request.json`;
    writeFileSync(path.join(webRoot, file), JSON.stringify(withProcedure(procedure)));
    requests[name] = file;
  }
  const trusted = 'parity-v1alpha5-trusted-request.json';
  writeFileSync(path.join(webRoot, trusted), trustedSource);
  requests.trusted = trusted;
  return requests;
}

// writeDevelopmentRequests derives the Development Build Native/Chromium
// comparison fixtures from the checked-in positive request: the canonical order,
// a fully reordered order and a negative closure that removes the transitive
// Schema package.
function writeDevelopmentRequests() {
  const source = JSON.parse(readFileSync(path.join(repoRoot, 'docs', 'design', 'fixtures', 'registration-development-request.json'), 'utf8'));
  const write = (name, value) => {
    writeFileSync(path.join(webRoot, name), JSON.stringify(value));
    return name;
  };
  const reordered = JSON.parse(JSON.stringify(source));
  reordered.files.reverse();
  reordered.snapshots.reverse();
  for (const snapshot of reordered.snapshots) snapshot.files.reverse();
  reordered.lock.roots.reverse();
  reordered.lock.packages.reverse();
  for (const entry of reordered.lock.packages) (entry.dependencies || []).reverse();
  const negative = JSON.parse(JSON.stringify(source));
  negative.snapshots = negative.snapshots.filter(snapshot => snapshot.key.packageId !== 'urn:example:ca:types');
  negative.lock.packages = negative.lock.packages.filter(entry => entry.key.packageId !== 'urn:example:ca:types');
  for (const entry of negative.lock.packages) {
    if (entry.key.packageId === 'urn:example:ca:tools') instrumentDropTypes(entry);
  }
  return {
    base: write('parity-development-request.json', source),
    reordered: write('parity-development-reordered-request.json', reordered),
    negative: write('parity-development-negative-request.json', negative),
  };
}

function instrumentDropTypes(entry) {
  entry.dependencies = [];
}

// nativeDevelopmentReference runs the real Native Development entry through the
// CLI and returns its BuildResult JSON. A failed build exits 1 with a complete
// structured result, so both exit codes are accepted.
function nativeDevelopmentReference(requestName, outName) {
  const result = spawnSync('go', [
    'run', './cmd/skillc', 'build-development',
    `--request=${path.join(webRoot, requestName)}`,
    `--out=${path.join(webRoot, outName)}`,
    '--json',
  ], { cwd: repoRoot, env, encoding: 'utf8', maxBuffer: 64 << 20 });
  if (!result.stdout) fail(`Native Development reference produced no output: ${result.stderr}`);
  return JSON.parse(result.stdout);
}

function browserArtifactHashes(build) {
  return (build.outputFiles || []).map(file => ({
    path: file.path,
    sha256: createHash('sha256').update(Buffer.from(file.bytes, 'base64')).digest('hex'),
  })).sort((left, right) => (left.path < right.path ? -1 : left.path > right.path ? 1 : 0));
}

function browserArtifactJSON(build, filePath) {
  const file = (build.outputFiles || []).find(candidate => candidate.path === filePath);
  if (!file) return null;
  return JSON.parse(Buffer.from(file.bytes, 'base64').toString('utf8'));
}

// nativeRequestReference runs the Native Go reference against an explicit
// BuildRequest JSON through parity-replay's --request mode, the same strict
// DecodeBuildRequestJSON path the CLI and WASM bridge use.
function nativeRequestReference(requestPath) {
  const stdout = execFileSync('go', ['run', './cmd/parity-replay', `--request=${requestPath}`], { cwd: repoRoot, env }).toString();
  return JSON.parse(stdout);
}

function stable(value) {
  if (Array.isArray(value)) return `[${value.map(stable).join(',')}]`;
  if (value && typeof value === 'object') {
    return `{${Object.keys(value).sort().map(key => `${JSON.stringify(key)}:${stable(value[key])}`).join(',')}}`;
  }
  return JSON.stringify(value);
}

function compare(label, nativeValue, browserValue) {
  if (stable(nativeValue) !== stable(browserValue)) {
    console.error(`MISMATCH in ${label}`);
    console.error(`  native : ${stable(nativeValue).slice(0, 2000)}`);
    console.error(`  browser: ${stable(browserValue).slice(0, 2000)}`);
    fail(`${label} projection differs between Native and Chromium Worker`);
  }
  console.log(`ok: ${label} projection identical`);
}

function contentType(file) {
  if (file.endsWith('.wasm')) return 'application/wasm';
  if (file.endsWith('.js')) return 'text/javascript';
  if (file.endsWith('.html')) return 'text/html; charset=utf-8';
  if (file.endsWith('.json')) return 'application/json';
  return 'application/octet-stream';
}

async function startServer() {
  const server = createServer((req, res) => {
    const requested = (req.url || '/').split('?')[0];
    const relative = requested === '/' ? '/parity.html' : requested;
    const absolute = path.join(webRoot, relative);
    if (!absolute.startsWith(webRoot)) {
      res.writeHead(403).end();
      return;
    }
    try {
      const body = readFileSync(absolute);
      res.writeHead(200, { 'content-type': contentType(absolute) });
      res.end(body);
    } catch {
      res.writeHead(404).end();
    }
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  return { server, port: server.address().port };
}

async function cdpEvaluate(discoveryUrl, pageUrl, globalName = '__parity') {
  const version = await (await fetch(discoveryUrl)).json();
  const socket = new WebSocket(version.webSocketDebuggerUrl);
  const pending = new Map();
  let nextId = 1;
  let sessionId = null;
  socket.onmessage = event => {
    const message = JSON.parse(String(event.data));
    if (message.id && pending.has(message.id)) {
      pending.get(message.id)(message);
      pending.delete(message.id);
    }
  };
  await new Promise((resolve, reject) => {
    socket.onopen = resolve;
    socket.onerror = () => reject(new Error('CDP socket error'));
  });
  const send = (method, params = {}, useSession = false) => new Promise((resolve, reject) => {
    const id = nextId++;
    pending.set(id, message => (message.error ? reject(new Error(`${method}: ${message.error.message}`)) : resolve(message.result)));
    const payload = { id, method, params };
    if (useSession && sessionId) payload.sessionId = sessionId;
    socket.send(JSON.stringify(payload));
  });
  const target = await send('Target.createTarget', { url: 'about:blank' });
  const attached = await send('Target.attachToTarget', { targetId: target.targetId, flatten: true });
  sessionId = attached.sessionId;
  try {
    await send('Page.enable', {}, true);
    await send('Page.navigate', { url: pageUrl }, true);
    const deadline = Date.now() + 40000;
    while (Date.now() < deadline) {
      await new Promise(resolve => setTimeout(resolve, 250));
      const outcome = await send('Runtime.evaluate', {
        expression: `window.${globalName} ? JSON.stringify(window.${globalName}) : null`,
        returnByValue: true,
      }, true);
      const value = outcome.result?.value;
      if (value && value !== 'null') {
        return JSON.parse(value);
      }
    }
    const debug = await send('Runtime.evaluate', {
      expression: 'JSON.stringify({title: document.title, dataset: document.documentElement.dataset.parity || null, debug: window.__parityDebug || null, body: document.body ? document.body.innerText.slice(0,200) : null})',
      returnByValue: true,
    }, true);
    return { ok: false, error: 'timeout waiting for window.__parity: ' + JSON.stringify(debug.result?.value) };
  } finally {
    try { await send('Target.closeTarget', { targetId: target.targetId }); } catch {}
    socket.close();
  }
}

async function main() {
  buildWasm();
  writeRequest();
  const native = nativeReference();
  const { server, port } = await startServer();
  const profile = path.join(webRoot, 'chrome-profile');
  const debugPort = 9222 + Math.floor(Math.random() * 500);
  const chrome = spawn(chromePath, [
    '--headless=new', '--disable-gpu', '--no-sandbox', '--no-first-run', '--no-default-browser-check',
    `--remote-debugging-port=${debugPort}`, `--user-data-dir=${profile}`, 'about:blank',
  ], { stdio: 'ignore' });
  try {
    await new Promise(resolve => setTimeout(resolve, 1200));
    const browser = await cdpEvaluate(`http://127.0.0.1:${debugPort}/json/version`, `http://127.0.0.1:${port}/parity.html`);
    if (!browser.ok) fail(`Chromium Worker error: ${browser.error}`);
    compare('check', native.check, browser.check);
    compare('build.success', native.build.success, browser.build.success);
    compare('build.apiVersion', native.build.apiVersion, browser.build.apiVersion);
    compare('build.diagnostics', native.build.diagnostics, browser.build.diagnostics);
    compare('build.dependencyClosure', native.build.dependencyClosure, browser.build.dependencyClosure);
    const nativePaths = (native.build.outputFiles || []).map(file => file.path);
    const browserPaths = (browser.build.outputFiles || []).map(file => file.path);
    compare('build.outputFiles.paths', nativePaths, browserPaths);
    compare('inspect', native.inspect, browser.inspect);

    const studio = await cdpEvaluate(`http://127.0.0.1:${debugPort}/json/version`, `http://127.0.0.1:${port}/studio.html`, '__studio');
    if (!studio.ok) fail(`Studio integration error: ${studio.error}`);
    const expectedDiagnostics = (native.build.diagnostics || []).map(diagnostic => ({
      code: diagnostic.code, phase: diagnostic.phase, path: diagnostic.path,
      line: diagnostic.line ?? 0, column: diagnostic.column ?? 0,
    }));
    compare('studio.diagnostics', expectedDiagnostics, studio.diagnostics);
    compare('studio.buildSuccess', native.build.success, studio.buildSuccess);
    compare('studio.artifactPaths', nativePaths, studio.artifactPaths);
    if (!studio.artifactReadOnly) fail('Studio artifact view must be read-only');
    if (studio.renderedRows !== expectedDiagnostics.length) {
      fail(`Studio rendered ${studio.renderedRows} diagnostic rows, expected ${expectedDiagnostics.length}`);
    }
    if (studio.reimplementedParser) fail('Studio must not reimplement the compiler parser');
    console.log('ok: studio diagnostics/artifact read-only integration identical');

    // Successful-Build phase: same request with trusted bindings + CapabilityContracts.
    const successRequestPath = path.join(webRoot, 'parity-success-request.json');
    const nativeSuccess = JSON.parse(execFileSync('go', ['run', './cmd/parity-replay', 'web/parity-success', `--emit-request=${successRequestPath}`], { cwd: repoRoot, env }).toString());
    if (!nativeSuccess.build.success) fail('successful parity fixture did not Build successfully under Native Go');
    const browserSuccess = await cdpEvaluate(`http://127.0.0.1:${debugPort}/json/version`, `http://127.0.0.1:${port}/parity.html?request=parity-success-request.json`);
    if (!browserSuccess.ok) fail(`Chromium Worker success-path error: ${browserSuccess.error}`);
    if (browserSuccess.artifactError !== '') fail(`Chromium Worker artifact verification failed: ${browserSuccess.artifactError}`);
    console.log('ok: Chromium Worker artifact integrity verification PASS');
    compare('success.check', nativeSuccess.check, browserSuccess.check);
    compare('success.build.success', nativeSuccess.build.success, browserSuccess.build.success);
    compare('success.build.apiVersion', nativeSuccess.build.apiVersion, browserSuccess.build.apiVersion);
    compare('success.build.diagnostics', nativeSuccess.build.diagnostics, browserSuccess.build.diagnostics);
    compare('success.build.dependencyClosure', nativeSuccess.build.dependencyClosure, browserSuccess.build.dependencyClosure);
    const nativeArtifact = nativeSuccess.build.outputFiles;
    const browserArtifact = browserSuccess.build.outputFiles || [];
    if (nativeArtifact.length === 0) fail('successful Native Build emitted no artifact files');
    compare('success.build.outputFiles.paths', nativeArtifact.map(file => file.path), browserArtifact.map(file => file.path));
    for (const file of browserArtifact) {
      const expected = nativeArtifact.find(candidate => candidate.path === file.path);
      if (!expected) fail(`unexpected browser artifact file ${file.path}`);
      const actualHash = createHash('sha256').update(Buffer.from(file.bytes, 'base64')).digest('hex');
      if (actualHash !== expected.sha256) fail(`artifact bytes differ for ${file.path}`);
    }
    console.log(`ok: successful-Build artifact byte parity (${browserArtifact.length} files)`);

    const studioSuccess = await cdpEvaluate(`http://127.0.0.1:${debugPort}/json/version`, `http://127.0.0.1:${port}/studio.html?request=parity-success-request.json`, '__studio');
    if (!studioSuccess.ok) fail(`Studio success-path error: ${studioSuccess.error}`);
    compare('studio-success.buildSuccess', true, studioSuccess.buildSuccess);
    compare('studio-success.artifactPaths', nativeArtifact.map(file => file.path), studioSuccess.artifactPaths);
    if (!studioSuccess.artifactReadOnly) fail('Studio success-path artifact view must be read-only');
    console.log('ok: Studio success-path artifact presentation identical');

    // v1alpha5 candidate-target artifact phase (WP8 completion). Check
    // diagnostics, successful Build artifact bytes and the linked Ingress
    // metadata of a Wait-bearing Skill are compared between Native Go and the
    // real Chromium Worker. The trusted fixture is SYNTHETIC and compiler-owned;
    // it is not a real Owner publication or qualification claim.
    const v1alpha5Requests = writeV1Alpha5Requests();
    const nativeV1Alpha5Valid = nativeRequestReference(path.join(webRoot, v1alpha5Requests.valid));
    const nativeV1Alpha5Invalid = nativeRequestReference(path.join(webRoot, v1alpha5Requests.invalid));
    const nativeV1Alpha5Trusted = nativeRequestReference(path.join(webRoot, v1alpha5Requests.trusted));

    // (a) Wait-free Procedure inside the complete trusted package: zero-diagnostic\n    // successful Build.
    if (nativeV1Alpha5Valid.check.length !== 0) {
      fail(`v1alpha5 valid procedure must Check with zero diagnostics, got ${stable(nativeV1Alpha5Valid.check)}`);
    }
    if (!nativeV1Alpha5Valid.build.success) {
      fail(`v1alpha5 valid Build must succeed, diagnostics = ${stable(nativeV1Alpha5Valid.build.diagnostics)}`);
    }
    if (nativeV1Alpha5Valid.build.apiVersion !== v1alpha5ResultAPIVersion) {
      fail(`v1alpha5 BuildResult apiVersion = ${JSON.stringify(nativeV1Alpha5Valid.build.apiVersion)}`);
    }
    if (nativeV1Alpha5Valid.build.diagnostics.length !== 0) {
      fail(`v1alpha5 valid Build diagnostics = ${stable(nativeV1Alpha5Valid.build.diagnostics)}`);
    }
    if ((nativeV1Alpha5Valid.build.outputFiles || []).length !== 3) {
      fail(`v1alpha5 valid Build emitted ${(nativeV1Alpha5Valid.build.outputFiles || []).length} files, want 3`);
    }
    console.log('ok: v1alpha5 valid Check returns zero diagnostics');
    console.log('ok: v1alpha5 valid Build succeeds with deterministic artifact');

    // (b) Trusted Wait-bearing fixture: zero-diagnostic successful Build with the
    // measured dependency closure and linked Ingress identity.
    if (nativeV1Alpha5Trusted.check.length !== 0) {
      fail(`v1alpha5 trusted fixture must Check with zero diagnostics, got ${stable(nativeV1Alpha5Trusted.check)}`);
    }
    if (!nativeV1Alpha5Trusted.build.success) {
      fail(`v1alpha5 trusted Build must succeed, diagnostics = ${stable(nativeV1Alpha5Trusted.build.diagnostics)}`);
    }
    if (nativeV1Alpha5Trusted.build.apiVersion !== v1alpha5ResultAPIVersion) {
      fail(`v1alpha5 trusted BuildResult apiVersion = ${JSON.stringify(nativeV1Alpha5Trusted.build.apiVersion)}`);
    }
    if (stable(nativeV1Alpha5Trusted.build.dependencyClosure) !== stable(['integration', 'tools'])) {
      fail(`v1alpha5 trusted dependencyClosure = ${stable(nativeV1Alpha5Trusted.build.dependencyClosure)}`);
    }
    const trustedPaths = (nativeV1Alpha5Trusted.build.outputFiles || []).map(file => file.path);
    for (const required of ['manifest.json', 'skills/example/compiled/procedure.json', 'skills/example/compiled/ingress-links.json']) {
      if (!trustedPaths.includes(required)) fail(`v1alpha5 trusted artifact is missing ${required}`);
    }
    console.log(`ok: v1alpha5 trusted Build succeeds with dependency closure (${trustedPaths.length} files)`);

    // (c) Invalid Procedure inside the complete trusted package: deterministic
    // located WAIT_* diagnostics and no artifact. The full-language pipeline also
    // reports inherited package/structure diagnostics for this deliberately broken
    // Procedure, so the Wait contract assertions below pin exactly the WAIT_*
    // identities; Native/Chromium equality still covers every diagnostic.
    const invalidWaitDiagnostics = nativeV1Alpha5Invalid.check.filter(diagnostic =>
      String(diagnostic.code).startsWith('WAIT_'));
    if (invalidWaitDiagnostics.length === 0) fail('v1alpha5 invalid Wait procedure unexpectedly produced no WAIT_* diagnostic');
    for (const diagnostic of invalidWaitDiagnostics) {
      if (!(diagnostic.line > 0) || !(diagnostic.column > 0)) fail(`v1alpha5 diagnostic ${diagnostic.code} lacks line/column`);
      if (diagnostic.path !== 'skills/example/assurance/procedure.yaml') fail(`v1alpha5 diagnostic path = ${JSON.stringify(diagnostic.path)}`);
    }
    if (stable(nativeV1Alpha5Invalid.build.diagnostics) !== stable(nativeV1Alpha5Invalid.check)) {
      fail('v1alpha5 invalid Check and failed Build diagnostics must be identical');
    }
    if (nativeV1Alpha5Invalid.build.success) fail('invalid v1alpha5 fixture unexpectedly Build succeeded');
    if ((nativeV1Alpha5Invalid.build.outputFiles || []).length !== 0) fail('failed v1alpha5 Build emitted artifact files');
    if (nativeV1Alpha5Invalid.build.apiVersion !== v1alpha5ResultAPIVersion) {
      fail(`v1alpha5 invalid BuildResult apiVersion = ${JSON.stringify(nativeV1Alpha5Invalid.build.apiVersion)}`);
    }
    console.log(`ok: v1alpha5 invalid WAIT_* diagnostics positioned and deterministic (${nativeV1Alpha5Invalid.check.length} diagnostics)`);

    // Native vs real Chromium Worker: valid successful Build.
    const browserV1Alpha5Valid = await cdpEvaluate(`http://127.0.0.1:${debugPort}/json/version`, `http://127.0.0.1:${port}/parity.html?request=${v1alpha5Requests.valid}`);
    if (!browserV1Alpha5Valid.ok) fail(`Chromium Worker v1alpha5 valid error: ${browserV1Alpha5Valid.error}`);
    if (browserV1Alpha5Valid.artifactError !== '') fail(`v1alpha5 artifact verification failed in Chromium Worker: ${browserV1Alpha5Valid.artifactError}`);
    console.log('ok: v1alpha5 Chromium Worker VerifyArtifact PASS');
    compare('v1alpha5 check', nativeV1Alpha5Valid.check, browserV1Alpha5Valid.check);
    compare('v1alpha5 build.success', nativeV1Alpha5Valid.build.success, browserV1Alpha5Valid.build.success);
    compare('v1alpha5 build.apiVersion', nativeV1Alpha5Valid.build.apiVersion, browserV1Alpha5Valid.build.apiVersion);
    compare('v1alpha5 build.diagnostics', nativeV1Alpha5Valid.build.diagnostics, browserV1Alpha5Valid.build.diagnostics);
    compare('v1alpha5 build.dependencyClosure', nativeV1Alpha5Valid.build.dependencyClosure, browserV1Alpha5Valid.build.dependencyClosure);
    compare('v1alpha5 build.outputFiles.paths',
      (nativeV1Alpha5Valid.build.outputFiles || []).map(file => file.path),
      (browserV1Alpha5Valid.build.outputFiles || []).map(file => file.path));
    const browserV1Alpha5ValidHashes = browserArtifactHashes(browserV1Alpha5Valid.build);
    compare('v1alpha5 successful-Build artifact hashes', nativeV1Alpha5Valid.build.outputFiles, browserV1Alpha5ValidHashes);
    console.log(`ok: v1alpha5 successful-Build artifact byte parity (${browserV1Alpha5ValidHashes.length} files)`);
    compare('v1alpha5 inspect', nativeV1Alpha5Valid.inspect, browserV1Alpha5Valid.inspect);

    // Native vs real Chromium Worker: invalid fixture.
    const browserV1Alpha5Invalid = await cdpEvaluate(`http://127.0.0.1:${debugPort}/json/version`, `http://127.0.0.1:${port}/parity.html?request=${v1alpha5Requests.invalid}`);
    if (!browserV1Alpha5Invalid.ok) fail(`Chromium Worker v1alpha5 invalid error: ${browserV1Alpha5Invalid.error}`);
    compare('v1alpha5 invalid-diagnostics', nativeV1Alpha5Invalid.check, browserV1Alpha5Invalid.check);
    compare('v1alpha5 invalid build.diagnostics', nativeV1Alpha5Invalid.build.diagnostics, browserV1Alpha5Invalid.build.diagnostics);
    compare('v1alpha5 invalid inspect', nativeV1Alpha5Invalid.inspect, browserV1Alpha5Invalid.inspect);

    // Native vs real Chromium Worker: trusted Wait-bearing successful Build.
    const browserV1Alpha5Trusted = await cdpEvaluate(`http://127.0.0.1:${debugPort}/json/version`, `http://127.0.0.1:${port}/parity.html?request=${v1alpha5Requests.trusted}`);
    if (!browserV1Alpha5Trusted.ok) fail(`Chromium Worker v1alpha5 trusted error: ${browserV1Alpha5Trusted.error}`);
    if (browserV1Alpha5Trusted.artifactError !== '') fail(`v1alpha5 trusted artifact verification failed in Chromium Worker: ${browserV1Alpha5Trusted.artifactError}`);
    compare('v1alpha5 trusted check', nativeV1Alpha5Trusted.check, browserV1Alpha5Trusted.check);
    compare('v1alpha5 trusted build.success', nativeV1Alpha5Trusted.build.success, browserV1Alpha5Trusted.build.success);
    compare('v1alpha5 trusted build.apiVersion', nativeV1Alpha5Trusted.build.apiVersion, browserV1Alpha5Trusted.build.apiVersion);
    compare('v1alpha5 trusted build.diagnostics', nativeV1Alpha5Trusted.build.diagnostics, browserV1Alpha5Trusted.build.diagnostics);
    compare('v1alpha5 trusted dependencyClosure', nativeV1Alpha5Trusted.build.dependencyClosure, browserV1Alpha5Trusted.build.dependencyClosure);
    compare('v1alpha5 trusted build.outputFiles.paths',
      (nativeV1Alpha5Trusted.build.outputFiles || []).map(file => file.path),
      (browserV1Alpha5Trusted.build.outputFiles || []).map(file => file.path));
    const browserV1Alpha5TrustedHashes = browserArtifactHashes(browserV1Alpha5Trusted.build);
    compare('v1alpha5 trusted successful-Build artifact hashes', nativeV1Alpha5Trusted.build.outputFiles, browserV1Alpha5TrustedHashes);
    console.log(`ok: v1alpha5 trusted successful-Build artifact byte parity (${browserV1Alpha5TrustedHashes.length} files)`);

    // Byte parity implies the Native manifest equals the browser manifest; the
    // browser manifest is decoded here to assert the linked Ingress identity and
    // dependency closure explicitly.
    const trustedManifest = browserArtifactJSON(browserV1Alpha5Trusted.build, 'manifest.json');
    if (!trustedManifest) fail('v1alpha5 trusted artifact has no manifest.json');
    if (trustedManifest.apiVersion !== 'connectionagent.io/compiler-artifact/v1alpha5' ||
        trustedManifest.languageTarget !== v1alpha5LanguageTarget) {
      fail(`v1alpha5 trusted manifest identity = ${JSON.stringify({ apiVersion: trustedManifest.apiVersion, languageTarget: trustedManifest.languageTarget })}`);
    }
    if (stable((trustedManifest.dependencyClosure || []).map(entry => entry.dependencyId)) !== stable(['integration', 'tools'])) {
      fail(`v1alpha5 trusted manifest dependencyClosure = ${stable(trustedManifest.dependencyClosure)}`);
    }
    const trustedSkill = (trustedManifest.skills || [])[0];
    const trustedWait = trustedSkill && (trustedSkill.waitLinks || [])[0];
    if (!trustedWait) fail('v1alpha5 trusted manifest has no Wait link');
    const trustedIngress = trustedWait.ingress || {};
    if (trustedIngress.dependencyAlias !== 'integration' ||
        trustedIngress.snapshotPath !== 'ingress/catalog.yaml' ||
        trustedIngress.operationId !== 'nsmf-pdusession.update-sm-context' ||
        trustedIngress.normalizedSchemaRef !== 'urn:example:schema:update-normalized' ||
        !String(trustedIngress.normalizedSchemaClosureDigest || '').startsWith('sha256:')) {
      fail(`v1alpha5 trusted Ingress identity = ${stable(trustedIngress)}`);
    }
    if (!trustedWait.response || trustedWait.response.mode !== 'none') {
      fail(`v1alpha5 trusted Wait response metadata = ${stable(trustedWait.response)}`);
    }
    if ((trustedManifest.outputFiles || []).length !== browserV1Alpha5TrustedHashes.length - 1) {
      fail('v1alpha5 trusted manifest outputFiles does not list every emitted file');
    }
    console.log('ok: v1alpha5 trusted linked Ingress metadata identical');
    console.log('ok: v1alpha5 Chromium Worker VerifyArtifact PASS');

    // Development Build phase: Compile -> Interface Link -> Build -> Verify.
    // The Native CLI and the real Chromium Worker must produce identical status
    // fields, diagnostics, artifactDigest and complete artifact bytes for the
    // canonical and reordered requests, and identical failure diagnostics for the
    // incomplete interface closure.
    const developmentRequests = writeDevelopmentRequests();
    const nativeDevelopment = nativeDevelopmentReference(developmentRequests.base, 'development-native-out');
    const nativeDevelopmentReordered = nativeDevelopmentReference(developmentRequests.reordered, 'development-native-reordered-out');
    const nativeDevelopmentNegative = nativeDevelopmentReference(developmentRequests.negative, 'development-native-negative-out');
    if (!nativeDevelopment.success) fail('Development positive fixture did not build natively: ' + stable(nativeDevelopment.diagnostics));
    if (nativeDevelopment.executionStatus !== 'NOT_ADMITTED') fail('Development executionStatus is not NOT_ADMITTED');
    if (nativeDevelopment.artifactDigest !== nativeDevelopmentReordered.artifactDigest) fail('reordered Development input changed the Native artifactDigest');
    if (stable(nativeDevelopment.outputFiles) !== stable(nativeDevelopmentReordered.outputFiles)) fail('reordered Development input changed Native artifact bytes');
    const baseArtifact = await cdpEvaluate('http://127.0.0.1:' + debugPort + '/json/version', 'http://127.0.0.1:' + port + '/parity.html?development=1&request=' + developmentRequests.base);
    if (!baseArtifact.ok) fail('Chromium Worker Development error: ' + baseArtifact.error);
    if (baseArtifact.verifyError !== '') fail('Chromium Worker Development Verify failed: ' + baseArtifact.verifyError);
    const browserDevelopment = baseArtifact.build;
    compare('development build.success', nativeDevelopment.success, browserDevelopment.success);
    compare('development build.sourceStatus', nativeDevelopment.sourceStatus, browserDevelopment.sourceStatus);
    compare('development build.interfaceLinkStatus', nativeDevelopment.interfaceLinkStatus, browserDevelopment.interfaceLinkStatus);
    compare('development build.artifactStatus', nativeDevelopment.artifactStatus, browserDevelopment.artifactStatus);
    compare('development build.executionStatus', nativeDevelopment.executionStatus, browserDevelopment.executionStatus);
    compare('development build.diagnostics', nativeDevelopment.diagnostics, browserDevelopment.diagnostics);
    compare('development build.artifactDigest', nativeDevelopment.artifactDigest, browserDevelopment.artifactDigest);
    compare('development build.outputFiles', nativeDevelopment.outputFiles, browserDevelopment.outputFiles);
    const developmentObligations = browserArtifactJSON(browserDevelopment, 'obligations.json');
    if (!Array.isArray(developmentObligations) || developmentObligations.length === 0) fail('Development artifact has no obligations');
    if (!developmentObligations.some(obligation => obligation.rule === 'EVIDENCE_GUARANTEE' && obligation.status === 'UNRESOLVED')) {
      fail('Development artifact has no UNRESOLVED EVIDENCE_GUARANTEE obligation');
    }
    console.log('ok: Development LINKED/BUILT/NOT_ADMITTED byte parity (' + browserDevelopment.outputFiles.length + ' files)');
    const reorderedBrowser = await cdpEvaluate('http://127.0.0.1:' + debugPort + '/json/version', 'http://127.0.0.1:' + port + '/parity.html?development=1&request=' + developmentRequests.reordered);
    if (!reorderedBrowser.ok) fail('Chromium Worker Development reordered error: ' + reorderedBrowser.error);
    compare('development reordered build.success', true, reorderedBrowser.build.success);
    compare('development reordered artifact bytes', browserDevelopment.outputFiles, reorderedBrowser.build.outputFiles);
    console.log('ok: Development reordered input produces identical artifact bytes');
    if (nativeDevelopmentNegative.success) fail('Development negative fixture unexpectedly built');
    if (nativeDevelopmentNegative.diagnostics.length === 0) fail('Development negative fixture produced no diagnostics');
    const negativeBrowser = await cdpEvaluate('http://127.0.0.1:' + debugPort + '/json/version', 'http://127.0.0.1:' + port + '/parity.html?development=1&request=' + developmentRequests.negative);
    if (!negativeBrowser.ok) fail('Chromium Worker Development negative error: ' + negativeBrowser.error);
    compare('development negative build.success', false, negativeBrowser.build.success);
    compare('development negative diagnostics', nativeDevelopmentNegative.diagnostics, negativeBrowser.build.diagnostics);
    console.log('ok: Development failure diagnostics identical between Native and Chromium Worker');

    console.log('PASS: Native and real Chromium Worker Development Build parity');
    console.log('PASS: Native and real Chromium Worker v1alpha5 artifact parity');

    console.log('PASS: Native and real Chromium Worker projections are identical');
  } finally {
    chrome.kill();
    server.close();
  }
}

main().catch(error => fail(error.stack || String(error)));


