package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// extract unpacks the gzipped tarball file into dir, which must exist.
// Entries that would land outside dir, and links that point outside it, are
// refused. What is written belongs to whoever runs this, root in the image
// build, whatever owners the tarball records. Files get mode 0755 if the
// tarball marks them executable and 0644 otherwise, and keep their times.
func extract(file, dir string) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		name := path.Clean(strings.TrimPrefix(hdr.Name, "./"))
		if !filepath.IsLocal(name) {
			return fmt.Errorf("%s: unsafe path %q", filepath.Base(file), hdr.Name)
		}
		if name == "." {
			continue
		}
		if err := extractEntry(root, tr, hdr, name); err != nil {
			return fmt.Errorf("%s: %s: %w", filepath.Base(file), hdr.Name, err)
		}
	}
}

func extractEntry(root *os.Root, tr *tar.Reader, hdr *tar.Header, name string) error {
	if dir := path.Dir(name); dir != "." {
		if err := root.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	switch hdr.Typeflag {
	case tar.TypeDir:
		return root.MkdirAll(name, 0o755)
	case tar.TypeReg:
		mode := os.FileMode(0o644)
		if hdr.Mode&0o111 != 0 {
			mode = 0o755
		}
		f, err := root.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		if err := root.Chmod(name, mode); err != nil {
			return err
		}
		return root.Chtimes(name, hdr.ModTime, hdr.ModTime)
	case tar.TypeSymlink:
		if path.IsAbs(hdr.Linkname) || !filepath.IsLocal(path.Join(path.Dir(name), hdr.Linkname)) {
			return fmt.Errorf("link to %q points outside the tarball", hdr.Linkname)
		}
		return root.Symlink(hdr.Linkname, name)
	case tar.TypeLink:
		target := path.Clean(hdr.Linkname)
		if !filepath.IsLocal(target) {
			return fmt.Errorf("link to %q points outside the tarball", hdr.Linkname)
		}
		return root.Link(target, name)
	default:
		return fmt.Errorf("unsupported entry type %q", hdr.Typeflag)
	}
}
