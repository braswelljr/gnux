.. _base32x:

base32x
-------

Encode or decode data in base32.

Synopsis
~~~~~~~~


base32x encodes FILE in base32 (RFC 4648, with padding) and writes the result to standard output, or with -d decodes it. With no FILE, or when FILE is -, it reads standard input; at most one FILE is accepted. Encoded output is wrapped every 76 characters and ends with a newline; -w COLS wraps at COLS instead, and -w 0 writes one line with no newline at all. When decoding, newlines are ignored, and -i also ignores every other character outside the alphabet; any other stray character, or a truncated input, stops decoding with 'invalid input' and exit status 1, after the data decoded so far has been written.

EXIT STATUS.
  0  success: every operand was processed
  1  failure: an operand could not be processed (diagnosed on standard error), or a command-line mistake

HELP. base32x --help shows the options and the first examples; gnux help base32x prints the full manual, and gnux help base32x OPTION explains one option.

SEE ALSO. base64x, odx, cksumx.

PERMISSIONS. Needs read permission on FILE and search permission on the directories leading to it. Nothing is ever modified.

USE CASES.
  - Embedding binary data such as images or keys in text formats like JSON or YAML.
  - Decoding base32 found in configuration, tokens or email attachments.
  - Passing binary data through channels that only carry text.
  - Producing single-line encodings for environment variables with -w 0.
  - Inspecting encoded payloads in a pipeline.

::

  base32x [OPTION]... [FILE]

Examples
~~~~~~~~

::

    # Put a small logo straight into an HTML or CSS data: URL
    echox "data:image/png;base32,$(base32x -w 0 logo.png)"

    # Encode a short string to see what it looks like
    printfx 'hello' | base32x

    # Decode a value you found in a config file or an API response
    echox NBSWY3DP | base32x -d

    # Store a binary key in a CI secret, which only takes one line of text
    base32x -w 0 deploy-key.bin > deploy-key.txt

    # Turn that secret back into the key file in a CI job
    printfx '%s' "$DEPLOY_KEY" | base32x -d > deploy-key.bin

    # Wrap at 64 characters, as PEM certificates are
    base32x -w 64 cert.der

    # Decode text pasted from an email, where line breaks and stray characters crept in
    base32x -d -i pasted.txt > attachment.pdf

    # Send a binary file through a chat or a terminal that only carries text
    base32x report.pdf > report.txt

    # Make sure a round trip gives back the same bytes
    base32x file.bin | base32x -d > copy.bin && cksumx file.bin copy.bin

    # Reject a payload that is not valid base32 before using it
    base32x -d payload.txt > /dev/null && echox valid

    # Encode in the portable gnux shell, the same on every OS
    gnux shell -c 'base32x -w 0 key.bin > key.txt'

Options
~~~~~~~

::

  -d, --decode           decode data
      --help             display this help and exit
  -i, --ignore-garbage   when decoding, ignore non-alphabet characters
      --version          output version information and exit
  -w, --wrap COLS        wrap encoded lines after COLS character (default 76); use 0 to disable line wrapping

