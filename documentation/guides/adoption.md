# Adoption

How to adopt gnux next to the GNU coreutils you already know. gnux does not replace them: it brings the same interface to the places they are missing (Windows, containers without coreutils, the browser, Go programs), so the GNU tools stay where they are and gnux is added where it helps. This guide sets the two side by side: how each of the 108 gnux tools maps onto its GNU counterpart, which flags it supports, every documented deviation (the places where gnux must choose differently because it cannot see option order, or because Windows cannot express a feature), the GNU options that are not implemented yet, the places where gnux deliberately does better than GNU, and the three coreutils tools left out by design. If a behavior is not listed here as a difference, gnux aims to match GNU exactly, down to the wording of its diagnostics and its exit statuses.

## The short version

- Add an `x`: `rm` is `rmx`, `cp` is `cpx`, `ls` is `lsx`. The GNU names also work as aliases (`gnux rm`, or a link named `rm`), and inside `gnux shell` both names are the same command.
- The flags, their meanings, the diagnostics and the exit statuses are GNU's.
- An option gnux does not implement is **refused** with `unknown flag`, never silently ignored, so a script that relies on one fails loudly instead of doing the wrong thing.
- Where GNU lets the last of two conflicting options win, gnux picks the **most cautious** one for `rmx`, `cpx`, `mvx`, `lnx`, `pwdx`, `chmodx`, `chownx`, `chgrpx`, `dux` and `readlinkx`. `lsx`, `headx`, `tailx`, and the unit options of `dux` and `dfx`, track option order and keep GNU's last-one-wins.
- Where GNU has a known rough edge, gnux fixes it and says so: recursive `chmodx`, `chownx` and `chgrpx` refuse `/` by default, `dux` reads directories in parallel and lists them in a stable order, `nicex` and `nohupx` really work on Windows, and `killx 1234 -9` never signals process group 9.

## Tool by tool

