# Terminal view

On a terminal, gnux shows what the tools print in color and in a layout made for reading: `lsx` colors every column and can draw trees, show icons and git status, `dfx`, `wcx` and `statx` draw framed tables and panels, `dux` shows each folder's share of the space, `grepx` heads each file's matches with its name, the checksum tools color OK and FAILED, and the file tools color their `-v` reports. Anything piped or redirected stays exactly GNU's, so scripts are unaffected. This guide shows the view, how to pick one of the built-in themes (in a dark and a light variant that follows your system), how to make your choices the default, and how to turn it off.

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

`dfx` draws its table in a frame, with a usage bar after `Use%`, green, then yellow from 75%
and red from 90%. A table wider than your terminal shortens its longest column with `…`, so the
frame never wraps:

```text
$ dfx -h / /System/Volumes/Data
╭────────────────┬──────┬──────┬───────┬──────┬──────────────┬─────────────────╮
│ Filesystem     │ Size │ Used │ Avail │ Use% │              │ Mounted on      │
├────────────────┼──────┼──────┼───────┼──────┼──────────────┼─────────────────┤
│ /dev/disk3s1s1 │ 927G │ 618G │  309G │  67% │ ████████░░░░ │ /               │
│ /dev/disk3s5   │ 927G │ 618G │  309G │  67% │ ████████░░░░ │ /System/Volume… │
╰────────────────┴──────┴──────┴───────┴──────┴──────────────┴─────────────────╯
```

`wcx` does the same for its counts, the total set apart at the bottom:

```text
$ wcx go.mod Makefile README.md
╭───────┬───────┬───────┬───────────╮
│ Lines │ Words │ Bytes │ File      │
├───────┼───────┼───────┼───────────┤
│    20 │    44 │   537 │ go.mod    │
│   259 │  1378 │ 11884 │ Makefile  │
│   615 │  4566 │ 31366 │ README.md │
├───────┼───────┼───────┼───────────┤
│   894 │  5988 │ 43787 │ total     │
╰───────┴───────┴───────┴───────────╯
```

`statx` gives each file a panel of its own, the mode painted as `lsx` paints it (`statx -f`
adds a bar of the space in use):

```text
$ statx notes.txt
╭─ notes.txt ─────────────────────────────────────────╮
│ Type      regular file                              │
│ Size      537 bytes, 8 blocks of 512, IO block 4096 │
│ Mode      0644 -rw-r--r--                           │
│ Owner     ada (501)                                 │
│ Group     staff (20)                                │
│ Links     1                                         │
│ Inode     73018274 on device 1,17                   │
│ Accessed  2026-10-19 22:10:56.548756669 +0000       │
│ Modified  2026-10-19 22:10:55.449463899 +0000       │
│ Changed   2026-10-19 22:10:55.449463899 +0000       │
│ Born      2026-10-19 22:10:55.449434316 +0000       │
╰─────────────────────────────────────────────────────╯
```

`statx -c FORMAT`, `--printf` and `-t`, and `wcx --total=only`, print as GNU does. `dux` aligns the
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

`findx --tree` draws what a search found the same way, under each starting point. The
directories leading to a match are dimmed, so you can tell them from the matches, and a count
follows:

```text
$ findx --tree core -name 'box*.go' -o -name 'table.go'
core
└── internal
    └── visual
        ├── box.go
        ├── box_test.go
        └── table.go

0 directories, 3 files
```

The view options go before the starting points (`findx --tree --icons . -name '*.go'`). Without
`--tree`, `findx` still prints one path per line, as GNU find does, with each name in its
type's color and the directories in front of it dimmed. `-print0`, `-printf` and `-ls` never
change.

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

`ddx status=progress` draws a bar while it copies, rewritten in place, with how much is done,
the speed and the time left. You can tell at a glance whether writing a 4 GB image to a USB
stick will take one minute or twenty:

```text
$ ddx if=debian-13.iso of=/dev/sdb bs=4M status=progress
██████████████▍░░░░░░░░░  60%  2.4 GB of 4.0 GB  31 MB/s  0:52 left
```

