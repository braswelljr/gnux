export type PlaygroundExample = {
  label: string;
  command: string;
  /** The tools the command uses; it is only offered when all of them exist. */
  needs: string[];
};

/**
 * Commands offered in the playground's side panel. They are filtered against
 * the tool list the WebAssembly build reports, so an example never names a tool
 * that build does not have.
 */
export const examples: PlaygroundExample[] = [
  { label: 'List everything', command: 'lsx -la', needs: ['lsx'] },
  { label: 'Read the welcome note', command: 'catx README.txt', needs: ['catx'] },
  { label: 'Where am I?', command: 'pwdx', needs: ['pwdx'] },
  { label: 'First lines of a log', command: 'headx -n 5 logs/app.log', needs: ['headx'] },
  { label: 'Last lines of a log', command: 'tailx -n 3 logs/app.log', needs: ['tailx'] },
  { label: 'Count lines', command: 'wcx -l logs/app.log notes/*.txt', needs: ['wcx'] },
  {
    label: 'Pipe tools together',
    command: 'catx logs/app.log | headx -n 12 | tailx -n 2',
    needs: ['catx', 'headx', 'tailx']
  },
  { label: 'Stop an endless stream', command: 'yesx gnux | headx -n 3', needs: ['yesx', 'headx'] },
  {
    label: 'Write and append',
    command: 'echox first > out.txt && echox second >> out.txt && catx out.txt',
    needs: ['echox', 'catx']
  },
  { label: 'Glob', command: 'echox notes/*.txt', needs: ['echox'] },
  {
    label: 'Make, fill and remove a tree',
    command: 'mkdirx -p build/out && touchx build/out/a.txt build/b.txt && rmx -rv build',
    needs: ['mkdirx', 'touchx', 'rmx']
  },
  {
    label: 'Copy a directory',
    command: 'cpx -r notes backup && lsx -l backup',
    needs: ['cpx', 'lsx']
  },
  {
    label: 'Rename a file',
    command: 'touchx draft.md && mvx -v draft.md final.md',
    needs: ['touchx', 'mvx']
  },
  { label: 'Follow a symbolic link', command: 'realpathx latest.log', needs: ['realpathx'] },
  {
    label: 'Link a file',
    command: 'lnx -s notes/todo.txt todo && lsx -l todo',
    needs: ['lnx', 'lsx']
  },
  { label: 'Variables', command: 'NAME=world; echox "hello $NAME"', needs: ['echox'] },
  {
    label: 'A failing command',
    command: 'rmx nope || echox "exit status: $?"',
    needs: ['rmx', 'echox']
  },
  {
    label: 'Path parts',
    command: 'basenamex src/hello.go .go; dirnamex src/hello.go',
    needs: ['basenamex', 'dirnamex']
  },
  { label: 'The environment', command: 'envx', needs: ['envx'] },
  {
    label: 'Loop over files',
    command: 'for f in notes/*.txt; do echox "$(wcx -l < $f) $f"; done',
    needs: ['echox', 'wcx']
  },
  {
    label: 'Branch on a test',
    command: 'if [ -d logs ]; then echox "logs is a directory"; else echox "no logs"; fi',
    needs: ['echox', 'testx']
  },
  { label: 'Shell syntax', command: 'help', needs: [] }
];

/** Shell builtins, plus the playground's own `clear`, for tab completion. `/` commands complete on their own. */
export const builtins = ['cd', 'exit', 'help', 'set', 'shift', 'break', 'continue', 'clear'];
