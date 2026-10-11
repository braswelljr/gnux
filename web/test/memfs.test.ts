/**
 * Unit tests for the parts of the in-memory filesystem that the gnux tools do
 * not reach on their own: descriptor offsets, exclusive creation, the callback
 * surface and the error codes Go maps to errno values.
 */
import assert from 'node:assert/strict';
import { describe, test } from 'node:test';
import { constants, FsError, MemFS, S_IFCHR, S_IFMT } from '../lib/playground/memfs.ts';

const { O_RDONLY, O_WRONLY, O_RDWR, O_CREAT, O_EXCL, O_TRUNC, O_APPEND, O_DIRECTORY } = constants;

const bytes = (s: string) => new TextEncoder().encode(s);

function throwsCode(fn: () => unknown, code: string) {
  assert.throws(
    fn,
    (err: unknown) => err instanceof FsError && err.code === code,
    `expected ${code}`
  );
}

/** A filesystem whose root the test user owns. */
function fresh(): MemFS {
  const fs = new MemFS();
  fs.asRoot(() => {
    fs.mkdirSync('/work', 0o755);
    fs.chownSync('/work', fs.uid, fs.gid);
  });
  fs.chdir('/work');
  return fs;
}

describe('MemFS', () => {
  test('sequential reads and writes advance the descriptor offset', () => {
    const fs = fresh();
    const fd = fs.openSync('f', O_RDWR | O_CREAT, 0o644);
    assert.equal(fs.writeSync(fd, bytes('hello '), 0, 6, null), 6);
    assert.equal(fs.writeSync(fd, bytes('world'), 0, 5, null), 5);
    fs.closeSync(fd);

    const rd = fs.openSync('f', O_RDONLY);
    const buf = new Uint8Array(4);
    assert.equal(fs.readSync(rd, buf, 0, 4, null), 4);
    assert.equal(new TextDecoder().decode(buf), 'hell');
    assert.equal(fs.readSync(rd, buf, 0, 4, null), 4);
    assert.equal(new TextDecoder().decode(buf), 'o wo');
    assert.equal(fs.readSync(rd, buf, 0, 4, null), 3);
    assert.equal(fs.readSync(rd, buf, 0, 4, null), 0, 'end of file reads as 0');
    fs.closeSync(rd);
  });

  test('positional reads and writes leave the offset alone', () => {
    const fs = fresh();
    fs.writeFileSync('f', '0123456789');
    const fd = fs.openSync('f', O_RDWR);
    const buf = new Uint8Array(3);
    assert.equal(fs.readSync(fd, buf, 0, 3, 7), 3);
    assert.equal(new TextDecoder().decode(buf), '789');
    assert.equal(fs.readSync(fd, buf, 0, 3, 100), 0);
    assert.equal(fs.writeSync(fd, bytes('AB'), 0, 2, 4), 2);
    assert.equal(fs.readSync(fd, buf, 0, 3, null), 3);
    assert.equal(new TextDecoder().decode(buf), '012', 'the offset did not move');
    fs.closeSync(fd);
    assert.equal(fs.readTextSync('f'), '0123AB6789');
  });

  test('a write past the end fills the gap with zeros', () => {
    const fs = fresh();
    const fd = fs.openSync('f', O_WRONLY | O_CREAT);
    fs.writeSync(fd, bytes('x'), 0, 1, 3);
    fs.closeSync(fd);
    assert.deepEqual([...fs.readFileSync('f')], [0, 0, 0, 120]);
  });

  test('O_APPEND always writes at the end', () => {
    const fs = fresh();
    fs.writeFileSync('f', 'abc');
    const fd = fs.openSync('f', O_WRONLY | O_APPEND);
    fs.writeSync(fd, bytes('1'), 0, 1, null);
    fs.writeSync(fd, bytes('2'), 0, 1, 0);
    fs.closeSync(fd);
    assert.equal(fs.readTextSync('f'), 'abc12');
  });

  test('O_TRUNC empties and O_EXCL refuses an existing name', () => {
    const fs = fresh();
    fs.writeFileSync('f', 'abc');
    fs.closeSync(fs.openSync('f', O_WRONLY | O_TRUNC));
    assert.equal(fs.statSync('f').size, 0);
    throwsCode(() => fs.openSync('f', O_WRONLY | O_CREAT | O_EXCL), 'EEXIST');
    fs.symlinkSync('missing', 'dangling');
    throwsCode(() => fs.openSync('dangling', O_WRONLY | O_CREAT | O_EXCL), 'EEXIST');
    // Without O_EXCL, creating through a dangling link creates its target.
    fs.closeSync(fs.openSync('dangling', O_WRONLY | O_CREAT));
    assert.ok(fs.existsSync('missing'));
  });

  test('truncate shrinks, grows with zeros, and old bytes do not return', () => {
    const fs = fresh();
    fs.writeFileSync('f', 'abcdef');
    fs.truncateSync('f', 2);
    assert.equal(fs.readTextSync('f'), 'ab');
    fs.truncateSync('f', 4);
    assert.deepEqual([...fs.readFileSync('f')], [97, 98, 0, 0]);
    const fd = fs.openSync('f', O_RDWR);
    fs.ftruncateSync(fd, 1);
    fs.closeSync(fd);
    assert.equal(fs.readTextSync('f'), 'a');
  });

  test('open enforces the access mode', () => {
    const fs = fresh();
    fs.writeFileSync('f', 'abc');
    const rd = fs.openSync('f', O_RDONLY);
    throwsCode(() => fs.writeSync(rd, bytes('x'), 0, 1, null), 'EBADF');
    const wr = fs.openSync('f', O_WRONLY);
    throwsCode(() => fs.readSync(wr, new Uint8Array(1), 0, 1, null), 'EBADF');
    fs.closeSync(rd);
    fs.closeSync(wr);
    throwsCode(() => fs.closeSync(rd), 'EBADF');
    throwsCode(() => fs.fstatSync(999), 'EBADF');
  });

  test('directories open read-only and only as directories', () => {
    const fs = fresh();
    fs.mkdirSync('d');
    fs.writeFileSync('f', '');
    const fd = fs.openSync('d', O_RDONLY | O_DIRECTORY);
    assert.ok(fs.fstatSync(fd).isDirectory());
    throwsCode(() => fs.readSync(fd, new Uint8Array(1), 0, 1, null), 'EISDIR');
    fs.closeSync(fd);
    throwsCode(() => fs.openSync('d', O_WRONLY), 'EISDIR');
    throwsCode(() => fs.openSync('f', O_RDONLY | O_DIRECTORY), 'ENOTDIR');
    throwsCode(() => fs.openSync('f/x', O_RDONLY), 'ENOTDIR');
    throwsCode(() => fs.openSync('f/', O_RDONLY), 'ENOTDIR');
    throwsCode(() => fs.openSync('nope/x', O_WRONLY | O_CREAT), 'ENOENT');
    throwsCode(() => fs.openSync('', O_RDONLY), 'ENOENT');
  });

  test('standard streams: 0 is at end of file, 1 and 2 go to the console', () => {
    const lines: [number, string][] = [];
    const fs = new MemFS({ onConsole: (fd, line) => lines.push([fd, line]) });
    assert.equal(fs.readSync(0, new Uint8Array(8), 0, 8, null), 0);
    assert.equal(fs.writeSync(1, bytes('partial')), 7);
    assert.deepEqual(lines, []);
    assert.equal(fs.writeSync(1, bytes(' line\nnext')), 10);
    assert.equal(fs.writeSync(2, bytes('oops\n')), 5);
    assert.deepEqual(lines, [
      [1, 'partial line'],
      [2, 'oops']
    ]);
    assert.equal(fs.fstatSync(1).mode & S_IFMT, S_IFCHR);
    fs.closeSync(1);
  });

  test('an open file outlives its name', () => {
    const fs = fresh();
    fs.writeFileSync('f', 'still here');
    const fd = fs.openSync('f', O_RDONLY);
    fs.unlinkSync('f');
    assert.equal(fs.existsSync('f'), false);
    assert.equal(fs.fstatSync(fd).nlink, 0);
    const buf = new Uint8Array(10);
    assert.equal(fs.readSync(fd, buf, 0, 10, 0), 10);
    assert.equal(new TextDecoder().decode(buf), 'still here');
    fs.closeSync(fd);
  });

  test('hard links share an inode and count their names', () => {
    const fs = fresh();
    fs.writeFileSync('a', 'x');
    fs.linkSync('a', 'b');
    assert.equal(fs.statSync('a').nlink, 2);
    assert.equal(fs.statSync('a').ino, fs.statSync('b').ino);
    fs.writeFileSync('b', 'changed');
    assert.equal(fs.readTextSync('a'), 'changed');
    fs.unlinkSync('a');
    assert.equal(fs.statSync('b').nlink, 1);
    fs.mkdirSync('d');
    throwsCode(() => fs.linkSync('d', 'd2'), 'EPERM');
    throwsCode(() => fs.linkSync('b', 'b'), 'EEXIST');
    throwsCode(() => fs.linkSync('nope', 'c'), 'ENOENT');
  });

  test('symbolic links resolve in stat and open but not lstat or readlink', () => {
    const fs = fresh();
    fs.mkdirSync('d');
    fs.writeFileSync('d/f', 'data');
    fs.symlinkSync('d', 'ld');
    fs.symlinkSync('/work/d/f', 'abs');
    fs.symlinkSync('../d/f', 'd/rel');
    assert.ok(fs.statSync('ld').isDirectory());
    assert.ok(fs.lstatSync('ld').isSymbolicLink());
    assert.equal(fs.lstatSync('ld').size, 1);
    assert.equal(fs.readlinkSync('ld'), 'd');
    assert.equal(fs.readTextSync('ld/f'), 'data');
    assert.equal(fs.readTextSync('abs'), 'data');
    assert.equal(fs.readTextSync('ld/rel'), 'data');
    assert.ok(fs.lstatSync('ld/').isDirectory(), 'a trailing slash follows the link');
    throwsCode(() => fs.readlinkSync('d'), 'EINVAL');
    throwsCode(() => fs.symlinkSync('x', 'ld'), 'EEXIST');
    fs.chdir('ld');
    assert.equal(fs.cwd(), '/work/d', 'the working directory is the physical path');
    fs.chdir('/work');

    fs.symlinkSync('b', 'a');
    fs.symlinkSync('a', 'b');
    throwsCode(() => fs.statSync('a'), 'ELOOP');
    assert.ok(fs.lstatSync('a').isSymbolicLink());
    fs.symlinkSync('nowhere', 'dangling');
    throwsCode(() => fs.statSync('dangling'), 'ENOENT');
  });

  test('directory link counts follow their subdirectories', () => {
    const fs = fresh();
    fs.mkdirSync('d');
    assert.equal(fs.statSync('d').nlink, 2);
    fs.mkdirSync('d/a');
    fs.mkdirSync('d/b');
    fs.writeFileSync('d/f', '');
    assert.equal(fs.statSync('d').nlink, 4);
    fs.rmdirSync('d/a');
    assert.equal(fs.statSync('d').nlink, 3);
    fs.renameSync('d/b', 'b');
    assert.equal(fs.statSync('d').nlink, 2);
    assert.deepEqual(fs.readdirSync('d'), ['f']);
  });

  test('rmdir and unlink report the right errors', () => {
    const fs = fresh();
    fs.mkdirSync('d');
    fs.writeFileSync('d/f', '');
    throwsCode(() => fs.rmdirSync('d'), 'ENOTEMPTY');
    throwsCode(() => fs.rmdirSync('d/f'), 'ENOTDIR');
    throwsCode(() => fs.rmdirSync('nope'), 'ENOENT');
    throwsCode(() => fs.rmdirSync('.'), 'EINVAL');
    throwsCode(() => fs.rmdirSync('/'), 'EBUSY');
    throwsCode(() => fs.unlinkSync('d'), 'EISDIR');
    throwsCode(() => fs.unlinkSync('nope'), 'ENOENT');
    throwsCode(() => fs.mkdirSync('d'), 'EEXIST');
    throwsCode(() => fs.mkdirSync('nope/x'), 'ENOENT');
    fs.mkdirSync('trailing/');
    assert.ok(fs.statSync('trailing').isDirectory());
  });

  test('rename replaces files, refuses bad directory moves and keeps inodes', () => {
    const fs = fresh();
    fs.writeFileSync('a', 'A');
    fs.writeFileSync('b', 'B');
    const ino = fs.statSync('a').ino;
    fs.renameSync('a', 'b');
    assert.equal(fs.readTextSync('b'), 'A');
    assert.equal(fs.statSync('b').ino, ino);
    assert.equal(fs.existsSync('a'), false);
    fs.renameSync('b', 'b');

    fs.mkdirSync('d');
    fs.mkdirSync('d/sub');
    fs.mkdirSync('full');
    fs.writeFileSync('full/x', '');
    fs.mkdirSync('empty');
    throwsCode(() => fs.renameSync('d', 'd/sub/inner'), 'EINVAL');
    throwsCode(() => fs.renameSync('d', 'full'), 'ENOTEMPTY');
    throwsCode(() => fs.renameSync('d', 'b'), 'ENOTDIR');
    throwsCode(() => fs.renameSync('b', 'd'), 'EISDIR');
    throwsCode(() => fs.renameSync('nope', 'x'), 'ENOENT');
    fs.renameSync('d', 'empty');
    assert.deepEqual(fs.readdirSync('empty'), ['sub']);

    // A moved directory keeps working as the working directory.
    fs.chdir('empty/sub');
    fs.renameSync('/work/empty', '/work/moved');
    assert.equal(fs.cwd(), '/work/moved/sub');
    fs.chdir('..');
    assert.equal(fs.cwd(), '/work/moved');
  });

  test('a removed working directory has no path', () => {
    const fs = fresh();
    fs.mkdirSync('gone');
    fs.chdir('gone');
    fs.rmdirSync('/work/gone');
    throwsCode(() => fs.cwd(), 'ENOENT');
    throwsCode(() => fs.statSync('x'), 'ENOENT');
    assert.ok(fs.statSync('/work').isDirectory());
    throwsCode(() => fs.chdir('/work/gone'), 'ENOENT');
    throwsCode(() => fs.chdir('/nope'), 'ENOENT');
  });

  test('permissions are checked for the one user', () => {
    const fs = fresh();
    throwsCode(() => fs.writeFileSync('/system', ''), 'EACCES');
    throwsCode(() => fs.mkdirSync('/system'), 'EACCES');
    fs.writeFileSync('ro', 'x', 0o444);
    throwsCode(() => fs.openSync('ro', O_WRONLY), 'EACCES');
    assert.equal(fs.readTextSync('ro'), 'x');
    fs.chmodSync('ro', 0o000);
    throwsCode(() => fs.openSync('ro', O_RDONLY), 'EACCES');
    assert.equal(fs.statSync('ro').mode & 0o7777, 0);
    fs.mkdirSync('locked');
    fs.writeFileSync('locked/f', '');
    fs.chmodSync('locked', 0o555);
    throwsCode(() => fs.unlinkSync('locked/f'), 'EACCES');
    throwsCode(() => fs.writeFileSync('locked/g', ''), 'EACCES');
    fs.chmodSync('locked', 0o000);
    throwsCode(() => fs.statSync('locked/f'), 'EACCES');
    throwsCode(() => fs.readdirSync('locked'), 'EACCES');
    throwsCode(() => fs.chmodSync('/', 0o777), 'EPERM');
    throwsCode(() => fs.chownSync('ro', 0, 0), 'EPERM');
    fs.chownSync('ro', 0xffffffff, 0xffffffff);
  });

  test('the umask applies to new files and directories', () => {
    const fs = fresh();
    assert.equal(fs.umask(0o077), 0o022);
    fs.writeFileSync('f', '');
    fs.mkdirSync('d');
    assert.equal(fs.statSync('f').mode & 0o777, 0o600);
    assert.equal(fs.statSync('d').mode & 0o777, 0o700);
    assert.equal(fs.umask(), 0o077);
  });

  test('utimes takes seconds and stat reports milliseconds', () => {
    const fs = fresh();
    fs.writeFileSync('f', '');
    fs.utimesSync('f', 1_000_000_000, 1_500_000_000.5);
    const st = fs.statSync('f');
    assert.equal(st.atimeMs, 1_000_000_000_000);
    assert.equal(st.mtimeMs, 1_500_000_000_500);
    assert.equal(st.blksize, 4096);
    assert.equal(st.dev, 1);
  });

  test('running out of space is ENOSPC and frees on unlink', () => {
    const fs = new MemFS({ maxBytes: 1024, uid: 0 });
    fs.asRoot(() => fs.writeFileSync('/a', new Uint8Array(1000)));
    throwsCode(() => fs.asRoot(() => fs.writeFileSync('/b', new Uint8Array(100))), 'ENOSPC');
    fs.asRoot(() => fs.unlinkSync('/a'));
    fs.asRoot(() => fs.writeFileSync('/b', new Uint8Array(1024)));
  });

  test('resolvePath normalizes against the working directory', () => {
    const fs = fresh();
    assert.equal(fs.resolvePath('a/../b/./c'), '/work/b/c');
    assert.equal(fs.resolvePath('/x//y/'), '/x/y');
    assert.equal(fs.resolvePath('..', '..', '..'), '/');
  });

  test('the callback surface passes results and coded errors', () => {
    const fs = fresh();
    const node = fs.nodeFs();
    const calls: unknown[][] = [];
    node.mkdir('d', 0o755, (...args) => calls.push(args));
    node.stat('d', (err, st) => calls.push([err, st?.isDirectory()]));
    node.stat('nope', (err) => calls.push([(err as FsError).code]));
    node.open('f', O_WRONLY | O_CREAT, 0o644, (_err, fd) => {
      node.write(fd as number, bytes('hi'), 0, 2, null, (err2, n) => calls.push([err2, n]));
      node.close(fd as number, (err3) => calls.push([err3]));
    });
    node.readdir('.', (err, names) => calls.push([err, names]));
    assert.deepEqual(calls, [
      [null, undefined],
      [null, true],
      ['ENOENT'],
      [null, 2],
      [null],
      [null, ['d', 'f']]
    ]);

    // A callback that throws is not called a second time with its own error.
    let count = 0;
    assert.throws(() =>
      node.stat('d', () => {
        count++;
        throw new Error('from the callback');
      })
    );
    assert.equal(count, 1);

    const proc = fs.nodeProcess();
    assert.equal(proc.cwd(), '/work');
    proc.chdir('d');
    assert.equal(proc.cwd(), '/work/d');
    assert.throws(
      () => proc.chdir('/nope'),
      (err: unknown) => (err as FsError).code === 'ENOENT'
    );
    assert.deepEqual([proc.getuid(), proc.getgid(), proc.getgroups()], [1000, 1000, [1000]]);
    assert.equal(fs.nodePath().resolve('x'), '/work/d/x');
  });
});
