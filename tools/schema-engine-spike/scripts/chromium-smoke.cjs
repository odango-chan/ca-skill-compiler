const fs = require("node:fs");
const http = require("node:http");
const path = require("node:path");

const root = process.env.SPIKE_WEB_ROOT;
if (!root) throw new Error("set SPIKE_WEB_ROOT to the staged WASM artifacts directory");
const debugPort = Number(process.env.CHROMIUM_DEBUG_PORT || 9222);
const mime = {".html": "text/html; charset=utf-8", ".js": "text/javascript; charset=utf-8", ".wasm": "application/wasm"};
const server = http.createServer((req, res) => {
  const pathname = decodeURIComponent(new URL(req.url, "http://127.0.0.1").pathname);
  const target = path.resolve(root, `.${pathname}`);
  if (!target.startsWith(path.resolve(root) + path.sep)) { res.writeHead(403).end(); return; }
  fs.readFile(target, (err, data) => {
    if (err) { res.writeHead(404).end(); return; }
    res.writeHead(200, {"Content-Type": mime[path.extname(target)] || "application/octet-stream", "Cache-Control": "no-store"});
    res.end(data);
  });
});

class CDP {
  constructor(url) {
    this.ws = new WebSocket(url);
    this.nextID = 0;
    this.pending = new Map();
    this.ready = new Promise((resolve, reject) => {
      this.ws.addEventListener("open", resolve, {once: true});
      this.ws.addEventListener("error", reject, {once: true});
    });
    this.ws.addEventListener("message", (event) => {
      const msg = JSON.parse(event.data);
      if (!msg.id) return;
      const pending = this.pending.get(msg.id);
      if (!pending) return;
      this.pending.delete(msg.id);
      if (msg.error) pending.reject(new Error(JSON.stringify(msg.error)));
      else pending.resolve(msg.result);
    });
  }
  async command(method, params = {}) {
    await this.ready;
    const id = ++this.nextID;
    return new Promise((resolve, reject) => {
      this.pending.set(id, {resolve, reject});
      this.ws.send(JSON.stringify({id, method, params}));
    });
  }
  async evaluate(expression) {
    const result = await this.command("Runtime.evaluate", {expression, returnByValue: true, awaitPromise: true});
    if (result.exceptionDetails) throw new Error(result.exceptionDetails.text || "page evaluation failed");
    return result.result.value;
  }
  close() { this.ws.close(); }
}

const delay = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
server.listen(0, "127.0.0.1", async () => {
  const address = server.address();
  let targetID;
  let cdp;
  try {
    const url = `http://127.0.0.1:${address.port}/index.html`;
    const created = await fetch(`http://127.0.0.1:${debugPort}/json/new?${encodeURIComponent(url)}`, {method: "PUT"});
    if (!created.ok) throw new Error(`DevTools target creation failed: ${created.status} ${await created.text()}`);
    const target = await created.json();
    targetID = target.id;
    cdp = new CDP(target.webSocketDebuggerUrl);
    await cdp.command("Page.enable");
    await cdp.command("Runtime.enable");
    await cdp.command("Page.navigate", {url});
    const deadline = Date.now() + 60000;
    let state = "pending";
    let report = null;
    while (Date.now() < deadline) {
      const snapshot = await cdp.evaluate("JSON.stringify({state:document.body&&document.body.dataset.test,text:document.querySelector('#result')&&document.querySelector('#result').textContent})");
      const parsed = JSON.parse(snapshot);
      state = parsed.state;
      if (parsed.text) {
        try { report = JSON.parse(parsed.text); } catch {}
      }
      if (state && state !== "pending") break;
      await delay(200);
    }
    if (state !== "pass") throw new Error(`Chromium worker state=${state}: ${JSON.stringify(report)}`);
    process.stdout.write(JSON.stringify(report, null, 2) + "\n");
  } catch (error) {
    process.exitCode = 1;
    console.error(error);
  } finally {
    if (cdp) cdp.close();
    if (targetID) {
      await fetch(`http://127.0.0.1:${debugPort}/json/close/${targetID}`).catch(() => {});
    }
    server.close();
  }
});