When the copy ends, the bar shows its last state and GNU's statistics follow. The total is
known when the input is a regular file or `count=` is given; reading from a pipe, the line
shows what was copied, the speed and the time so far. With standard error redirected to a log
file, `ddx` writes GNU's own progress line instead, once a second.

`shredx -v` draws each pass the same way, on a line of its own:

```text
$ shredx -v -u old-backup.tar
shredx: old-backup.tar: pass 1/3 (random) ████████████████████ 100%  1.2GiB/1.2GiB
shredx: old-backup.tar: pass 2/3 (ffffff) ███████▋░░░░░░░░░░░░  38%  466MiB/1.2GiB
```

`uniqx -c` draws its counts as a histogram when it writes to the terminal. Each count is
colored by its share of the largest one, with a bar of that share. Here is how a web server's
responses split between status codes:

```text
$ cutx -d' ' -f9 access.log | sortx | uniqx -c
      8 ████████████████████ 200
      1 ██▌░░░░░░░░░░░░░░░░░ 301
      2 █████░░░░░░░░░░░░░░░ 304
      3 ███████▌░░░░░░░░░░░░ 404
      1 ██▌░░░░░░░░░░░░░░░░░ 500
```

The bars need the largest count, so `uniqx -c` writes them when its input ends (or after the
first 4096 groups, for a long input). Piped on, into `sortx -rn` say, it prints GNU's
`      8 200` lines.

`odx` colors each byte by what it is: zero bytes dimmed, printable text, whitespace, other
control characters and bytes above 0x7f each in a color of their own. A config file that a
parser rejects, though it looks fine in an editor, shows its byte order mark (`ef bb bf`, above
0x7f) and its Windows line ending (`0d 0a`, whitespace) apart from the text around them:

```text
$ odx -A x -t x1z -N 12 config.yaml
000000 ef bb bf 6e 61 6d 65 3a 20 78 0d 0a              >...name: x..<
00000c
```

`uptimex` draws a panel, with each load average as a bar against the number of CPUs. When a
build has every core busy, the 1-minute bar fills and turns yellow, then red:

```text
$ uptimex
╭─ up 2 days, 6:50 ──────────────────────╮
│ Time    06:46:12                       │
│ Users   1                              │
│ Load     1 min  ██▌░░░░░░░░░░░░░  2.35 │
│          5 min  ███▌░░░░░░░░░░░░  3.30 │
│         15 min  ███▏░░░░░░░░░░░░  2.97 │
│ CPUs    15                             │
╰────────────────────────────────────────╯
```

`whox` and `pinkyx` show who is logged in as a table, with your own sessions in the color for
what is yours, so on a shared server you can tell at a glance which logins are not yours:

```text
$ whox -u
╭──────┬─────────┬──────────────┬───────┬──────┬───────────────╮
│ NAME │ LINE    │ TIME         │ IDLE  │  PID │ COMMENT       │
├──────┼─────────┼──────────────┼───────┼──────┼───────────────┤
│ ada  │ pts/0   │ Nov  2 09:14 │ .     │ 4211 │ (10.0.0.23)   │
│ lin  │ pts/1   │ Nov  2 08:02 │ 01:12 │ 3980 │ (192.168.1.7) │
╰──────┴─────────┴──────────────┴───────┴──────┴───────────────╯
```

`idx` shows your IDs as a panel, one group a line, the primary one highlighted; when a program
runs with someone else's rights (setuid), the effective IDs get lines of their own, in red:

```text
$ idx
╭─ ada ─────────────────╮
│ User    501  ada      │
│ Group    20  staff    │
│ Groups   20  staff    │
│          12  everyone │
│          80  admin    │
╰───────────────────────╯
```

`usersx` and `groupsx` keep GNU's line and color your own name and your primary group.

`envx` and `printenvx`, listing the environment, color each name apart from its value, and dim
the `:` between the directories of `PATH` and every other `*PATH` variable, so you can find the
one directory you are looking for in a long search path. `printenvx PATH`, which prints just the
value, and `-0` stay GNU's.

