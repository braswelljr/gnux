package visual

import (
	"path"
	"strings"
)

// Kind is what a file is, for picking its icon.
type Kind int

const (
	// KindFile is a regular file.
	KindFile Kind = iota
	// KindDir is a directory.
	KindDir
	// KindLink is a symbolic link to anything but a directory.
	KindLink
	// KindLinkDir is a symbolic link to a directory.
	KindLinkDir
	// KindExec is a regular file with an execute bit set.
	KindExec
	// KindDevice is a block or character device.
	KindDevice
	// KindPipe is a FIFO.
	KindPipe
	// KindSocket is a socket.
	KindSocket
)

// The icons are Nerd Font glyphs (https://www.nerdfonts.com), which need a
// patched font in the terminal; IconWidth is what each takes with its
// trailing space.
const (
	iconFile    = ""
	iconDir     = ""
	iconLink    = ""
	iconLinkDir = ""
	iconExec    = ""
	iconDevice  = ""
	iconPipe    = "\U000f07e5"
	iconSocket  = ""

	iconArchive = ""
	iconAudio   = ""
	iconBook    = ""
	iconC       = ""
	iconConfig  = ""
	iconCpp     = ""
	iconCSS     = ""
	iconDB      = ""
	iconDocker  = ""
	iconFont    = ""
	iconGit     = ""
	iconGo      = ""
	iconHTML    = ""
	iconImage   = ""
	iconJava    = ""
	iconJS      = ""
	iconJSON    = ""
	iconKey     = ""
	iconLicense = ""
	iconLock    = ""
	iconLua     = ""
	iconMake    = ""
	iconMD      = ""
	iconNix     = ""
	iconPDF     = ""
	iconPHP     = ""
	iconPython  = ""
	iconReact   = ""
	iconRuby    = ""
	iconRust    = ""
	iconShell   = ""
	iconSheet   = ""
	iconSwift   = ""
	iconText    = ""
	iconTS      = ""
	iconVideo   = ""
	iconVim     = ""
	iconWindows = ""
	iconWord    = ""
	iconSlides  = ""
	iconKotlin  = ""
	iconYAML    = ""
	iconTOML    = ""
	iconSQL     = ""
	iconZig     = ""

	iconDirGit     = ""
	iconDirGitHub  = ""
	iconDirConfig  = ""
	iconDirNPM     = ""
	iconDirTrash   = ""
	iconDirHome    = ""
	iconDirDown    = ""
	iconDirMusic   = ""
	iconDirPicture = ""
	iconDirVideo   = ""
	iconDirDocs    = ""
)

// IconWidth is the number of columns an icon and its trailing space take.
const IconWidth = 2

// byName holds the icons of files known by their whole name, lower-cased.
var byName = map[string]string{
	".gitignore": iconGit, ".gitattributes": iconGit, ".gitmodules": iconGit, ".gitconfig": iconGit,
	".bashrc": iconShell, ".zshrc": iconShell, ".profile": iconShell, ".bash_profile": iconShell,
	".vimrc": iconVim, ".editorconfig": iconConfig, ".env": iconConfig, ".npmrc": iconConfig,
	"dockerfile": iconDocker, "containerfile": iconDocker, "docker-compose.yml": iconDocker,
	"compose.yaml": iconDocker, "compose.yml": iconDocker,
	"makefile": iconMake, "gnumakefile": iconMake, "cmakelists.txt": iconMake, "justfile": iconMake,
	"license": iconLicense, "copying": iconLicense, "license.md": iconLicense,
	"go.mod": iconGo, "go.sum": iconGo, "go.work": iconGo,
	"cargo.toml": iconRust, "cargo.lock": iconLock, "package-lock.json": iconLock,
	"pnpm-lock.yaml": iconLock, "yarn.lock": iconLock, "flake.lock": iconLock,
	"flake.nix": iconNix, "gemfile": iconRuby, "rakefile": iconRuby,
}

