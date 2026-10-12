.. _base64x:

base64x
-------

Encode or decode data in base64.

Synopsis
~~~~~~~~


base64x encodes FILE in base64 (RFC 4648, with padding) and writes the result to standard output, or with -d decodes it. With no FILE, or when FILE is -, it reads standard input; at most one FILE is accepted. Encoded output is wrapped every 76 characters and ends with a newline; -w COLS wraps at COLS instead, and -w 0 writes one line with no newline at all. When decoding, newlines are ignored, and -i also ignores every other character outside the alphabet; any other stray character, or a truncated input, stops decoding with 'invalid input' and exit status 1, after the data decoded so far has been written.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. base64x --help shows the options and the first examples; gnux help base64x prints the full manual, and gnux help base64x OPTION explains one option.

SEE ALSO. base32x, odx, cksumx.

PERMISSIONS. Needs read permission on FILE and search permission on the directories leading to it. Nothing is ever modified.

USE CASES.
  - Embedding binary data such as images or keys in text formats like JSON or YAML.
  - Decoding base64 found in configuration, tokens or email attachments.
  - Passing binary data through channels that only carry text.
  - Producing single-line encodings for environment variables with -w 0.
  - Inspecting encoded payloads in a pipeline.

::

  base64x [OPTION]... [FILE]

Examples
~~~~~~~~

::

    # Encode a file
    base64x image.png

    # Encode a string
    printfx 'hello' | base64x

    # Decode a string
    echox aGVsbG8= | base64x -d

    # Decode a file back to binary
    base64x -d image.txt > image.png

    # Encode on a single line, without a newline
    base64x -w 0 key.bin

    # Wrap encoded lines at 64 characters
    base64x -w 64 cert.der

    # Decode, skipping characters outside the alphabet
    base64x -d -i messy.txt

    # Store an encoded file in a variable
    data=$(base64x -w 0 key.bin)

    # Round-trip a file and compare the checksums
    base64x file.bin | base64x -d > copy.bin && cksumx file.bin copy.bin

    # Check that input decodes cleanly
    base64x -d payload.txt > /dev/null && echox valid

    # Encode in the portable gnux shell
    gnux shell -c 'base64x -w 0 key.bin > key.txt'

Options
~~~~~~~

::

  -d, --decode           decode data
      --help             display this help and exit
  -i, --ignore-garbage   when decoding, ignore non-alphabet characters
      --version          output version information and exit
  -w, --wrap COLS        wrap encoded lines after COLS character (default 76); use 0 to disable line wrapping