`gnux shell`'s prompt shows the working directory and, after a command fails, its status; the
shell's own `--pretty` and `--theme` apply to every tool you run in the session.

`datex` and `unamex` keep GNU's text even on a terminal, since scripts capture it
(`deadline=$(datex -d 'next friday' +%F)`); ask for the view with `--pretty`. `GNUX_PRETTY` does
not turn it on. `datex --pretty` shows one moment every way you might need it, and how far it is
from now:

```text
$ datex --pretty -d '2026-12-25 18:00' +'%A %d %B'
╭─ Friday 25 December ───────────────────────────╮
│ Local     Fri Dec 25 18:00:00 CET 2026         │
│ UTC       Fri Dec 25 17:00:00 UTC 2026         │
│ ISO 8601  2026-12-25T18:00:00+01:00            │
│ Unix      1798218000                           │
│ Day       Friday, day 359 of the year, week 52 │
│ Zone      CET (UTC+01:00)                      │
│ From now  in 53 days 8 hours                   │
╰────────────────────────────────────────────────╯
```

`unamex --pretty` shows the system as a panel titled with the host's name, every field when you
name none.

`commx` puts a header row over its three columns, naming the files, and colors each column, so
a line only in the new list stands apart from one in both even when the lines are long. GNU's
columns are one tab apart, so a name wider than a tab stop is cut to fit over its column:

```text
$ commx old.txt new.txt
old.txt new.txt both
ada
                lin
        mei
```

`joinx` colors each field by the file it came from, and the join field by whether it paired
both files or came from one alone (`-a`), so you can see which side a value is from.

`killx -l` lists every signal as a table of number, name and description, so you can see what
`killx -s HUP` will do before you send it. The names are colored by what the signal does: `KILL`
and `STOP`, which no program can catch, stand out from the rest. `killx -l 9` still prints just
`KILL`.

`sttyx -a` shows the terminal's settings as a panel titled with the device: the control
characters in an aligned grid, then the control, input, output and local flags, each group on
lines of its own, the flags that are off dimmed. When a program has left the terminal in a
strange state, you can see at a glance that `echo` or `icanon` is off:

```text
$ sttyx -a
╭─ standard input ─────────────────────────────────────────────────────────╮
│ Speed       38400 baud                                                   │
│ Size        24 rows, 80 columns                                          │
│                                                                          │
│ Characters  intr    ^C      quit    ^\      erase   ^?      kill    ^U   │
│             eof     ^D      eol     <undef> eol2    <undef> start   ^Q   │
│             stop    ^S      susp    ^Z      rprnt   ^R      werase  ^W   │
│             lnext   ^V      discard ^O      min     1       time    0    │
│                                                                          │
│ Control     -parenb -parodd cs8 hupcl -cstopb cread -clocal -crtscts     │
│ Input       -ignbrk brkint -ignpar -parmrk -inpck -istrip -inlcr -igncr  │
│             icrnl ixon -ixoff -iuclc ixany imaxbel iutf8                 │
│ Output      opost -olcuc -ocrnl onlcr -onocr -onlret -ofill -ofdel nl0   │
│             cr0 tab0 bs0 vt0 ff0                                         │
│ Local       isig icanon iexten echo echoe echok -echonl -noflsh -xcase   │
│             -tostop -echoprt echoctl echoke -flusho -extproc             │
╰──────────────────────────────────────────────────────────────────────────╯
```

`sttyx -g` and `sttyx` with no option print GNU's text.

`headx` and `tailx` color their `==> FILE <==` headers. `rmx`, `cpx`, `mvx`, `lnx`,
`installx`, `mkdirx`, `rmdirx`, `chmodx`, `chownx` and `chgrpx` color their `-v` reports: a
removal in red, a creation in green, each quoted name, and the `->` arrows. In both, the words
are GNU's; only the color is added.

## Themes

`gnux theme` opens a picker on your terminal, framed to fit it: tabs along the top group the
themes (All, Adaptive, Dark, Light), a search line filters them as you type, the list shows each
theme with its kind, and the theme under the cursor is drawn beside it on a long listing, Git's
marks, a check's verdicts, usage bars and an environment line. A footer names the keys.