| GNU | gnux | Supported options | Reference |
|-----|------|-------------------|-----------|
| `arch` | `archx` | none | [archx](https://gnux.vercel.app/docs/tools/archx) |
| `b2sum` | `b2sumx` | `-b`, `-c`, `--ignore-missing`, `-l BITS`, `--quiet`, `--status`, `--strict`, `--tag`, `-t`, `-w`, `-z`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [b2sumx](https://gnux.vercel.app/docs/tools/b2sumx) |
| `base32` | `base32x` | `-d`, `-i`, `-w COLS` | [base32x](https://gnux.vercel.app/docs/tools/base32x) |
| `base64` | `base64x` | `-d`, `-i`, `-w COLS` | [base64x](https://gnux.vercel.app/docs/tools/base64x) |
| `basename` | `basenamex` | `-a`, `-s SUFFIX`, `-z` | [basenamex](https://gnux.vercel.app/docs/tools/basenamex) |
| `basenc` | `basencx` | `--base64`, `--base64url`, `--base32`, `--base32hex`, `--base16`, `--base2msbf`, `--base2lsbf`, `--z85`, `-d`, `-i`, `-w COLS` | [basencx](https://gnux.vercel.app/docs/tools/basencx) |
| `cat` | `catx` | `-A`, `-b`, `-e`, `-E`, `-n`, `-s`, `-t`, `-T`, `-u`, `-v` | [catx](https://gnux.vercel.app/docs/tools/catx) |
| `chgrp` | `chgrpx` | `-c`, `--dereference`, `-L`, `-H`, `--from=CURRENT_OWNER:CURRENT_GROUP`, `-h`, `--no-preserve-root`, `-P`, `--preserve-root`, `--quiet`, `-R`, `--reference=RFILE`, `-f`, `-v`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [chgrpx](https://gnux.vercel.app/docs/tools/chgrpx) |
| `chmod` | `chmodx` | `-c`, `--dereference`, `-L`, `-H`, `-h`, `-P`, `--no-preserve-root`, `--preserve-root`, `--quiet`, `-R`, `--reference=RFILE`, `-f`, `-v`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [chmodx](https://gnux.vercel.app/docs/tools/chmodx) |
| `chown` | `chownx` | `-c`, `--dereference`, `-L`, `-H`, `--from=CURRENT_OWNER:CURRENT_GROUP`, `-h`, `--no-preserve-root`, `-P`, `--preserve-root`, `--quiet`, `-R`, `--reference=RFILE`, `-f`, `-v`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [chownx](https://gnux.vercel.app/docs/tools/chownx) |
| `cksum` | `cksumx` | `-a TYPE`, `--base64`, `-c`, `--ignore-missing`, `-l BITS`, `--quiet`, `--raw`, `--status`, `--strict`, `--tag`, `--untagged`, `-w`, `-z`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [cksumx](https://gnux.vercel.app/docs/tools/cksumx) |
| `comm` | `commx` | `--check-order`, `--nocheck-order`, `--output-delimiter=STR`, `-3`, `-1`, `-2`, `--total`, `-z` | [commx](https://gnux.vercel.app/docs/tools/commx) |
| `cp` | `cpx` | `-a`, `--backup[=CONTROL]`, `-b`, `-d`, `-f`, `-H`, `-i`, `-l`, `-L`, `-n`, `-p`, `-P`, `-r`/`-R`, `-s`, `-t`, `-T`, `-u[=WHEN]`, `-v`, `-x`, `--parents`, `--preserve[=LIST]`, `--no-preserve=LIST`, `--remove-destination`, `-S SUFFIX`, `--strip-trailing-slashes`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [cpx](https://gnux.vercel.app/docs/tools/cpx) |
| `csplit` | `csplitx` | `-n DIGITS`, `-z`, `-k`, `-f PREFIX`, `-s`, `-q`, `-b FORMAT`, `--suppress-matched` | [csplitx](https://gnux.vercel.app/docs/tools/csplitx) |
| `cut` | `cutx` | `-b LIST`, `-c LIST`, `-d DELIM`, `-f LIST`, `-n`, `-s`, `-z`, `--complement`, `--output-delimiter=STRING` | [cutx](https://gnux.vercel.app/docs/tools/cutx) |
| `date` | `datex` | `-d STRING`, `-f DATEFILE`, `-I[=FMT]`, `-r FILE`, `--resolution`, `--rfc-3339=FMT`, `-R`, `-s STRING`, `--universal`, `-u` | [datex](https://gnux.vercel.app/docs/tools/datex) |
| `dd` | `ddx` | operands `if=`, `of=`, `bs=`, `ibs=`, `obs=`, `cbs=`, `skip=`/`iseek=`, `seek=`/`oseek=`, `count=`, `conv=`, `iflag=`, `oflag=`, `status=` | [ddx](https://gnux.vercel.app/docs/tools/ddx) |
| `df` | `dfx` | `-a`, `-B SIZE`, `-x TYPE`, `-h`, `-i`, `-k`, `-l`, `--no-sync`, `--output[=FIELD_LIST]`, `-P`, `-T`, `-H`, `--sync`, `-v`, `--total`, `-t TYPE`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [dfx](https://gnux.vercel.app/docs/tools/dfx) |
| `dir` | `dirx` | every `lsx` option | [dirx](https://gnux.vercel.app/docs/tools/dirx) |
| `dircolors` | `dircolorsx` | `-b`, `-c`, `-p`, `--print-ls-colors` | [dircolorsx](https://gnux.vercel.app/docs/tools/dircolorsx) |
| `dirname` | `dirnamex` | `-z` | [dirnamex](https://gnux.vercel.app/docs/tools/dirnamex) |
| `du` | `dux` | `-a`, `-A`, `-B SIZE`, `-b`, `-l`, `-L`, `-D`, `--exclude=PATTERN`, `-X FILE`, `--files0-from=F`, `-h`, `--inodes`, `-k`, `-d N`, `-m`, `-P`, `-0`, `-x`, `-S`, `--si`, `-s`, `-t SIZE`, `--time[=WORD]`, `--time-style=STYLE`, `-c`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [dux](https://gnux.vercel.app/docs/tools/dux) |
| `echo` | `echox` | `-n`, `-e`, `-E` | [echox](https://gnux.vercel.app/docs/tools/echox) |
| `env` | `envx` | `-0`, `-a ARG`, `-C DIR`, `-i`/`-`, `-S STRING`, `-u NAME`, `-v` | [envx](https://gnux.vercel.app/docs/tools/envx) |
| `expand` | `expandx` | `-i`, `-t N, LIST` | [expandx](https://gnux.vercel.app/docs/tools/expandx) |
| `expr` | `exprx` | EXPRESSION: `\|`, `&`, comparisons, `+ - * / %`, `:`, `match`, `substr`, `index`, `length`, `+ TOKEN`, parentheses | [exprx](https://gnux.vercel.app/docs/tools/exprx) |
| `factor` | `factorx` | `-h` | [factorx](https://gnux.vercel.app/docs/tools/factorx) |
| `false` | `falsex` | none (arguments ignored) | [falsex](https://gnux.vercel.app/docs/tools/falsex) |
| `find` (findutils) | `findx` | An expression, not options: `-H`, `-L`, `-P`; `( )`, `!`, `-a`, `-o`, `,`; tests such as `-name`, `-path`, `-regex`, `-type`, `-size`, `-newer`, `-mtime`, `-perm`, `-user`, `-empty`; `-maxdepth`, `-mindepth`, `-depth`, `-xdev`; actions `-print`, `-print0`, `-printf`, `-ls`, `-delete`, `-prune`, `-quit`, `-exec`, `-execdir`, `-ok` | [findx](https://gnux.vercel.app/docs/tools/findx) |
| `fmt` | `fmtx` | `-c`, `-g STRING`, `-p STRING`, `-s`, `-t`, `-u`, `-w WIDTH` | [fmtx](https://gnux.vercel.app/docs/tools/fmtx) |
| `fold` | `foldx` | `-b`, `-s`, `-w WIDTH` | [foldx](https://gnux.vercel.app/docs/tools/foldx) |
| `grep` (GNU grep) | `grepx` | `-E`, `-F`, `-G`, `-e PATTERNS`, `-f FILE`, `-i`, `-v`, `-w`, `-x`, `-c`, `-l`, `-L`, `-m NUM`, `-o`, `-q`, `-s`, `-b`, `-H`, `-h`, `-n`, `-T`, `-Z`, `-z`, `-A NUM`, `-B NUM`, `-C NUM`, `-NUM`, `-r`, `-R`, `-d ACTION`, `-D ACTION`, `-a`, `-I`, `-U`, `--binary-files=TYPE`, `--color[=WHEN]`, `--include=GLOB`, `--exclude=GLOB`, `--exclude-from=FILE`, `--exclude-dir=GLOB`, `--group-separator=SEP`, `--no-group-separator`, `--label=LABEL`, `--line-buffered`, `--no-ignore-case`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [grepx](https://gnux.vercel.app/docs/tools/grepx) |
| `groups` | `groupsx` | none | [groupsx](https://gnux.vercel.app/docs/tools/groupsx) |
| `head` | `headx` | `-c [-]NUM`, `-n [-]NUM`, `-NUM`, `-q`, `-v`, `-z`, size suffixes; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [headx](https://gnux.vercel.app/docs/tools/headx) |
| `hostid` | `hostidx` | none | [hostidx](https://gnux.vercel.app/docs/tools/hostidx) |
| `hostname` | `hostnamex` | none | [hostnamex](https://gnux.vercel.app/docs/tools/hostnamex) |
| `id` | `idx` | `-a`, `-Z`, `-g`, `-G`, `-n`, `-r`, `-u`, `-z` | [idx](https://gnux.vercel.app/docs/tools/idx) |
| `install` | `installx` | `--backup[=CONTROL]`, `-b`, `-C`, `-c`, `-D`, `-d`, `-g GROUP`, `-m MODE`, `-T`, `-o OWNER`, `-p`, `-s`, `-S SUFFIX`, `-t DIRECTORY`, `-v`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [installx](https://gnux.vercel.app/docs/tools/installx) |
| `join` | `joinx` | `--check-order`, `-e EMPTY`, `-1 FIELD`, `-2 FIELD`, `--header`, `-i`, `-j STRINGARRAY`, `--nocheck-order`, `-v STRINGARRAY`, `-o FORMAT`, `-t CHAR`, `-a FILENUM`, `-z` | [joinx](https://gnux.vercel.app/docs/tools/joinx) |
| `kill` | `killx` | `-l`, `-n NUMBER`, `-s STRING`, `-L` | [killx](https://gnux.vercel.app/docs/tools/killx) |
| `link` | `linkx` | none | [linkx](https://gnux.vercel.app/docs/tools/linkx) |
| `ln` | `lnx` | `--backup[=CONTROL]`, `-b`, `-f`, `-i`, `-L`, `-n`, `-P`, `-r`, `-s`, `-S SUFFIX`, `-t`, `-T`, `-v`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [lnx](https://gnux.vercel.app/docs/tools/lnx) |
| `logname` | `lognamex` | none | [lognamex](https://gnux.vercel.app/docs/tools/lognamex) |
| `ls` | `lsx` | `-1`, `-a`, `-A`, `-B`, `-b`, `-c`, `-C`, `-d`, `-F`, `-g`, `-G`, `-h`, `-H`, `-i`, `-I`, `-l`, `-L`, `-m`, `-n`, `-o`, `-p`, `-q`, `-Q`, `-r`, `-R`, `-s`, `-S`, `-t`, `-T`, `-u`, `-U`, `-v`, `-w`, `-x`, `-X`, `--author`, `--block-size`, `--color`, `-D`/`--dired`, `--file-type`, `--format`, `--full-time`, `--group-directories-first`, `--hide`, `--hyperlink`, `--indicator-style`, `-k`, `-N`, `--quoting-style`, `--show-control-chars`, `--si`, `--sort`, `--time`, `--time-style`, `--zero`; plus gnux's own terminal view: `--pretty`, `--theme`, `--tree`, `--level`, `--icons`, `--git`, `--git-ignore`, `--header`, `--color-scale`, `--only-dirs`, `--only-files`, `--total-size`, `--octal-permissions` ([Terminal view](terminal-view.md)) | [lsx](https://gnux.vercel.app/docs/tools/lsx) |
| `md5sum` | `md5sumx` | `-b`, `-c`, `--ignore-missing`, `--quiet`, `--status`, `--strict`, `--tag`, `-t`, `-w`, `-z`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [md5sumx](https://gnux.vercel.app/docs/tools/md5sumx) |
| `mkdir` | `mkdirx` | `-m MODE`, `-p`, `-v`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [mkdirx](https://gnux.vercel.app/docs/tools/mkdirx) |
| `mkfifo` | `mkfifox` | `-m MODE`, `-Z`, `--context[=CTX]` | [mkfifox](https://gnux.vercel.app/docs/tools/mkfifox) |
| `mknod` | `mknodx` | `-m MODE`, `-Z`, `--context[=CTX]` | [mknodx](https://gnux.vercel.app/docs/tools/mknodx) |
| `mktemp` | `mktempx` | `-p DIR`, `-d`, `-u`, `-t`, `-q`, `--suffix=SUFF`, `--tmpdir[=DIR]` | [mktempx](https://gnux.vercel.app/docs/tools/mktempx) |
| `mv` | `mvx` | `--backup[=CONTROL]`, `-b`, `-f`, `-i`, `-n`, `--no-copy`, `-S SUFFIX`, `-t`, `-T`, `-u[=WHEN]`, `-v`, `--strip-trailing-slashes`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [mvx](https://gnux.vercel.app/docs/tools/mvx) |
| `nice` | `nicex` | `-n N` | [nicex](https://gnux.vercel.app/docs/tools/nicex) |
| `nl` | `nlx` | `-b STYLE`, `-f STYLE`, `-h STYLE`, `-l NUMBER`, `-i STRING`, `-p`, `-n FORMAT`, `-s STRING`, `-w NUMBER`, `-d CC`, `-v STRING` | [nlx](https://gnux.vercel.app/docs/tools/nlx) |
| `nohup` | `nohupx` | none | [nohupx](https://gnux.vercel.app/docs/tools/nohupx) |
| `nproc` | `nprocx` | `--all`, `--ignore=N` | [nprocx](https://gnux.vercel.app/docs/tools/nprocx) |
| `numfmt` | `numfmtx` | `--debug`, `-d X`, `--field=FIELDS`, `--format=FORMAT`, `--from=UNIT`, `--from-unit=N`, `--grouping`, `--header[=N]`, `--invalid=MODE`, `--padding=N`, `--round=METHOD`, `--suffix=SUFFIX`, `--to=UNIT`, `--to-unit=N`, `-z` | [numfmtx](https://gnux.vercel.app/docs/tools/numfmtx) |
| `od` | `odx` | `-A RADIX`, `-c[=STRING]`, `-s[=STRING]`, `-i[=STRING]`, `-l[=STRING]`, `--endian=ORDER`, `-f[=STRING]`, `-t TYPE`, `-x[=STRING]`, `-a[=STRING]`, `-o[=STRING]`, `-b[=STRING]`, `-v`, `-N BYTES`, `-j BYTES`, `-S BYTES`, `-d[=STRING]`, `-w BYTES` | [odx](https://gnux.vercel.app/docs/tools/odx) |
| `paste` | `pastex` | `-d LIST`, `-s`, `-z` | [pastex](https://gnux.vercel.app/docs/tools/pastex) |
| `pathchk` | `pathchkx` | `-p`, `-P`, `--portability` | [pathchkx](https://gnux.vercel.app/docs/tools/pathchkx) |
| `pinky` | `pinkyx` | `-l`, `-b`, `-h`, `-p`, `-s`, `-f`, `-w`, `-i`, `-q`, `--lookup` | [pinkyx](https://gnux.vercel.app/docs/tools/pinkyx) |
| `pr` | `prx` | `+FIRST[:LAST]`, `--pages`, `-COLUMN`, `--columns`, `-a`, `-c`, `-d`, `-D FORMAT`, `-e[CHAR[WIDTH]]`, `-F`/`-f`, `-h HEADER`, `-i[CHAR[WIDTH]]`, `-J`, `-l LENGTH`, `-m`, `-n[SEP[DIGITS]]`, `-N NUMBER`, `-o MARGIN`, `-r`, `-s[CHAR]`, `-S[STRING]`, `-t`, `-T`, `-v`, `-w WIDTH`, `-W WIDTH` | [prx](https://gnux.vercel.app/docs/tools/prx) |
| `printenv` | `printenvx` | `-0` | [printenvx](https://gnux.vercel.app/docs/tools/printenvx) |
| `printf` | `printfx` | FORMAT [ARGUMENT]... (no options) | [printfx](https://gnux.vercel.app/docs/tools/printfx) |
| `ptx` | `ptxx` | `-A`, `-G`, `-F STRING`, `-M STRING`, `-O`, `-R`, `-S REGEXP`, `-T`, `--format=FORMAT`, `-W REGEXP`, `-b FILE`, `-f`, `-g NUMBER`, `-i FILE`, `-o FILE`, `-r`, `-t`, `-w NUMBER` | [ptxx](https://gnux.vercel.app/docs/tools/ptxx) |
| `pwd` | `pwdx` | `-L`, `-P` | [pwdx](https://gnux.vercel.app/docs/tools/pwdx) |
| `readlink` | `readlinkx` | `-f`, `-e`, `-m`, `-n`, `-q`, `-s`, `-v`, `-z` | [readlinkx](https://gnux.vercel.app/docs/tools/readlinkx) |
| `realpath` | `realpathx` | `-e`, `-L`, `-m`, `-P`, `-q`, `-s`/`--no-symlinks`, `-z`, `--relative-to`, `--relative-base` | [realpathx](https://gnux.vercel.app/docs/tools/realpathx) |
| `rm` | `rmx` | `-d`, `-f`, `-i`, `-I`, `-r`/`-R`, `-v`, `--interactive[=WHEN]`, `--one-file-system`, `--preserve-root`, `--no-preserve-root`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [rmx](https://gnux.vercel.app/docs/tools/rmx) |
| `rmdir` | `rmdirx` | `-p`, `-v`, `--ignore-fail-on-non-empty`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [rmdirx](https://gnux.vercel.app/docs/tools/rmdirx) |
| `sed` (GNU sed) | `sedx` | `-n`, `-e SCRIPT`, `-f SCRIPT-FILE`, `-E`/`-r`, `-i[SUFFIX]`, `-s`, `-z`, `-u`, `-l N`, `-b`, `--follow-symlinks`, `--posix`, `--sandbox`; the whole script language except the `e` command | [sedx](https://gnux.vercel.app/docs/tools/sedx) |
| `seq` | `seqx` | `-f FORMAT`, `-s STRING`, `-w` | [seqx](https://gnux.vercel.app/docs/tools/seqx) |
| `sha1sum` | `sha1sumx` | `-b`, `-c`, `--ignore-missing`, `--quiet`, `--status`, `--strict`, `--tag`, `-t`, `-w`, `-z`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [sha1sumx](https://gnux.vercel.app/docs/tools/sha1sumx) |
| `sha224sum` | `sha224sumx` | `-b`, `-c`, `--ignore-missing`, `--quiet`, `--status`, `--strict`, `--tag`, `-t`, `-w`, `-z`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [sha224sumx](https://gnux.vercel.app/docs/tools/sha224sumx) |
| `sha256sum` | `sha256sumx` | `-b`, `-c`, `--ignore-missing`, `--quiet`, `--status`, `--strict`, `--tag`, `-t`, `-w`, `-z`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [sha256sumx](https://gnux.vercel.app/docs/tools/sha256sumx) |
| `sha384sum` | `sha384sumx` | `-b`, `-c`, `--ignore-missing`, `--quiet`, `--status`, `--strict`, `--tag`, `-t`, `-w`, `-z`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [sha384sumx](https://gnux.vercel.app/docs/tools/sha384sumx) |
| `sha512sum` | `sha512sumx` | `-b`, `-c`, `--ignore-missing`, `--quiet`, `--status`, `--strict`, `--tag`, `-t`, `-w`, `-z`; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [sha512sumx](https://gnux.vercel.app/docs/tools/sha512sumx) |
| `shred` | `shredx` | `-f`, `-n N`, `--random-source=FILE`, `-s N`, `-u`, `--remove[=HOW]`, `-v`, `-x`, `-z` | [shredx](https://gnux.vercel.app/docs/tools/shredx) |
| `shuf` | `shufx` | `-e`, `-n COUNT`, `-i LO-HI`, `-o FILE`, `--random-source=FILE`, `-r`, `-z` | [shufx](https://gnux.vercel.app/docs/tools/shufx) |
| `sleep` | `sleepx` | NUMBER with `s`, `m`, `h`, `d`; fractions, exponents, hex floats, `infinity` | [sleepx](https://gnux.vercel.app/docs/tools/sleepx) |
| `sort` | `sortx` | `-b`, `-c`/`--check[=WHEN]`, `-C`, `-d`, `-f`, `-g`, `-h`, `-i`, `-k KEYDEF`, `-m`, `-M`, `-n`, `-o FILE`, `-r`, `-R`, `-s`, `-t SEP`, `-u`, `-V`, `-z`, `--files0-from=F`, `--random-source=FILE`, `--sort=WORD`; `-S`, `-T`, `--parallel`, `--batch-size`, `--compress-program` accepted and ignored | [sortx](https://gnux.vercel.app/docs/tools/sortx) |
| `split` | `splitx` | `--additional-suffix=SUFFIX`, `-b SIZE`, `-e`, `-x[=FROM]`, `-C SIZE`, `-l NUMBER`, `-n CHUNKS`, `-d[=FROM]`, `-t SEP`, `-a N`, `-u`, `--verbose` | [splitx](https://gnux.vercel.app/docs/tools/splitx) |
| `stat` | `statx` | `--cached=STRING`, `-L`, `-f`, `-c FORMAT`, `--printf=FORMAT`, `-t` | [statx](https://gnux.vercel.app/docs/tools/statx) |
| `stdbuf` | `stdbufx` | `-i MODE`, `-o MODE`, `-e MODE` | [stdbufx](https://gnux.vercel.app/docs/tools/stdbufx) |
| `stty` | `sttyx` | `-a`, `-g`, `-F DEVICE`, and every setting GNU stty has on the platform | [sttyx](https://gnux.vercel.app/docs/tools/sttyx) |
| `sum` | `sumx` | `-r`, `-s` | [sumx](https://gnux.vercel.app/docs/tools/sumx) |
| `sync` | `syncx` | `-d`, `-f` | [syncx](https://gnux.vercel.app/docs/tools/syncx) |
| `tac` | `tacx` | `-b`, `-r`, `-s STRING` | [tacx](https://gnux.vercel.app/docs/tools/tacx) |
| `tail` | `tailx` | `-c [+]NUM`, `-f[=HOW]`, `-F`, `-n [+]NUM`, `-q`, `-s N`, `-v`, `-z`, `--max-unchanged-stats=N`, `--pid`, `--retry`, size suffixes; plus gnux's own `--pretty`, `--theme` ([Terminal view](terminal-view.md)) | [tailx](https://gnux.vercel.app/docs/tools/tailx) |
| `tee` | `teex` | `-a`, `-i`, `-p`, `--output-error[=MODE]` | [teex](https://gnux.vercel.app/docs/tools/teex) |
| `test`, `[` | `testx`, `[` | EXPRESSION: string, integer and file tests, `-l STRING`, `-nt`/`-ot`/`-ef`, `!`, `-a`, `-o`, parentheses | [testx](https://gnux.vercel.app/docs/tools/testx) |
| `timeout` | `timeoutx` | `--foreground`, `-k DURATION`, `--preserve-status`, `-s SIGNAL`, `-v` | [timeoutx](https://gnux.vercel.app/docs/tools/timeoutx) |
| `touch` | `touchx` | `-a`, `-c`, `-d STRING`, `-h`, `-m`, `-r FILE`, `-t STAMP`, `--time=WORD` | [touchx](https://gnux.vercel.app/docs/tools/touchx) |
| `tr` | `trx` | `-c`/`-C`, `-d`, `-s`, `-t`; SET ranges, escapes, `[:CLASS:]`, `[=C=]`, `[C*N]`, `[C*]` | [trx](https://gnux.vercel.app/docs/tools/trx) |
| `true` | `truex` | none (arguments ignored) | [truex](https://gnux.vercel.app/docs/tools/truex) |
| `truncate` | `truncatex` | `-o`, `-c`, `-r RFILE`, `-s SIZE` | [truncatex](https://gnux.vercel.app/docs/tools/truncatex) |
| `tsort` | `tsortx` | none | [tsortx](https://gnux.vercel.app/docs/tools/tsortx) |
| `tty` | `ttyx` | `--quiet`, `-s` | [ttyx](https://gnux.vercel.app/docs/tools/ttyx) |
| `uname` | `unamex` | `-a`, `-i`, `-s`, `-r`, `-v`, `-m`, `-n`, `-o`, `-p` | [unamex](https://gnux.vercel.app/docs/tools/unamex) |
| `unexpand` | `unexpandx` | `-a`, `--first-only`, `-t N, LIST` | [unexpandx](https://gnux.vercel.app/docs/tools/unexpandx) |
| `uniq` | `uniqx` | `-c`, `-d`, `-D`, `-f N`, `-i`, `-s N`, `-u`, `-w N`, `-z`, `--all-repeated[=METHOD]`, `--group[=METHOD]` | [uniqx](https://gnux.vercel.app/docs/tools/uniqx) |
| `unlink` | `unlinkx` | none | [unlinkx](https://gnux.vercel.app/docs/tools/unlinkx) |
| `uptime` | `uptimex` | none | [uptimex](https://gnux.vercel.app/docs/tools/uptimex) |
| `users` | `usersx` | none | [usersx](https://gnux.vercel.app/docs/tools/usersx) |
| `vdir` | `vdirx` | every `lsx` option | [vdirx](https://gnux.vercel.app/docs/tools/vdirx) |
| `wc` | `wcx` | `-c`, `-l`, `-L`, `-m`, `-w`, `--files0-from=F`, `--total=WHEN` | [wcx](https://gnux.vercel.app/docs/tools/wcx) |
| `who` | `whox` | `-a`, `-b`, `-q`, `-d`, `-H`, `-l`, `--lookup`, `-T`, `--message`, `-m`, `-p`, `-r`, `-s`, `-t`, `-u`, `-w` | [whox](https://gnux.vercel.app/docs/tools/whox) |
| `whoami` | `whoamix` | none | [whoamix](https://gnux.vercel.app/docs/tools/whoamix) |
| `xargs` (findutils) | `xargsx` | `-0`, `-a FILE`, `-d CHARACTER`, `-E END`, `-e[END]`, `-I R`, `-i[R]`, `-L MAX-LINES`, `-l[MAX-LINES]`, `-n MAX-ARGS`, `-P MAX-PROCS`, `-p`, `-r`, `-s MAX-CHARS`, `-t`, `-x`, `--show-limits` | [xargsx](https://gnux.vercel.app/docs/tools/xargsx) |
| `yes` | `yesx` | STRING... (no options) | [yesx](https://gnux.vercel.app/docs/tools/yesx) |

Every tool also takes `--help` and `--version`. For `echox`, `printfx`, `truex`, `falsex`, `yesx`, `seqx`, `exprx`, `testx` and `ddx`, as in GNU, they are honored only as the sole argument.

## Documented deviations

These are the places where gnux deliberately differs from GNU. Each is stated in the tool's reference page and in section 2 of the [functional requirements](../../docs/requirements/functional-requirements.md).

### Conflicting options resolve toward the most cautious

GNU lets the last of conflicting options win. The command-line framework gnux uses does not report option order, so for these tools a fixed precedence applies, always toward the option that destroys least:

| Tool | Precedence (strongest first) | Example |
|------|------------------------------|---------|
| `rmx` | `-i` > `--interactive=WHEN` > `-I` > `-f` | `rmx -if x` prompts, where `rm -if x` would not. `-f` still silences "No such file" errors |
| `mvx` | `-n` > `-i` > `-f` | `mvx -fn a b` never overwrites |
| `cpx` | `-n` > `-i`; `-P`/`-d` > `-H` > `-L` | `cpx -in a b` never overwrites and never asks |
| `lnx` | `-i` > `-f`; `-L` > `-P` | `lnx -fi a b` asks before replacing |
| `pwdx` | `-L` over `-P` | `pwdx -PL` and `pwdx -LP` both print `$PWD` when it is valid |
| `realpathx` | `-e` over `-m` | |
| `readlinkx` | `-e` > `-f` > `-m`; `-v` over `-q`/`-s` | `readlinkx -m -e x` requires every component to exist |
| `chmodx` | `-P` > `-H` > `-L`; `-h` over `--dereference`; `--preserve-root` over `--no-preserve-root` | `chmodx -RLP g+w d` follows no symbolic link |
| `chownx`, `chgrpx` | `-P` > `-H` > `-L`; `-h` over `--dereference`; `--preserve-root` over `--no-preserve-root` | `chownx -h --dereference u l` changes the link itself |
| `dux` | `-P` > `-H`/`-D` > `-L` | `dux -LP` follows no symbolic link |
| `grepx` | `-F` > `-E` > `-G`; `-H` > `-h`; `--no-ignore-case` > `-i`; `-A`/`-B` > `-C` > `-NUM`; `-I` > `-a` > `--binary-files`; `-R` > `-r` > `-d`; `-l` > `-L` | `grepx -hH x a` prints the file name |
| `xargsx` | `-I`/`-i` > `-L`/`-l` > `-n`; `-0` > `-d` | `xargsx -n 2 -I {} echo {}` runs one command per line and warns that `-n` is ignored |
| `commx`, `joinx` | `--check-order` over `--nocheck-order` | |
| `cksumx` | `--tag` over `--untagged` | |
| `sumx` | `-s` over `-r` | `sumx -s -r f` prints the System V checksum |
| digest tools (`md5sumx`, `sha*sumx`, `b2sumx`) | `-t` (text) over `-b` (binary) | `sha256sumx -bt f` prints the text-mode mark |

`lsx`, `headx` and `tailx` do track order: `lsx -lC` is columns, `lsx -Cl` is long, exactly as in GNU. So do the unit options of `dux` and `dfx` (`-h`, `-k`, `-m`, `-B`, `--si`), `splitx -d`/`-x`, `statx -c`/`--printf`, and the `-NUM` forms of `foldx`, `fmtx` and `splitx`.

The rule is in the project's [rules](../../docs/rules/README.md) as "conflicts resolve toward the most cautious option".

### Behavior differences, tool by tool

| Tool | Difference |
|------|------------|
| `catx` | `-e`, `-t` and `-u` also answer to the long names `--show-nonprinting-ends`, `--show-nonprinting-tabs` and `--unbuffered`, which GNU does not have |
| `cpx` | Option precedence as above. `-d`, `-H` and `-p` also answer to `--no-dereference-links`, `--dereference-args` and `--preserve-default`. On Windows only the read-only attribute and the modification time are carried over |
| `cutx` | `-c` counts bytes, as GNU's cut does. The same kind of list given twice keeps the last instead of refusing |
| `envx` | GNU `env` replaces itself with the command; `envx` runs it as a child process on the same streams and exits with its status (128 plus the signal number when a signal killed it), so it behaves the same on platforms that cannot replace a running process |
| `lnx` | Option precedence as above. On Windows, symbolic links need Developer Mode or an elevated prompt, a link's target is stored with backslashes, and replacing a link to a directory (`-sfn`) is not atomic: the old link is removed before the new one is renamed in |
| `lsx` | Names sort by plain byte comparison (the C locale) whatever `LANG` says, so uppercase sorts before lowercase. On Windows the owner, group, inode and block columns hold placeholders, and every directory is colored as a plain directory (Windows has no other-writable or sticky bits). Several single-letter options also have gnux long names: `--long`, `--one-per-line`, `--sort-size`, `--sort-time`, `--sort-version`, `--sort-extension`, `--unsorted`, `--columns`, `--across`, `--commas`, `--slash`, `--ctime`, `--atime`, `--long-no-owner`, `--long-no-group`. Names are read as UTF-8 in every locale, so in the C locale a printable accented letter is shown where GNU escapes its bytes |
| `dirx`, `vdirx` | As `lsx`: they are `lsx -C -b` and `lsx -l -b`, and an option that picks a format or quoting replaces those defaults |
| `mkdirx` | On Windows only the read-only attribute is representable, so most of a `-m` mode has no effect |
| `mvx` | Option precedence as above |
| `printfx` | Floating-point values are held in double precision, not long double. `\c` ends output with the status the run already had rather than always 0. `%b` and `%q` also accept a width and precision |
| `pwdx` | `-L` wins when both are given |
| `realpathx` | `-e` wins over `-m` |
| `rmx` | Option precedence as above. `-i` and `-I` also answer to `--prompt-each` and `--prompt-once`. Off Unix, `--one-file-system` has no effect (no device numbers). Independent arguments are removed concurrently when nothing is prompted or printed |
| `seqx` | `-f` values are formatted from double precision, not long double. An infinite FIRST or INCREMENT is refused as an invalid argument |
| `sortx` | Comparison follows the C locale (byte order) whatever `LANG` says. `-S`, `-T`, `--parallel`, `--batch-size` and `--compress-program` are accepted and ignored. `-c` wins over `-C` when both are given |
| `tailx` | Following is done by polling at the `-s` interval (no inotify). `-F` also answers to `--follow-retry`. `--follow=name` checks the name on every poll, so `--max-unchanged-stats` is validated but changes nothing |
| `teex` | `-i` is accepted but has no effect (tools run in-process). An explicit `--output-error=MODE` wins over `-p` in either order. A closed pipe without `--output-error` ends quietly with status 0 instead of SIGPIPE |
| `touchx` | `-d` accepts `YYYY-MM-DD`, `YYYY-MM-DD hh:mm[:ss]`, RFC 3339, `@SECONDS` and `now`, not GNU's free-form date phrases such as `yesterday` or `next tuesday`. `-h` works only on Linux; elsewhere the link is reported rather than its target touched. `-a`, `-m` and `-t` also answer to `--access-time`, `--modification-time` and `--stamp` |
| `uniqx` | `-s` and `-w` count bytes and `-i` folds ASCII letters only (GNU's C locale) |
| `chmodx` | Refuses to work recursively on `/` unless `--no-preserve-root`, where GNU chmod leaves that safeguard off by default. A mode written as an option (`chmodx -w f`) is diagnosed for umask surprises exactly as in GNU. Where links have no mode of their own (Linux), `-h` reports a link as unchanged, as GNU does; on macOS and the BSDs it changes the link. On Windows only the owner's write bit has an effect |
| `chownx`, `chgrpx` | Refuse to work recursively on `/` unless `--no-preserve-root` (GNU's default is the opposite). On Windows, which has no Unix owners, every change fails with "Operation not supported" |
| `cksumx` | `-a` accepts unambiguous abbreviations, as GNU's argmatch does; `--tag` wins over `--untagged` |
| `commx`, `joinx` | Lines compare byte by byte (the C locale). `-1`, `-2`, `-3` of commx also answer to `--suppress-file1`, `--suppress-file2`, `--suppress-common`. joinx refuses the obsolescent `-j1 FIELD` and `-j2 FIELD` |
| `csplitx`, `nlx` | Regular expressions are POSIX basic ones translated to Go's RE2, so back references (`\1`) are refused |
| `datex` | `-d` understands the common forms (ISO dates, `@SECONDS`, `yesterday`, `next friday`, `3 days ago`, `tomorrow 09:00`) but not all of GNU's grammar. It never sets the clock: `-s` and a `MMDDhhmm` operand fail with "Operation not supported". `--debug` is not implemented |
| `ddx` | See its reference page: `status=progress` prints only the final statistics; some conversions and flags (`ascii`, `ebcdic`, `sparse`, `direct`, ...) are refused |
| `dfx` | On Windows, lists every drive with the same columns, and skips a card reader or DVD drive with no medium. `--sync` is accepted and has no effect |
| `dux` | Reads directories in parallel and lists entries in name order, so the output and which path a hard-linked file is counted under are the same on every run (GNU lists them in file-system order) |
| digest tools (`md5sumx`, `sha*sumx`, `b2sumx`) | `-t` wins over `-b` when both are given |
| `expandx`, `unexpandx` | GNU's comma form `-4,8` is not accepted, and two `-N` options in a row run together (`-4 -8` reads as `-48`); write `-t 4,8` |
| `exprx` | Strings are measured and compared in bytes; back references and `[= =]`/`[. .]` are not supported in a regex |
| `findx` | Directory entries are visited in name order, and paths are joined with `/` on every platform. A command named after a gnux tool runs in-process (under `-execdir` it is given the full path); a command that cannot be run makes the status 1. Regular expressions are RE2, so back references are refused. `-newerXt` takes a fixed set of date spellings. Off Unix there are no owners, inodes or devices. `findx -L` detects loops; `findx` itself always follows GNU's `-P`/`-H`/`-L` last-one-wins, because it parses its own command line |
| `grepx` | Patterns are translated to RE2: back references are refused, `\<` and `\>` behave as `\b`, a repetition count above 1000 is too big. Input is treated as UTF-8 and only a NUL byte makes a file binary. Directory entries are searched in name order. A missing pattern is a usage error in gnux's form. `GREP_COLOR` is not read |
| `sedx` | The `e` command is refused: sedx does not run commands. Regular expressions are RE2: back references inside a pattern are refused when the script is parsed (`\1` in a replacement works), `\<` and `\>` behave as `\b`. `--posix` changes only `n` and `N` at end of input. `-i` keeps the permission bits but not the owner, group or extended attributes |
| `hostnamex` | Never sets the host name: `NAME` fails with "Operation not supported" |
| `idx` | `-Z` (SELinux) fails as on a kernel without SELinux. On Windows the IDs are SIDs |
| `killx` | Options are read wherever they appear before `--`, so `killx 1234 -9` sends KILL to 1234 rather than signaling process group 9. On Windows only KILL and TERM (both terminate) and 0 can be sent |
| `nicex` | Runs COMMAND as a child and sets its niceness as it starts. On Windows the niceness maps onto priority classes (idle to high, never realtime), so it really lowers a job's priority there |
| `nohupx` | Runs COMMAND as a child. On Windows it starts COMMAND detached from the console, so closing the window or pressing Ctrl-C no longer ends it |
| `nprocx` | Both counts come from Go's `runtime.NumCPU`, so `--all` cannot count offline processors |
| `odx` | Characters are read in the C locale; long double is read as double; `-S` and `-w` need a value; the traditional offset operand and `--traditional` are not implemented |
| `findx` | `-fstype`, `-context`, `-used`, `-files0-from`, `-D`, `%F` and `%Z` in `-printf` (each refused by name) |
| `grepx` | `-P` (refused with a message), `-u` |
| `sedx` | `--debug`; the `e` command and the `e` flag of `s` (refused when the script is parsed) |
| `xargsx` | `-o`, `--open-tty`, `--process-slot-var` (each refused by name) |
| `shufx` | The order drawn from a given `--random-source` differs from GNU's, though it is just as reproducible |
| `splitx` | `--filter` is not implemented; `-u` has no effect. Two `-N` options in a row run together |
| `statx` | The device is shown as `major,minor`, as GNU 9 shows it. Missing data is shown honestly: `-` for an unknown birth time, `?` for owners on Windows. `%C` (SELinux) is not supported; `--cached` has no effect. "cannot stat", not Linux's "cannot statx", on every platform |
| `tacx` | `-r` uses Go's RE2 syntax rather than GNU's regex dialect |
| `testx` | `[` is reached as `gnux [ ... ]`, a link named `[` or in `gnux shell`, not as a tool of its own; off Unix `-O` and `-G` are false and `-r`, `-w`, `-x` read the permission bits |
| `lognamex` | Without cgo there is no `getlogin`: on macOS and the BSDs a process with no terminal on standard input has no login name |
| `timeoutx` | On Windows, which has no signals, the limit still works: TERM and KILL both end COMMAND |
| `ttyx` | Finds the name without the C library; on Windows a console prints as `CON` |
| `unamex` | Prints the same fields on every platform; on Windows the kernel name is `Windows_NT` and the operating system `MS/Windows` |
| `uptimex` | Windows keeps no load average, so that part is left out there |
| `whox` | Reads Linux utmp, macOS utmpx and FreeBSD utx.active without the C library. On Windows it lists the signed-in console and Remote Desktop sessions; FILE is not supported there |
| `xargsx` | A command named after a gnux tool runs in-process; anything else is a program on `PATH`. `-P 0` runs at most 128 commands at once. The default size limit is a fixed figure, not asked of the system. A NUL in the input outside `-0` mode is dropped with GNU's warning |
| `basencx` | Decodes and encodes with the same engine as `base64x` and `base32x`; the last encoding option wins, as in GNU |
| `dircolorsx` | The built-in database is GNU coreutils 9.7's. `--sh` and `--csh` also answer to `--bourne-shell` and `--c-shell`, as in GNU; the last of `-b` and `-c` wins |
| `hostidx` | Windows has no `gethostid`: the identifier is derived from the host's IPv4 address, as glibc does when `/etc/hostid` is missing |
| `mkfifox`, `mknodx` | `-Z` and `--context` are accepted and ignored with GNU's warning, as on a kernel without SELinux. Windows has no special files: every name fails with "Operation not supported" |
| `numfmtx` | Computes in the 64-bit-mantissa precision of x86-64's long double, so a plain number of more than 18 digits needs `--to` (GNU on arm64 Linux, with a 113-bit long double, allows more). The C locale: `--grouping` adds no separators |
| `pathchkx` | `-p` and `-P` also answer to `--basic` and `--extra`. On Windows the limits are NTFS's: 255 characters per component, 259 in all |
| `pinkyx` | The long names of its letter options (`--long`, `--omit-heading`, ...) are gnux's own. The shell in `-l` is read from `/etc/passwd`, so it is blank for users kept elsewhere (macOS directory services, LDAP) and on Windows |
| `prx` | The header's date is in the C locale's words |
| `ptxx` | Regular expressions are in GNU ptx's Emacs syntax, run by Go's RE2: back references are refused, and `\<` and `\>` behave as `\b` |
| `shredx` | The order of the fixed patterns differs from GNU's run for run, as GNU's own does. A bad block is reported and ends the file's shredding, where GNU skips the block. On Windows, a run's directory renames are not flushed one by one |
| `stdbufx` | A command named after a gnux tool runs in-process, unchanged: gnux tools already write each line as soon as their input runs dry. Any other command needs GNU's `libstdbuf.so`, found beside gnux or where coreutils packages install it, or stdbufx refuses (status 125), as GNU does when it is missing; Windows has none |
| `sttyx` | On macOS and the BSDs the `PENDIN` flag, which the kernel raises by itself, is left out of the check that every setting was taken. When the Linux kernel quietly ignores a setting (`evenp` on a pseudo-terminal), sttyx says it was unable to perform all requested operations where glibc's `tcsetattr` makes GNU say "Invalid argument". Windows has no termios: every setting is checked, then fails with "Operation not supported" |
| `syncx` | Off Linux, `-d` flushes as plain `fsync` and `-f` flushes every file system. On Windows a run with no FILE succeeds without flushing anything: there is no unprivileged call that flushes every volume |
| `basenamex`, `base32x`, `base64x`, `dirnamex`, `echox`, `falsex`, `foldx`, `fmtx`, `headx`, `installx`, `linkx`, `mktempx`, `pastex`, `printenvx`, `readlinkx`, `rmdirx`, `sleepx`, `trx`, `truex`, `truncatex`, `unlinkx`, `wcx`, `whoamix`, `yesx` | None known beyond what their reference pages state |

## Not implemented yet

These GNU options are refused with `unrecognized option` (or `invalid option` for a letter):

| Tool | Not implemented |
|------|-----------------|
| `cpx` | `--reflink`, `--sparse`, `--attributes-only`, `--copy-contents`, `--keep-directory-symlink`, `-Z`, `--context`, `--debug` |
| `envx` | `--block-signal`, `--default-signal`, `--ignore-signal`, `--list-signal-handling` |
| `lnx` | `-d`, `-F`, `--directory` |
| `lsx` | `-f`, `-Z`, `--context` |
| `mkdirx` | `-Z`, `--context` |
| `mvx` | `-Z`, `--context`, `--exchange`, `--debug` |
| `realpathx` | `-E`, `--canonicalize` (the default behavior anyway) |
| `rmx` | `--preserve-root=all` |
| `sortx` | `--debug` |
| `touchx` | `-f` (GNU accepts and ignores it) |
| `uniqx` | the obsolete `-N` and `+N` forms of `-f` and `-s` |
| `wcx` | `--debug` |
| `datex` | `--debug` |
| `splitx` | `--filter`, `--io-blksize` |
| `odx` | `--traditional` and the `[+]OFFSET[.][b]` operand |

If a script uses one, the fix is usually simple: replace `ls -f` with `lsx -aU`.

## Things that look different but are not

- **Diagnostics name the x tool.** `rmx: cannot remove 'x': No such file or directory` rather than `rm: ...`, even when invoked as `rm`. Everything after the prefix is GNU's wording.
- **`--version`** prints `rmx (gnux) VERSION`, in GNU's `NAME (PACKAGE) VERSION` form.
- **`--help`** is gnux's own text: a description, usage, aliases, examples and the flag list.
- **`-h` is free** on every tool, because `--help` has no short form: `lsx -h` and `touchx -h` mean what they mean in GNU.
- **Write-protected prompts** appear only when standard input is a terminal, as in GNU: piping into `rmx` or running it in CI never prompts unless `-i` or `-I` is given.

## Adopting gnux in a shell script

1. Call the x names (`rmx` for `rm`) where the script should run the same everywhere, or leave the names as they are and run the script with `gnux shell`, where both names work. The GNU tools on the machine are left untouched.
2. Check every option against the "not implemented" table.
3. If the script uses only pipelines, `&&`, `||`, `;`, redirection, variables, globs, `$(...)`, `if`, `for`, `while` and `case`, it can run under `gnux shell` and becomes portable to Windows. Check two things first: an unquoted `$(...)` is split into lines, not words, and a variable's value is never re-split. Functions, arithmetic (`$((...))`), `until`, here-documents, backquotes and `( )` grouping are not part of the [gnux shell language](portable-scripting.md), so those scripts stay with bash, calling the x tools.
4. Calls to other programs (`git`, `go`, `make`) go through `envx` inside `gnux shell`: `envx git status`. `xargsx` and `findx -exec` also run a program when the command they are given is not a gnux tool.

## What is not in gnux yet

gnux covers 108 tools today: all of coreutils but three, plus `grepx`, `sedx`, `findx` and `xargsx` from GNU grep, GNU sed and GNU findutils. `chroot`, `chcon` and `runcon` are out of scope by design: they change system-wide state that gnux leaves to the system's own tools.

## See also

- [Functional requirements](../../docs/requirements/functional-requirements.md), section 2: each tool's required behavior, deviations and missing options.
- [Rules](../../docs/rules/README.md): the conventions behind the deviations.
- [Troubleshooting](troubleshooting.md): what an `unrecognized option` or other error means.
