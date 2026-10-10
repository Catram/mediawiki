package main

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tarball writes a gzipped tarball of entries to a temporary file.
func tarball(t *testing.T, entries ...*tar.Header) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "test.tar.gz")
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for _, h := range entries {
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Size > 0 {
			if _, err := tw.Write([]byte(strings.Repeat("x", int(h.Size)))); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, c := range []interface{ Close() error }{tw, gz, f} {
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return file
}

func TestExtract(t *testing.T) {
	file := tarball(t,
		&tar.Header{Typeflag: tar.TypeDir, Name: "Ext/", Mode: 0o777},
		&tar.Header{Typeflag: tar.TypeReg, Name: "Ext/a.php", Mode: 0o666, Size: 3},
		&tar.Header{Typeflag: tar.TypeReg, Name: "Ext/bin/run", Mode: 0o4775, Size: 1},
		&tar.Header{Typeflag: tar.TypeSymlink, Name: "Ext/b.php", Linkname: "a.php"},
		&tar.Header{Typeflag: tar.TypeLink, Name: "Ext/c.php", Linkname: "Ext/a.php"},
	)
	dir := t.TempDir()
	if err := extract(file, dir); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]os.FileMode{
		"Ext/a.php":   0o644,
		"Ext/bin/run": 0o755,
		"Ext/c.php":   0o644,
	} {
		fi, err := os.Lstat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode() != want {
			t.Errorf("%s: mode %v, want %v", name, fi.Mode(), want)
		}
	}
	if l, err := os.Readlink(filepath.Join(dir, "Ext/b.php")); err != nil || l != "a.php" {
		t.Errorf("Ext/b.php: link %q, %v", l, err)
	}
}

func TestExtractRefusesEscapes(t *testing.T) {
	for name, h := range map[string]*tar.Header{
		"dot-dot":       {Typeflag: tar.TypeReg, Name: "../evil", Mode: 0o644},
		"absolute":      {Typeflag: tar.TypeReg, Name: "/evil", Mode: 0o644},
		"symlink out":   {Typeflag: tar.TypeSymlink, Name: "Ext/l", Linkname: "../../evil"},
		"symlink abs":   {Typeflag: tar.TypeSymlink, Name: "Ext/l", Linkname: "/etc/passwd"},
		"hard link out": {Typeflag: tar.TypeLink, Name: "Ext/l", Linkname: "../evil"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := extract(tarball(t, h), t.TempDir()); err == nil {
				t.Error("extracted")
			}
		})
	}
}

func TestExtractThroughSymlink(t *testing.T) {
	file := tarball(t,
		&tar.Header{Typeflag: tar.TypeSymlink, Name: "Ext/l", Linkname: "."},
		&tar.Header{Typeflag: tar.TypeReg, Name: "Ext/l/f", Mode: 0o644, Size: 1},
	)
	dir := t.TempDir()
	if err := extract(file, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Ext/f")); err != nil {
		t.Error(err)
	}
}