| Key                                 | Does                                         |
|-------------------------------------|----------------------------------------------|
| Up, Down (Ctrl-P, Ctrl-N)           | move through the list                        |
| Page Up, Page Down, Home, End       | move a page, or to either end                |
| Tab, Shift-Tab (Right, Left)        | switch group                                 |
| any text, Backspace, Ctrl-U         | filter by name, edit or clear the filter     |
| Enter                               | keep the theme for every tool                |
| Esc, Ctrl-C                         | clear the filter, then leave unchanged       |

Adaptive holds each family's own name (it takes the dark or the light variant to suit your
background) with `default` and `mono`, which use your terminal's colors. Choosing `default`
removes the saved theme, so you are back where you started. The picker draws on the terminal's
alternate screen, so your scrollback is as you left it, and a terminal too small for the frame
gets a line asking for more room.

```bash
gnux theme                 # pick one with a live preview
gnux theme set solarized   # keep one without the picker, from a script or your dotfiles
gnux theme show            # the theme in use, and where it comes from
gnux theme list            # every theme on a sample row (gnux themes is the same)
gnux theme reset           # back to the default colors (as is gnux theme set default)
```

The theme you keep is saved as its name in one small file: `$XDG_CONFIG_HOME/gnux/theme`, else
`~/.config/gnux/theme` (`%AppData%\gnux\theme` on Windows), or `theme` in `GNUX_CONFIG_DIR`
when you set that. Only the name of a built-in theme is accepted, only you may read or write
the file, and it is replaced whole when it changes. A tool reads it only when its output is
colored, so pipes and scripts never touch it. Each way of naming a theme wins over the next:
`--theme=NAME` for one run, then `GNUX_THEME`, then the saved theme, then `default`. The help
screens follow it too: `--help`, `gnux help` and the gnux mark take their heading, command and
dimmed colors from the theme in use.

What keeps it safe:

- The file can only name a built-in theme. It never holds color codes, so nothing written into
  it can reach your terminal as a command; a file holding anything else, larger than 64 bytes or
  not a regular file is reported once and ignored, and the tools carry on in the default colors.
- It is opened without blocking and checked as the file that was opened, so a named pipe put in
  its place cannot make a tool wait, and nothing can be swapped in between the check and the read.
- It is written to a temporary file in the same directory and renamed into place, so a tool
  never reads half of it, and a link left at its path is replaced, not followed. The file is
  readable and writable by you only, in a directory only you may enter.
- The picker echoes only printable text, keeps the filter to 32 characters, reads the keyboard
  through a fixed buffer, and restores the terminal whatever ends it: Enter, Esc, Ctrl-C, an
  error, or a SIGTERM or SIGHUP.

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
| `material`      | Material Darker and Material Lighter     |
| `tomorrow`      | Tomorrow Night and Tomorrow              |
| `selenized`     | Selenized Dark and Selenized Light       |
| `flexoki`       | Flexoki Dark and Flexoki Light           |
| `xcode`         | Xcode's Default Dark and Default Light   |
| `ios`           | Apple's system colors in dark and light mode |
| `colorful`      | gnux's own: every hue at full strength   |
| `monochrome`    | gnux's own: shades of gray, no hues      |

`dracula`, `nord`, `monokai`, `kanagawa`, `palenight`, `ayu-mirage`, `zenburn`, `synthwave`
(after SynthWave '84), `horizon` and `moonlight` come in one, dark, variant. `default` uses
the 16 basic terminal colors, so it follows your terminal's own scheme, and `mono` uses bold,
underline and dim only: no color at all, for monochrome terminals and color blindness. The named themes use 24-bit color, which current terminals show, Windows
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
grepx, headx, tailx), `ok` and `er` (OK and FAILED, a creation and a removal), `pa` (a name
in a `-v` report), `ev` (a variable's name in `envx` and `printenvx`), `f1` `f2` `fb` (from the first file, the
second, or both, in `commx` and `joinx`) and `yz` `yp` `yw` `yc` `yh` (bytes in an `odx` dump: zero, printable,
whitespace, other control characters, and above 0x7f).

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