// byExt holds the icons of files known by their extension, lower-cased and
// without the dot.
var byExt = map[string]string{
	"go": iconGo, "rs": iconRust, "py": iconPython, "pyc": iconPython, "rb": iconRuby,
	"js": iconJS, "mjs": iconJS, "cjs": iconJS, "ts": iconTS, "mts": iconTS, "cts": iconTS,
	"jsx": iconReact, "tsx": iconReact, "java": iconJava, "jar": iconJava, "kt": iconKotlin,
	"c": iconC, "h": iconC, "cc": iconCpp, "cpp": iconCpp, "cxx": iconCpp, "hpp": iconCpp,
	"swift": iconSwift, "lua": iconLua, "php": iconPHP, "zig": iconZig, "nix": iconNix,
	"html": iconHTML, "htm": iconHTML, "css": iconCSS, "scss": iconCSS, "sass": iconCSS,
	"json": iconJSON, "jsonc": iconJSON, "yml": iconYAML, "yaml": iconYAML, "toml": iconTOML,
	"ini": iconConfig, "conf": iconConfig, "cfg": iconConfig, "xml": iconConfig,
	"md": iconMD, "mdx": iconMD, "markdown": iconMD, "rst": iconText, "txt": iconText,
	"log": iconText, "csv": iconSheet, "tsv": iconSheet, "xls": iconSheet, "xlsx": iconSheet,
	"doc": iconWord, "docx": iconWord, "ppt": iconSlides, "pptx": iconSlides, "pdf": iconPDF,
	"epub": iconBook, "sh": iconShell, "bash": iconShell, "zsh": iconShell, "fish": iconShell,
	"ps1": iconShell, "bat": iconWindows, "cmd": iconWindows, "exe": iconWindows, "dll": iconWindows,
	"vim": iconVim, "sql": iconSQL, "db": iconDB, "sqlite": iconDB, "sqlite3": iconDB,
	"lock": iconLock, "key": iconKey, "pem": iconKey, "crt": iconKey, "pub": iconKey, "asc": iconKey,
	"zip": iconArchive, "tar": iconArchive, "gz": iconArchive, "tgz": iconArchive, "bz2": iconArchive,
	"xz": iconArchive, "zst": iconArchive, "7z": iconArchive, "rar": iconArchive, "deb": iconArchive,
	"rpm": iconArchive, "apk": iconArchive, "dmg": iconArchive, "iso": iconArchive,
	"png": iconImage, "jpg": iconImage, "jpeg": iconImage, "gif": iconImage, "svg": iconImage,
	"webp": iconImage, "ico": iconImage, "bmp": iconImage, "tiff": iconImage, "heic": iconImage,
	"mp3": iconAudio, "flac": iconAudio, "wav": iconAudio, "ogg": iconAudio, "m4a": iconAudio,
	"mp4": iconVideo, "mkv": iconVideo, "mov": iconVideo, "webm": iconVideo, "avi": iconVideo,
	"ttf": iconFont, "otf": iconFont, "woff": iconFont, "woff2": iconFont,
	"wasm": iconConfig, "mod": iconGo,
}

// byDir holds the icons of directories known by their name, lower-cased.
var byDir = map[string]string{
	".git": iconDirGit, ".github": iconDirGitHub, ".config": iconDirConfig, "node_modules": iconDirNPM,
	".trash": iconDirTrash, "trash": iconDirTrash, "home": iconDirHome, "downloads": iconDirDown,
	"music": iconDirMusic, "pictures": iconDirPicture, "videos": iconDirVideo, "movies": iconDirVideo,
	"documents": iconDirDocs, "docs": iconDirDocs,
}

// Icon returns the glyph for a file called name (a base name or a path) of
// the given kind, without the trailing space.
func Icon(name string, kind Kind) string {
	base := strings.ToLower(path.Base(strings.ReplaceAll(name, "\\", "/")))
	switch kind {
	case KindDir:
		if icon, ok := byDir[base]; ok {
			return icon
		}
		return iconDir
	case KindLinkDir:
		return iconLinkDir
	case KindLink:
		return iconLink
	case KindDevice:
		return iconDevice
	case KindPipe:
		return iconPipe
	case KindSocket:
		return iconSocket
	}
	if icon, ok := byName[base]; ok {
		return icon
	}
	if i := strings.LastIndexByte(base, '.'); i >= 0 {
		if icon, ok := byExt[base[i+1:]]; ok {
			return icon
		}
	}
	if kind == KindExec {
		return iconExec
	}
	if strings.HasPrefix(base, "readme") {
		return iconMD
	}
	return iconFile
}
