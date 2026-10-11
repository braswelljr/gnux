import assert from 'node:assert/strict';
import { describe, test } from 'node:test';
import { parseAnsi, stripAnsi } from '../lib/playground/ansi.ts';
import { complete } from '../lib/playground/complete.ts';
import { createSeededFS, HOME } from '../lib/playground/seed.ts';

const commands = ['catx', 'cpx', 'echox', 'lsx', 'cd', 'help'];

function tab(line: string, fs = createSeededFS()) {
  return complete({ line, cursor: line.length, commands, fs, home: HOME });
}

describe('tab completion', () => {
  test('completes a unique command name', () => {
    assert.deepEqual(tab('ec'), { line: 'echox ', cursor: 6, candidates: [] });
    assert.equal(tab('lsx | ca').line, 'lsx | catx ');
    assert.equal(tab('truex && l').line, 'truex && lsx ');
  });

  test('lists ambiguous commands and extends to their common prefix', () => {
    const r = tab('c');
    assert.equal(r.line, 'c');
    assert.deepEqual(r.candidates, ['catx', 'cd', 'cpx']);
  });

  test('completes file names in argument position', () => {
    assert.equal(tab('catx REA').line, 'catx README.txt ');
    assert.equal(tab('lsx no').line, 'lsx notes/');
    assert.equal(tab('catx notes/to').line, 'catx notes/todo.txt ');
    assert.equal(tab('catx ~/notes/sh').line, 'catx ~/notes/shopping.txt ');
    assert.equal(tab(`catx ${HOME}/lo`).line, `catx ${HOME}/logs/`);
    assert.equal(tab('echox hi > READ').line, 'echox hi > README.txt ');
  });

  test('lists ambiguous files', () => {
    const r = tab('catx l');
    assert.equal(r.line, 'catx l');
    assert.deepEqual(r.candidates, ['latest.log', 'logs/']);
    assert.deepEqual(tab('catx notes/').candidates, ['ideas.md', 'shopping.txt', 'todo.txt']);
  });

  test('cd completes directories only', () => {
    assert.equal(tab('cd l').line, 'cd logs/');
  });

  test('hides dotfiles until asked and escapes special characters', () => {
    const fs = createSeededFS();
    fs.writeFileSync('.hidden', '');
    fs.writeFileSync('my file.txt', '');
    assert.equal(tab('catx m', fs).line, 'catx my\\ file.txt ');
    assert.equal(tab('catx .h', fs).line, 'catx .hidden ');
    assert.ok(!tab('catx ', fs).candidates.includes('.hidden'));
    assert.equal(tab('catx my\\ f', fs).line, 'catx my\\ file.txt ');
  });

  test('completes in the middle of a line', () => {
    const line = 'catx REA | wcx';
    const r = complete({ line, cursor: 8, commands, fs: createSeededFS(), home: HOME });
    assert.equal(r.line, 'catx README.txt  | wcx');
    assert.equal(r.cursor, 16);
  });

  test('leaves the line alone when nothing matches', () => {
    assert.deepEqual(tab('catx zzz'), { line: 'catx zzz', cursor: 8, candidates: [] });
    assert.equal(tab('catx nowhere/x').line, 'catx nowhere/x');
  });
});

describe('ANSI escapes', () => {
  test('are stripped', () => {
    assert.equal(stripAnsi('\x1b[01;34mlogs\x1b[0m\n\x1b]0;title\x07x\x1b[2K'), 'logs\nx');
  });

  test('become styled spans', () => {
    assert.deepEqual(parseAnsi('a\x1b[01;34mlogs\x1b[0m b\x1b[91mc\x1b[39md'), [
      { text: 'a' },
      { text: 'logs', bold: true, fg: 4 },
      { text: ' b' },
      { text: 'c', fg: 9 },
      { text: 'd' }
    ]);
    assert.deepEqual(parseAnsi('\x1b[38;5;196mx\x1b[m'), [{ text: 'x' }]);
  });
});
