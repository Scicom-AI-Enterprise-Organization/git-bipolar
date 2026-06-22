#!/usr/bin/env node
// sync-web.mjs — keep the web installer (docs/index.html) byte-for-byte in
// sync with the canonical shell blocks defined in internal/shell/shell.go.
//
// The Go file is the single source of truth. This script extracts the
// `git-dispatch` managed block and the git-aliases block, then injects them
// into the two `<script type="text/plain">` placeholders in docs/index.html.
//
//   Run:  node scripts/sync-web.mjs      (or `make sync-web`)
//
// It is idempotent and fails loudly if anything can't be found.

import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const shellGo = resolve(root, 'internal/shell/shell.go');
const htmlPath = resolve(root, 'docs/index.html');

const go = readFileSync(shellGo, 'utf8');

// Pull a backtick-delimited Go const string by name.
function rawConst(name) {
  const m = go.match(new RegExp('const\\s+' + name + '\\s*=\\s*`([\\s\\S]*?)`'));
  if (!m) throw new Error(`could not find const ${name} in ${shellGo}`);
  return m[1];
}

function scalarConst(name) {
  const m = go.match(new RegExp('const\\s+' + name + '\\s*=\\s*"([^"]*)"'));
  if (!m) throw new Error(`could not find const ${name} in ${shellGo}`);
  return m[1];
}

// ManagedBlock = BlockStart + "\n" + shellFunction + "\n" + BlockEnd
const managedBlock =
  scalarConst('BlockStart') + '\n' + rawConst('shellFunction') + '\n' + scalarConst('BlockEnd');

// AliasesBlock = AliasesBlockStart + "\n" + aliasesContent + "\n" + AliasesBlockEnd
const aliasesBlock =
  scalarConst('AliasesBlockStart') + '\n' + rawConst('aliasesContent') + '\n' + scalarConst('AliasesBlockEnd');

// A raw-text <script> element ends only at the literal "</script" — guard against it.
for (const [label, body] of [['block', managedBlock], ['aliases', aliasesBlock]]) {
  if (/<\/script/i.test(body)) {
    throw new Error(`${label} contains "</script" which would break the HTML embedding`);
  }
}

let html = readFileSync(htmlPath, 'utf8');

function inject(id, content) {
  const re = new RegExp(
    '(<script type="text/plain" id="' + id + '">)[\\s\\S]*?(</script>)'
  );
  if (!re.test(html)) throw new Error(`placeholder script#${id} not found in ${htmlPath}`);
  // Wrap in newlines for readability; the page .trim()s before use.
  // NB: use a replacer FUNCTION, not a string — a string replacement would
  // interpret `$1`/`$2` inside `content` (the shell block has `$1`) as
  // capture-group references and corrupt the block.
  html = html.replace(re, (_m, open, close) => open + '\n' + content + '\n' + close);
}

const before = html;
inject('bipolar-block', managedBlock);
inject('bipolar-aliases', aliasesBlock);

// --check: verify the committed file is already in sync (for CI), don't write.
if (process.argv.includes('--check')) {
  if (html !== before) {
    console.error('docs/index.html is OUT OF SYNC with internal/shell/shell.go.');
    console.error('Run `make sync-web` and commit the result.');
    process.exit(1);
  }
  console.log('docs/index.html is in sync with internal/shell/shell.go');
  process.exit(0);
}

writeFileSync(htmlPath, html);
console.log('synced docs/index.html from internal/shell/shell.go');
console.log(`  git-dispatch block: ${managedBlock.split('\n').length} lines`);
console.log(`  aliases block:      ${aliasesBlock.split('\n').length} lines`);
