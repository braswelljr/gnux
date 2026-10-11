// Runs before `pnpm dev` and `pnpm build`: regenerates what the site takes from
// the Go project at the repository root: the documentation content and the
// WebAssembly build of gnux.
//
// The documentation is committed, so a host without Go can still build from
// it; gnux.wasm is not. When Go is missing and something has to be built (a
// fresh Vercel build, say), the Go release named in ../go.mod is downloaded
// from go.dev, checked against the SHA-256 go.dev publishes, and kept under
// node_modules/.cache for the next build. A machine with Go uses its own.
//
// A failed step is only fatal when the file it would have produced is not
// there at all.
//
// Set GNUX_SKIP_GEN=1 to skip generation, and GNUX_FETCH_GO=0 to never
// download Go.
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { copyFileSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const web = join(dirname(fileURLToPath(import.meta.url)), '..');
const root = join(web, '..');
const cache = join(web, 'node_modules', '.cache', 'gnux');

/** Whether `binary` can be started at all, whatever it then prints or returns. */
function has(binary) {
  const probe = spawnSync(binary, ['version'], {
    stdio: 'ignore',
    shell: process.platform === 'win32'
  });
  // On Windows the shell starts either way, and reports a missing program as 9009 or 1.
  return !probe.error && (process.platform !== 'win32' || probe.status === 0);
}

/** Runs a command from the repository root, printing its output; true on success. */
function run(binary, args, env = {}) {
  const result = spawnSync(binary, args, {
    cwd: root,
    stdio: 'inherit',
    env: { ...process.env, ...env },
    shell: process.platform === 'win32'
  });
  return !result.error && result.status === 0;
}

/** A command's standard output, trimmed, or '' when it fails. */
function output(binary, args, env = {}) {
  const result = spawnSync(binary, args, {
    cwd: root,
    encoding: 'utf8',
    env: { ...process.env, ...env },
    shell: process.platform === 'win32'
  });
  return result.status === 0 ? result.stdout.trim() : '';
}

/** The Go version go.mod asks for, as a release name: `go 1.25.0` is go1.25.0. */
function goRelease() {
  const line = readFileSync(join(root, 'go.mod'), 'utf8').match(/^go\s+(\d+\.\d+(?:\.\d+)?)\s*$/m);
  if (!line) throw new Error('no go directive in go.mod');
  return `go${line[1]}`;
}

/**
 * Downloads and unpacks the Go release go.mod names, unless an earlier build
 * already did, and returns its `go` binary. The archive is checked against the
 * checksum go.dev publishes for it before anything is unpacked.
 */
/** Where the downloaded Go for this host lives, or null when go.dev has none for it. */
function goDownload() {
  const release = goRelease();
  const os = { linux: 'linux', darwin: 'darwin' }[process.platform];
  const arch = { x64: 'amd64', arm64: 'arm64' }[process.arch];
  if (!os || !arch) return null;
  const file = `${release}.${os}-${arch}.tar.gz`;
  const dir = join(cache, `${release}.${os}-${arch}`);
  return { file, dir, go: join(dir, 'go', 'bin', 'go') };
}

async function fetchGo() {
  const target = goDownload();
  if (!target) {
    throw new Error(`no Go download for ${process.platform}/${process.arch}; install Go instead`);
  }
  const { file, dir, go } = target;
  if (existsSync(go)) return go;

  console.log(`gen: go not found, downloading ${file} from go.dev`);

  const listing = await fetch('https://go.dev/dl/?mode=json&include=all');
  if (!listing.ok) throw new Error(`go.dev/dl answered ${listing.status}`);
  const releases = await listing.json();
  const expected = releases
    .flatMap((r) => (Array.isArray(r.files) ? r.files : []))
    .find((f) => f.filename === file)?.sha256;
  if (!expected) throw new Error(`go.dev lists no ${file}`);

  const archive = await fetch(`https://go.dev/dl/${file}`);
  if (!archive.ok) throw new Error(`downloading ${file}: HTTP ${archive.status}`);
  const bytes = Buffer.from(await archive.arrayBuffer());
  const actual = createHash('sha256').update(bytes).digest('hex');
  if (actual !== expected) {
    throw new Error(`${file} does not match its published checksum (got ${actual})`);
  }

  rmSync(dir, { recursive: true, force: true });
  mkdirSync(dir, { recursive: true });
  const tarball = join(dir, file);
  writeFileSync(tarball, bytes);
  const unpacked = spawnSync('tar', ['-xzf', tarball, '-C', dir], { stdio: 'inherit' });
  rmSync(tarball, { force: true });
  if (unpacked.status !== 0 || !existsSync(go)) throw new Error(`unpacking ${file} failed`);
  return go;
}

/** The version stamped into the wasm build, as `make web/wasm` stamps the binary. */
function version() {
  return (
    output('git', ['describe', '--tags', '--always', '--dirty']) ||
    process.env.VERCEL_GIT_COMMIT_SHA?.slice(0, 7) ||
    'dev'
  );
}

const steps = [
  {
    name: 'docs',
    output: 'content/docs/index.mdx',
    build: (go, env) => run(go, ['run', './tools/gendocs', 'web/content/docs', 'mdx'], env)
  },
  {
    // The same build as `make web/wasm`, without needing make.
    name: 'wasm',
    output: 'public/gnux.wasm',
    build: (go, env) => {
      const ldflags = `-s -w -X github.com/braswelljr/gnux/internal/common.Version=${version()}`;
      const built = run(
        go,
        [
          'build',
          '-trimpath',
          '-buildvcs=false',
          '-ldflags',
          ldflags,
          '-o',
          'web/public/gnux.wasm',
          './tools/wasm'
        ],
        { ...env, GOOS: 'js', GOARCH: 'wasm' }
      );
      if (!built) return false;
      const goroot = output(go, ['env', 'GOROOT'], env);
      copyFileSync(
        join(goroot, 'lib', 'wasm', 'wasm_exec.js'),
        join(web, 'public', 'wasm_exec.js')
      );
      return true;
    }
  }
];

const missingOutputs = () =>
  steps.map((step) => step.output).filter((file) => !existsSync(join(web, file)));

const only = process.argv.slice(2);
const selected = only.length > 0 ? steps.filter((step) => only.includes(step.name)) : steps;

if (process.env.GNUX_SKIP_GEN === '1') {
  console.log('gen: GNUX_SKIP_GEN=1, using the committed content/docs and public/gnux.wasm');
} else {
  let go = has('go') ? 'go' : null;
  // A downloaded Go keeps its build and module caches beside it, so the next
  // build on the same host (Vercel restores node_modules) starts warm, and it
  // never switches itself to another toolchain.
  let env = {};
  // One downloaded earlier is used whenever it is there, so the wasm keeps up
  // with the code; a new download only happens when something is missing.
  const cached = goDownload();
  const wanted = missingOutputs().length > 0 || (cached !== null && existsSync(cached.go));
  if (!go && wanted && process.env.GNUX_FETCH_GO !== '0') {
    try {
      go = await fetchGo();
      env = {
        GOTOOLCHAIN: 'local',
        GOCACHE: join(cache, 'go-build'),
        GOMODCACHE: join(cache, 'go-mod')
      };
    } catch (err) {
      console.warn(`gen: could not download Go: ${err instanceof Error ? err.message : err}`);
    }
  }

  if (!go) {
    console.log('gen: go not found, using the committed content/docs and public/gnux.wasm');
  } else {
    for (const step of selected) {
      console.log(`gen: ${step.name}`);
      if (!step.build(go, env)) {
        console.warn(`gen: ${step.name} failed; continuing with the existing ${step.output}`);
      }
    }
  }
}

const missing = missingOutputs();
if (missing.length > 0) {
  console.error(
    `gen: missing ${missing.join(' and ')}. Install Go and run \`pnpm gen\` in web/ ` +
      '(or `make web/docs web/wasm` at the repository root) to produce them.'
  );
  process.exit(1);
}
