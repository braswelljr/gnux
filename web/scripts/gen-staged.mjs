// The pre-commit hook's generation step (lefthook.yml): when a commit changes
// what the website is generated from, regenerates the docs, the YAML reference
// and, for a change to the program, the wasm, and stages the result. Vercel
// deploys these committed files and generates nothing.
//
// The wasm is stamped with the latest tag, not the commit, so that a commit
// which does not touch the program never produces a new 14 MB binary.
import { execFileSync, spawnSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const git = (...args) => execFileSync('git', args, { cwd: root, encoding: 'utf8' }).trim();
const lines = (text) => text.split('\n').filter(Boolean);

const isProgram = (f) =>
  !f.startsWith('web/') && (f === 'go.mod' || f === 'go.sum' || f.endsWith('.go'));
const isDocsSource = (f) =>
  isProgram(f) || f === 'CHANGELOG.md' || f.startsWith('documentation/guides/');

const staged = lines(git('diff', '--cached', '--name-only'));
const sources = staged.filter(isDocsSource);
if (sources.length === 0) process.exit(0);

// Generation reads the working tree, so a partly staged file would put output
// in the commit that does not match the source beside it.
const unstaged = new Set(lines(git('diff', '--name-only')));
const partial = sources.filter((f) => unstaged.has(f));
if (partial.length > 0) {
  console.error(`gen-staged: stage or stash the rest of: ${partial.join(', ')}`);
  process.exit(1);
}

const wasm = staged.some((f) => isProgram(f) && !f.endsWith('_test.go'));
const steps = wasm ? ['docs', 'yaml', 'wasm'] : ['docs', 'yaml'];
const result = spawnSync('node', [join(root, 'web/scripts/gen.mjs'), ...steps], {
  cwd: root,
  stdio: 'inherit',
  env: {
    ...process.env,
    GNUX_GEN_STRICT: '1',
    GNUX_WASM_VERSION: git('describe', '--tags', '--abbrev=0')
  }
});
if (result.status !== 0) process.exit(result.status ?? 1);

const outputs = ['web/content/docs', 'web/public/reference'];
if (wasm) outputs.push('web/public/gnux.wasm', 'web/public/wasm_exec.js');
git('add', '-A', '--', ...outputs);
