# Terminal view

On a terminal, gnux shows its listings the way eza, dust and duf do: `lsx` colors every column and can draw trees, show icons and git status, `dfx` draws a usage bar per disk, and `dux` shows each folder's share of the space. Anything piped or redirected stays exactly GNU's, so scripts are unaffected. This guide shows the view, how to make your choices the default with the built-in themes and a few environment variables, and how to turn it off.

## What you see by default

The view is on whenever standard output is a terminal (`--pretty=auto`). In `lsx -l` that
means:

- each permission letter colored (read, write, execute, special bits), the link count, an owner
  or group that is yours, the size and the date in the theme's colors;
- a header row naming the columns (`--header`);
- sizes like `1.4K` and `23M` (`-h`), unless you chose a unit with `--si`, `--block-size`,
  `LS_BLOCK_SIZE` or `BLOCK_SIZE`;
- directories listed first (`--group-directories-first`).

Each of those follows an option you give, so `lsx -l --color=never` is the view without color,
and `lsx -l --si` keeps powers of 1000.

`dfx` adds a bar after `Use%`, green, then yellow from 75% and red from 90%. `dux` aligns the
sizes, colors them by magnitude, and gives each line a bar and the percentage of its folder:

```text
$ dux -h -d 1 core/internal
8.0K  ▍░░░░░░░░░░░░░░░   2%  core/internal/bre
 48K  ██▏░░░░░░░░░░░░░  13%  core/internal/fsstat
 48K  ██▏░░░░░░░░░░░░░  13%  core/internal/gitstat
 12K  ▌░░░░░░░░░░░░░░░   3%  core/internal/human
 40K  █▊░░░░░░░░░░░░░░  11%  core/internal/sysstat
 44K  ██░░░░░░░░░░░░░░  12%  core/internal/utmp
356K  ████████████████ 100%  core/internal
```

(Run in gnux's own source tree, some lines left out; on a terminal the sizes and names are in
color too.)

Piped or redirected output is GNU's: `lsx -l | grepx x` reads the same columns as GNU ls
prints, and so does every script, on every machine.

## Trees

`--tree` draws each directory's contents beneath it; `--level=DEPTH` stops after that many
levels (it also limits `-R`):

```text
$ lsx --tree --level=2 --group-directories-first
.
├── node_modules
│   └── react
├── public
│   └── index.html
├── src
│   ├── components
│   ├── App.tsx
│   └── api.ts
├── README.md
├── debug.log
└── package.json
```

