import { type TypingScene } from '@/hooks/use-typewriter';

/*
 * The sessions the landing page types out. Every output here is what the real
 * gnux printed for that command, captured from a build of this repository
 * (the user and group in `lsx -la` are generalized to "you" and "staff").
 */

/** The hero terminal, typed on the keyboard beneath it. */
export const heroSession: TypingScene[] = [
  {
    command: 'lsx -la',
    output: `total 8
drwxr-xr-x  6 you staff 192 Oct 11 19:18 .
drwxr-xr-x 26 you staff 832 Oct 11 19:18 ..
-rw-r--r--  1 you staff   7 Oct 11 19:18 README.md
drwxr-xr-x  4 you staff 128 Oct 11 19:18 build
-rw-r--r--  1 you staff  50 Oct 11 19:18 notes.txt`
  },
  { command: 'catx notes.txt | wcx -l', output: '5' },
  {
    command: 'rmx -rv build/',
    output: `removed 'build/main.o'
removed 'build/out/app'
removed directory 'build/out'
removed directory 'build/'`
  },
  {
    command: "gnux shell -c 'mkdirx -p src && touchx src/main.go && lsx src'",
    output: 'main.go'
  },
  { command: 'yesx | headx -3', output: 'y\ny\ny' }
];

/** "As commands": each tool on its own, as the GNU original would be used. */
export const commandsSession: TypingScene[] = [
  { command: 'cpx -rv src backup', output: "'src' -> 'backup'\n'src/main.go' -> 'backup/main.go'" },
  { command: 'mvx -v notes.txt todo.txt', output: "renamed 'notes.txt' -> 'todo.txt'" },
  { command: 'headx -n 2 todo.txt', output: 'buy milk\nship gnux' }
];

/** "As a Go library": the engine behind rmx, called from a program. */
export const libraryDemo = {
  file: 'main.go',
  code: `// rmx -rv build
engine := rm.New(
    rm.Options{Recursive: true, Verbose: true},
    core.Stdio(),
)
err := engine.Run([]string{"build"})`,
  run: {
    command: 'go run .',
    output: "removed 'build/main.o'\nremoved directory 'build'"
  }
};

/** "As a portable shell": the gnux shell's prompt, variables and connectors. */
export const shellSession: TypingScene[] = [
  { command: 'OUT=dist' },
  {
    command: 'rmx -rf $OUT && mkdirx -pv $OUT/bin',
    output: "mkdirx: created directory 'dist'\nmkdirx: created directory 'dist/bin'"
  },
  { command: 'echox "built $OUT"', output: 'built dist' }
];

/** "Same script, every OS": one session, typed on three platforms. */
export const everywhereSession: TypingScene[] = [
  { command: 'catx todo.txt | headx -n 2', output: 'buy milk\nship gnux' },
  {
    command: "gnux shell -c 'mkdirx -pv dist/bin'",
    output: "mkdirx: created directory 'dist'\nmkdirx: created directory 'dist/bin'"
  }
];

/** The platforms, and the prompt each one's usual shell shows. */
export const platforms = [
  { name: 'Linux', shell: 'bash', prompt: 'you@linux:~$' },
  { name: 'macOS', shell: 'zsh', prompt: 'you@mac ~ %' },
  { name: 'Windows', shell: 'PowerShell', prompt: 'PS C:\\Users\\you>' }
];

/** The 404 page's terminal: looking for the page that is not there. */
export const notFoundSession: TypingScene[] = [
  {
    command: 'lsx this-page',
    output: "lsx: cannot access 'this-page': No such file or directory"
  },
  { command: 'echox $?', output: '2' }
];
