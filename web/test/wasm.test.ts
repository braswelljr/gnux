/**
 * Drives the real gnux.wasm against the in-memory filesystem and checks both
 * what the tools print and what they leave behind. This is the proof that
 * lib/playground/memfs.ts honors the contract Go's js/wasm runtime expects.
 */
import assert from 'node:assert/strict';
import { after, before, describe, test } from 'node:test';
import { S_IFDIR, S_IFLNK, S_IFMT, S_IFREG } from '../lib/playground/memfs.ts';
import { HOME } from '../lib/playground/seed.ts';
import { boot, type Harness } from './harness.ts';

describe('gnux.wasm on the in-memory filesystem', () => {
  let h: Harness;

  before(async () => {
    h = await boot();
  });
  after(() => h.dispose());

  /** Runs a line and fails with its stderr if it does not exit 0. */
  async function ok(line: string): Promise<string> {
    const r = await h.run(line);
    assert.equal(r.status, 0, `\`${line}\` exited ${r.status}: ${r.stderr}`);
    assert.equal(r.stderr, '', `\`${line}\` wrote to stderr`);
    return r.stdout;
  }

  test('publishes its API', () => {
    assert.equal(typeof h.version, 'string');
    assert.notEqual(h.version, '');
    for (const tool of ['pwdx', 'mkdirx', 'touchx', 'rmx', 'rmdirx', 'echox', 'realpathx']) {
      assert.ok(h.has(tool), `missing ${tool}`);
    }
    assert.equal(h.cwd(), HOME);
  });

  test('pwdx prints the working directory', async () => {
    assert.equal(await ok('pwdx'), `${HOME}\n`);
  });

  test('tools answer to their GNU names too', async () => {
    assert.equal(await ok('pwd'), `${HOME}\n`);
  });

  test('mkdirx -p creates every level', async () => {
    await ok('mkdirx -p a/b/c');
    for (const dir of ['a', 'a/b', 'a/b/c']) {
      const st = h.fs.statSync(`${HOME}/${dir}`);
      assert.equal(st.mode & S_IFMT, S_IFDIR, dir);
      assert.equal(st.mode & 0o777, 0o755, `${dir} mode honors the umask`);
    }
    assert.equal(h.fs.statSync(`${HOME}/a`).nlink, 3, 'a directory counts its subdirectories');
    // Without -p an existing directory is an error.
    const again = await h.run('mkdirx a');
    assert.equal(again.status, 1);
    assert.match(again.stderr, /^mkdirx: cannot create directory 'a': File exists/);
  });

  test('touchx creates a file and rmx removes it', async () => {
    await ok('touchx f');
    const st = h.fs.statSync(`${HOME}/f`);
    assert.equal(st.mode & S_IFMT, S_IFREG);
    assert.equal(st.size, 0);
    assert.equal(st.mode & 0o777, 0o644);
    await ok('rmx f');
    assert.equal(h.fs.existsSync(`${HOME}/f`), false);
  });

  test('touchx -d sets the modification time', async () => {
    await ok("touchx -d '2001-02-03 04:05:06Z' dated");
    const st = h.fs.statSync(`${HOME}/dated`);
    assert.equal(st.mtimeMs, Date.UTC(2001, 1, 3, 4, 5, 6));
    await ok('rmx dated');
  });

  test('> writes a file and >> appends to it', async () => {
    await ok('echox hi > out.txt');
    assert.equal(h.fs.readTextSync(`${HOME}/out.txt`), 'hi\n');
    await ok('echox there >> out.txt');
    assert.equal(h.fs.readTextSync(`${HOME}/out.txt`), 'hi\nthere\n');
    // A second > truncates.
    await ok('echox x > out.txt');
    assert.equal(h.fs.readTextSync(`${HOME}/out.txt`), 'x\n');
  });

  test('cd carries over between lines', async () => {
    await ok('cd a/b');
    assert.equal(await ok('pwdx'), `${HOME}/a/b\n`);
    assert.equal(h.cwd(), `${HOME}/a/b`);
    assert.equal(h.fs.cwd(), `${HOME}/a/b`);
    await ok('echox rel > here.txt');
    assert.equal(
      h.fs.readTextSync(`${HOME}/a/b/here.txt`),
      'rel\n',
      'relative paths resolve against cwd'
    );
    await ok('cd ..');
    assert.equal(await ok('pwdx'), `${HOME}/a\n`);
    await ok('cd');
    assert.equal(await ok('pwdx'), `${HOME}\n`);
    const bad = await h.run('cd nowhere');
    assert.equal(bad.status, 1);
    assert.match(bad.stderr, /^gnux: cd: nowhere: No such file or directory/);
  });

  test('variables carry over and expand', async () => {
    await ok('NAME=world');
    assert.equal(await ok('echox "hello $NAME"'), 'hello world\n');
    assert.equal(await ok('echox ~'), `${HOME}\n`);
  });

  test('globbing expands against the virtual filesystem', async () => {
    assert.equal(await ok('echox *.txt'), 'README.txt out.txt\n');
    assert.equal(await ok('echox notes/*.txt'), 'notes/shopping.txt notes/todo.txt\n');
    assert.equal(await ok('echox *.nomatch'), '*.nomatch\n');
  });

  test('pipelines connect tools', async (t) => {
    // The last command's status and output are the pipeline's. (Whatever the
    // first echox says about its unread output goes to stderr.)
    const pipe = await h.run('echox a | echox b');
    assert.equal(pipe.status, 0);
    assert.equal(pipe.stdout, 'b\n');
    if (!h.has('headx')) return t.diagnostic('headx unavailable: skipped yesx | headx');
    assert.equal(await ok('yesx | headx -n 3'), 'y\ny\ny\n');
    if (h.has('catx') && h.has('wcx')) {
      assert.equal((await ok('catx logs/app.log | wcx -l')).trim(), '40');
    }
  });

  test('&&, || and ; sequence commands by status', async () => {
    assert.equal(await ok('truex && echox yes'), 'yes\n');
    assert.equal(await ok('falsex || echox no'), 'no\n');
    const r = await h.run('falsex && echox skipped');
    assert.deepEqual(r, { status: 1, stdout: '', stderr: '' });
    assert.equal(await ok('falsex; echox $?'), '1\n');
  });

  test('a failing command reports a status and a diagnostic', async () => {
    const r = await h.run('rmx nope');
    assert.equal(r.status, 1);
    assert.equal(r.stdout, '');
    assert.ok(r.stderr.startsWith('rmx: cannot remove'), r.stderr);
    assert.match(r.stderr, /No such file or directory/);
    assert.deepEqual(await h.run('rmx -f nope'), { status: 0, stdout: '', stderr: '' });
    const unknown = await h.run('frobnicate');
    assert.equal(unknown.status, 127);
    assert.match(unknown.stderr, /command not found/);
  });

  test('rmx refuses a directory without -r', async () => {
    const r = await h.run('rmx a');
    assert.equal(r.status, 1);
    assert.match(r.stderr, /^rmx: cannot remove 'a': Is a directory/);
    assert.ok(h.fs.existsSync(`${HOME}/a`));
  });

  test('rmdirx refuses a non-empty directory', async () => {
    const r = await h.run('rmdirx a');
    assert.equal(r.status, 1);
    assert.match(r.stderr, /^rmdirx: failed to remove 'a': Directory not empty/);
    assert.ok(h.fs.existsSync(`${HOME}/a/b/c`));
    await ok('rmdirx a/b/c');
    assert.equal(h.fs.existsSync(`${HOME}/a/b/c`), false);
    assert.equal(h.fs.statSync(`${HOME}/a/b`).nlink, 2);
  });

  test('rmx -r removes a whole tree', async () => {
    await ok('mkdirx -p tree/x/y tree/z');
    await ok('touchx tree/1 tree/x/2 tree/x/y/3 tree/z/4');
    await ok('rmx -r tree a');
    assert.equal(h.fs.existsSync(`${HOME}/tree`), false);
    assert.equal(h.fs.existsSync(`${HOME}/a`), false);
  });

  test('rmx -rv reports each removal, children first', async () => {
    await ok('mkdirx -p v/w');
    await ok('touchx v/w/f');
    const out = await ok('rmx -rv v');
    assert.deepEqual(out.trimEnd().split('\n'), [
      "removed 'v/w/f'",
      "removed directory 'v/w'",
      "removed directory 'v'"
    ]);
  });

  test('realpathx resolves symbolic links', async () => {
    // The seed links latest.log -> logs/app.log.
    assert.equal(h.fs.lstatSync(`${HOME}/latest.log`).mode & S_IFMT, S_IFLNK);
    assert.equal(h.fs.statSync(`${HOME}/latest.log`).mode & S_IFMT, S_IFREG);
    assert.equal(await ok('realpathx latest.log'), `${HOME}/logs/app.log\n`);

    // A link to a directory, an absolute link, and a chain of the two.
    h.fs.symlinkSync('notes', `${HOME}/n`);
    h.fs.symlinkSync(`${HOME}/n/todo.txt`, `${HOME}/todo`);
    assert.equal(await ok('realpathx n/todo.txt'), `${HOME}/notes/todo.txt\n`);
    assert.equal(await ok('realpathx todo'), `${HOME}/notes/todo.txt\n`);
    assert.equal(await ok('cd n && pwdx -P'), `${HOME}/notes\n`);
    await ok('cd');

    // Removing a link removes the link, not what it points to.
    await ok('rmx n todo');
    assert.equal(h.fs.existsSync(`${HOME}/n`), false);
    assert.ok(h.fs.existsSync(`${HOME}/notes/todo.txt`));
  });

  test('a symbolic link loop is an error, not a hang', async () => {
    h.fs.symlinkSync('loop2', `${HOME}/loop1`);
    h.fs.symlinkSync('loop1', `${HOME}/loop2`);
    const r = await h.run('realpathx loop1');
    assert.equal(r.status, 1);
    assert.match(r.stderr, /^realpathx: /);
    await ok('rmx loop1 loop2');
  });

  test('permission errors surface as diagnostics', async () => {
    const r = await h.run('touchx /root-owned');
    assert.equal(r.status, 1);
    assert.match(r.stderr, /^touchx: cannot touch '\/root-owned': Permission denied/);
    await ok('touchx /tmp/scratch');
    await ok('rmx /tmp/scratch');
  });

  test('a redirect to a bad path fails the command', async () => {
    const r = await h.run('echox x > missing/dir/file');
    assert.equal(r.status, 1);
    assert.match(r.stderr, /^gnux: missing\/dir\/file: No such file or directory/);
  });

  test('basenamex and dirnamex', async () => {
    assert.equal(await ok('basenamex /usr/lib/libc.so .so'), 'libc\n');
    assert.equal(await ok('dirnamex /usr/lib/libc.so'), '/usr/lib\n');
  });

  // The tools below were added to the wasm build after the first ones; each
  // test skips itself if the build under test does not have its tool.

  test('catx reads files and standard input', async (t) => {
    if (!h.has('catx')) return t.skip('catx unavailable');
    assert.equal(
      await ok('catx notes/shopping.txt'),
      h.fs.readTextSync(`${HOME}/notes/shopping.txt`)
    );
    assert.equal(await ok('catx < notes/todo.txt'), h.fs.readTextSync(`${HOME}/notes/todo.txt`));
    assert.equal(
      await ok('catx latest.log'),
      h.fs.readTextSync(`${HOME}/logs/app.log`),
      'open follows links'
    );
    await ok('catx notes/todo.txt notes/shopping.txt > joined.txt');
    assert.equal(
      h.fs.readTextSync(`${HOME}/joined.txt`),
      h.fs.readTextSync(`${HOME}/notes/todo.txt`) + h.fs.readTextSync(`${HOME}/notes/shopping.txt`)
    );
    await ok('rmx joined.txt');
  });

  test('headx, tailx and wcx read the seeded log', async (t) => {
    if (!h.has('headx') || !h.has('tailx') || !h.has('wcx'))
      return t.skip('headx/tailx/wcx unavailable');
    const lines = h.fs.readTextSync(`${HOME}/logs/app.log`).trimEnd().split('\n');
    assert.equal(lines.length, 40);
    assert.equal(await ok('headx -n 3 logs/app.log'), `${lines.slice(0, 3).join('\n')}\n`);
    // tailx seeks from the end of the file, which exercises positional reads.
    assert.equal(await ok('tailx -n 3 logs/app.log'), `${lines.slice(-3).join('\n')}\n`);
    assert.equal(await ok('tailx -c 5 notes/shopping.txt'), 'eggs\n');
    assert.match(await ok('wcx -l logs/app.log'), /^\s*40 logs\/app\.log\n$/);
    const bytes = h.fs.statSync(`${HOME}/logs/app.log`).size;
    assert.match(await ok('wcx -c logs/app.log'), new RegExp(`^\\s*${bytes} logs/app\\.log\\n$`));
  });

  test('cpx copies files and trees', async (t) => {
    if (!h.has('cpx')) return t.skip('cpx unavailable');
    await ok('cpx README.txt copy.txt');
    assert.equal(h.fs.readTextSync(`${HOME}/copy.txt`), h.fs.readTextSync(`${HOME}/README.txt`));
    await ok('cpx -r notes backup');
    assert.deepEqual(
      h.fs.readdirSync(`${HOME}/backup`).sort(),
      h.fs.readdirSync(`${HOME}/notes`).sort()
    );
    assert.equal(
      h.fs.readTextSync(`${HOME}/backup/todo.txt`),
      h.fs.readTextSync(`${HOME}/notes/todo.txt`)
    );
    const st = h.fs.statSync(`${HOME}/backup/todo.txt`);
    assert.notEqual(st.ino, h.fs.statSync(`${HOME}/notes/todo.txt`).ino, 'a copy is a new file');
    await ok('rmx -r copy.txt backup');
  });

  test('mvx renames files and directories', async (t) => {
    if (!h.has('mvx')) return t.skip('mvx unavailable');
    await ok('mkdirx -p m/sub && echox data > m/sub/f && echox old > target');
    const ino = h.fs.statSync(`${HOME}/m/sub/f`).ino;
    await ok('mvx m/sub/f m/g');
    assert.equal(h.fs.existsSync(`${HOME}/m/sub/f`), false);
    assert.equal(h.fs.statSync(`${HOME}/m/g`).ino, ino, 'a rename keeps the inode');
    await ok('mvx m/g target');
    assert.equal(h.fs.readTextSync(`${HOME}/target`), 'data\n', 'a rename replaces its target');
    await ok('mvx m moved');
    assert.ok(h.fs.existsSync(`${HOME}/moved/sub`));
    assert.equal(h.fs.existsSync(`${HOME}/m`), false);
    await ok('cd moved/sub && pwdx && cd');
    const into = await h.run('mvx moved moved/sub/inner');
    assert.equal(into.status, 1, 'a directory cannot move into itself');
    await ok('rmx -r moved target');
  });

  test('lnx makes hard and symbolic links', async (t) => {
    if (!h.has('lnx')) return t.skip('lnx unavailable');
    await ok('echox one > orig');
    await ok('lnx orig hard');
    assert.equal(h.fs.statSync(`${HOME}/orig`).nlink, 2);
    assert.equal(h.fs.statSync(`${HOME}/hard`).ino, h.fs.statSync(`${HOME}/orig`).ino);
    await ok('echox two >> hard');
    assert.equal(h.fs.readTextSync(`${HOME}/orig`), 'one\ntwo\n', 'hard links share their data');

    await ok('lnx -s orig soft');
    assert.equal(h.fs.readlinkSync(`${HOME}/soft`), 'orig');
    assert.equal(h.fs.lstatSync(`${HOME}/soft`).mode & S_IFMT, S_IFLNK);
    assert.equal(await ok('realpathx soft'), `${HOME}/orig\n`);

    await ok('rmx orig');
    assert.equal(h.fs.statSync(`${HOME}/hard`).nlink, 1);
    assert.equal(
      h.fs.readTextSync(`${HOME}/hard`),
      'one\ntwo\n',
      'data survives while a name remains'
    );
    // Like GNU realpath, a missing last component is fine; -e insists on it.
    assert.equal(await ok('realpathx soft'), `${HOME}/orig\n`);
    const dangling = await h.run('realpathx -e soft');
    assert.equal(dangling.status, 1);
    assert.match(dangling.stderr, /^realpathx: soft: No such file or directory/);
    await ok('rmx hard soft');
  });

  test('lsx lists the virtual filesystem', async (t) => {
    if (!h.has('lsx')) return t.skip('lsx unavailable');
    assert.deepEqual((await ok('lsx notes')).trimEnd().split(/\s+/), [
      'ideas.md',
      'shopping.txt',
      'todo.txt'
    ]);
    const long = await ok('lsx -la');
    assert.match(long, /^drwxr-xr-x\s+\d+ .* \.$/m);
    assert.match(long, /^lrwxrwxrwx\s+1 .* latest\.log -> logs\/app\.log$/m);
    assert.match(long, /^-rw-r--r--\s+1 .* README\.txt$/m);
    const missing = await h.run('lsx nowhere');
    assert.equal(missing.status, 2);
    assert.match(missing.stderr, /^lsx: cannot access 'nowhere': No such file or directory/);
  });

  test('sleepx waits on the event loop', async (t) => {
    if (!h.has('sleepx')) return t.skip('sleepx unavailable');
    const start = performance.now();
    await ok('sleepx 0.05');
    assert.ok(performance.now() - start >= 40);
  });

  test('runaway output is cut off instead of hanging', async () => {
    const r = await h.run('yesx');
    assert.match(r.stderr, /output truncated/);
    assert.ok(r.stdout.length <= 1 << 20);
  });

  test('filling the filesystem fails cleanly', async () => {
    const r = await h.run('yesx > big');
    assert.equal(r.status, 1);
    assert.match(r.stderr, /No space left on device/);
    await ok('rmx big');
    await ok('echox still-works > small');
    assert.equal(h.fs.readTextSync(`${HOME}/small`), 'still-works\n');
    await ok('rmx small');
  });

  test('the home directory ends as it began, plus out.txt', async () => {
    assert.deepEqual(h.fs.readdirSync(HOME).sort(), [
      'README.txt',
      'latest.log',
      'logs',
      'notes',
      'out.txt',
      'src'
    ]);
  });

  test('sessions are independent', async () => {
    const other = await boot();
    try {
      assert.equal(other.fs.existsSync(`${HOME}/out.txt`), false);
      const r = await other.run('echox second > second.txt && pwdx');
      assert.equal(r.stdout, `${HOME}\n`);
      assert.ok(other.fs.existsSync(`${HOME}/second.txt`));
      assert.equal(h.fs.existsSync(`${HOME}/second.txt`), false);
      // The first session still works, against its own files.
      assert.equal(await ok('catx out.txt'), 'x\n');
    } finally {
      other.dispose();
    }
  });
});
