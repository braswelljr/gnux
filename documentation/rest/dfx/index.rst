.. _dfx:

dfx
---

Report file system space usage.

Synopsis
~~~~~~~~


dfx shows, for each mounted file system, its size, the space used and available, the percentage used and where it is mounted. With FILE operands it shows only the file systems holding them. Sizes are in 1024-byte blocks by default; -h prints them like 23G (--si/-H in powers of 1000), -k and -m use KiB and MiB, and -B SIZE any unit. Use% is used/(used+available), rounded up, so space reserved for the superuser shows as used. -i reports inodes instead of blocks, -T adds the file system type, -P uses the POSIX format, and --output=FIELDS chooses the columns (source, fstype, itotal, iused, iavail, ipcent, size, used, avail, pcent, file, target). -t TYPE and -x TYPE select or exclude file system types, -l leaves out network file systems, and --total adds a grand total. Without -a, pseudo file systems (proc, sysfs, devfs, ...), file systems with no blocks and duplicate mounts of the same device are left out; -a shows them all, with '-' for any that cannot be examined. The block size defaults to DF_BLOCK_SIZE, BLOCK_SIZE or BLOCKSIZE when set. GNU df only runs where coreutils is installed; dfx reports the mounted file systems on Linux, macOS and FreeBSD, and on Windows lists every drive (C:\, D:\, network drives) with the same columns, quietly skipping a card reader or DVD drive with no medium. --sync and --no-sync are accepted and have no effect, and -v is ignored, as in GNU.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. dfx --help shows the options and the first examples; gnux help dfx prints the full manual, and gnux help dfx OPTION explains one option.

SEE ALSO. dux, statx, lsx.

PERMISSIONS. Needs no special permission; a mount point the user cannot examine is left out (or shown with '-' under -a).

USE CASES.
  - Checking how much free space is left before a large download, build or backup.
  - Finding which disk is filling up on a server.
  - Checking which file system a directory lives on, and its type.
  - Running out of inodes despite free space: checking with -i.
  - Alerting from a script when a disk passes a usage threshold.

::

  dfx [OPTION]... [FILE]...

Examples
~~~~~~~~

::

    # See at a glance how full every disk is
    dfx -h

    # Before a 20 GB download, check the free space where it will land
    dfx -h ~/Downloads

    # Show only real local disks, with their file system types
    dfx -hlT

    # A server says 'No space left on device' but df shows free space: check the inodes
    dfx -i /var

    # Alert from a cron job when the root disk passes 90%
    [ "$(dfx --output=pcent / | tailx -n 1 | trx -dc 0-9)" -gt 90 ] && echox 'root disk above 90%'

    # Print just the available space, in megabytes, for a script
    dfx -BM --output=avail /data | tailx -n 1

    # See which device and file system type a directory is on
    dfx --output=source,fstype,target /var/lib/docker

    # Total the space across all local disks
    dfx -hl --total

    # Check only the network shares
    dfx -h -t nfs -t cifs

    # Leave temporary file systems out of the report
    dfx -h -x tmpfs -x devtmpfs

    # Produce stable, parseable output with the POSIX format
    dfx -P | tailx -n +2

    # Show disk space in the portable gnux shell, the same on every OS
    gnux shell -c 'dfx -h . && dux -sh .'

Options
~~~~~~~

::

  -a, --all                     include pseudo, duplicate, inaccessible file systems
  -B, --block-size SIZE         scale sizes by SIZE before printing them; e.g., '-BM' prints sizes in units of 1,048,576 bytes
  -x, --exclude-type TYPE       limit listing to file systems not of type TYPE
      --help                    display this help and exit
  -h, --human-readable          print sizes in powers of 1024 (e.g., 1023M)
  -i, --inodes                  list inode information instead of block usage
  -k, --kilobytes               like --block-size=1K
  -l, --local                   limit listing to local file systems
      --no-sync                 do not invoke sync before getting usage info (default)
      --output FIELD_LIST[= ]   use the output format defined by FIELD_LIST, or print all fields if FIELD_LIST is omitted
  -P, --portability             use the POSIX output format
  -T, --print-type              print file system type
  -H, --si                      print sizes in powers of 1000 (e.g., 1.1G)
      --sync                    invoke sync before getting usage info (no effect)
  -v, --sysv                    (ignored)
      --total                   elide all entries insignificant to available space, and produce a grand total
  -t, --type TYPE               limit listing to file systems of type TYPE
      --version                 output version information and exit

