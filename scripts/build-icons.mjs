/**
 * Icon rasterizer for Xin Skill Vault.
 *
 * Serves brand/ + a generator page, the browser renders each size on canvas
 * and POSTs the PNG back. Run this, open /gen, then stop it.
 *
 *   node scripts/build-icons.mjs
 *   open http://127.0.0.1:4317/gen
 */
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const BRAND = path.join(ROOT, 'brand');
const OUT = path.join(BRAND, 'icons');
fs.mkdirSync(OUT, { recursive: true });

// size, filename, withRocket
const PLAN = [
  [16, 'icon-16.png', false],
  [24, 'icon-24.png', false],
  [32, 'icon-32.png', false],
  [48, 'icon-48.png', true],
  [64, 'icon-64.png', true],
  [128, 'icon-128.png', true],
  [180, 'icon-180.png', true],
  [256, 'icon-256.png', true],
  [512, 'icon-512.png', true],
  [1024, 'icon-1024.png', true],
];

const GEN = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>rendering…</title>
<style>html,body{margin:0;background:#fff;font:12px/1.5 system-ui}canvas{display:none}</style>
</head><body>
<pre id="log">rendering…</pre>
<script>
const PLAN = ${JSON.stringify(PLAN)};
const log = (m) => { document.getElementById('log').textContent += '\\n' + m; };
(async () => {
  for (const [size, name, rocket] of PLAN) {
    const img = new Image();
    img.src = rocket ? '/logo.svg' : '/logo-small.svg';
    await img.decode();
    const c = document.createElement('canvas');
    c.width = c.height = size;
    const ctx = c.getContext('2d');
    ctx.imageSmoothingEnabled = true;
    ctx.imageSmoothingQuality = 'high';
    ctx.drawImage(img, 0, 0, size, size);
    const blob = await new Promise((r) => c.toBlob(r, 'image/png'));
    await fetch('/save?name=' + name, { method: 'POST', body: blob });
    log(size + 'px ✓');
  }
  document.title = 'DONE';
  log('all done');
})().catch((e) => { log('ERR ' + e.message); document.title = 'ERROR'; });
</script></body></html>`;

const MIME = { '.svg': 'image/svg+xml', '.png': 'image/png', '.html': 'text/html; charset=utf-8' };

http.createServer(async (req, res) => {
  const url = new URL(req.url, 'http://127.0.0.1');

  if (req.method === 'POST' && url.pathname === '/save') {
    const name = url.searchParams.get('name') || 'out.png';
    if (!/^[\w.-]+$/.test(name)) { res.writeHead(400).end('bad name'); return; }
    const chunks = [];
    for await (const chunk of req) chunks.push(chunk);
    const buf = Buffer.concat(chunks);
    fs.writeFileSync(path.join(OUT, name), buf);
    console.log('saved', name, buf.length, 'bytes');
    res.writeHead(200, { 'content-type': 'text/plain' }).end('ok');
    return;
  }

  if (url.pathname === '/gen') {
    res.writeHead(200, { 'content-type': MIME['.html'] }).end(GEN);
    return;
  }

  const rel = decodeURIComponent(url.pathname).replace(/^\/+/, '');
  const file = path.join(BRAND, rel);
  if (!file.startsWith(BRAND) || !fs.existsSync(file) || !fs.statSync(file).isFile()) {
    res.writeHead(404).end('not found');
    return;
  }
  res.writeHead(200, { 'content-type': MIME[path.extname(file)] || 'application/octet-stream' });
  fs.createReadStream(file).pipe(res);
}).listen(4317, '127.0.0.1', () => {
  console.log('icon server: http://127.0.0.1:4317/gen');
  console.log('out dir:', OUT);
});
