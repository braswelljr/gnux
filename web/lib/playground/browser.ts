/**
 * The browser half of booting gnux: fetching Go's loader script and the
 * WebAssembly module from the site's own `public/` directory.
 */
import { bootGnux, type GnuxSession } from './runtime.ts';
import { createSeededFS, ENV } from './seed.ts';

const WASM_URL = '/gnux.wasm';
const LOADER_URL = '/wasm_exec.js';

let loader: Promise<void> | undefined;
let compiled: WebAssembly.Module | undefined;

/** Adds `wasm_exec.js` to the page once; it defines `globalThis.Go`. */
function loadGoRuntime(): Promise<void> {
  loader ??= new Promise<void>((resolve, reject) => {
    const script = document.createElement('script');
    script.src = LOADER_URL;
    script.async = true;
    script.onload = () => resolve();
    script.onerror = () => {
      loader = undefined; // Let a later attempt try again.
      script.remove();
      reject(new Error(`Could not load ${LOADER_URL}`));
    };
    document.head.appendChild(script);
  });
  return loader;
}

async function fetchWasm(): Promise<Response> {
  const response = await fetch(WASM_URL);
  if (!response.ok) {
    throw new Error(`Could not download ${WASM_URL} (HTTP ${response.status})`);
  }
  return response;
}

/**
 * Instantiates the module, compiling it while it downloads where the browser
 * and the server's Content-Type allow, and from a buffer otherwise. The
 * compiled module is kept, so "Reset" starts a fresh instance without a second
 * download.
 */
async function instantiate(imports: WebAssembly.Imports): Promise<WebAssembly.Instance> {
  if (compiled) return WebAssembly.instantiate(compiled, imports);

  if (typeof WebAssembly.instantiateStreaming === 'function') {
    try {
      const result = await WebAssembly.instantiateStreaming(fetchWasm(), imports);
      compiled = result.module;
      return result.instance;
    } catch (err) {
      // Typically a wrong Content-Type; the buffered path below does not care.
      console.warn('gnux: streaming instantiation failed, retrying from a buffer', err);
    }
  }

  const bytes = await (await fetchWasm()).arrayBuffer();
  const result = await WebAssembly.instantiate(bytes, imports);
  compiled = result.module;
  return result.instance;
}

/** Starts a fresh gnux on a freshly seeded in-memory filesystem. */
export function bootInBrowser(): Promise<GnuxSession> {
  if (typeof WebAssembly !== 'object') {
    return Promise.reject(new Error('This browser does not support WebAssembly.'));
  }
  return bootGnux({ fs: createSeededFS(), env: ENV, loadGoRuntime, instantiate });
}
