.. _unamex:

unamex
------

Print system information.

Synopsis
~~~~~~~~


unamex prints information about the machine and operating system. With no option it prints the kernel name, as -s does. The selected fields are printed on one line, separated by spaces, in a fixed order whatever the order of the options: kernel name (-s), network node host name (-n), kernel release (-r), kernel version (-v), machine hardware name (-m), processor type (-p), hardware platform (-i) and operating system (-o). -a prints them all, leaving out -p and -i when they are unknown, as GNU does. The values come from uname(2) on Linux, sysctl on macOS and FreeBSD, and the registry on Windows, so unamex prints the same fields on every platform: on Windows the kernel name is 'Windows_NT' and the operating system 'MS/Windows'; in the browser playground the fields describe the WebAssembly runtime.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. unamex --help shows the options and the first examples; gnux help unamex prints the full manual, and gnux help unamex OPTION explains one option.

SEE ALSO. hostnamex, nprocx, uptimex.

PERMISSIONS. Needs no permissions.

USE CASES.
  - Branching a portable script on the operating system or architecture.
  - Recording the platform in bug reports and build logs.
  - Picking the right download for the machine's architecture with -m.
  - Checking the kernel release before relying on a kernel feature.
  - Getting the same answer on Linux, macOS, FreeBSD and Windows.

::

  unamex [OPTION]...

Examples
~~~~~~~~

::

    # Print the kernel name
    unamex

    # Print everything
    unamex -a

    # Print the machine architecture
    unamex -m

    # Print the kernel release
    unamex -r

    # Print the operating system
    unamex -o

    # Print the kernel name and release
    unamex -sr

    # Print the host name
    unamex -n

    # Branch a script on the operating system
    case "$(unamex -s)" in Linux) echox linux ;; Darwin) echox macos ;; esac

    # Pick a download for this machine
    curl -LO "https://example.com/tool-$(unamex -s)-$(unamex -m).tar.gz"

    # Record the platform in a log
    unamex -a >> build.log

    # Print system information in the portable gnux shell
    gnux shell -c 'unamex -srm'

Options
~~~~~~~

::

  -a, --all                 print all information, in the following order, except omit -p and -i if unknown
  -i, --hardware-platform   print the hardware platform (non-portable)
      --help                display this help and exit
  -s, --kernel-name         print the kernel name
  -r, --kernel-release      print the kernel release
  -v, --kernel-version      print the kernel version
  -m, --machine             print the machine hardware name
  -n, --nodename            print the network node hostname
  -o, --operating-system    print the operating system
  -p, --processor           print the processor type (non-portable)
      --version             output version information and exit

