package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Pack writes dir to archive as a gzipped tar with dir's own name at the top
// (v1.2/...), and archive.sha256 beside it in sha256sum's format. The archive
// is the same for the same pages: entries are in name order, with a fixed time
// and fixed modes.
func Pack(dir, archive string) error {
	if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
		return err
	}
	f, err := os.Create(archive)
	if err != nil {
		return err
	}
	sum := sha256.New()
	gz := gzip.NewWriter(io.MultiWriter(f, sum))
	tw := tar.NewWriter(gz)
	base := filepath.Dir(filepath.Clean(dir))
	stamp := time.Unix(0, 0).UTC()
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if d.IsDir() {
			return tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: name + "/", Mode: 0o755, ModTime: stamp, Format: tar.FormatPAX})
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: name, Mode: 0o644, Size: int64(len(data)), ModTime: stamp, Format: tar.FormatPAX}); err != nil {
			return err
		}
		_, err = tw.Write(data)
		return err
	})
	if err := closeAll(walkErr, tw, gz, f); err != nil {
		return err
	}
	line := hex.EncodeToString(sum.Sum(nil)) + "  " + filepath.Base(archive) + "\n"
	return os.WriteFile(archive+".sha256", []byte(line), 0o644)
}

// closeAll closes each in turn and returns the first error, starting with err.
func closeAll(err error, closers ...io.Closer) error {
	for _, c := range closers {
		if cerr := c.Close(); err == nil {
			err = cerr
		}
	}
	return err
}
