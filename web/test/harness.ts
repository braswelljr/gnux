/**
 * Boots the real gnux.wasm in Node against the in-memory filesystem, through
 * the same `bootGnux` the playground page uses.
 */
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import vm from 'node:vm';
import type { MemFS } from '../lib/playground/memfs.ts';
import { bootGnux } from '../lib/playground/runtime.ts';
import { createSeededFS, ENV } from '../lib/playground/seed.ts';

const publicDir = fileURLToPath(new URL('../public/', import.meta.url));

let wasmModule: WebAssembly.Module | undefined;

export interface Result {
  status: number;
  stdout: string;
  stderr: string;
}

export interface Harness {
  fs: MemFS;
  tools: string[];
  version: string;
  cwd(): string;
  has(tool: string): boolean;
  run(line: string): Promise<Result>;
  dispose(): void;
}

export async function boot(): Promise<Harness> {
  const fs = createSeededFS();
  const session = await bootGnux({
    fs,
    env: ENV,
    loadGoRuntime: async () => {
      // wasm_exec.js is a classic script that defines globalThis.Go.
      vm.runInThisContext(readFileSync(`${publicDir}wasm_exec.js`, 'utf8'), {
        filename: 'wasm_exec.js'
      });
    },
    instantiate: async (imports) => {
      wasmModule ??= await WebAssembly.compile(readFileSync(`${publicDir}gnux.wasm`));
      return WebAssembly.instantiate(wasmModule, imports);
    }
  });

  const decoder = [new TextDecoder(), new TextDecoder()];
  let out = ['', ''];
  session.onOutput((fd, chunk) => {
    out[fd - 1] += decoder[fd - 1].decode(chunk, { stream: true });
  });

  return {
    fs,
    tools: [...session.tools],
    version: session.version,
    cwd: () => session.cwd(),
    has: (tool) => session.tools.includes(tool),
    async run(line) {
      out = ['', ''];
      const status = await session.run(line);
      return { status, stdout: out[0], stderr: out[1] };
    },
    dispose: () => session.dispose()
  };
}
