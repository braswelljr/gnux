package main

import "testing"

const sample = `# Changelog

Intro.

## [Unreleased]

### Added

- Something new.

## [1.2.0] - 2026-11-02

### Fixed

- A bug.

## [1.1.0] - 2026-10-20

- Earlier.

## [1.0.0]

[Unreleased]: https://github.com/braswelljr/gnux/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/braswelljr/gnux/compare/v1.1.0...v1.2.0
`

func TestSection(t *testing.T) {
	tests := []struct {
		version, want string
		ok            bool
	}{
		{"v1.2.0", "### Fixed\n\n- A bug.\n", true},
		{"1.2.0", "### Fixed\n\n- A bug.\n", true},
		{"v1.1.0", "- Earlier.\n", true},
		{"Unreleased", "### Added\n\n- Something new.\n", true},
		// A heading with nothing under it is not release notes.
		{"v1.0.0", "", false},
		{"v9.9.9", "", false},
		// 1.2 must not match the 1.2.0 heading.
		{"v1.2", "", false},
	}
	for _, tt := range tests {
		got, ok := Section(sample, tt.version)
		if got != tt.want || ok != tt.ok {
			t.Errorf("Section(%q) = %q, %v; want %q, %v", tt.version, got, ok, tt.want, tt.ok)
		}
	}
}
