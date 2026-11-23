#!/usr/bin/env node
// Local-document integrity and navigation check. Does not use the network.
// Run from the repository root: node scripts/check-doc-links.mjs
import { execFileSync } from 'node:child_process';
import { existsSync, readFileSync, statSync } from 'node:fs';
import path from 'node:path';

const root = process.cwd();
const tracked = new Set(execFileSync('git', ['ls-files', '-z'], { cwd: root })
  .toString('utf8').split('\0').filter(Boolean).map(p => p.replaceAll('\\', '/')));

const shouldAudit = p =>
  p === 'README.md' || p === 'README.zh-CN.md' ||
  (p.startsWith('docs/') && p.endsWith('.md')) ||
  (p.startsWith('contracts/') && /\/README\.md$/.test(p)) ||
  (p.startsWith('examples/') && /\/README(?:\.zh-CN)?\.md$/.test(p));

const errors = [];
const warnings = [];
let checkedLinks = 0;
let files = 0;
const contentByPath = new Map();

function ensureFile(file) {
  if (!tracked.has(file)) {
    errors.push(`missing maintained document: ${file}`);
    return '';
  }
  const content = readFileSync(path.join(root, file), 'utf8');
  contentByPath.set(file, content);
  return content;
}

for (const file of [...tracked].filter(shouldAudit).sort()) {
  files++;
  const source = ensureFile(file);
  let inFence = false;
  let delimiter = '';
  let lineNo = 0;
  for (const line of source.split(/\r?\n/)) {
    lineNo++;
    const fence = line.match(/^\s{0,3}(?:>\s*)?(`{3,}|~{3,})/);
    if (fence) {
      if (!inFence) {
        inFence = true;
        delimiter = fence[1][0];
      } else if (fence[1][0] === delimiter) {
        inFence = false;
      }
      continue;
    }
    if (inFence) continue;
    // Inline links/images plus reference-style definitions.
    const links = [...line.matchAll(/!?\[[^\]\n]*\]\((<[^>]+>|[^)]*)\)/g)]
      .map(m => m[1]);
    const reference = line.match(/^\s{0,3}\[[^\]]+\]:\s*(<[^>]+>|\S+)/);
    if (reference) links.push(reference[1]);
    for (const value of links) {
      let uri = value.trim().replace(/^<|>$/g, '').split(/\s+["']/)[0];
      if (!uri || uri.startsWith('#') || uri.startsWith('//') ||
          /^[a-zA-Z][a-zA-Z\d+.-]*:/.test(uri)) continue;
      try { uri = decodeURIComponent(uri); } catch {
        errors.push(`${file}:${lineNo}: invalid URI encoding ${uri}`);
        continue;
      }
      const pathname = uri.split(/[?#]/, 1)[0];
      if (!pathname) continue;
      // Relative Markdown URLs are resolved against their source file, as
      // GitHub Markdown does. A leading slash is repository-root relative.
      const target = pathname.startsWith('/')
        ? path.resolve(root, '.' + pathname)
        : path.resolve(root, path.dirname(file), pathname);
      const relative = path.relative(root, target).replaceAll('\\', '/');
      if (relative.startsWith('../') || relative === '..') {
        errors.push(`${file}:${lineNo}: link escapes repository: ${uri}`);
        continue;
      }
      if (!existsSync(target)) {
        errors.push(`${file}:${lineNo}: broken local link ${uri}`);
        continue;
      }
      if (statSync(target).isFile() && !tracked.has(relative)) {
        warnings.push(`${file}:${lineNo}: link targets an untracked file ${uri}`);
      }
      checkedLinks++;
    }
  }
}

const authoritative = [
  'README.md','README.zh-CN.md','docs/README.md','docs/README.zh-CN.md',
  'docs/WORK.md','docs/cli.md','docs/cli.zh-CN.md',
  'docs/design/README.md','docs/design/README.zh-CN.md',
  'docs/workstreams/README.md','docs/workstreams/README.zh-CN.md',
  'examples/README.md','examples/README.zh-CN.md',
  'docs/releases/ca-skill-compiler-v2.1.0.md',
];
for (const file of authoritative) {
  const body = contentByPath.get(file) ?? ensureFile(file);
  if (body && !body.includes('v2.1.0')) {
    errors.push(`${file}: current maintained entry must identify released v2.1.0`);
  }
}
for (const file of tracked) {
  if (!/^docs\/workstreams\/.*\.md$/.test(file) ||
      /^docs\/workstreams\/README(?:\.zh-CN)?\.md$/.test(file)) continue;
  const text = contentByPath.get(file) ?? ensureFile(file);
  if (!text.slice(0,850).includes('historical-docs-v2.1.0')) {
    errors.push(`${file}: missing historical/non-normative header`);
  }
}

for (const warning of warnings) console.warn('WARN:', warning);
for (const error of errors) console.error('ERROR:', error);
console.log(`documentation integrity: ${files} Markdown files; ${checkedLinks} local links; ${errors.length} errors, ${warnings.length} warnings`);
if (errors.length > 0) process.exitCode = 1;
