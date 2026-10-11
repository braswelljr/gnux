/**
 * Boots the gnux WebAssembly build against a {@link MemFS}.
 *
 * Go's js/wasm runtime reads the Node-style `fs`, `process` and `path` globals
 * once, while the program starts, and keeps those references. So each boot
 * installs the objects for its own filesystem, starts the program, and takes
 * the API it publishes; an earlier instance keeps talking to its own
 * filesystem and is simply dropped.
 *
 * Nothing here touches the DOM or the network: the caller supplies how to load
 * `wasm_exec.js` and how to instantiate the module, so the same code runs in
 * the browser and under `node --test`.
 */
import { arrayOf, isFunction, isNumber, isRecord, isString, prop } from '../guards.ts';
import type { MemFS } from './memfs.ts';

/** Receives the bytes gnux writes to standard output (1) or standard error (2). */
export type OutputHandler = (fd: 1 | 2, chunk: Uint8Array) => void;

/** The parts of Go's `wasm_exec.js` loader this module uses. */
interface GoInstance {
  argv: string[];
  env: Record<string, string>;
  exit: (code: number) => void;
  importObject: WebAssembly.Imports;
  run(instance: WebAssembly.Instance): Promise<void>;
  _scheduledTimeouts?: unknown;
}

type Globals = {
  Go?: new () => GoInstance;
  fs?: unknown;
  path?: unknown;
  process?: unknown;
  /** Set by `tools/wasm/main.go` once the program is ready. Unknown until checked. */
  gnux?: unknown;
};

export interface BootOptions {
  fs: MemFS;
  env: Record<string, string>;
  /** Evaluates `wasm_exec.js` so that `globalThis.Go` is defined. Called once per page. */
  loadGoRuntime: () => Promise<void>;
  /** Instantiates `gnux.wasm` with Go's import object. */
  instantiate: (imports: WebAssembly.Imports) => Promise<WebAssembly.Instance>;
}

/** A running gnux program and the filesystem it sees. */
export interface GnuxSession {
  /** The gnux version the build reports. */
  version: string;
  /** Every tool in the build, by its gnux name (`rmx`, `lsx`, ...). */
  tools: string[];
  fs: MemFS;
  /**
   * Runs one shell line in the session, where variables and `cd` carry over,
   * and resolves with its exit status.
   */
  run(line: string): Promise<number>;
  /** The shell's working directory. */
  cwd(): string;
  /** Sets (or, with undefined, removes) the receiver of the program's output. */
  onOutput(handler: OutputHandler | undefined): void;
  /**
   * Stops the instance as far as a single-threaded page can: pending timers
   * (a `sleepx` in flight) are cancelled and output is disconnected.
   */
  dispose(): void;
}

export async function bootGnux(options: BootOptions): Promise<GnuxSession> {
  const g = globalThis as unknown as Globals;
  const memfs = options.fs;

  // `fs` and `path` are not browser globals, and the Go loader also looks up
  // `fs` at call time (for runtime panics), so they stay installed.
  g.fs = memfs.nodeFs();
  g.path = memfs.nodePath();

  if (!g.Go) {
    // The loader installs a stub `process` if there is none; a stub without
    // `env` left behind on the page could confuse unrelated code, so remove it.
    const hadProcess = g.process !== undefined;
    await options.loadGoRuntime();
    if (!hadProcess) delete g.process;
    if (!g.Go) throw new Error('wasm_exec.js did not define globalThis.Go');
  }

  const go = new g.Go();
  go.argv = ['gnux'];
  go.env = { ...options.env };
  let exited = false;
  go.exit = () => {
    exited = true;
  };

  const instance = await options.instantiate(go.importObject);

  // Go captures `process` during startup, which runs synchronously inside
  // `go.run` until main parks itself. Swapping the global for just that long
  // keeps the page's (or Node's) own `process` intact everywhere else.
  // The globals are assigned again because another boot may have replaced them
  // while the module was downloading.
  const realProcess = g.process;
  g.fs = memfs.nodeFs();
  g.path = memfs.nodePath();
  g.process = memfs.nodeProcess();
  delete g.gnux;
  let published: unknown;
  try {
    void go.run(instance);
    published = g.gnux;
  } finally {
    if (realProcess === undefined) delete g.process;
    else g.process = realProcess;
    delete g.gnux;
  }

  if (exited) throw new Error('gnux.wasm exited while starting');
  const api = readApi(published);

  return {
    version: api.version,
    tools: api.tools,
    fs: memfs,
    async run(line) {
      const status: unknown = await api.run(line);
      // A status that is not a number means the contract changed; report failure.
      return isNumber(status) ? status : 1;
    },
    cwd() {
      const dir = api.cwd();
      return isString(dir) ? dir : '/';
    },
    onOutput(handler) {
      api.object.onOutput =
        handler &&
        ((fd: unknown, chunk: unknown) => {
          if ((fd === 1 || fd === 2) && chunk instanceof Uint8Array) handler(fd, chunk);
        });
    },
    dispose() {
      api.object.onOutput = undefined;
      const timeouts = go._scheduledTimeouts;
      if (timeouts instanceof Map) {
        for (const id of timeouts.values()) clearTimeout(id);
        timeouts.clear();
      }
    }
  };
}

/**
 * Checks the object the program published as `globalThis.gnux` against the
 * contract in `tools/wasm/main.go`. It comes from another program, so nothing
 * about it is taken on trust.
 */
function readApi(value: unknown) {
  if (!isRecord(value)) throw new Error('gnux.wasm started but did not publish its API');

  const version = prop(value, 'version');
  const run = prop(value, 'run');
  const cwd = prop(value, 'cwd');
  if (!isFunction(run) || !isFunction(cwd)) {
    throw new Error('gnux.wasm published an API without run() and cwd()');
  }

  return {
    object: value,
    version: isString(version) ? version : 'unknown',
    tools: arrayOf(prop(value, 'tools'), isString),
    run: (line: string): unknown => run.call(value, line),
    cwd: (): unknown => cwd.call(value)
  };
}
