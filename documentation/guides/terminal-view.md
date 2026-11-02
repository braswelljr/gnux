# Terminal view

On a terminal, gnux shows what the tools print in color and in a layout made for reading: `lsx` colors every column and can draw trees, show icons and git status, `dfx` draws a usage bar per disk, `dux` shows each folder's share of the space, `grepx` heads each file's matches with its name, the checksum tools color OK and FAILED, and the file tools color their `-v` reports. Anything piped or redirected stays exactly GNU's, so scripts are unaffected. This guide shows the view, how to pick one of the built-in themes (in a dark and a light variant that follows your system), how to make your choices the default, and how to turn it off.

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

`--git` adds two letters after the date: what is staged, then what is not. `-` is
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

## The other tools

`grepx` prints each file's name once, as a heading over its lines, numbers the lines, and
leaves a blank line between files:

```text
$ grepx -r 'visual.Use(' core --include='*.go'
core/df/df.go
640:	theme := visual.Use(d.Pretty && d.Color, d.Theme, d.Getenv, d.diag.Warnf)

core/du/du.go
179:	d.theme = visual.Use(d.Pretty && d.Color, d.Theme, d.Getenv, d.diag.Warnf)
```

(Some files left out.) `-c`, `-l`, `-L`, `-q`, `-Z` and `-z` print GNU's format, and
`--color=never` keeps the headings without color.

The checksum tools (`md5sumx`, `sha1sumx` to `sha512sumx`, `b2sumx`, `cksumx`) color `OK`
green and `FAILED` red in `-c` output and end each list with a count, so one bad download
in a long list stands out:

```text
$ sha256sumx -c checksums.txt
gnux-darwin-arm64.tar.gz: OK
gnux-linux-amd64.tar.gz: OK
gnux-windows-amd64.zip: FAILED
2 OK, 1 FAILED
sha256sumx: WARNING: 1 computed checksum did NOT match
```

`headx` and `tailx` color their `==> FILE <==` headers. `rmx`, `cpx`, `mvx`, `lnx`,
`installx`, `mkdirx`, `rmdirx`, `chmodx`, `chownx` and `chgrpx` color their `-v` reports: a
removal in red, a creation in green, each quoted name, and the `->` arrows. In both, the words
are GNU's; only the color is added.

## Themes

`--theme=NAME` picks a built-in theme, and `gnux themes` shows every one of them, each on a
sample row in its own colors, with a panel naming the theme in use and the background gnux
has read:

```bash
gnux themes
```

Most themes follow an editor theme and come in a dark and a light variant, named
`FAMILY-dark` and `FAMILY-light`. Give the family's own name and gnux picks the variant that
suits your background:

| Family          | After                                    |
|-----------------|------------------------------------------|
| `vscode`        | VS Code's Dark+ and Light+               |
| `github`        | GitHub Dark and GitHub Light             |
| `one`           | One Dark and One Light                   |
| `solarized`     | Solarized Dark and Solarized Light       |
| `gruvbox`       | Gruvbox Dark and Gruvbox Light           |
| `catppuccin`    | Catppuccin Mocha and Catppuccin Latte    |
| `tokyo-night`   | Tokyo Night and Tokyo Night Day          |
| `ayu`           | Ayu Dark and Ayu Light (and `ayu-mirage`) |
| `rose-pine`     | Rosé Pine and Rosé Pine Dawn             |
| `everforest`    | Everforest Dark and Everforest Light     |
| `night-owl`     | Night Owl and Light Owl                  |
| `high-contrast` | High-contrast colors on dark and on light |

`dracula`, `nord`, `monokai`, `kanagawa`, `palenight` and `ayu-mirage` come in one, dark,
variant. `default` uses the 16 basic terminal colors, so it follows your terminal's own
scheme, and `mono` uses bold, underline and dim only: no hues, for monochrome terminals and
color blindness. The named themes use 24-bit color, which current terminals show, Windows
Terminal included.

### Dark or light

A family's variant follows the first of these that says something:

1. `GNUX_BACKGROUND=dark` or `GNUX_BACKGROUND=light`;
2. `COLORFGBG`, which many terminals (rxvt, Konsole, iTerm2) set to their colors;
3. the system's appearance: dark mode on macOS and on Windows, or a dark GTK theme
   (`GTK_THEME`) on a Linux desktop;
4. otherwise dark, which most terminals are.

So with `GNUX_THEME=github`, switching macOS or Windows to dark mode switches your listings
too. If your terminal keeps its own colors whatever the system does, say which it is once,
with `GNUX_BACKGROUND`.

## Making it your default

Four variables carry your choices to every shell and every tool that has the view:

| Variable       | Values                                  | Default  |
|----------------|-----------------------------------------|----------|
| `GNUX_PRETTY`  | `auto` (on a terminal), `always`, `never` | `auto`   |
| `GNUX_THEME`   | `NAME`, `NAME:KEY=CODE:...` or `KEY=CODE:...` | `default` |
| `GNUX_BACKGROUND` | `dark`, `light`: which variant a family takes | from the system |
| `GNUX_ICONS`   | `auto`, `always`, `never` (lsx)         | `never`  |
| `NO_COLOR`     | any value: keep the view, without color | unset    |

For example, with a Nerd Font and a system that switches between light and dark during the
day, add to `~/.bashrc` or `~/.zshrc`:

```bash
export GNUX_THEME=github GNUX_ICONS=auto
```

and in PowerShell, to `$PROFILE`:

```powershell
$env:GNUX_THEME = 'github'; $env:GNUX_ICONS = 'auto'
```

An option always wins over the variable, for one command: `lsx --theme=nord`,
`lsx --icons=never`.

### Changing single colors

After the theme's name, `GNUX_THEME` takes `KEY=CODE` pairs: a two-letter key below and an
SGR code (`1` bold, `2` dim, `4` underline, `31` to `37` and `90` to `97` the basic colors,
`38;2;R;G;B` any color). A code of `0` switches a color off.

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
modified, deleted, renamed, type change, ignored, conflict, clean), `fh` (a file heading:
grepx, headx, tailx), `ok` and `er` (OK and FAILED, a creation and a removal) and `pa` (a name
in a `-v` report).

## Turning it off

- For one command: `lsx --pretty=never`.
- Everywhere: `export GNUX_PRETTY=never`. gnux then looks exactly like GNU coreutils.
- In `grepx` only, keep the headings but drop the colors: `--color=never`.
- Keep the layout but drop the colors: `NO_COLOR=1`.

## Through a pipe

A pipe turns the view off, which keeps scripts safe. To page through it, ask for it:

```bash
lsx -l --pretty=always /etc | less -R
```

`--pretty=always` also colors through the pipe. `lsx --zero`, `dux -0`, `dfx -P` and the
`-z` of `grepx`, `headx`, `tailx` and the checksum tools always print GNU's format, whatever
you set.

## Related

- [Cookbook](cookbook.md): everyday recipes with the same tools.
- [Troubleshooting](troubleshooting.md): what an error or exit status means. If icons show as
  empty boxes, the terminal's font is not a Nerd Font: install one, or set `GNUX_ICONS=never`.
- The [lsx reference](https://gnux.vercel.app/docs/tools/lsx), with every option.
