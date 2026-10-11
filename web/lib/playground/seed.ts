/**
 * The files a fresh playground starts with: a small home directory with enough
 * in it that the first commands a visitor tries have something to act on.
 */
import { MemFS } from './memfs.ts';

export const HOME = '/home/guest';

/** The environment the gnux process starts with. */
export const ENV: Record<string, string> = {
  HOME,
  USER: 'guest',
  LOGNAME: 'guest',
  SHELL: '/bin/gnux',
  TMPDIR: '/tmp',
  LANG: 'C.UTF-8',
  TERM: 'dumb'
};

const README = `Welcome to the gnux playground.

This is the real gnux, compiled to WebAssembly and running in your browser
against a filesystem that lives in memory. Nothing here touches your disk, and
"Reset" brings every file back.

Things to try:

  lsx -la
  catx notes/todo.txt
  headx -n 5 logs/app.log
  wcx -l logs/app.log
  mkdirx -p build/out && touchx build/out/a.txt && rmx -rv build
  echox "hello" > hello.txt && catx hello.txt

Type "help" for the shell's syntax, and "TOOL --help" for a tool's options.
`;

const TODO = `- [x] rewrite rm in Go
- [x] one binary, every platform
- [ ] try the playground
- [ ] read the docs
`;

const IDEAS = `Portable build scripts
=====================

A script written for "gnux shell" runs the same on Linux, macOS and Windows,
because every command in it is a gnux tool running in-process.

    mkdirx -p dist && cpx -r assets dist/ && lsx -l dist
`;

const SHOPPING = `apples
bread
coffee
dark chocolate
eggs
`;

const HELLO_GO = `package main

import "fmt"

func main() {
	fmt.Println("hello from gnux")
}
`;

const CLEAN_GO = `package main

import (
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/rm"
)

// rmx -rf build dist
func main() {
	engine := rm.New(rm.Options{Recursive: true, Force: true}, core.Stdio())
	_ = engine.Run([]string{"build", "dist"})
}
`;

const BUILD_SCRIPT = `# A portable build script: run it with "gnux shell build.gnux".
OUT=dist
rmx -rf $OUT
mkdirx -p $OUT/bin
echox "built at $OUT" > $OUT/BUILD
`;

/** A deterministic 40-line log, for headx, tailx and wcx. */
function appLog(): string {
  const levels = ['INFO', 'INFO', 'DEBUG', 'INFO', 'WARN', 'INFO', 'INFO', 'ERROR'];
  const messages = [
    'server listening on :8080',
    'GET /docs 200 12ms',
    'cache miss key=tools/rmx',
    'GET /playground 200 31ms',
    'slow request path=/gnux.wasm took=1.2s',
    'GET /docs/tools/lsx 200 9ms',
    'worker 3 finished job=reindex',
    'upstream timed out, retrying (attempt 2)'
  ];
  const lines: string[] = [];
  for (let i = 0; i < 40; i++) {
    const minute = String(Math.floor(i / 4)).padStart(2, '0');
    const second = String((i * 13) % 60).padStart(2, '0');
    lines.push(
      `2026-01-15T09:${minute}:${second}Z ${levels[i % levels.length].padEnd(5)} ${messages[i % messages.length]}`
    );
  }
  return `${lines.join('\n')}\n`;
}

/** Builds the starting filesystem, with the working directory set to {@link HOME}. */
export function createSeededFS(options: ConstructorParameters<typeof MemFS>[0] = {}): MemFS {
  const fs = new MemFS(options);

  // System directories belong to root, as they would on a real machine.
  fs.asRoot(() => {
    fs.mkdirSync('/home', 0o755);
    fs.mkdirSync('/tmp', 0o777);
    fs.chmodSync('/tmp', 0o1777);
    fs.mkdirSync(HOME, 0o755);
    fs.chownSync(HOME, fs.uid, fs.gid);
  });

  fs.chdir(HOME);
  fs.writeFileSync('README.txt', README, 0o644);

  fs.mkdirSync('notes', 0o755);
  fs.writeFileSync('notes/todo.txt', TODO, 0o644);
  fs.writeFileSync('notes/ideas.md', IDEAS, 0o644);
  fs.writeFileSync('notes/shopping.txt', SHOPPING, 0o644);

  fs.mkdirSync('src', 0o755);
  fs.writeFileSync('src/hello.go', HELLO_GO, 0o644);
  fs.writeFileSync('src/clean.go', CLEAN_GO, 0o644);
  fs.writeFileSync('src/build.gnux', BUILD_SCRIPT, 0o644);

  fs.mkdirSync('logs', 0o755);
  fs.writeFileSync('logs/app.log', appLog(), 0o644);

  // A symbolic link, so realpathx and lsx -l have something to resolve.
  fs.symlinkSync('logs/app.log', 'latest.log');

  return fs;
}
