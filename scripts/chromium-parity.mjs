// Chromium Worker parity harness (WP11).
//
// It builds the Go/WASM compiler, serves a static page, launches real headless
// Chromium, runs Check/Build/Inspect through the same Web Worker transport used by
// Skill Studio, and compares the browser results with the Native Go reference for
// the same BuildRequest. Run from the repository root:
//
//   GOTOOLCHAIN=go1.27.4 node scripts/chromium-parity.mjs
//
// Exit code 0 only when every compared projection is identical.

import { spawn, execFileSync } from 'node:child_process';
import { createServer } from 'node:http';
import { copyFileSync, mkdirSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import path from 'node:path';

const repoRoot = process.cwd();
const webRoot = path.join(repoRoot, '.parity-web');
const projectRoot = path.join(repoRoot, 'web', 'parity-project');
const chromePath = process.env.CHROME_PATH || 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const env = { ...process.env, GOTOOLCHAIN: 'go1.27.4', GOWORK: 'off', CGO_ENABLED: '0' };

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
    apiVersion: 'connectionagent.io/compiler-build-request/v1alpha4',
    languageTarget: 'connectionagent.io/v1alpha4',
    files: collectFiles(projectRoot),
    resolvedDependencies: [],
  };
  writeFileSync(path.join(webRoot, 'parity-request.json'), JSON.stringify(request));
}

function nativeReference() {
  const stdout = execFileSync('go', ['run', './cmd/parity-replay', projectRoot], { cwd: repoRoot, env }).toString();
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

    console.log('PASS: Native and real Chromium Worker projections are identical');
  } finally {
    chrome.kill();
    server.close();
  }
}

main().catch(error => fail(error.stack || String(error)));