With `-l`, the whole tree is one table, so every column lines up. Where the locale is not UTF-8
(`LANG=C`), the branches are drawn as `|--` and `` `-- ``.

## Git status

`--git` adds eza's two letters after the date: what is staged, then what is not. `-` is
unchanged, `N` new, `M` modified, `D` deleted, `T` type changed, `I` ignored and `U`
conflicted. A directory shows the most significant letter of what is inside it.
`--git-ignore` leaves out what `.gitignore` excludes:

```text
$ lsx -l --tree --level=2 --git --git-ignore -g src
Permissions Links Group Size Modified         Git Name
drwxr-xr-x      5 staff  160 2026-11-01 23:38 -M  src
drwxr-xr-x      3 staff   96 2026-11-01 23:38 --  ├── components
-rw-r--r--      1 staff   33 2026-11-01 23:38 --  │   └── Button.tsx
-rw-r--r--      1 staff   32 2026-11-01 23:38 -M  ├── App.tsx
-rw-r--r--      1 staff   10 2026-11-01 23:38 -N  └── api.ts
```

gnux reads the repository itself, so this works with no git installed, on every platform.
Renames show as a deletion and a new file, and a file whose only change is a line-ending
conversion shows as `M`.

## More of what lsx can show

| Option                     | What it shows                                                        |
|----------------------------|----------------------------------------------------------------------|
| `--icons[=WHEN]`           | A file-type icon before each name; needs a [Nerd Font](https://www.nerdfonts.com) |
| `--color-scale[=WHICH]`    | Sizes colored by magnitude and dates by age (`size`, `age`, `all`)   |
| `--total-size`             | A folder's size as the total of everything in it (and sorted on with `-S`) |
| `--only-dirs`, `--only-files` | Only folders, or everything else                                  |
| `--octal-permissions`      | The mode in octal too, such as `0644`                                |
| `--header`                 | The column names, also without the rest of the view                  |

A scenario that uses several at once: your downloads folder is full, and you want its biggest
folders first, sized by their contents:

```bash
lsx -lS --total-size --only-dirs --color-scale=size ~/Downloads
```

## Themes

`--theme=NAME` picks one of the built-in themes:

| Theme        | Looks like                                                            |
|--------------|-----------------------------------------------------------------------|
| `default`    | eza's colors in the 16 basic terminal colors, so it follows your terminal's scheme |
| `mono`       | Bold, underline and dim only: no hues, for monochrome terminals and color blindness |
| `dracula`    | The Dracula palette                                                   |
| `nord`       | The Nord palette                                                      |
| `gruvbox`    | The Gruvbox dark palette                                              |
| `solarized`  | The Solarized palette                                                 |
| `catppuccin` | The Catppuccin Mocha palette                                          |

The named themes use 24-bit color, which current terminals show, Windows Terminal included.

## Making it your default

Four variables carry your choices to every shell and every tool that has the view:

| Variable       | Values                                  | Default  |
|----------------|-----------------------------------------|----------|
| `GNUX_PRETTY`  | `auto` (on a terminal), `always`, `never` | `auto`   |
| `GNUX_THEME`   | `NAME`, `NAME:KEY=CODE:...` or `KEY=CODE:...` | `default` |
| `GNUX_ICONS`   | `auto`, `always`, `never` (lsx)         | `never`  |
| `NO_COLOR`     | any value: keep the view, without color | unset    |

For example, with a Nerd Font and a dark terminal, add to `~/.bashrc` or `~/.zshrc`:

```bash
export GNUX_THEME=dracula GNUX_ICONS=auto
```

and in PowerShell, to `$PROFILE`:

```powershell
$env:GNUX_THEME = 'dracula'; $env:GNUX_ICONS = 'auto'
```

An option always wins over the variable, for one command: `lsx --theme=nord`,
`lsx --icons=never`.

### Changing single colors

After the theme's name, `GNUX_THEME` takes `KEY=CODE` pairs, with eza's keys and SGR codes,
and `EZA_COLORS` is read too, so a theme written for eza carries over. A code of `0` switches a
color off.

```bash
# Nord, with dates dimmed and your own files in cyan
export GNUX_THEME='nord:da=2;34:uu=36'
```

The keys: `ur` `uw` `ux` `ue` (the owner's read, write, execute, and search on a directory),
`gr` `gw` `gx` and `tr` `tw` `tx` (group and others), `su` (setuid, setgid, sticky), `xx`
(punctuation), `di` `ln` `pi` `so` `bd` (the type letter), `sn` and `nb` `nk` `nm` `ng` `nt`
(sizes, and the size scale), `da` and `dh` `dd` `dw` `dm` `dy` `do` (dates, and the age scale
from an hour to older than a year), `uu` `un` (yours, others'), `lc` (links), `in` (inode),
`bl` (blocks), `oc` (octal), `hd` (headers), `tb` (tree branches), `bf` `bw` `ba` `be` (bars:
filled, warning, alert, empty) and `ga` `gm` `gd` `gv` `gt` `gi` `gc` `g-` (git: new,
modified, deleted, renamed, type change, ignored, conflict, clean).

## Turning it off

- For one command: `lsx --pretty=never`.
- Everywhere: `export GNUX_PRETTY=never`. gnux then looks exactly like GNU coreutils.
- Keep the layout but drop the colors: `NO_COLOR=1`.

## Through a pipe

A pipe turns the view off, which keeps scripts safe. To page through it, ask for it:

```bash
lsx -l --pretty=always /etc | less -R
```

`--pretty=always` also colors through the pipe. `lsx --zero`, `dux -0` and `dfx -P` always
print GNU's format, whatever you set.

## Related

- [Cookbook](cookbook.md): everyday recipes with the same tools.
- [Troubleshooting](troubleshooting.md): what an error or exit status means. If icons show as
  empty boxes, the terminal's font is not a Nerd Font: install one, or set `GNUX_ICONS=never`.
- The [lsx reference](https://gnux.vercel.app/docs/tools/lsx), with every option.
