import { project } from '@/config/site';

/** One step of getting gnux onto a platform: what to do, and the code that does it. */
export type InstallStep = {
  title: string;
  /** Why the step is there, in a sentence. */
  detail: string;
  /** The code block's title: the shell it runs in, or the file it goes in. */
  shell: string;
  code: string;
};

/** A platform on the install page: its own section, reachable from the page index. */
export type InstallPlatform = {
  /** The section's anchor: /install#macos. */
  id: string;
  name: string;
  /** One line under the name: how gnux arrives there. */
  summary: string;
  steps: InstallStep[];
  /** Said under the steps when something about the platform needs knowing. */
  note?: string;
};

const linkUnix = `bin="$(dirname "$(command -v gnux)")"
for t in $(gnux --list); do ln -sf gnux "$bin/$t"; done`;

const linkWindows = `$bin = Split-Path (Get-Command gnux).Source
foreach ($t in gnux --list) {
    Remove-Item "$bin\\$t.exe" -ErrorAction SilentlyContinue
    New-Item -ItemType HardLink -Path "$bin\\$t.exe" -Target "$bin\\gnux.exe" | Out-Null
}`;

const checkUnix = `rmx --version
lsx -la`;

const linkStep = (code: string, shell: string): InstallStep => ({
  title: 'Link the tool names',
  detail:
    'gnux picks the tool from the name it was started under, so each name linked to the binary is a command of its own: rmx, cpx, lsx and the rest.',
  shell,
  code
});

/** Every way to install gnux, by platform, as the README gives them. */
export const installPlatforms: InstallPlatform[] = [
  {
    id: 'macos',
    name: 'macOS',
    summary: 'A Homebrew cask, with shell completion for bash, zsh and fish.',
    steps: [
      {
        title: 'Install',
        detail: 'The cask installs the gnux binary and its completions.',
        shell: 'Terminal',
        code: 'brew install --cask braswelljr/tap/gnux'
      },
      linkStep(linkUnix, 'Terminal'),
      {
        title: 'Check',
        detail:
          'Every tool answers to --version; the second line should list the current directory.',
        shell: 'Terminal',
        code: checkUnix
      }
    ],
    note: 'The package managers are published from the first release after v0.0.1. Until then, use Go or a release archive.'
  },
  {
    id: 'windows',
    name: 'Windows',
    summary: 'winget or Scoop, then hard links: no WSL, no admin rights, no extra disk space.',
    steps: [
      {
        title: 'Install',
        detail: 'Either package manager installs the native, statically linked gnux.exe.',
        shell: 'PowerShell',
        code: `winget install braswelljr.gnux
# or
scoop bucket add braswelljr https://github.com/braswelljr/scoop-bucket
scoop install braswelljr/gnux`
      },
      linkStep(linkWindows, 'PowerShell'),
      {
        title: 'Check',
        detail: 'Open a new PowerShell window so it sees the new names, then run a tool.',
        shell: 'PowerShell',
        code: `rmx --version
lsx -la`
      }
    ],
    note: 'Hard links need neither Developer Mode nor elevation. Copying gnux.exe to each name also works, at the cost of disk space; rerun either loop after an upgrade.'
  },
  {
    id: 'linux',
    name: 'Linux and FreeBSD',
    summary: 'A release archive, checked against its checksums, for amd64, arm64 or riscv64.',
    steps: [
      {
        title: 'Download and verify',
        detail:
          'Pick the archive for your platform from the releases page; the checksum file proves it arrived intact.',
        shell: 'Shell',
        code: `v=v0.1.0
curl -LO "https://github.com/braswelljr/gnux/releases/download/$v/gnux-$v-linux-amd64.tar.gz"
curl -LO "https://github.com/braswelljr/gnux/releases/download/$v/checksums.txt"
sha256sum --check --ignore-missing checksums.txt`
      },
      {
        title: 'Install',
        detail: 'One static binary: no dependencies to install with it.',
        shell: 'Shell',
        code: `tar -xzf "gnux-$v-linux-amd64.tar.gz"
sudo mv "gnux-$v-linux-amd64/gnux" /usr/local/bin/`
      },
      linkStep(linkUnix.replace('ln -sf', 'sudo ln -sf'), 'Shell'),
      {
        title: 'Check',
        detail: 'Every tool answers to --version.',
        shell: 'Shell',
        code: checkUnix
      }
    ],
    note: 'Homebrew works on Linux too: brew install --cask braswelljr/tap/gnux.'
  },
  {
    id: 'go',
    name: 'Any platform, with Go',
    summary: 'Builds the latest version from source with the Go toolchain, 1.25 or newer.',
    steps: [
      {
        title: 'Install',
        detail: 'The binary lands in $(go env GOPATH)/bin, which should be on your PATH.',
        shell: 'Shell',
        code: project.install
      },
      linkStep(linkUnix, 'Shell'),
      {
        title: 'Check',
        detail: 'Every tool answers to --version.',
        shell: 'Shell',
        code: checkUnix
      }
    ]
  },
  {
    id: 'github-actions',
    name: 'GitHub Actions',
    summary: 'The setup-gnux action: the same tools on Linux, macOS and Windows runners.',
    steps: [
      {
        title: 'Add the action',
        detail:
          'It installs a verified release and puts gnux and every tool name on PATH, so later steps just use them.',
        shell: '.github/workflows/ci.yml',
        code: `- uses: braswelljr/gnux@v0
  with:
    version: latest
- run: rmx -rf build && mkdirx -p build && cpx -r assets build/`
      }
    ],
    note: 'Set gnu-names: true to also link rm, cp, ls and the rest ahead of the runner’s own tools.'
  },
  {
    id: 'docker',
    name: 'Docker',
    summary: 'Build the image from a checkout and mount the directory to work on.',
    steps: [
      {
        title: 'Build',
        detail: 'The image carries gnux and every tool name.',
        shell: 'Shell',
        code: `git clone https://github.com/braswelljr/gnux && cd gnux
docker build -t gnux .`
      },
      {
        title: 'Run',
        detail: 'Mount the current directory at /work, where the tools start.',
        shell: 'Shell',
        code: `docker run --rm -v "$(pwd):/work" gnux lsx -la
docker run --rm -v "$(pwd):/work" --entrypoint rmx gnux -rv build/`
      }
    ]
  }
];
